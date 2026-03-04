-- Migration: Create Contacts Schema (CRM Base Entity)
-- Contains public.contacts and RLS policies

-- 1. Create public.contacts
CREATE TABLE IF NOT EXISTS public.contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Identity fields
    name TEXT NOT NULL,
    email TEXT NULL,
    phone TEXT NULL,
    
    -- Classification & CRM fields
    company_name TEXT NULL,
    contact_type TEXT NOT NULL DEFAULT 'customer',
    lifecycle_stage TEXT DEFAULT 'lead',
    notes TEXT NULL,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_contact_type CHECK (contact_type IN ('customer', 'supplier', 'both', 'lead')),
    CONSTRAINT chk_lifecycle_stage CHECK (lifecycle_stage IN ('lead', 'prospect', 'customer', 'inactive'))
);

-- 2. Indexing for performance and multi-tenant queries
-- Index to quickly find contacts by tenant (critical for SaaS)
CREATE INDEX IF NOT EXISTS contacts_tenant_id_idx ON public.contacts (tenant_id);

-- Composite index for searching by name within a tenant (using lower() for case-insensitive search if needed, but standard B-Tree covers exact matches. Adding an expression index for lowercased name specifically for case-insensitive LIKE/ILIKE searches or exact matches)
CREATE INDEX IF NOT EXISTS contacts_tenant_id_name_idx ON public.contacts (tenant_id, lower(name));

-- Indexes for finding contacts globally or cross-tenant by email/phone
CREATE INDEX IF NOT EXISTS contacts_email_idx ON public.contacts (email);
CREATE INDEX IF NOT EXISTS contacts_phone_idx ON public.contacts (phone);

-- 3. Triggers for updated_at (reusing set_updated_at function)
DROP TRIGGER IF EXISTS set_contacts_updated_at ON public.contacts;
CREATE TRIGGER set_contacts_updated_at
    BEFORE UPDATE ON public.contacts
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- 4. Enable Row Level Security (RLS)
ALTER TABLE public.contacts ENABLE ROW LEVEL SECURITY;

-- 5. RLS Policies

-- Read access: Users can view contacts only if they belong to the same tenant.
DROP POLICY IF EXISTS "Members can view contacts in their tenant" ON public.contacts;
CREATE POLICY "Members can view contacts in their tenant"
    ON public.contacts FOR SELECT
    USING (public.is_tenant_member(tenant_id));

-- Write access (insert/update/delete): Only tenant owners, admins, and staff can modify contacts.
DROP POLICY IF EXISTS "Staff and above can manage contacts" ON public.contacts;
CREATE POLICY "Staff and above can manage contacts"
    ON public.contacts FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

-- Service role access: Full access for the service role
DROP POLICY IF EXISTS "Service role has full access to contacts" ON public.contacts;
CREATE POLICY "Service role has full access to contacts"
    ON public.contacts FOR ALL TO service_role USING (true) WITH CHECK (true);
