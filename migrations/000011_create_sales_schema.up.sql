-- Migration: Create Sales Schema (POS/ERP)
-- Contains public.sales_orders, public.sales_order_lines, RLS policies, consistency guards, and auto-totals triggers.

-- ==============================================================================
-- PART A: SALES ORDERS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.sales_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES public.locations(id) ON DELETE RESTRICT,
    contact_id UUID NULL REFERENCES public.contacts(id) ON DELETE SET NULL,
    staff_id UUID NULL REFERENCES public.staff(id) ON DELETE SET NULL,
    
    order_number TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    
    -- Money
    subtotal NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_total NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax_total NUMERIC(12,2) NOT NULL DEFAULT 0,
    total NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'COP',
    
    -- Optional
    notes TEXT NULL,
    external_ref TEXT NULL,
    
    -- Audit
    created_by UUID NULL REFERENCES public.profiles(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_sales_order_status CHECK (status IN ('draft', 'confirmed', 'paid', 'cancelled', 'refunded')),
    CONSTRAINT chk_sales_order_amounts CHECK (subtotal >= 0 AND discount_total >= 0 AND tax_total >= 0 AND total >= 0),
    CONSTRAINT uq_tenant_order_number UNIQUE (tenant_id, order_number)
);

-- Indexing for SaaS performance (sales_orders)
CREATE INDEX IF NOT EXISTS sales_orders_tenant_id_idx ON public.sales_orders(tenant_id);
CREATE INDEX IF NOT EXISTS sales_orders_tenant_location_created_idx ON public.sales_orders(tenant_id, location_id, created_at DESC);
CREATE INDEX IF NOT EXISTS sales_orders_tenant_status_created_idx ON public.sales_orders(tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS sales_orders_tenant_contact_created_idx ON public.sales_orders(tenant_id, contact_id, created_at DESC);

-- Trigger for update_at
DROP TRIGGER IF EXISTS set_sales_orders_updated_at ON public.sales_orders;
CREATE TRIGGER set_sales_orders_updated_at
    BEFORE UPDATE ON public.sales_orders
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for sales_orders
ALTER TABLE public.sales_orders ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view sales_orders in their tenant" ON public.sales_orders;
CREATE POLICY "Members can view sales_orders in their tenant"
    ON public.sales_orders FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Staff and above can manage sales_orders" ON public.sales_orders;
CREATE POLICY "Staff and above can manage sales_orders"
    ON public.sales_orders FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

DROP POLICY IF EXISTS "Service role has full access to sales_orders" ON public.sales_orders;
CREATE POLICY "Service role has full access to sales_orders"
    ON public.sales_orders FOR ALL TO service_role USING (true) WITH CHECK (true);

-- ==============================================================================
-- PART B: SALES ORDER LINES
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.sales_order_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    sales_order_id UUID NOT NULL REFERENCES public.sales_orders(id) ON DELETE CASCADE,
    catalog_item_id UUID NOT NULL REFERENCES public.catalog_items(id) ON DELETE RESTRICT,
    
    -- Line values
    quantity NUMERIC(12,2) NOT NULL DEFAULT 1,
    unit_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_total NUMERIC(12,2) NOT NULL DEFAULT 0,
    
    -- Optional
    notes TEXT NULL,
    
    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    CONSTRAINT chk_sol_quantity CHECK (quantity > 0),
    CONSTRAINT chk_sol_unit_price CHECK (unit_price >= 0),
    CONSTRAINT chk_sol_discount CHECK (discount >= 0),
    CONSTRAINT chk_sol_tax CHECK (tax >= 0),
    CONSTRAINT chk_sol_line_total CHECK (line_total >= 0),
    CONSTRAINT uq_sales_order_catalog_item UNIQUE (sales_order_id, catalog_item_id)
);

-- Indexing for SaaS performance (sales_order_lines)
CREATE INDEX IF NOT EXISTS sales_order_lines_order_idx ON public.sales_order_lines(sales_order_id);
CREATE INDEX IF NOT EXISTS sales_order_lines_tenant_item_idx ON public.sales_order_lines(tenant_id, catalog_item_id);

-- Trigger for update_at
DROP TRIGGER IF EXISTS set_sales_order_lines_updated_at ON public.sales_order_lines;
CREATE TRIGGER set_sales_order_lines_updated_at
    BEFORE UPDATE ON public.sales_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();

-- RLS for sales_order_lines
ALTER TABLE public.sales_order_lines ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Members can view sales_order_lines in their tenant" ON public.sales_order_lines;
CREATE POLICY "Members can view sales_order_lines in their tenant"
    ON public.sales_order_lines FOR SELECT
    USING (public.is_tenant_member(tenant_id));

DROP POLICY IF EXISTS "Staff and above can manage sales_order_lines" ON public.sales_order_lines;
CREATE POLICY "Staff and above can manage sales_order_lines"
    ON public.sales_order_lines FOR ALL
    USING (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']))
    WITH CHECK (public.has_tenant_role(tenant_id, ARRAY['owner', 'admin', 'staff']));

DROP POLICY IF EXISTS "Service role has full access to sales_order_lines" ON public.sales_order_lines;
CREATE POLICY "Service role has full access to sales_order_lines"
    ON public.sales_order_lines FOR ALL TO service_role USING (true) WITH CHECK (true);

-- ==============================================================================
-- PART C: CROSS-TENANT SAFETY GUARDS
-- ==============================================================================

-- 1. Validate Sales Order Consistency
CREATE OR REPLACE FUNCTION public.check_sales_order_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_location_tenant UUID;
    v_contact_tenant UUID;
    v_staff_tenant UUID;
BEGIN
    -- Check Location
    SELECT tenant_id INTO v_location_tenant FROM public.locations WHERE id = NEW.location_id;
    IF v_location_tenant != NEW.tenant_id THEN 
        RAISE EXCEPTION 'Location does not belong to the same tenant' USING ERRCODE = 'P0002'; 
    END IF;

    -- Check Contact (if present)
    IF NEW.contact_id IS NOT NULL THEN
        SELECT tenant_id INTO v_contact_tenant FROM public.contacts WHERE id = NEW.contact_id;
        IF v_contact_tenant != NEW.tenant_id THEN 
            RAISE EXCEPTION 'Contact does not belong to the same tenant' USING ERRCODE = 'P0002'; 
        END IF;
    END IF;

    -- Check Staff (if present)
    IF NEW.staff_id IS NOT NULL THEN
        SELECT tenant_id INTO v_staff_tenant FROM public.staff WHERE id = NEW.staff_id;
        IF v_staff_tenant != NEW.tenant_id THEN 
            RAISE EXCEPTION 'Staff member does not belong to the same tenant' USING ERRCODE = 'P0002'; 
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_sales_order_consistency ON public.sales_orders;
CREATE TRIGGER validate_sales_order_consistency
    BEFORE INSERT OR UPDATE ON public.sales_orders
    FOR EACH ROW
    EXECUTE FUNCTION public.check_sales_order_tenant_consistency();

-- 2. Validate Sales Order Lines Consistency
CREATE OR REPLACE FUNCTION public.check_sales_order_line_tenant_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_order_tenant UUID;
    v_catalog_tenant UUID;
BEGIN
    -- The tenant_id of the line MUST match the tenant_id of the parent order
    SELECT tenant_id INTO v_order_tenant FROM public.sales_orders WHERE id = NEW.sales_order_id;
    IF v_order_tenant != NEW.tenant_id THEN
        RAISE EXCEPTION 'Sales order line tenant_id must match parent sales order tenant_id' USING ERRCODE = 'P0002';
    END IF;

    -- Check Catalog Item
    SELECT tenant_id INTO v_catalog_tenant FROM public.catalog_items WHERE id = NEW.catalog_item_id;
    IF v_catalog_tenant != NEW.tenant_id THEN 
        RAISE EXCEPTION 'Catalog item does not belong to the same tenant' USING ERRCODE = 'P0002'; 
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS validate_sales_order_line_consistency ON public.sales_order_lines;
CREATE TRIGGER validate_sales_order_line_consistency
    BEFORE INSERT OR UPDATE ON public.sales_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION public.check_sales_order_line_tenant_consistency();

-- ==============================================================================
-- PART D: TOTALS MANAGEMENT (OPTION 1 - DB COMPUTED)
-- ==============================================================================

-- Recalculates the parent sales_order totals automatically whenever a line is added, updated, or removed.
-- Recommended for database integrity to ensure header and lines are historically perfectly balanced.
CREATE OR REPLACE FUNCTION public.recalculate_sales_order_totals()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_order_id UUID;
    v_subtotal NUMERIC(12,2) := 0;
    v_discount NUMERIC(12,2) := 0;
    v_tax NUMERIC(12,2) := 0;
    v_total NUMERIC(12,2) := 0;
BEGIN
    -- Determine the order ID depending on the operation
    IF TG_OP = 'DELETE' THEN
        v_order_id := OLD.sales_order_id;
    ELSE
        v_order_id := NEW.sales_order_id;
    END IF;

    -- Aggregate the lines
    SELECT 
        COALESCE(SUM(quantity * unit_price), 0),
        COALESCE(SUM(discount), 0),
        COALESCE(SUM(tax), 0),
        COALESCE(SUM(line_total), 0)
    INTO
        v_subtotal,
        v_discount,
        v_tax,
        v_total
    FROM public.sales_order_lines
    WHERE sales_order_id = v_order_id;

    -- Update the parent sales_order
    UPDATE public.sales_orders
    SET 
        subtotal = v_subtotal,
        discount_total = v_discount,
        tax_total = v_tax,
        total = v_total
    WHERE id = v_order_id;

    RETURN NULL; -- AFTER trigger can return NULL
END;
$$;

DROP TRIGGER IF EXISTS trigger_recalculate_sales_order_totals ON public.sales_order_lines;
CREATE TRIGGER trigger_recalculate_sales_order_totals
    AFTER INSERT OR UPDATE OR DELETE ON public.sales_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION public.recalculate_sales_order_totals();

-- ==============================================================================
-- END OF MIGRATION
-- ==============================================================================
