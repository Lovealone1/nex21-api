-- Migration: Create Accounts Schema
-- Introduces public.accounts and links it to public.transactions with backfilling.

-- ==============================================================================
-- PART A: CREATE ACCOUNTS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Identity
    name TEXT NOT NULL,
    code TEXT NULL,
    
    -- Type & Currency
    account_type TEXT NOT NULL DEFAULT 'cash',
    currency TEXT NOT NULL DEFAULT 'COP',
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_default BOOLEAN NOT NULL DEFAULT false,
    
    -- Optional metadata
    provider TEXT NULL,
    notes TEXT NULL,
    
    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_account_type CHECK (account_type IN ('cash', 'bank', 'wallet', 'card_terminal', 'gateway', 'other'))
);

-- Partial Unique Constraints
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_default_account 
    ON public.accounts (tenant_id) 
    WHERE is_default = true;

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_account_code 
    ON public.accounts (tenant_id, code) 
    WHERE code IS NOT NULL;

-- Indexing for SaaS performance (accounts)
CREATE INDEX IF NOT EXISTS accounts_tenant_id_idx ON public.accounts(tenant_id);
CREATE INDEX IF NOT EXISTS accounts_tenant_type_idx ON public.accounts(tenant_id, account_type);
CREATE INDEX IF NOT EXISTS accounts_tenant_name_idx ON public.accounts(tenant_id, lower(name));

-- Trigger for update_at
DROP TRIGGER IF EXISTS set_accounts_updated_at ON public.accounts;
CREATE TRIGGER set_accounts_updated_at
    BEFORE UPDATE ON public.accounts
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for accounts
ALTER TABLE public.accounts ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view accounts in their tenant" ON public.accounts;
CREATE POLICY "Members can view accounts in their tenant"
    ON public.accounts FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and Admin can manage accounts" ON public.accounts;
CREATE POLICY "Owner and Admin can manage accounts"
    ON public.accounts FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

DROP POLICY IF EXISTS "Service role has full access to accounts" ON public.accounts;
CREATE POLICY "Service role has full access to accounts"
    ON public.accounts FOR ALL TO service_role USING (true) WITH CHECK (true);

-- ==============================================================================
-- PART B: BIND ACCOUNTS TO TRANSACTIONS & BACKFILL
-- ==============================================================================

-- 1) Add the column as nullable initially
-- We use RESTRICT instead of SET NULL because if an account is deleted, we shouldn't orphan financial records. 
-- For a SaaS, it's better to soft-delete (is_active=false) accounts rather than hard delete them if they have transactions.
ALTER TABLE public.transactions 
ADD COLUMN IF NOT EXISTS account_id UUID NULL REFERENCES public.accounts(id) ON DELETE RESTRICT;

-- 2) Backfill Strategy
-- Create a default 'Caja Principal' account for every tenant that doesn't have a default account yet.
INSERT INTO public.accounts (tenant_id, name, account_type, is_default)
SELECT t.id, 'Caja Principal', 'cash', true
FROM public.tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM public.accounts a 
    WHERE a.tenant_id = t.id AND a.is_default = true
);

-- Update all existing transactions that have no account_id, assigning them to their tenant's default account.
UPDATE public.transactions t
SET account_id = a.id
FROM public.accounts a
WHERE a.tenant_id = t.tenant_id 
  AND a.is_default = true 
  AND t.account_id IS NULL;

-- Now that all transactions are guaranteed to have an account_id, enforce NOT NULL
ALTER TABLE public.transactions 
ALTER COLUMN account_id SET NOT NULL;

-- 3) Add new index
CREATE INDEX IF NOT EXISTS transactions_tenant_account_paid_at_idx 
ON public.transactions(tenant_id, account_id, paid_at DESC);

-- ==============================================================================
-- PART C: CROSS-TENANT SAFETY GUARDS UPDATE
-- ==============================================================================

-- Overwrite the existing consistency function to include the new account_id check.
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
BEGIN
    -- Check Account (NEW!)
    IF NEW.account_id IS NOT NULL THEN
        SELECT tenant_id INTO v_account_tenant FROM public.accounts WHERE id = NEW.account_id;
        IF v_account_tenant != NEW.tenant_id THEN
            RAISE EXCEPTION 'Account does not belong to the same tenant' USING ERRCODE = 'P0002';
        END IF;
    END IF;

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

-- Note: The trigger "validate_transaction_consistency" is already bound to this function from the previous migration.
-- By using 'CREATE OR REPLACE', the database will now use this updated logic seamlessly.

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
