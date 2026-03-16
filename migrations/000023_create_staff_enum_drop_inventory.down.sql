-- Revert: Create Staff Role ENUM and Drop Inventory Items
-- 1. Revert staff_role column to TEXT with check constraint
ALTER TABLE public.staff ALTER COLUMN staff_role SET DEFAULT 'staff';
ALTER TABLE public.staff 
    ALTER COLUMN staff_role TYPE TEXT 
    USING staff_role::TEXT;

ALTER TABLE public.staff ADD CONSTRAINT chk_staff_role CHECK (staff_role IN ('owner', 'admin', 'staff'));

-- 2. Drop the ENUM type
DROP TYPE IF EXISTS public.staff_role_type;

-- 3. Note: inventory_items table is NOT recreated here as it would require complex data recovery.
-- This rollback focuses on the type system of the staff table.
