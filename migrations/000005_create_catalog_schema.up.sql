-- Migration: Create Catalog Schema (catalog items)
-- Contains public.catalog_items and RLS policies

-- 1. Create public.catalog_items
CREATE TABLE IF NOT EXISTS public.catalog_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Classification
    item_type TEXT NOT NULL,
    
    -- Identity fields
    name TEXT NOT NULL,
    description TEXT NULL,
    sku TEXT NULL,
    
    -- Pricing
    price NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'COP',
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_item_type CHECK (item_type IN ('product', 'service'))
);

-- 2. Indexing for performance and multi-tenant queries
-- Index to quickly find catalog items by tenant (critical for SaaS)
CREATE INDEX IF NOT EXISTS catalog_items_tenant_id_idx ON public.catalog_items (tenant_id);

-- Composite index for finding specific types of items within a tenant
CREATE INDEX IF NOT EXISTS catalog_items_tenant_type_idx ON public.catalog_items (tenant_id, item_type);

-- Expression index for searching items by name within a tenant 
CREATE INDEX IF NOT EXISTS catalog_items_tenant_name_idx ON public.catalog_items (tenant_id, lower(name));

-- Optional unique index: Skus should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS catalog_items_tenant_sku_idx ON public.catalog_items (tenant_id, sku) WHERE sku IS NOT NULL;

-- 3. Triggers for updated_at (reusing set_updated_at function)
DROP TRIGGER IF EXISTS set_catalog_items_updated_at ON public.catalog_items;
CREATE TRIGGER set_catalog_items_updated_at
    BEFORE UPDATE ON public.catalog_items
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- 4. Enable Row Level Security (RLS)
ALTER TABLE public.catalog_items ENABLE ROW LEVEL SECURITY;

-- 5. RLS Policies

-- Read access: Users can read catalog items only if they belong to the same tenant.
DROP POLICY IF EXISTS "Members can view catalog items in their tenant" ON public.catalog_items;
CREATE POLICY "Members can view catalog items in their tenant"
    ON public.catalog_items FOR SELECT
    USING (public.is_tenant_member(tenant_id));

-- Write access: Only tenant owners, admins, and staff can insert/update/delete catalog items.
DROP POLICY IF EXISTS "Staff and above can manage catalog items" ON public.catalog_items;
CREATE POLICY "Staff and above can manage catalog items"
    ON public.catalog_items FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

-- Service role access: Full access for the service role
DROP POLICY IF EXISTS "Service role has full access to catalog items" ON public.catalog_items;
CREATE POLICY "Service role has full access to catalog items"
    ON public.catalog_items FOR ALL TO service_role USING (true) WITH CHECK (true);
