-- Migration: Separate Products and Services
-- Replaces public.catalog_items with public.products and public.services

-- 1. Create public.products (Tracks Stock)
CREATE TABLE IF NOT EXISTS public.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Identity fields
    name TEXT NOT NULL,
    description TEXT NULL,
    sku TEXT NULL,
    
    -- Pricing & Inventory
    price NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'COP',
    quantity INT NOT NULL DEFAULT 0,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for Products
CREATE INDEX IF NOT EXISTS products_tenant_id_idx ON public.products (tenant_id);
CREATE INDEX IF NOT EXISTS products_tenant_name_idx ON public.products (tenant_id, lower(name));
CREATE UNIQUE INDEX IF NOT EXISTS products_tenant_sku_idx ON public.products (tenant_id, sku) WHERE sku IS NOT NULL;

-- Triggers for Products
DROP TRIGGER IF EXISTS set_products_updated_at ON public.products;
CREATE TRIGGER set_products_updated_at
    BEFORE UPDATE ON public.products
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for Products
ALTER TABLE public.products ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS "Members can view products in their tenant" ON public.products;
CREATE POLICY "Members can view products in their tenant" ON public.products FOR SELECT USING (public.is_tenant_member(tenant_id));
DROP POLICY IF EXISTS "Staff and above can manage products" ON public.products;
CREATE POLICY "Staff and above can manage products" ON public.products FOR ALL USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff'])) WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));
DROP POLICY IF EXISTS "Service role has full access to products" ON public.products;
CREATE POLICY "Service role has full access to products" ON public.products FOR ALL TO service_role USING (true) WITH CHECK (true);


-- 2. Modify public.services (No Stock) - Table already exists from 000009
ALTER TABLE public.services ADD COLUMN IF NOT EXISTS sku TEXT NULL;

-- Indexes for Services
CREATE UNIQUE INDEX IF NOT EXISTS services_tenant_sku_idx ON public.services (tenant_id, sku) WHERE sku IS NOT NULL;

-- 3. Drop old public.catalog_items table (assuming no data yet)
DROP TABLE IF EXISTS public.catalog_items CASCADE;
