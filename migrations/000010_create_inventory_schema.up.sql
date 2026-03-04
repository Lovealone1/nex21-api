-- Migration: Create Inventory Schema
-- Contains public.inventory_items, RLS policies, and strict cross-tenant guards

-- 1. Create public.inventory_items
CREATE TABLE IF NOT EXISTS public.inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES public.locations(id) ON DELETE CASCADE,
    catalog_item_id UUID NOT NULL REFERENCES public.catalog_items(id) ON DELETE RESTRICT,
    
    -- Stock fields
    quantity NUMERIC(12,2) NOT NULL DEFAULT 0,
    min_quantity NUMERIC(12,2) NOT NULL DEFAULT 0,
    
    -- Optional fields
    unit_cost NUMERIC(12,2) NULL,
    last_counted_at TIMESTAMPTZ NULL,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_inventory_quantity CHECK (quantity >= 0),
    CONSTRAINT chk_inventory_min_quantity CHECK (min_quantity >= 0),
    CONSTRAINT uq_inventory_location_item UNIQUE (tenant_id, location_id, catalog_item_id)
);

-- 2. Indexing for performance and multi-tenant queries (ERP usage)
CREATE INDEX IF NOT EXISTS inventory_items_tenant_id_idx ON public.inventory_items (tenant_id);
CREATE INDEX IF NOT EXISTS inventory_items_tenant_location_idx ON public.inventory_items (tenant_id, location_id);
CREATE INDEX IF NOT EXISTS inventory_items_tenant_catalog_item_idx ON public.inventory_items (tenant_id, catalog_item_id);

-- 3. Triggers for updated_at
DROP TRIGGER IF EXISTS set_inventory_items_updated_at ON public.inventory_items;
CREATE TRIGGER set_inventory_items_updated_at
    BEFORE UPDATE ON public.inventory_items
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- 4. Enable Row Level Security (RLS)
ALTER TABLE public.inventory_items ENABLE ROW LEVEL SECURITY;

-- 5. RLS Policies

-- Read access
DROP POLICY IF EXISTS "Members can view inventory in their tenant" ON public.inventory_items;
CREATE POLICY "Members can view inventory in their tenant"
    ON public.inventory_items FOR SELECT
    USING (public.is_tenant_member(tenant_id));

-- Write access: Owner, Admin, and Staff can manage inventory
DROP POLICY IF EXISTS "Staff and above can manage inventory" ON public.inventory_items;
CREATE POLICY "Staff and above can manage inventory"
    ON public.inventory_items FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

-- Service role access
DROP POLICY IF EXISTS "Service role has full access to inventory" ON public.inventory_items;
CREATE POLICY "Service role has full access to inventory"
    ON public.inventory_items FOR ALL TO service_role USING (true) WITH CHECK (true);

-- 6. CROSS-TENANT DATA CONTAMINATION GUARD
-- A strict trigger that ensures catalog_items and locations belong exactly to the same tenant 
-- reported by the inventory_item row.
CREATE OR REPLACE FUNCTION public.check_inventory_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_catalog_tenant UUID;
    v_location_tenant UUID;
BEGIN
    -- Check Catalog Item (Product)
    SELECT tenant_id INTO v_catalog_tenant FROM public.catalog_items WHERE id = NEW.catalog_item_id;
    IF v_catalog_tenant != NEW.tenant_id THEN 
        RAISE EXCEPTION 'Catalog item does not belong to the same tenant' USING ERRCODE = 'P0002'; 
    END IF;
    
    -- Check Location
    SELECT tenant_id INTO v_location_tenant FROM public.locations WHERE id = NEW.location_id;
    IF v_location_tenant != NEW.tenant_id THEN 
        RAISE EXCEPTION 'Location does not belong to the same tenant' USING ERRCODE = 'P0002'; 
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_inventory_consistency ON public.inventory_items;
CREATE TRIGGER validate_inventory_consistency
    BEFORE INSERT OR UPDATE ON public.inventory_items
    FOR EACH ROW
    EXECUTE FUNCTION public.check_inventory_tenant_consistency();

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
