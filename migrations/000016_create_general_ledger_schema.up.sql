-- Migration: Create General Ledger Schema
-- Introduces chart of accounts, ledger journals, and ledger entries for double-entry accounting.

-- ==============================================================================
-- PART A: CHART OF ACCOUNTS (COA)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.chart_of_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NULL,
    
    account_type TEXT NOT NULL,
    normal_balance TEXT NOT NULL,
    
    parent_id UUID NULL REFERENCES public.chart_of_accounts(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_coa_type CHECK (account_type IN ('asset', 'liability', 'equity', 'income', 'expense')),
    CONSTRAINT chk_coa_balance CHECK (normal_balance IN ('debit', 'credit')),
    CONSTRAINT uq_tenant_coa_code UNIQUE (tenant_id, code)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_coa_name ON public.chart_of_accounts (tenant_id, lower(name));

CREATE INDEX IF NOT EXISTS coa_tenant_code_idx ON public.chart_of_accounts(tenant_id, code);
CREATE INDEX IF NOT EXISTS coa_tenant_type_idx ON public.chart_of_accounts(tenant_id, account_type);
CREATE INDEX IF NOT EXISTS coa_tenant_parent_idx ON public.chart_of_accounts(tenant_id, parent_id);

DROP TRIGGER IF EXISTS set_coa_updated_at ON public.chart_of_accounts;
CREATE TRIGGER set_coa_updated_at
    BEFORE UPDATE ON public.chart_of_accounts
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.chart_of_accounts ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view COA in tenant" ON public.chart_of_accounts;
CREATE POLICY "Members can view COA in tenant"
    ON public.chart_of_accounts FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and Admin can manage COA" ON public.chart_of_accounts;
CREATE POLICY "Owner and Admin can manage COA"
    ON public.chart_of_accounts FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner','admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner','admin']));

DROP POLICY IF EXISTS "Service role has full access to COA" ON public.chart_of_accounts;
CREATE POLICY "Service role has full access to COA"
    ON public.chart_of_accounts FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Consistency Guard
CREATE OR REPLACE FUNCTION public.check_coa_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_parent_tenant UUID;
BEGIN
    IF NEW.parent_id IS NOT NULL THEN
        SELECT tenant_id INTO v_parent_tenant FROM public.chart_of_accounts WHERE id = NEW.parent_id;
        IF v_parent_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Parent account does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_coa_consistency ON public.chart_of_accounts;
CREATE TRIGGER validate_coa_consistency
    BEFORE INSERT OR UPDATE ON public.chart_of_accounts
    FOR EACH ROW
    EXECUTE FUNCTION public.check_coa_tenant_consistency();


-- ==============================================================================
-- PART B: LEDGER JOURNALS (HEADERS)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.ledger_journals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    journal_number TEXT NOT NULL,
    description TEXT NULL,
    
    status TEXT NOT NULL DEFAULT 'draft',
    journal_date DATE NOT NULL DEFAULT current_date,
    posted_at TIMESTAMPTZ NULL,
    posted_by UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    
    source_type TEXT NULL,
    source_id UUID NULL,
    
    total_debit NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_credit NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'COP',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_journal_status CHECK (status IN ('draft', 'posted', 'void')),
    CONSTRAINT chk_journal_source CHECK ( (source_type IS NULL AND source_id IS NULL) OR (source_type IS NOT NULL AND source_id IS NOT NULL) ),
    CONSTRAINT chk_journal_source_type CHECK (source_type IS NULL OR source_type IN ('transaction', 'sale', 'purchase', 'expense', 'payroll', 'appointment', 'manual')),
    CONSTRAINT uq_tenant_journal_number UNIQUE (tenant_id, journal_number)
);

CREATE INDEX IF NOT EXISTS ledger_journals_tenant_date_idx ON public.ledger_journals(tenant_id, journal_date DESC);
CREATE INDEX IF NOT EXISTS ledger_journals_tenant_status_idx ON public.ledger_journals(tenant_id, status, journal_date DESC);
CREATE INDEX IF NOT EXISTS ledger_journals_tenant_source_idx ON public.ledger_journals(tenant_id, source_type, source_id);

DROP TRIGGER IF EXISTS set_journals_updated_at ON public.ledger_journals;
CREATE TRIGGER set_journals_updated_at
    BEFORE UPDATE ON public.ledger_journals
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.ledger_journals ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view journals in tenant" ON public.ledger_journals;
CREATE POLICY "Members can view journals in tenant"
    ON public.ledger_journals FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and Admin can manage journals" ON public.ledger_journals;
CREATE POLICY "Owner and Admin can manage journals"
    ON public.ledger_journals FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner','admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner','admin']));

DROP POLICY IF EXISTS "Service role has full access to journals" ON public.ledger_journals;
CREATE POLICY "Service role has full access to journals"
    ON public.ledger_journals FOR ALL TO service_role USING (true) WITH CHECK (true);


-- ==============================================================================
-- PART C: LEDGER ENTRIES (LINES)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    journal_id UUID NOT NULL REFERENCES public.ledger_journals(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES public.chart_of_accounts(id) ON DELETE RESTRICT,
    
    description TEXT NULL,
    debit NUMERIC(12,2) NOT NULL DEFAULT 0,
    credit NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'COP',
    
    contact_id UUID NULL REFERENCES public.contacts(id) ON DELETE SET NULL,
    location_id UUID NULL REFERENCES public.locations(id) ON DELETE SET NULL,
    staff_id UUID NULL REFERENCES public.staff(id) ON DELETE SET NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_entry_positive CHECK (debit >= 0 AND credit >= 0),
    CONSTRAINT chk_entry_exclusive CHECK ( (debit = 0 AND credit > 0) OR (debit > 0 AND credit = 0) ),
    CONSTRAINT chk_entry_non_zero CHECK (debit + credit > 0)
);

CREATE INDEX IF NOT EXISTS ledger_entries_journal_idx ON public.ledger_entries(journal_id);
CREATE INDEX IF NOT EXISTS ledger_entries_tenant_account_idx ON public.ledger_entries(tenant_id, account_id);
CREATE INDEX IF NOT EXISTS ledger_entries_tenant_contact_idx ON public.ledger_entries(tenant_id, contact_id);
CREATE INDEX IF NOT EXISTS ledger_entries_tenant_location_idx ON public.ledger_entries(tenant_id, location_id);
CREATE INDEX IF NOT EXISTS ledger_entries_tenant_staff_idx ON public.ledger_entries(tenant_id, staff_id);

DROP TRIGGER IF EXISTS set_entries_updated_at ON public.ledger_entries;
CREATE TRIGGER set_entries_updated_at
    BEFORE UPDATE ON public.ledger_entries
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.ledger_entries ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view entries in tenant" ON public.ledger_entries;
CREATE POLICY "Members can view entries in tenant"
    ON public.ledger_entries FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owner and Admin can manage entries" ON public.ledger_entries;
CREATE POLICY "Owner and Admin can manage entries"
    ON public.ledger_entries FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner','admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner','admin']));

DROP POLICY IF EXISTS "Service role has full access to entries" ON public.ledger_entries;
CREATE POLICY "Service role has full access to entries"
    ON public.ledger_entries FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Consistency Guard
CREATE OR REPLACE FUNCTION public.check_entry_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_journal_tenant UUID;
    v_coa_tenant UUID;
    v_contact_tenant UUID;
    v_location_tenant UUID;
    v_staff_tenant UUID;
BEGIN
    SELECT tenant_id INTO v_journal_tenant FROM public.ledger_journals WHERE id = NEW.journal_id;
    IF v_journal_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Journal does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;

    SELECT tenant_id INTO v_coa_tenant FROM public.chart_of_accounts WHERE id = NEW.account_id;
    IF v_coa_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Account does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;

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
        IF v_staff_tenant != NEW.tenant_id THEN RAISE EXCEPTION 'Staff does not belong to the same tenant' USING ERRCODE = 'P0002'; END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_entry_consistency ON public.ledger_entries;
CREATE TRIGGER validate_entry_consistency
    BEFORE INSERT OR UPDATE ON public.ledger_entries
    FOR EACH ROW
    EXECUTE FUNCTION public.check_entry_tenant_consistency();


-- ==============================================================================
-- PART D: ENFORCE BALANCED JOURNALS & TOTALS CACHE
-- ==============================================================================

-- 1. Auto-Reconcile/Cache Totals in parent
CREATE OR REPLACE FUNCTION public.recalculate_journal_totals()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_journal_id UUID;
    v_tot_debit NUMERIC(12,2) := 0;
    v_tot_credit NUMERIC(12,2) := 0;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_journal_id := OLD.journal_id;
    ELSE
        v_journal_id := NEW.journal_id;
    END IF;

    SELECT COALESCE(SUM(debit), 0), COALESCE(SUM(credit), 0)
    INTO v_tot_debit, v_tot_credit
    FROM public.ledger_entries
    WHERE journal_id = v_journal_id;

    UPDATE public.ledger_journals
    SET total_debit = v_tot_debit,
        total_credit = v_tot_credit
    WHERE id = v_journal_id;

    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trigger_recalculate_journal_totals ON public.ledger_entries;
CREATE TRIGGER trigger_recalculate_journal_totals
    AFTER INSERT OR UPDATE OR DELETE ON public.ledger_entries
    FOR EACH ROW
    EXECUTE FUNCTION public.recalculate_journal_totals();

-- 2. Guard/Enforce Balanced Posting on the Journal level
CREATE OR REPLACE FUNCTION public.enforce_journal_balanced_on_post()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    IF NEW.status = 'posted' AND OLD.status != 'posted' THEN
        -- Verify Balance
        IF NEW.total_debit != NEW.total_credit THEN
            RAISE EXCEPTION 'Cannot post unbalanced journal (Debit: %, Credit: %)', NEW.total_debit, NEW.total_credit USING ERRCODE = 'P0006';
        END IF;

        IF NEW.total_debit <= 0 THEN
            RAISE EXCEPTION 'Cannot post journal with zero or negative totals' USING ERRCODE = 'P0006';
        END IF;

        NEW.posted_at := now();
        
        -- Try to grab the user ID safely if executed directly by an authenticated user
        -- (Safe fallback if outside of direct API hit, created_by not in schema though so we just grab auth.uid())
        BEGIN
            NEW.posted_by := auth.uid();
        EXCEPTION WHEN OTHERS THEN
            -- In cases where this isn't run through PostgREST (e.g. background job), this fails gracefully.
            NULL;
        END;
    END IF;

    -- If unposting (e.g., reverting to draft if allowed by business rules or moving to void)
    IF NEW.status != 'posted' AND (OLD.status = 'posted' OR OLD.posted_at IS NOT NULL) THEN
        NEW.posted_at := NULL;
        NEW.posted_by := NULL;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trigger_guard_journal_posting ON public.ledger_journals;
CREATE TRIGGER trigger_guard_journal_posting
    BEFORE UPDATE ON public.ledger_journals
    FOR EACH ROW
    EXECUTE FUNCTION public.enforce_journal_balanced_on_post();

-- 3. Block mutating lines in posted journals
CREATE OR REPLACE FUNCTION public.prevent_edit_posted_entries()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_status TEXT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        SELECT status INTO v_status FROM public.ledger_journals WHERE id = OLD.journal_id;
    ELSE
        SELECT status INTO v_status FROM public.ledger_journals WHERE id = NEW.journal_id;
    END IF;

    IF v_status = 'posted' THEN
        RAISE EXCEPTION 'Cannot modify ledger entries for a journal that is already posted' USING ERRCODE = 'P0007';
    END IF;

    IF TG_OP = 'DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END;
$$;

DROP TRIGGER IF EXISTS trigger_prevent_posting_entry_edit ON public.ledger_entries;
CREATE TRIGGER trigger_prevent_posting_entry_edit
    BEFORE INSERT OR UPDATE OR DELETE ON public.ledger_entries
    FOR EACH ROW
    EXECUTE FUNCTION public.prevent_edit_posted_entries();

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
