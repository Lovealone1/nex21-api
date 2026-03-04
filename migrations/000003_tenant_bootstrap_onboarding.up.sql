-- Migration: Tenant Bootstrap Onboarding
-- Contains the RPC for transactional tenant creation and helper for domain resolution

-- 1. Bootstrap Tenant Function
CREATE OR REPLACE FUNCTION public.bootstrap_tenant(
    p_name text,
    p_slug text,
    p_domain text
)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_user_id uuid;
    v_tenant_id uuid;
    v_slug text;
    v_domain text;
BEGIN
    -- 1. Authentication gate
    v_user_id := auth.uid();
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'Not authenticated' USING ERRCODE = 'P0001'; -- Using custom code or standard exception
    END IF;

    -- 2. Normalize inputs
    v_slug := lower(trim(p_slug));
    v_domain := lower(trim(p_domain));

    IF p_name IS NULL OR trim(p_name) = '' THEN
        RAISE EXCEPTION 'Tenant name cannot be empty';
    END IF;

    IF v_slug IS NULL OR v_slug = '' OR v_slug !~ '^[a-z0-9-]+$' THEN
        RAISE EXCEPTION 'Invalid slug format. Use only lowercase letters, numbers, and hyphens.';
    END IF;

    IF v_domain IS NULL OR v_domain = '' THEN
        RAISE EXCEPTION 'Domain cannot be empty';
    END IF;

    -- 3. Idempotency Check
    -- Check if this specific user already owns a tenant with the given slug or domain.
    -- If so, return the existing tenant_id to make the function idempotent for retries.
    SELECT t.id INTO v_tenant_id
    FROM public.tenants t
    JOIN public.memberships m ON t.id = m.tenant_id
    LEFT JOIN public.tenant_domains td ON t.id = td.tenant_id
    WHERE m.user_id = v_user_id
        AND m.role = 'owner'
        AND m.status = 'active'
        AND (t.slug = v_slug OR td.domain = v_domain)
    LIMIT 1;

    IF v_tenant_id IS NOT NULL THEN
        RETURN v_tenant_id;
    END IF;

    -- Ensure the slug or domain are not taken by someone else
    IF EXISTS (SELECT 1 FROM public.tenants WHERE slug = v_slug) THEN
        RAISE EXCEPTION 'Slug is already taken' USING ERRCODE = 'unique_violation';
    END IF;

    IF EXISTS (SELECT 1 FROM public.tenant_domains WHERE domain = v_domain) THEN
        RAISE EXCEPTION 'Domain is already taken' USING ERRCODE = 'unique_violation';
    END IF;

    -- 4. Transactional Creation
    -- Insert tenant
    INSERT INTO public.tenants (name, slug, status)
    VALUES (trim(p_name), v_slug, 'active')
    RETURNING id INTO v_tenant_id;

    -- Insert primary domain
    INSERT INTO public.tenant_domains (tenant_id, domain, domain_type, is_primary)
    VALUES (v_tenant_id, v_domain, 'subdomain', true);

    -- Insert owner membership
    INSERT INTO public.memberships (tenant_id, user_id, role, status)
    VALUES (v_tenant_id, v_user_id, 'owner', 'active');

    RETURN v_tenant_id;
END;
$$;

-- Secure the function: Remove public execution rights, grant to authenticated only
REVOKE ALL ON FUNCTION public.bootstrap_tenant(text, text, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.bootstrap_tenant(text, text, text) TO authenticated;

-- 2. Domain Resolution Helper
-- Justification: A function is preferred here because domain resolution typically happens
-- unauthenticated (e.g., to load tenant branding prior to login or API calls). Using a SECURITY DEFINER
-- function allows us to safely expose just the tenant_id for an exact domain match without
-- bypassing or complicating RLS on the underlying tables for the anonymous role.
CREATE OR REPLACE FUNCTION public.resolve_tenant_id_by_domain(p_domain text)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_tenant_id uuid;
    v_domain text;
BEGIN
    v_domain := lower(trim(p_domain));

    SELECT td.tenant_id INTO v_tenant_id
    FROM public.tenant_domains td
    JOIN public.tenants t ON td.tenant_id = t.id
    WHERE td.domain = v_domain
      AND t.status = 'active'
    LIMIT 1;

    RETURN v_tenant_id;
END;
$$;

-- Allow anon and authenticated to resolve tenant domains
REVOKE ALL ON FUNCTION public.resolve_tenant_id_by_domain(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.resolve_tenant_id_by_domain(text) TO anon, authenticated;
