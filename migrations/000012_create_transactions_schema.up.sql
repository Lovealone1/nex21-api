-- Migration: Create Unified Transactions Schema (Payments)
-- Adapts sales_orders and introduces public.transactions with polymorphic targets.

-- ==============================================================================
-- PART A: ADAPT SALES ORDERS
-- ==============================================================================

ALTER TABLE public.sales_orders 
ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ NULL,
ADD COLUMN IF NOT EXISTS payment_status TEXT NOT NULL DEFAULT 'unpaid';

-- Add constraints for the new columns
ALTER TABLE public.sales_orders
ADD CONSTRAINT chk_sales_order_payment_status 
CHECK (payment_status IN ('unpaid', 'paid', 'refunded'));

-- Add performance index for payment times
CREATE INDEX IF NOT EXISTS sales_orders_tenant_paid_at_idx 
ON public.sales_orders(tenant_id, paid_at DESC);

-- ==============================================================================
-- PART B: CREATE UNIFIED TRANSACTIONS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Polymorphic Target (What this pays for)
    appointment_id UUID NULL REFERENCES public.appointments(id) ON DELETE SET NULL,
    sales_order_id UUID NULL REFERENCES public.sales_orders(id) ON DELETE SET NULL,
    purchase_order_id UUID NULL, -- Placeholder for future module
    
    -- Party
    contact_id UUID NULL REFERENCES public.contacts(id) ON DELETE SET NULL,
    
    -- Operational Metadata
    location_id UUID NULL REFERENCES public.locations(id) ON DELETE SET NULL,
    staff_id UUID NULL REFERENCES public.staff(id) ON DELETE SET NULL,
    created_by UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    
    -- Payment Core
    direction TEXT NOT NULL, -- 'in' or 'out'
    amount NUMERIC(12,2) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'COP',
    method TEXT NOT NULL DEFAULT 'cash',
    status TEXT NOT NULL DEFAULT 'completed',
    
    -- Timing & External Info
    paid_at TIMESTAMPTZ NULL,
    reference TEXT NULL,
    notes TEXT NULL,
    metadata JSONB NULL,
    
    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_transactions_amount CHECK (amount > 0),
    CONSTRAINT chk_transactions_direction CHECK (direction IN ('in', 'out')),
    CONSTRAINT chk_transactions_method CHECK (method IN ('cash', 'transfer', 'card', 'other')),
    CONSTRAINT chk_transactions_status CHECK (status IN ('pending', 'completed', 'failed', 'cancelled', 'refunded')),
    
    -- Ensures a transaction links exactly to ONE module entity (appointment, sale, or purchase)
    CONSTRAINT chk_transactions_target CHECK (
        ( (appointment_id IS NOT NULL)::int + 
          (sales_order_id IS NOT NULL)::int + 
          (purchase_order_id IS NOT NULL)::int ) = 1
    )
);

-- Indexing for SaaS performance (transactions)
CREATE INDEX IF NOT EXISTS transactions_tenant_paid_at_idx ON public.transactions(tenant_id, paid_at DESC);
CREATE INDEX IF NOT EXISTS transactions_tenant_status_paid_at_idx ON public.transactions(tenant_id, status, paid_at DESC);
CREATE INDEX IF NOT EXISTS transactions_sales_order_idx ON public.transactions(sales_order_id);
CREATE INDEX IF NOT EXISTS transactions_appointment_idx ON public.transactions(appointment_id);
CREATE INDEX IF NOT EXISTS transactions_contact_idx ON public.transactions(tenant_id, contact_id, paid_at DESC);

-- Trigger for update_at
DROP TRIGGER IF EXISTS set_transactions_updated_at ON public.transactions;
CREATE TRIGGER set_transactions_updated_at
    BEFORE UPDATE ON public.transactions
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for transactions
ALTER TABLE public.transactions ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view transactions in their tenant" ON public.transactions;
CREATE POLICY "Members can view transactions in their tenant"
    ON public.transactions FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Staff and above can manage transactions" ON public.transactions;
CREATE POLICY "Staff and above can manage transactions"
    ON public.transactions FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

DROP POLICY IF EXISTS "Service role has full access to transactions" ON public.transactions;
CREATE POLICY "Service role has full access to transactions"
    ON public.transactions FOR ALL TO service_role USING (true) WITH CHECK (true);

-- ==============================================================================
-- PART C: CROSS-TENANT SAFETY GUARDS
-- ==============================================================================

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
BEGIN
    -- Check Sales Order
    IF NEW.sales_order_id IS NOT NULL THEN
        SELECT tenant_id INTO v_sales_tenant FROM public.sales_orders WHERE id = NEW.sales_order_id;
        IF v_sales_tenant != NEW.tenant_id THEN 
            RAISE EXCEPTION 'Sales order does not belong to the same tenant' USING ERRCODE = 'P0002'; 
        END IF;
    END IF;

    -- Check Appointment
    IF NEW.appointment_id IS NOT NULL THEN
        SELECT tenant_id INTO v_appt_tenant FROM public.appointments WHERE id = NEW.appointment_id;
        IF v_appt_tenant != NEW.tenant_id THEN 
            RAISE EXCEPTION 'Appointment does not belong to the same tenant' USING ERRCODE = 'P0002'; 
        END IF;
    END IF;

    -- Check Contact
    IF NEW.contact_id IS NOT NULL THEN
        SELECT tenant_id INTO v_contact_tenant FROM public.contacts WHERE id = NEW.contact_id;
        IF v_contact_tenant != NEW.tenant_id THEN 
            RAISE EXCEPTION 'Contact does not belong to the same tenant' USING ERRCODE = 'P0002'; 
        END IF;
    END IF;

    -- Check Location
    IF NEW.location_id IS NOT NULL THEN
        SELECT tenant_id INTO v_location_tenant FROM public.locations WHERE id = NEW.location_id;
        IF v_location_tenant != NEW.tenant_id THEN 
            RAISE EXCEPTION 'Location does not belong to the same tenant' USING ERRCODE = 'P0002'; 
        END IF;
    END IF;

    -- Check Staff
    IF NEW.staff_id IS NOT NULL THEN
        SELECT tenant_id INTO v_staff_tenant FROM public.staff WHERE id = NEW.staff_id;
        IF v_staff_tenant != NEW.tenant_id THEN 
            RAISE EXCEPTION 'Staff member does not belong to the same tenant' USING ERRCODE = 'P0002'; 
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_transaction_consistency ON public.transactions;
CREATE TRIGGER validate_transaction_consistency
    BEFORE INSERT OR UPDATE ON public.transactions
    FOR EACH ROW
    EXECUTE FUNCTION public.check_transaction_tenant_consistency();

-- ==============================================================================
-- PART D: ENFORCE SALES "PAID" RULES
-- ==============================================================================

-- 1) Recalculate Sales Order payment status when transactions change
CREATE OR REPLACE FUNCTION public.sync_sales_order_payment_status()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_order_id UUID;
    v_total NUMERIC(12,2);
    v_paid NUMERIC(12,2);
    v_last_paid_at TIMESTAMPTZ;
BEGIN
    -- Determine the target sales_order_id
    IF TG_OP = 'DELETE' THEN
        v_order_id := OLD.sales_order_id;
    ELSE
        v_order_id := NEW.sales_order_id;
    END IF;

    IF v_order_id IS NULL THEN
        RETURN NULL;
    END IF;

    -- Get the order total
    SELECT total INTO v_total FROM public.sales_orders WHERE id = v_order_id;

    -- Sum completed IN transactions for this order
    SELECT 
        COALESCE(SUM(amount), 0),
        MAX(paid_at)
    INTO v_paid, v_last_paid_at
    FROM public.transactions
    WHERE sales_order_id = v_order_id 
      AND direction = 'in' 
      AND status = 'completed';

    -- Update parent order
    IF v_paid >= v_total AND v_total > 0 THEN
        UPDATE public.sales_orders 
        SET payment_status = 'paid', paid_at = COALESCE(v_last_paid_at, now())
        WHERE id = v_order_id;
    ELSE
        UPDATE public.sales_orders 
        SET payment_status = 'unpaid', paid_at = NULL
        WHERE id = v_order_id;
    END IF;

    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trigger_sync_sales_payment ON public.transactions;
CREATE TRIGGER trigger_sync_sales_payment
    AFTER INSERT OR UPDATE OR DELETE ON public.transactions
    FOR EACH ROW
    EXECUTE FUNCTION public.sync_sales_order_payment_status();

-- 2) Prevent manual override of payment_status to 'paid' if logic forbids it
CREATE OR REPLACE FUNCTION public.guard_sales_order_payment_status()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_paid NUMERIC(12,2);
BEGIN
    -- Only check if we are transitioning to 'paid' or 'payment_status'='paid'
    IF NEW.payment_status = 'paid' AND (OLD.payment_status IS DISTINCT FROM 'paid' OR NEW.payment_status IS DISTINCT FROM OLD.payment_status) THEN
        -- Verify actual sum
        SELECT COALESCE(SUM(amount), 0) INTO v_paid
        FROM public.transactions
        WHERE sales_order_id = NEW.id 
          AND direction = 'in' 
          AND status = 'completed';
          
        IF v_paid < NEW.total OR NEW.total = 0 THEN
            RAISE EXCEPTION 'Cannot manually mark sales order as paid without completed transactions covering the full total' USING ERRCODE = 'P0005';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trigger_guard_sales_payment_status ON public.sales_orders;
CREATE TRIGGER trigger_guard_sales_payment_status
    BEFORE UPDATE ON public.sales_orders
    FOR EACH ROW
    EXECUTE FUNCTION public.guard_sales_order_payment_status();

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
