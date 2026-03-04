-- Migration: Create Appointments Schema
-- Contains public.appointments with advanced scheduling constraints and RLS policies

-- 1. Enable extension for advanced exclusion constraints (required for tstzrange overlapping detection)
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- 2. Create public.appointments
CREATE TABLE IF NOT EXISTS public.appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES public.locations(id) ON DELETE RESTRICT,
    staff_id UUID NOT NULL REFERENCES public.staff(id) ON DELETE RESTRICT,
    contact_id UUID NOT NULL REFERENCES public.contacts(id) ON DELETE RESTRICT,
    service_id UUID NOT NULL REFERENCES public.catalog_items(id) ON DELETE RESTRICT,
    
    -- Time
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    
    -- Status
    status TEXT NOT NULL DEFAULT 'scheduled',
    
    -- Optional metadata
    notes TEXT NULL,
    external_ref TEXT NULL,
    metadata JSONB NULL,
    cancelled_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    
    -- Audit
    created_by UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Constraints
    CONSTRAINT chk_time_logic CHECK (end_at > start_at),
    CONSTRAINT chk_appointment_status CHECK (status IN ('scheduled', 'confirmed', 'cancelled', 'completed', 'no_show'))
);

-- 3. Overlap Prevention (Critical)
-- We use Postgres Exclusion Constraints to guarantee no double-booking per staff member within a tenant.
-- The && operator checks if two tstzrange (timestamp with timezone range) overlap.
-- We only exclude overlaps if BOTH appointments are active ('scheduled' or 'confirmed').
ALTER TABLE public.appointments ADD CONSTRAINT prevent_staff_double_booking
EXCLUDE USING gist (
    tenant_id WITH =,
    staff_id WITH =,
    tstzrange(start_at, end_at, '[)') WITH &&
)
WHERE (status IN ('scheduled', 'confirmed'));

-- 4. Service Validation Approach
-- Approach A: Trigger.
-- Justification: We use a trigger because Supabase's FK constraints cannot enforce cross-table column checks (catalog_items.item_type = 'service') simply without complicated workarounds like composite foreign keys that pollute the design. A trigger provides a clean, domain-level validation precisely when the appointment is created or modified.
CREATE OR REPLACE FUNCTION public.check_appointment_is_service()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_item_type TEXT;
BEGIN
    SELECT item_type INTO v_item_type
    FROM public.catalog_items
    WHERE id = NEW.service_id AND tenant_id = NEW.tenant_id;
    
    -- Ensure the item exists AND belongs to the exact same tenant
    IF v_item_type IS NULL THEN
        RAISE EXCEPTION 'Service not found or cross-tenant contamination detected';
    END IF;

    IF v_item_type != 'service' THEN
        RAISE EXCEPTION 'Appointments can only be booked for items of type ''service''';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_appointment_service ON public.appointments;
CREATE TRIGGER validate_appointment_service
    BEFORE INSERT OR UPDATE ON public.appointments
    FOR EACH ROW
    EXECUTE FUNCTION public.check_appointment_is_service();

-- 5. Indexing for performance and multi-tenant queries (ERP usage)
CREATE INDEX IF NOT EXISTS appointments_tenant_staff_start_idx ON public.appointments (tenant_id, staff_id, start_at);
CREATE INDEX IF NOT EXISTS appointments_tenant_location_start_idx ON public.appointments (tenant_id, location_id, start_at);
CREATE INDEX IF NOT EXISTS appointments_tenant_contact_start_idx ON public.appointments (tenant_id, contact_id, start_at);
CREATE INDEX IF NOT EXISTS appointments_tenant_status_start_idx ON public.appointments (tenant_id, status, start_at);

-- 6. Triggers for updated_at
DROP TRIGGER IF EXISTS set_appointments_updated_at ON public.appointments;
CREATE TRIGGER set_appointments_updated_at
    BEFORE UPDATE ON public.appointments
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- 7. Enable Row Level Security (RLS)
ALTER TABLE public.appointments ENABLE ROW LEVEL SECURITY;

-- 8. RLS Policies

-- Read access
DROP POLICY IF EXISTS "Members can view appointments in their tenant" ON public.appointments;
CREATE POLICY "Members can view appointments in their tenant"
    ON public.appointments FOR SELECT
    USING (public.is_tenant_member(tenant_id));

-- Write access: Owner, Admin, and Staff can manage appointments
DROP POLICY IF EXISTS "Staff and above can manage appointments" ON public.appointments;
CREATE POLICY "Staff and above can manage appointments"
    ON public.appointments FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

-- Service role access
DROP POLICY IF EXISTS "Service role has full access to appointments" ON public.appointments;
CREATE POLICY "Service role has full access to appointments"
    ON public.appointments FOR ALL TO service_role USING (true) WITH CHECK (true);
