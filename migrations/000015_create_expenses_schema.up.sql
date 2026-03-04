-- Migration: Create Expenses Schema
-- Introduces public.expense_categories, public.expenses, and bonds them 
-- as money-out vectors inside public.transactions.

-- ==============================================================================
-- PART A: EXPENSE CATEGORIES
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.expense_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    name TEXT NOT NULL,
    code TEXT NULL,
    description TEXT NULL,
    
    category_type TEXT NOT NULL DEFAULT 'opex',
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_expense_cat_type CHECK (category_type IN ('opex', 'cogs', 'other'))
);

-- Unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_expense_cat_name 
    ON public.expense_categories (tenant_id, lower(name));

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_expense_cat_code 
    ON public.expense_categories (tenant_id, code) 
    WHERE code IS NOT NULL;

-- Indexing for SaaS performance (expense categories)
CREATE INDEX IF NOT EXISTS expense_categories_tenant_idx ON public.expense_categories(tenant_id);
CREATE INDEX IF NOT EXISTS expense_categories_tenant_type_idx ON public.expense_categories(tenant_id, category_type);
CREATE INDEX IF NOT EXISTS expense_categories_tenant_name_idx ON public.expense_categories(tenant_id, lower(name));

-- Trigger for update_at
DROP TRIGGER IF EXISTS set_expense_cat_updated_at ON public.expense_categories;
CREATE TRIGGER set_expense_cat_updated_at
    BEFORE UPDATE ON public.expense_categories
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for expense_categories
ALTER TABLE public.expense_categories ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view categories in tenant" ON public.expense_categories;
CREATE POLICY "Members can view categories in tenant"
    ON public.expense_categories FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and admin can manage categories" ON public.expense_categories;
CREATE POLICY "Owner and admin can manage categories"
    ON public.expense_categories FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner','admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner','admin']));

DROP POLICY IF EXISTS "Service role has full access to expense categories" ON public.expense_categories;
CREATE POLICY "Service role has full access to expense categories"
    ON public.expense_categories FOR ALL TO service_role USING (true) WITH CHECK (true);


-- ==============================================================================
-- PART B: EXPENSES
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NULL REFERENCES public.locations(id) ON DELETE SET NULL,
    category_id UUID NOT NULL REFERENCES public.expense_categories(id) ON DELETE RESTRICT,
    supplier_contact_id UUID NULL REFERENCES public.contacts(id) ON DELETE SET NULL,
    
    -- Document info
    reference TEXT NULL,
    description TEXT NOT NULL,
    
    -- Dates
    expense_date DATE NOT NULL DEFAULT current_date,
    due_date DATE NULL,
    
    -- Status
    status TEXT NOT NULL DEFAULT 'draft',
    
    -- Money
    total NUMERIC(12,2) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'COP',
    
    -- Audit
    created_by UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_expense_total CHECK (total > 0),
    CONSTRAINT chk_expense_status CHECK (status IN ('draft', 'approved', 'paid', 'cancelled')),
    CONSTRAINT chk_expense_dates CHECK (due_date IS NULL OR due_date >= expense_date)
);

-- Indexing for SaaS performance (expenses)
CREATE INDEX IF NOT EXISTS expenses_tenant_date_idx ON public.expenses(tenant_id, expense_date DESC);
CREATE INDEX IF NOT EXISTS expenses_tenant_status_date_idx ON public.expenses(tenant_id, status, expense_date DESC);
CREATE INDEX IF NOT EXISTS expenses_tenant_category_date_idx ON public.expenses(tenant_id, category_id, expense_date DESC);
CREATE INDEX IF NOT EXISTS expenses_tenant_supplier_date_idx ON public.expenses(tenant_id, supplier_contact_id, expense_date DESC);

-- Trigger for update_at
DROP TRIGGER IF EXISTS set_expenses_updated_at ON public.expenses;
CREATE TRIGGER set_expenses_updated_at
    BEFORE UPDATE ON public.expenses
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for expenses
ALTER TABLE public.expenses ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view expenses in tenant" ON public.expenses;
CREATE POLICY "Members can view expenses in tenant"
    ON public.expenses FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and admin can manage expenses" ON public.expenses;
CREATE POLICY "Owner and admin can manage expenses"
    ON public.expenses FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner','admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner','admin']));

DROP POLICY IF EXISTS "Service role has full access to expenses" ON public.expenses;
CREATE POLICY "Service role has full access to expenses"
    ON public.expenses FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Consistency Guard
CREATE OR REPLACE FUNCTION public.check_expense_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_cat_tenant UUID;
    v_loc_tenant UUID;
    v_con_tenant UUID;
BEGIN
    -- Check Expense Category
    SELECT tenant_id INTO v_cat_tenant FROM public.expense_categories WHERE id = NEW.category_id;
    IF v_cat_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Category does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;

    -- Check Location
    IF NEW.location_id IS NOT NULL THEN
        SELECT tenant_id INTO v_loc_tenant FROM public.locations WHERE id = NEW.location_id;
        IF v_loc_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Location does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;

    -- Check Supplier Contact
    IF NEW.supplier_contact_id IS NOT NULL THEN
        SELECT tenant_id INTO v_con_tenant FROM public.contacts WHERE id = NEW.supplier_contact_id;
        IF v_con_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Supplier contact does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_expense_consistency ON public.expenses;
CREATE TRIGGER validate_expense_consistency
    BEFORE INSERT OR UPDATE ON public.expenses
    FOR EACH ROW
    EXECUTE FUNCTION public.check_expense_tenant_consistency();

-- ==============================================================================
-- PART C: BIND EXPENSES TO TRANSACTIONS (MONEY OUT)
-- ==============================================================================

-- 1. Alter public.transactions
ALTER TABLE public.transactions 
ADD COLUMN IF NOT EXISTS expense_id UUID NULL REFERENCES public.expenses(id) ON DELETE SET NULL;

-- 2. Update exclusive "exactly one target" constraint
ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS chk_transactions_target;
ALTER TABLE public.transactions ADD CONSTRAINT chk_transactions_target CHECK (
    ( (appointment_id IS NOT NULL)::int + 
      (sales_order_id IS NOT NULL)::int + 
      (purchase_order_id IS NOT NULL)::int +
      (payroll_run_id IS NOT NULL)::int +
      (expense_id IS NOT NULL)::int ) = 1
);

-- 3. Update Existing Transactions Consistency Check (Polymorphic Guard)
CREATE OR REPLACE FUNCTION public.check_transaction_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_account_tenant UUID;
    v_sales_tenant UUID;
    v_appt_tenant UUID;
    v_contact_tenant UUID;
    v_location_tenant UUID;
    v_staff_tenant UUID;
    v_payroll_tenant UUID;
    v_expense_tenant UUID;
BEGIN
    IF NEW.account_id IS NOT NULL THEN
        SELECT tenant_id INTO v_account_tenant FROM public.accounts WHERE id = NEW.account_id;
        IF v_account_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Account does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    
    IF NEW.sales_order_id IS NOT NULL THEN
        SELECT tenant_id INTO v_sales_tenant FROM public.sales_orders WHERE id = NEW.sales_order_id;
        IF v_sales_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Sales order does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    
    IF NEW.appointment_id IS NOT NULL THEN
        SELECT tenant_id INTO v_appt_tenant FROM public.appointments WHERE id = NEW.appointment_id;
        IF v_appt_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Appointment does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    
    IF NEW.payroll_run_id IS NOT NULL THEN
        SELECT tenant_id INTO v_payroll_tenant FROM public.payroll_runs WHERE id = NEW.payroll_run_id;
        IF v_payroll_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Payroll run does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    
    -- NEW: Expense Tenant Check
    IF NEW.expense_id IS NOT NULL THEN
        SELECT tenant_id INTO v_expense_tenant FROM public.expenses WHERE id = NEW.expense_id;
        IF v_expense_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Expense does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    
    IF NEW.contact_id IS NOT NULL THEN
        SELECT tenant_id INTO v_contact_tenant FROM public.contacts WHERE id = NEW.contact_id;
        IF v_contact_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Contact does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    
    IF NEW.location_id IS NOT NULL THEN
        SELECT tenant_id INTO v_location_tenant FROM public.locations WHERE id = NEW.location_id;
        IF v_location_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Location does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    
    IF NEW.staff_id IS NOT NULL THEN
        SELECT tenant_id INTO v_staff_tenant FROM public.staff WHERE id = NEW.staff_id;
        IF v_staff_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Staff member does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    
    RETURN NEW;
END;
$$;

-- 4. NEW: Guard forcing expenses to ONLY use OUT money transactions
CREATE OR REPLACE FUNCTION public.guard_transaction_expense_direction()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    IF NEW.expense_id IS NOT NULL AND NEW.direction != 'out' THEN
        RAISE EXCEPTION 'Transactions linked to expenses MUST be of direction OUT' USING ERRCODE = 'P0003';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_transaction_expense_direction ON public.transactions;
CREATE TRIGGER validate_transaction_expense_direction
    BEFORE INSERT OR UPDATE ON public.transactions
    FOR EACH ROW
    EXECUTE FUNCTION public.guard_transaction_expense_direction();

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
