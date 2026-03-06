-- Rollback: Add Tenant Enums and Booleans
-- Reverts 'is_active' back to 'status' TEXT, and ENUMs back to TEXT

-- 1. Restore helper functions
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

-- 2. Revert memberships table
-- 2a. Revert role ENUM to TEXT
ALTER TABLE public.memberships
    ALTER COLUMN role DROP DEFAULT,
    ALTER COLUMN role TYPE TEXT
        USING role::text,
    ALTER COLUMN role SET DEFAULT 'member';

-- 2b. Revert is_active BOOLEAN to status TEXT
ALTER TABLE public.memberships ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
UPDATE public.memberships SET status = CASE WHEN is_active THEN 'active' ELSE 'inactive' END;
ALTER TABLE public.memberships DROP COLUMN is_active;

-- 3. Revert tenants table
-- 3a. Revert plan ENUM to TEXT
ALTER TABLE public.tenants
    ALTER COLUMN plan DROP DEFAULT,
    ALTER COLUMN plan TYPE TEXT
        USING plan::text,
    ALTER COLUMN plan SET DEFAULT 'free';

-- 3b. Revert is_active BOOLEAN to status TEXT
ALTER TABLE public.tenants ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
UPDATE public.tenants SET status = CASE WHEN is_active THEN 'active' ELSE 'inactive' END;
ALTER TABLE public.tenants DROP COLUMN is_active;

-- 4. Drop ENUM types
-- Assuming no other tables use them!
DROP TYPE IF EXISTS public.membership_role;
DROP TYPE IF EXISTS public.tenant_plan;
