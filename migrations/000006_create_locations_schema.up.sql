-- Migration: Create Locations Schema
-- Contains public.locations and RLS policies

-- 1. Create public.locations
CREATE TABLE IF NOT EXISTS public.locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Identity fields
    name TEXT NOT NULL,
    code TEXT NULL,
    
    -- Optional contact fields
    phone TEXT NULL,
    email TEXT NULL,
    
    -- Address (simple version for now)
    address TEXT NULL,
    city TEXT NULL,
    country TEXT NULL,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_default BOOLEAN NOT NULL DEFAULT false,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2. Indexing for performance and multi-tenant queries
-- Index to quickly find locations by tenant (critical for SaaS)
CREATE INDEX IF NOT EXISTS locations_tenant_id_idx ON public.locations (tenant_id);

-- Expression index for searching locations by name within a tenant 
CREATE INDEX IF NOT EXISTS locations_tenant_name_idx ON public.locations (tenant_id, lower(name));

-- Optional unique index: Location codes should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS locations_tenant_code_idx ON public.locations (tenant_id, code) WHERE code IS NOT NULL;

-- 3. Triggers for updated_at (reusing set_updated_at function)
DROP TRIGGER IF EXISTS set_locations_updated_at ON public.locations;
CREATE TRIGGER set_locations_updated_at
    BEFORE UPDATE ON public.locations
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- 4. Enable Row Level Security (RLS)
ALTER TABLE public.locations ENABLE ROW LEVEL SECURITY;

-- 5. RLS Policies

-- Read access: Users can view locations only if they belong to the same tenant.
DROP POLICY IF EXISTS "Members can view locations in their tenant" ON public.locations;
CREATE POLICY "Members can view locations in their tenant"
    ON public.locations FOR SELECT
    USING (public.is_tenant_member(tenant_id));

-- Write access: Only tenant owners and admins can create/update/delete locations.
DROP POLICY IF EXISTS "Owners and admins can manage locations" ON public.locations;
CREATE POLICY "Owners and admins can manage locations"
    ON public.locations FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

-- Service role access: Full access for the service role
DROP POLICY IF EXISTS "Service role has full access to locations" ON public.locations;
CREATE POLICY "Service role has full access to locations"
    ON public.locations FOR ALL TO service_role USING (true) WITH CHECK (true);
