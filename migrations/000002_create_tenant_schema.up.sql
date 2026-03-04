-- Migration: Create Tenant Schema (SaaS Core)
-- Contains public.tenants, public.tenant_domains, public.memberships, and RLS policies

-- 1. Create public.tenants
CREATE TABLE IF NOT EXISTS public.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    plan TEXT NOT NULL DEFAULT 'free',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2. Create public.tenant_domains
CREATE TABLE IF NOT EXISTS public.tenant_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    domain TEXT NOT NULL UNIQUE,
    domain_type TEXT NOT NULL DEFAULT 'subdomain',
    is_primary BOOLEAN NOT NULL DEFAULT true,
    verified_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Constraint: Only one primary domain per tenant
CREATE UNIQUE INDEX IF NOT EXISTS tenant_domains_primary_idx 
    ON public.tenant_domains (tenant_id) 
    WHERE is_primary = true;

-- 3. Create public.memberships
CREATE TABLE IF NOT EXISTS public.memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES public.profiles(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, user_id)
);

-- Indexing for fast lookups (Auth & RLS performance)
CREATE INDEX IF NOT EXISTS memberships_user_id_idx ON public.memberships (user_id);
CREATE INDEX IF NOT EXISTS memberships_tenant_id_idx ON public.memberships (tenant_id);

-- 4. Triggers for updated_at (reusing set_updated_at from migration 000001)
DROP TRIGGER IF EXISTS set_tenants_updated_at ON public.tenants;
CREATE TRIGGER set_tenants_updated_at
    BEFORE UPDATE ON public.tenants
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

DROP TRIGGER IF EXISTS set_tenant_domains_updated_at ON public.tenant_domains;
CREATE TRIGGER set_tenant_domains_updated_at
    BEFORE UPDATE ON public.tenant_domains
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

DROP TRIGGER IF EXISTS set_memberships_updated_at ON public.memberships;
CREATE TRIGGER set_memberships_updated_at
    BEFORE UPDATE ON public.memberships
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- 5. RLS Helper Functions (SECURITY DEFINER to avoid infinite recursion in policies)
CREATE OR REPLACE FUNCTION public.is_tenant_member(p_tenant_id uuid)
RETURNS boolean AS $$
    SELECT EXISTS (
        SELECT 1 FROM public.memberships
        WHERE tenant_id = p_tenant_id
        AND user_id = auth.uid()
        AND status = 'active'
    );
$$ LANGUAGE sql SECURITY DEFINER STABLE SET search_path = public;

CREATE OR REPLACE FUNCTION public.has_tenant_role(p_tenant_id uuid, roles text[])
RETURNS boolean AS $$
    SELECT EXISTS (
        SELECT 1 FROM public.memberships
        WHERE tenant_id = p_tenant_id
        AND user_id = auth.uid()
        AND status = 'active'
        AND role = ANY(roles)
    );
$$ LANGUAGE sql SECURITY DEFINER STABLE SET search_path = public;

-- 6. Enable Row Level Security (RLS)
ALTER TABLE public.tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_domains ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.memberships ENABLE ROW LEVEL SECURITY;

-- Tenants Policies
DROP POLICY IF EXISTS "Members can view their tenants" ON public.tenants;
CREATE POLICY "Members can view their tenants"
    ON public.tenants FOR SELECT
    USING (public.is_tenant_member(id));

DROP POLICY IF EXISTS "Owners and Admins can update their tenants" ON public.tenants;
CREATE POLICY "Owners and Admins can update their tenants"
    ON public.tenants FOR UPDATE
    USING (public.has_tenant_role(id, ARRAY['owner', 'admin']));

DROP POLICY IF EXISTS "Service role has full access to tenants" ON public.tenants;
CREATE POLICY "Service role has full access to tenants"
    ON public.tenants FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Tenant Domains Policies
DROP POLICY IF EXISTS "Members can view domains" ON public.tenant_domains;
CREATE POLICY "Members can view domains"
    ON public.tenant_domains FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owners and Admins can manage domains" ON public.tenant_domains;
CREATE POLICY "Owners and Admins can manage domains"
    ON public.tenant_domains FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

DROP POLICY IF EXISTS "Service role has full access to domains" ON public.tenant_domains;
CREATE POLICY "Service role has full access to domains"
    ON public.tenant_domains FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Memberships Policies
-- Note: A user can read a membership if they are a member of the SAME tenant
DROP POLICY IF EXISTS "Members can view other members in same tenant" ON public.memberships;
CREATE POLICY "Members can view other members in same tenant"
    ON public.memberships FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Owners and Admins can manage memberships" ON public.memberships;
CREATE POLICY "Owners and Admins can manage memberships"
    ON public.memberships FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin']));

DROP POLICY IF EXISTS "Service role has full access to memberships" ON public.memberships;
CREATE POLICY "Service role has full access to memberships"
    ON public.memberships FOR ALL TO service_role USING (true) WITH CHECK (true);
