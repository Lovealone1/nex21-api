-- Migration: Create Services Schema & Recreate Appointments
-- 1. Create public.services
-- 2. Drop and Recreate public.appointments referencing public.services
-- 3. Add Consistency constraints and Overlap prevention

-- ==============================================================================
-- PART A: SERVICES
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NULL REFERENCES public.locations(id) ON DELETE SET NULL,
    
    -- Identity
    name TEXT NOT NULL,
    description TEXT NULL,
    category TEXT NULL,
    
    -- Scheduling defaults (essential for calendar logic)
    duration_minutes INT NOT NULL DEFAULT 30,
    buffer_minutes INT NOT NULL DEFAULT 0,
    
    -- Pricing
    price NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'COP',
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_duration CHECK (duration_minutes > 0),
    CONSTRAINT chk_buffer CHECK (buffer_minutes >= 0)
);

-- Indexing for SaaS performance (services)
CREATE INDEX IF NOT EXISTS services_tenant_id_idx ON public.services(tenant_id);
CREATE INDEX IF NOT EXISTS services_tenant_location_idx ON public.services(tenant_id, location_id);
CREATE INDEX IF NOT EXISTS services_tenant_name_idx ON public.services(tenant_id, lower(name));

-- Trigger for update_at
DROP TRIGGER IF EXISTS set_services_updated_at ON public.services;
CREATE TRIGGER set_services_updated_at
    BEFORE UPDATE ON public.services
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for services
ALTER TABLE public.services ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view services in their tenant" ON public.services;
CREATE POLICY "Members can view services in their tenant"
    ON public.services FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Staff and above can manage services" ON public.services;
CREATE POLICY "Staff and above can manage services"
    ON public.services FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

DROP POLICY IF EXISTS "Service role has full access to services" ON public.services;
CREATE POLICY "Service role has full access to services"
    ON public.services FOR ALL TO service_role USING (true) WITH CHECK (true);

-- ==============================================================================
-- PART B: APPOINTMENTS (Recreated to reference services instead of catalog_items)
-- ==============================================================================

-- Drop previous version if it exists to cleanly recreate the schema dependencies
DROP TABLE IF EXISTS public.appointments CASCADE;

-- Ensure btree_gist extension is available for overlapped time constraints
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE public.appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES public.locations(id) ON DELETE RESTRICT,
    staff_id UUID NOT NULL REFERENCES public.staff(id) ON DELETE RESTRICT,
    contact_id UUID NOT NULL REFERENCES public.contacts(id) ON DELETE RESTRICT,
    service_id UUID NOT NULL REFERENCES public.services(id) ON DELETE RESTRICT,
    
    -- Time
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    
    -- Status
    status TEXT NOT NULL DEFAULT 'scheduled',
    
    -- Optional metadata
    notes TEXT NULL,
    cancelled_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    created_by UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_time_logic CHECK (end_at > start_at),
    CONSTRAINT chk_appointment_status CHECK (status IN ('scheduled', 'confirmed', 'cancelled', 'completed', 'no_show'))
);

-- Overlap Prevention (Critical)
-- Uses Postgres Exclusion Constraints to guarantee no double-booking per staff member natively.
-- Excludes overlaps if BOTH appointments are active.
ALTER TABLE public.appointments ADD CONSTRAINT prevent_staff_double_booking
EXCLUDE USING gist (
    tenant_id WITH =,
    staff_id WITH =,
    tstzrange(start_at, end_at, '[)') WITH &&
)
WHERE (status IN ('scheduled', 'confirmed'));

-- Indexing for SaaS performance (appointments)
CREATE INDEX IF NOT EXISTS appointments_tenant_staff_start_idx ON public.appointments (tenant_id, staff_id, start_at);
CREATE INDEX IF NOT EXISTS appointments_tenant_location_start_idx ON public.appointments (tenant_id, location_id, start_at);
CREATE INDEX IF NOT EXISTS appointments_tenant_contact_start_idx ON public.appointments (tenant_id, contact_id, start_at);
CREATE INDEX IF NOT EXISTS appointments_tenant_status_start_idx ON public.appointments (tenant_id, status, start_at);

-- Trigger for update_at
DROP TRIGGER IF EXISTS set_appointments_updated_at ON public.appointments;
CREATE TRIGGER set_appointments_updated_at
    BEFORE UPDATE ON public.appointments
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for appointments
ALTER TABLE public.appointments ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view appointments in their tenant" ON public.appointments;
CREATE POLICY "Members can view appointments in their tenant"
    ON public.appointments FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Staff and above can manage appointments" ON public.appointments;
CREATE POLICY "Staff and above can manage appointments"
    ON public.appointments FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

DROP POLICY IF EXISTS "Service role has full access to appointments" ON public.appointments;
CREATE POLICY "Service role has full access to appointments"
    ON public.appointments FOR ALL TO service_role USING (true) WITH CHECK (true);

-- ==============================================================================
-- PART C: CROSS-TENANT DATA CONTAMINATION GUARDS
-- ==============================================================================

-- A robust appointment system in a SaaS must never allow mixing entities from different tenants
-- (e.g. creating an appointment for Tenant A using a service from Tenant B).
CREATE OR REPLACE FUNCTION public.check_appointment_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_service_tenant UUID;
    v_staff_tenant UUID;
    v_contact_tenant UUID;
    v_location_tenant UUID;
BEGIN
    -- Check Service
    SELECT tenant_id INTO v_service_tenant FROM public.services WHERE id = NEW.service_id;
    IF v_service_tenant != NEW.tenant_id THEN 
        RAISE EXCEPTION 'Service does not belong to the same tenant' USING ERRCODE = 'P0002'; 
    END IF;
    
    -- Check Staff
    SELECT tenant_id INTO v_staff_tenant FROM public.staff WHERE id = NEW.staff_id;
    IF v_staff_tenant != NEW.tenant_id THEN 
        RAISE EXCEPTION 'Staff member does not belong to the same tenant' USING ERRCODE = 'P0002'; 
    END IF;
    
    -- Check Contact
    SELECT tenant_id INTO v_contact_tenant FROM public.contacts WHERE id = NEW.contact_id;
    IF v_contact_tenant != NEW.tenant_id THEN 
        RAISE EXCEPTION 'Contact does not belong to the same tenant' USING ERRCODE = 'P0002'; 
    END IF;
    
    -- Check Location
    SELECT tenant_id INTO v_location_tenant FROM public.locations WHERE id = NEW.location_id;
    IF v_location_tenant != NEW.tenant_id THEN 
        RAISE EXCEPTION 'Location does not belong to the same tenant' USING ERRCODE = 'P0002'; 
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_appointment_consistency ON public.appointments;
CREATE TRIGGER validate_appointment_consistency
    BEFORE INSERT OR UPDATE ON public.appointments
    FOR EACH ROW
    EXECUTE FUNCTION public.check_appointment_tenant_consistency();

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
