-- Migration: Add Tenant Enums and Booleans
-- Upgrades 'status' TEXT to 'is_active' BOOLEAN, and TEXT roles/plans to ENUMs

-- 1. Create ENUM types
DO $$ BEGIN
    CREATE TYPE public.tenant_plan AS ENUM ('free', 'pro', 'enterprise');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE public.membership_role AS ENUM ('owner', 'admin', 'staff', 'member');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- 2. Migrate tenants table
-- 2a. Convert status to is_active
ALTER TABLE public.tenants ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
UPDATE public.tenants SET is_active = (status = 'active');
ALTER TABLE public.tenants DROP COLUMN status;

-- 2b. Convert plan TEXT to tenant_plan ENUM
ALTER TABLE public.tenants
    ALTER COLUMN plan DROP DEFAULT,
    ALTER COLUMN plan TYPE public.tenant_plan
        USING plan::public.tenant_plan,
    ALTER COLUMN plan SET DEFAULT 'free'::public.tenant_plan;

-- 3. Migrate memberships table
-- 3a. Convert status to is_active
ALTER TABLE public.memberships ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
UPDATE public.memberships SET is_active = (status = 'active');
ALTER TABLE public.memberships DROP COLUMN status;

-- 3b. Convert role TEXT to membership_role ENUM
ALTER TABLE public.memberships
    ALTER COLUMN role DROP DEFAULT,
    ALTER COLUMN role TYPE public.membership_role
        USING role::public.membership_role,
    ALTER COLUMN role SET DEFAULT 'member'::public.membership_role;

-- 4. Update helper functions with boolean states
CREATE OR REPLACE FUNCTION public.is_tenant_member(p_tenant_id uuid)
RETURNS boolean AS $$
    SELECT EXISTS (
        SELECT 1 FROM public.memberships
        WHERE tenant_id = p_tenant_id
        AND user_id = auth.uid()
        AND is_active = true
    );
$$ LANGUAGE sql SECURITY DEFINER STABLE SET search_path = public;

CREATE OR REPLACE FUNCTION public.has_tenant_role(p_tenant_id uuid, roles text[])
RETURNS boolean AS $$
    SELECT EXISTS (
        SELECT 1 FROM public.memberships
        WHERE tenant_id = p_tenant_id
        AND user_id = auth.uid()
        AND is_active = true
        AND role::text = ANY(roles)
    );
$$ LANGUAGE sql SECURITY DEFINER STABLE SET search_path = public;
