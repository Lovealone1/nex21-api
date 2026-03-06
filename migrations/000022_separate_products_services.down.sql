-- Rollback: Separate Products and Services
-- Reverts public.products and public.services back into public.catalog_items

-- 1. Drop the products table and revert services
DROP TABLE IF EXISTS public.products CASCADE;
ALTER TABLE public.services DROP COLUMN IF EXISTS sku CASCADE;

-- 2. Recreate public.catalog_items
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

-- Recreate indexes
CREATE INDEX IF NOT EXISTS catalog_items_tenant_id_idx ON public.catalog_items (tenant_id);
CREATE INDEX IF NOT EXISTS catalog_items_tenant_type_idx ON public.catalog_items (tenant_id, item_type);
CREATE INDEX IF NOT EXISTS catalog_items_tenant_name_idx ON public.catalog_items (tenant_id, lower(name));
CREATE UNIQUE INDEX IF NOT EXISTS catalog_items_tenant_sku_idx ON public.catalog_items (tenant_id, sku) WHERE sku IS NOT NULL;

-- Recreate triggers
DROP TRIGGER IF EXISTS set_catalog_items_updated_at ON public.catalog_items;
CREATE TRIGGER set_catalog_items_updated_at
    BEFORE UPDATE ON public.catalog_items
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- Recreate RLS
ALTER TABLE public.catalog_items ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS "Members can view catalog items in their tenant" ON public.catalog_items;
CREATE POLICY "Members can view catalog items in their tenant" ON public.catalog_items FOR SELECT USING (public.is_tenant_member(tenant_id));
DROP POLICY IF EXISTS "Staff and above can manage catalog items" ON public.catalog_items;
CREATE POLICY "Staff and above can manage catalog items" ON public.catalog_items FOR ALL USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff'])) WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));
DROP POLICY IF EXISTS "Service role has full access to catalog items" ON public.catalog_items;
CREATE POLICY "Service role has full access to catalog items" ON public.catalog_items FOR ALL TO service_role USING (true) WITH CHECK (true);
