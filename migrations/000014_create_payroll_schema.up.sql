-- Migration: Create Payroll Schema
-- Introduces dynamic compensation, payroll runs, items, and binds payments to unified transactions.

-- ==============================================================================
-- PART A: STAFF COMPENSATION CONFIGURATION
-- ==============================================================================

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS public.staff_compensation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES public.staff(id) ON DELETE CASCADE,
    
    scheme TEXT NOT NULL DEFAULT 'fixed',
    pay_frequency TEXT NOT NULL DEFAULT 'monthly',
    
    base_salary NUMERIC(12,2) NULL,
    hourly_rate NUMERIC(12,2) NULL,
    commission_pct NUMERIC(5,4) NULL,
    
    effective_from DATE NOT NULL DEFAULT current_date,
    effective_to DATE NULL,
    
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_comp_scheme CHECK (scheme IN ('fixed', 'hourly', 'commission', 'mixed')),
    CONSTRAINT chk_comp_frequency CHECK (pay_frequency IN ('daily', 'weekly', 'biweekly', 'monthly', 'custom')),
    CONSTRAINT chk_comp_base CHECK (base_salary IS NULL OR base_salary >= 0),
    CONSTRAINT chk_comp_hourly CHECK (hourly_rate IS NULL OR hourly_rate >= 0),
    CONSTRAINT chk_comp_comm CHECK (commission_pct IS NULL OR (commission_pct >= 0 AND commission_pct <= 1)),
    CONSTRAINT chk_comp_dates CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

-- Prevent overlapping effective periods per staff
ALTER TABLE public.staff_compensation ADD CONSTRAINT prevent_overlapping_compensation
EXCLUDE USING gist (
    tenant_id WITH =,
    staff_id WITH =,
    daterange(effective_from, COALESCE(effective_to, 'infinity'::date), '[)') WITH &&
);

CREATE INDEX IF NOT EXISTS staff_comp_tenant_staff_idx ON public.staff_compensation(tenant_id, staff_id);
CREATE INDEX IF NOT EXISTS staff_comp_tenant_active_idx ON public.staff_compensation(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS staff_comp_effective_from_idx ON public.staff_compensation(tenant_id, staff_id, effective_from DESC);

DROP TRIGGER IF EXISTS set_staff_comp_updated_at ON public.staff_compensation;
CREATE TRIGGER set_staff_comp_updated_at
    BEFORE UPDATE ON public.staff_compensation
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.staff_compensation ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view compensation in their tenant" ON public.staff_compensation;
CREATE POLICY "Members can view compensation in their tenant"
    ON public.staff_compensation FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and Admin can manage compensation" ON public.staff_compensation;
CREATE POLICY "Owner and Admin can manage compensation"
    ON public.staff_compensation FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

DROP POLICY IF EXISTS "Service role has full access to compensation" ON public.staff_compensation;
CREATE POLICY "Service role has full access to compensation"
    ON public.staff_compensation FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Consistency Guard
CREATE OR REPLACE FUNCTION public.check_staff_comp_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_staff_tenant UUID;
BEGIN
    SELECT tenant_id INTO v_staff_tenant FROM public.staff WHERE id = NEW.staff_id;
    IF v_staff_tenant != NEW.tenant_id THEN
        RAISE EXCEPTION 'Staff member does not belong to the same tenant' USING ERRCODE = 'P0002';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_staff_comp_consistency ON public.staff_compensation;
CREATE TRIGGER validate_staff_comp_consistency
    BEFORE INSERT OR UPDATE ON public.staff_compensation
    FOR EACH ROW
    EXECUTE FUNCTION public.check_staff_comp_tenant_consistency();

-- ==============================================================================
-- PART B: PAYROLL RUNS (PAY PERIODS)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.payroll_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NULL REFERENCES public.locations(id) ON DELETE SET NULL,
    
    frequency TEXT NOT NULL DEFAULT 'monthly',
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    pay_date DATE NOT NULL DEFAULT current_date,
    
    status TEXT NOT NULL DEFAULT 'draft',
    total NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'COP',
    notes TEXT NULL,
    
    created_by UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_pr_frequency CHECK (frequency IN ('daily', 'weekly', 'biweekly', 'monthly', 'custom')),
    CONSTRAINT chk_pr_status CHECK (status IN ('draft', 'approved', 'paid', 'cancelled')),
    CONSTRAINT chk_pr_dates CHECK (period_end >= period_start),
    CONSTRAINT uq_tenant_payroll_run UNIQUE (tenant_id, frequency, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS payroll_runs_tenant_period_idx ON public.payroll_runs(tenant_id, period_start DESC, period_end DESC);
CREATE INDEX IF NOT EXISTS payroll_runs_tenant_status_idx ON public.payroll_runs(tenant_id, status, pay_date DESC);

DROP TRIGGER IF EXISTS set_payroll_runs_updated_at ON public.payroll_runs;
CREATE TRIGGER set_payroll_runs_updated_at
    BEFORE UPDATE ON public.payroll_runs
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.payroll_runs ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view payroll runs in their tenant" ON public.payroll_runs;
CREATE POLICY "Members can view payroll runs in their tenant"
    ON public.payroll_runs FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and Admin can manage payroll runs" ON public.payroll_runs;
CREATE POLICY "Owner and Admin can manage payroll runs"
    ON public.payroll_runs FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

DROP POLICY IF EXISTS "Service role has full access to payroll runs" ON public.payroll_runs;
CREATE POLICY "Service role has full access to payroll runs"
    ON public.payroll_runs FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Consistency Guard
CREATE OR REPLACE FUNCTION public.check_payroll_run_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_loc_tenant UUID;
BEGIN
    IF NEW.location_id IS NOT NULL THEN
        SELECT tenant_id INTO v_loc_tenant FROM public.locations WHERE id = NEW.location_id;
        IF v_loc_tenant != NEW.tenant_id THEN
            RAISE EXCEPTION 'Location does not belong to the same tenant' USING ERRCODE = 'P0002';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_payroll_run_consistency ON public.payroll_runs;
CREATE TRIGGER validate_payroll_run_consistency
    BEFORE INSERT OR UPDATE ON public.payroll_runs
    FOR EACH ROW
    EXECUTE FUNCTION public.check_payroll_run_tenant_consistency();

-- ==============================================================================
-- PART C: PAYROLL ITEMS (LINES)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.payroll_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    payroll_run_id UUID NOT NULL REFERENCES public.payroll_runs(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES public.staff(id) ON DELETE RESTRICT,
    
    line_type TEXT NOT NULL DEFAULT 'earning',
    concept TEXT NOT NULL DEFAULT 'salary',
    amount NUMERIC(12,2) NOT NULL,
    
    appointment_id UUID NULL REFERENCES public.appointments(id) ON DELETE SET NULL,
    service_id UUID NULL REFERENCES public.services(id) ON DELETE SET NULL,
    notes TEXT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_pi_line_type CHECK (line_type IN ('earning', 'deduction')),
    CONSTRAINT chk_pi_concept CHECK (concept IN ('salary', 'commission', 'bonus', 'tips', 'overtime', 'advance', 'loan', 'other')),
    CONSTRAINT chk_pi_amount CHECK (amount > 0),
    CONSTRAINT uq_run_staff_concept_type UNIQUE (payroll_run_id, staff_id, concept, line_type)
);

CREATE INDEX IF NOT EXISTS payroll_items_run_idx ON public.payroll_items(payroll_run_id);
CREATE INDEX IF NOT EXISTS payroll_items_tenant_staff_idx ON public.payroll_items(tenant_id, staff_id);
CREATE INDEX IF NOT EXISTS payroll_items_tenant_concept_idx ON public.payroll_items(tenant_id, concept);

DROP TRIGGER IF EXISTS set_payroll_items_updated_at ON public.payroll_items;
CREATE TRIGGER set_payroll_items_updated_at
    BEFORE UPDATE ON public.payroll_items
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.payroll_items ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view payroll items in their tenant" ON public.payroll_items;
CREATE POLICY "Members can view payroll items in their tenant"
    ON public.payroll_items FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and Admin can manage payroll items" ON public.payroll_items;
CREATE POLICY "Owner and Admin can manage payroll items"
    ON public.payroll_items FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

DROP POLICY IF EXISTS "Service role has full access to payroll items" ON public.payroll_items;
CREATE POLICY "Service role has full access to payroll items"
    ON public.payroll_items FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Consistency Guard
CREATE OR REPLACE FUNCTION public.check_payroll_item_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_run_tenant UUID;
    v_staff_tenant UUID;
    v_appt_tenant UUID;
    v_srv_tenant UUID;
BEGIN
    SELECT tenant_id INTO v_run_tenant FROM public.payroll_runs WHERE id = NEW.payroll_run_id;
    IF v_run_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Payroll run does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    
    SELECT tenant_id INTO v_staff_tenant FROM public.staff WHERE id = NEW.staff_id;
    IF v_staff_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Staff does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    
    IF NEW.appointment_id IS NOT NULL THEN
        SELECT tenant_id INTO v_appt_tenant FROM public.appointments WHERE id = NEW.appointment_id;
        IF v_appt_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Appointment does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;

    IF NEW.service_id IS NOT NULL THEN
        SELECT tenant_id INTO v_srv_tenant FROM public.services WHERE id = NEW.service_id;
        IF v_srv_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Service does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_payroll_item_consistency ON public.payroll_items;
CREATE TRIGGER validate_payroll_item_consistency
    BEFORE INSERT OR UPDATE ON public.payroll_items
    FOR EACH ROW
    EXECUTE FUNCTION public.check_payroll_item_tenant_consistency();

-- ==============================================================================
-- PART D: PAYROLL TOTALS RECALCULATION
-- ==============================================================================

CREATE OR REPLACE FUNCTION public.recalculate_payroll_run_totals()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_run_id UUID;
    v_total NUMERIC(12,2) := 0;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_run_id := OLD.payroll_run_id;
    ELSE
        v_run_id := NEW.payroll_run_id;
    END IF;

    SELECT COALESCE(SUM(CASE WHEN line_type = 'earning' THEN amount ELSE -amount END), 0)
    INTO v_total
    FROM public.payroll_items
    WHERE payroll_run_id = v_run_id;

    -- Ensure total stringency (0 floor if deductions > earnings)
    IF v_total < 0 THEN v_total := 0; END IF;

    UPDATE public.payroll_runs
    SET total = v_total
    WHERE id = v_run_id;

    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trigger_recalculate_payroll_totals ON public.payroll_items;
CREATE TRIGGER trigger_recalculate_payroll_totals
    AFTER INSERT OR UPDATE OR DELETE ON public.payroll_items
    FOR EACH ROW
    EXECUTE FUNCTION public.recalculate_payroll_run_totals();

-- ==============================================================================
-- PART E: BIND PAYROLL TO TRANSACTIONS
-- ==============================================================================

ALTER TABLE public.transactions 
ADD COLUMN IF NOT EXISTS payroll_run_id UUID NULL REFERENCES public.payroll_runs(id) ON DELETE SET NULL;

ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS chk_transactions_target;
ALTER TABLE public.transactions ADD CONSTRAINT chk_transactions_target CHECK (
    ( (appointment_id IS NOT NULL)::int + 
      (sales_order_id IS NOT NULL)::int + 
      (purchase_order_id IS NOT NULL)::int +
      (payroll_run_id IS NOT NULL)::int ) = 1
);

-- Overwrite checking function to add payroll_runs to the transaction target rules
CREATE OR REPLACE FUNCTION public.check_transaction_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_sales_tenant UUID;
    v_appt_tenant UUID;
    v_contact_tenant UUID;
    v_location_tenant UUID;
    v_staff_tenant UUID;
    v_account_tenant UUID;
    v_payroll_tenant UUID;
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

-- Guard to enforce transactions when marking payroll 'paid'
CREATE OR REPLACE FUNCTION public.guard_payroll_run_paid_status()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_paid NUMERIC(12,2);
BEGIN
    IF NEW.status = 'paid' AND (OLD.status IS DISTINCT FROM 'paid' OR NEW.status IS DISTINCT FROM OLD.status) THEN
        SELECT COALESCE(SUM(amount), 0) INTO v_paid
        FROM public.transactions
        WHERE payroll_run_id = NEW.id 
          AND direction = 'out' 
          AND status = 'completed';
          
        IF v_paid < NEW.total OR NEW.total = 0 THEN
            RAISE EXCEPTION 'Cannot manually mark payroll run as paid without completed OUT transactions covering the full total' USING ERRCODE = 'P0005';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trigger_guard_payroll_paid_status ON public.payroll_runs;
CREATE TRIGGER trigger_guard_payroll_paid_status
    BEFORE UPDATE ON public.payroll_runs
    FOR EACH ROW
    EXECUTE FUNCTION public.guard_payroll_run_paid_status();

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
