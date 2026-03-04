-- Migration: Create Purchases Schema
-- Introduces public.purchase_orders, public.purchase_order_lines, auto-totals, and links 
-- to public.transactions for multi-payment tracking (installments/partial payments).

-- ==============================================================================
-- PART A: PURCHASE ORDERS (HEADERS)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES public.locations(id) ON DELETE RESTRICT,
    supplier_contact_id UUID NOT NULL REFERENCES public.contacts(id) ON DELETE RESTRICT,
    staff_id UUID NULL REFERENCES public.staff(id) ON DELETE SET NULL,
    
    order_number TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    
    -- Money Totals
    subtotal NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_total NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax_total NUMERIC(12,2) NOT NULL DEFAULT 0,
    total NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'COP',
    
    -- Dates
    ordered_at TIMESTAMPTZ NULL,
    received_at TIMESTAMPTZ NULL,
    
    -- Payment Tracking
    payment_status TEXT NOT NULL DEFAULT 'unpaid',
    paid_total NUMERIC(12,2) NOT NULL DEFAULT 0,
    balance_due NUMERIC(12,2) NOT NULL DEFAULT 0,
    due_date DATE NULL,
    
    -- Optional
    notes TEXT NULL,
    external_ref TEXT NULL,
    
    created_by UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_po_status CHECK (status IN ('draft', 'ordered', 'received', 'cancelled')),
    CONSTRAINT chk_po_payment_status CHECK (payment_status IN ('unpaid', 'partial', 'paid')),
    CONSTRAINT chk_po_money CHECK (subtotal >= 0 AND discount_total >= 0 AND tax_total >= 0 AND total >= 0),
    CONSTRAINT chk_po_payment_money CHECK (paid_total >= 0 AND balance_due >= 0),
    CONSTRAINT chk_po_due_date CHECK (due_date IS NULL OR due_date >= (created_at::DATE)),
    CONSTRAINT uq_tenant_po_number UNIQUE (tenant_id, order_number)
);

CREATE INDEX IF NOT EXISTS purchase_orders_tenant_id_idx ON public.purchase_orders(tenant_id);
CREATE INDEX IF NOT EXISTS purchase_orders_tenant_loc_created_idx ON public.purchase_orders(tenant_id, location_id, created_at DESC);
CREATE INDEX IF NOT EXISTS purchase_orders_tenant_status_created_idx ON public.purchase_orders(tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS purchase_orders_tenant_supplier_created_idx ON public.purchase_orders(tenant_id, supplier_contact_id, created_at DESC);
CREATE INDEX IF NOT EXISTS purchase_orders_tenant_payment_status_idx ON public.purchase_orders(tenant_id, payment_status, due_date);

DROP TRIGGER IF EXISTS set_po_updated_at ON public.purchase_orders;
CREATE TRIGGER set_po_updated_at
    BEFORE UPDATE ON public.purchase_orders
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.purchase_orders ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view purchase orders in their tenant" ON public.purchase_orders;
CREATE POLICY "Members can view purchase orders in their tenant"
    ON public.purchase_orders FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner, Admin, and Staff can manage purchase orders" ON public.purchase_orders;
CREATE POLICY "Owner, Admin, and Staff can manage purchase orders"
    ON public.purchase_orders FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

DROP POLICY IF EXISTS "Service role has full access to purchase orders" ON public.purchase_orders;
CREATE POLICY "Service role has full access to purchase orders"
    ON public.purchase_orders FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Consistency Guard
CREATE OR REPLACE FUNCTION public.check_purchase_order_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_loc_tenant UUID;
    v_con_tenant UUID;
    v_con_type TEXT;
    v_staff_tenant UUID;
BEGIN
    SELECT tenant_id INTO v_loc_tenant FROM public.locations WHERE id = NEW.location_id;
    IF v_loc_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Location does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;

    SELECT tenant_id, contact_type INTO v_con_tenant, v_con_type FROM public.contacts WHERE id = NEW.supplier_contact_id;
    IF v_con_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Contact does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    IF v_con_type NOT IN ('supplier', 'both') THEN RAISE EXCEPTION 'Contact is not classified as a supplier' USING ERRCODE = 'P0004'; END IF;

    IF NEW.staff_id IS NOT NULL THEN
        SELECT tenant_id INTO v_staff_tenant FROM public.staff WHERE id = NEW.staff_id;
        IF v_staff_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Staff does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_po_consistency ON public.purchase_orders;
CREATE TRIGGER validate_po_consistency
    BEFORE INSERT OR UPDATE ON public.purchase_orders
    FOR EACH ROW
    EXECUTE FUNCTION public.check_purchase_order_tenant_consistency();

-- ==============================================================================
-- PART B: PURCHASE ORDER LINES (DETAILS)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.purchase_order_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    purchase_order_id UUID NOT NULL REFERENCES public.purchase_orders(id) ON DELETE CASCADE,
    catalog_item_id UUID NOT NULL REFERENCES public.catalog_items(id) ON DELETE RESTRICT,
    
    quantity NUMERIC(12,2) NOT NULL DEFAULT 1,
    unit_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_total NUMERIC(12,2) NOT NULL DEFAULT 0,
    
    notes TEXT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_pol_quantity CHECK (quantity > 0),
    CONSTRAINT chk_pol_unit_cost CHECK (unit_cost >= 0),
    CONSTRAINT chk_pol_discount CHECK (discount >= 0),
    CONSTRAINT chk_pol_tax CHECK (tax >= 0),
    CONSTRAINT chk_pol_line_total CHECK (line_total >= 0),
    CONSTRAINT uq_po_catalog_item UNIQUE (purchase_order_id, catalog_item_id)
);

CREATE INDEX IF NOT EXISTS po_lines_order_idx ON public.purchase_order_lines(purchase_order_id);
CREATE INDEX IF NOT EXISTS po_lines_tenant_item_idx ON public.purchase_order_lines(tenant_id, catalog_item_id);

DROP TRIGGER IF EXISTS set_po_lines_updated_at ON public.purchase_order_lines;
CREATE TRIGGER set_po_lines_updated_at
    BEFORE UPDATE ON public.purchase_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.purchase_order_lines ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view purchase lines in their tenant" ON public.purchase_order_lines;
CREATE POLICY "Members can view purchase lines in their tenant"
    ON public.purchase_order_lines FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner, Admin, and Staff can manage purchase lines" ON public.purchase_order_lines;
CREATE POLICY "Owner, Admin, and Staff can manage purchase lines"
    ON public.purchase_order_lines FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

DROP POLICY IF EXISTS "Service role has full access to purchase lines" ON public.purchase_order_lines;
CREATE POLICY "Service role has full access to purchase lines"
    ON public.purchase_order_lines FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Consistency Guard
CREATE OR REPLACE FUNCTION public.check_po_line_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_po_tenant UUID;
    v_item_tenant UUID;
BEGIN
    SELECT tenant_id INTO v_po_tenant FROM public.purchase_orders WHERE id = NEW.purchase_order_id;
    IF v_po_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Purchase order does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;

    SELECT tenant_id INTO v_item_tenant FROM public.catalog_items WHERE id = NEW.catalog_item_id;
    IF v_item_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Catalog item does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_po_line_consistency ON public.purchase_order_lines;
CREATE TRIGGER validate_po_line_consistency
    BEFORE INSERT OR UPDATE ON public.purchase_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION public.check_po_line_tenant_consistency();

-- ==============================================================================
-- PART C: TOTALS MANAGEMENT (COMPUTED IN PARENT PO)
-- ==============================================================================

CREATE OR REPLACE FUNCTION public.recalculate_purchase_order_totals()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_po_id UUID;
    v_subtotal NUMERIC(12,2) := 0;
    v_discount NUMERIC(12,2) := 0;
    v_tax NUMERIC(12,2) := 0;
    v_total NUMERIC(12,2) := 0;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_po_id := OLD.purchase_order_id;
    ELSE
        v_po_id := NEW.purchase_order_id;
    END IF;

    SELECT 
        COALESCE(SUM(quantity * unit_cost), 0),
        COALESCE(SUM(discount), 0),
        COALESCE(SUM(tax), 0),
        COALESCE(SUM(line_total), 0)
    INTO v_subtotal, v_discount, v_tax, v_total
    FROM public.purchase_order_lines
    WHERE purchase_order_id = v_po_id;

    UPDATE public.purchase_orders
    SET subtotal = v_subtotal,
        discount_total = v_discount,
        tax_total = v_tax,
        total = v_total,
        balance_due = GREATEST(v_total - paid_total, 0)
    WHERE id = v_po_id;

    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trigger_recalculate_po_totals ON public.purchase_order_lines;
CREATE TRIGGER trigger_recalculate_po_totals
    AFTER INSERT OR UPDATE OR DELETE ON public.purchase_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION public.recalculate_purchase_order_totals();

-- ==============================================================================
-- PART D: BIND PURCHASES TO TRANSACTIONS (PARTIAL PAYMENTS)
-- ==============================================================================

-- If the target 'purchase_order_id' already physically existed from the earlier transaction schema,
-- this ALTER handles it seamlessly. If it didn't strictly map down a FK before this schema existed, it does now.
ALTER TABLE public.transactions 
DROP CONSTRAINT IF EXISTS transactions_purchase_order_id_fkey;

ALTER TABLE public.transactions 
ADD CONSTRAINT transactions_purchase_order_id_fkey 
FOREIGN KEY (purchase_order_id) REFERENCES public.purchase_orders(id) ON DELETE SET NULL;

-- Expand the Consistency Trigger to handle Purchase direction and validation
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
    v_purchase_tenant UUID;
    v_payroll_tenant UUID;
    v_expense_tenant UUID;
    v_contact_tenant UUID;
    v_location_tenant UUID;
    v_staff_tenant UUID;
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

    -- NEW: Purchase tenant check
    IF NEW.purchase_order_id IS NOT NULL THEN
        SELECT tenant_id INTO v_purchase_tenant FROM public.purchase_orders WHERE id = NEW.purchase_order_id;
        IF v_purchase_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Purchase order does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;

    IF NEW.payroll_run_id IS NOT NULL THEN
        SELECT tenant_id INTO v_payroll_tenant FROM public.payroll_runs WHERE id = NEW.payroll_run_id;
        IF v_payroll_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Payroll run does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;

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

-- Note: No need to DROP/CREATE the validate_transaction_consistency trigger since CREATE OR REPLACE FUNCTION gracefully swapped the logic.

-- Guard forcing Purchases to ONLY use OUT money transactions
CREATE OR REPLACE FUNCTION public.guard_transaction_purchase_direction()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    IF NEW.purchase_order_id IS NOT NULL AND NEW.direction != 'out' THEN
        RAISE EXCEPTION 'Transactions linked to purchases MUST be of direction OUT' USING ERRCODE = 'P0003';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_transaction_purchase_direction ON public.transactions;
CREATE TRIGGER validate_transaction_purchase_direction
    BEFORE INSERT OR UPDATE ON public.transactions
    FOR EACH ROW
    EXECUTE FUNCTION public.guard_transaction_purchase_direction();

-- ==============================================================================
-- PART E: DYNAMIC PURCHASE MULTI-PAYMENT SYNCHRONIZATION
-- ==============================================================================

CREATE OR REPLACE FUNCTION public.sync_purchase_order_payment_status()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_po_id UUID;
    v_total_paid NUMERIC(12,2) := 0;
    v_po_total NUMERIC(12,2) := 0;
BEGIN
    -- Determine the targeted Purchase Order ID based on the Operation
    IF TG_OP = 'DELETE' THEN
        v_po_id := OLD.purchase_order_id;
    ELSE
        v_po_id := NEW.purchase_order_id;
    END IF;

    -- If this transaction isn't linked to a Purchase Order, bail out explicitly.
    IF v_po_id IS NULL THEN
        RETURN NULL;
    END IF;

    -- Sum up all completed OUT money corresponding to this specific purchase order
    SELECT COALESCE(SUM(amount), 0) INTO v_total_paid
    FROM public.transactions
    WHERE purchase_order_id = v_po_id 
      AND status = 'completed'
      AND direction = 'out';

    -- Retrieve the original invoice total
    SELECT total INTO v_po_total FROM public.purchase_orders WHERE id = v_po_id;

    -- Update the core POS states via automated Ledger balancing mapping
    UPDATE public.purchase_orders
    SET 
        paid_total = v_total_paid,
        balance_due = GREATEST(v_po_total - v_total_paid, 0),
        payment_status = CASE 
            WHEN v_total_paid >= v_po_total AND v_po_total > 0 THEN 'paid'
            WHEN v_total_paid > 0 AND v_total_paid < v_po_total THEN 'partial'
            ELSE 'unpaid'
        END
    WHERE id = v_po_id;

    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trigger_sync_purchase_payment ON public.transactions;
CREATE TRIGGER trigger_sync_purchase_payment
    AFTER INSERT OR UPDATE OR DELETE ON public.transactions
    FOR EACH ROW
    EXECUTE FUNCTION public.sync_purchase_order_payment_status();

-- Guard against Manual Status Overrides (Enforce explicit payouts via transactions)
CREATE OR REPLACE FUNCTION public.guard_purchase_payment_status_override()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    -- Only block this if it's an isolated UPDATE that didn't trigger via our sync function naturally matching real payouts.
    IF NEW.payment_status = 'paid' AND (OLD.payment_status IS DISTINCT FROM 'paid') THEN
        IF NEW.paid_total < NEW.total OR NEW.total = 0 THEN
            RAISE EXCEPTION 'Cannot manually mark purchase order as paid without completed OUT transactions covering the full total' USING ERRCODE = 'P0005';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trigger_guard_purchase_payment_status ON public.purchase_orders;
CREATE TRIGGER trigger_guard_purchase_payment_status
    BEFORE UPDATE ON public.purchase_orders
    FOR EACH ROW
    EXECUTE FUNCTION public.guard_purchase_payment_status_override();

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
