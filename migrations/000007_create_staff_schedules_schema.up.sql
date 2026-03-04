-- Migration: Create Staff and Work Schedules Schema
-- Contains public.staff, public.work_schedules, and RLS policies

-- 1. Create public.staff
CREATE TABLE IF NOT EXISTS public.staff (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NULL REFERENCES public.locations(id) ON DELETE SET NULL,
    profile_id UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    
    -- Identity fields
    display_name TEXT NOT NULL,
    email TEXT NULL,
    phone TEXT NULL,
    
    -- Role & Status
    staff_role TEXT NOT NULL DEFAULT 'staff',
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_staff_role CHECK (staff_role IN ('owner', 'admin', 'staff'))
);

-- Ensure a profile can only be mapped to one staff record per tenant
CREATE UNIQUE INDEX IF NOT EXISTS staff_tenant_profile_idx ON public.staff (tenant_id, profile_id) WHERE profile_id IS NOT NULL;

-- 2. Indexing for public.staff
-- Core tenant isolation idx
CREATE INDEX IF NOT EXISTS staff_tenant_id_idx ON public.staff (tenant_id);
-- Fast lookup by location within a tenant
CREATE INDEX IF NOT EXISTS staff_tenant_location_idx ON public.staff (tenant_id, location_id);
-- Case-insensitive search by display name
CREATE INDEX IF NOT EXISTS staff_tenant_name_idx ON public.staff (tenant_id, lower(display_name));

-- 3. Create public.work_schedules
CREATE TABLE IF NOT EXISTS public.work_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES public.staff(id) ON DELETE CASCADE,
    location_id UUID NULL REFERENCES public.locations(id) ON DELETE SET NULL,
    
    -- Schedule details
    -- weekday: 0 = Sunday, 1 = Monday, ..., 6 = Saturday
    weekday SMALLINT NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_weekday_range CHECK (weekday >= 0 AND weekday <= 6),
    CONSTRAINT chk_time_logic CHECK (end_time > start_time),
    CONSTRAINT uq_tenant_staff_schedule UNIQUE (tenant_id, staff_id, weekday, start_time, end_time)
);

-- 4. Indexing for public.work_schedules
-- Query schedules for a specific staff member on a specific day
CREATE INDEX IF NOT EXISTS work_schedules_tenant_staff_weekday_idx ON public.work_schedules (tenant_id, staff_id, weekday);
-- Query available schedules at a specific location on a given day
CREATE INDEX IF NOT EXISTS work_schedules_tenant_location_weekday_idx ON public.work_schedules (tenant_id, location_id, weekday);

-- 5. Triggers for updated_at
DROP TRIGGER IF EXISTS set_staff_updated_at ON public.staff;
CREATE TRIGGER set_staff_updated_at
    BEFORE UPDATE ON public.staff
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

DROP TRIGGER IF EXISTS set_work_schedules_updated_at ON public.work_schedules;
CREATE TRIGGER set_work_schedules_updated_at
    BEFORE UPDATE ON public.work_schedules
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- 6. Enable Row Level Security (RLS)
ALTER TABLE public.staff ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.work_schedules ENABLE ROW LEVEL SECURITY;

-- 7. RLS Policies for public.staff

-- Read access
DROP POLICY IF EXISTS "Members can view staff in their tenant" ON public.staff;
CREATE POLICY "Members can view staff in their tenant"
    ON public.staff FOR SELECT
    USING (public.is_tenant_member(tenant_id));

-- Write access
DROP POLICY IF EXISTS "Owners and admins can manage staff" ON public.staff;
CREATE POLICY "Owners and admins can manage staff"
    ON public.staff FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

-- Service role access
DROP POLICY IF EXISTS "Service role has full access to staff" ON public.staff;
CREATE POLICY "Service role has full access to staff"
    ON public.staff FOR ALL TO service_role USING (true) WITH CHECK (true);

-- 8. RLS Policies for public.work_schedules

-- Read access
DROP POLICY IF EXISTS "Members can view schedules in their tenant" ON public.work_schedules;
CREATE POLICY "Members can view schedules in their tenant"
    ON public.work_schedules FOR SELECT
    USING (public.is_tenant_member(tenant_id));

-- Write access
DROP POLICY IF EXISTS "Owners and admins can manage schedules" ON public.work_schedules;
CREATE POLICY "Owners and admins can manage schedules"
    ON public.work_schedules FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

-- Service role access
DROP POLICY IF EXISTS "Service role has full access to schedules" ON public.work_schedules;
CREATE POLICY "Service role has full access to schedules"
    ON public.work_schedules FOR ALL TO service_role USING (true) WITH CHECK (true);
