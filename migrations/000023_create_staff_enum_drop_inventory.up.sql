-- Migration: Create Staff Role ENUM and Drop Inventory Items (Fixed Idempotent)

-- 1. Create the ENUM type for staff roles
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'staff_role_type') THEN
        CREATE TYPE public.staff_role_type AS ENUM ('owner', 'admin', 'manager', 'staff', 'receptionist');
    END IF;
END $$;

-- 2. Drop the obsolete inventory_items table
DROP TABLE IF EXISTS public.inventory_items CASCADE;
DROP FUNCTION IF EXISTS public.check_inventory_tenant_consistency CASCADE;

-- 3. Refactor the staff table to use the new ENUM
-- Drop default first to avoid type mismatch
ALTER TABLE public.staff ALTER COLUMN staff_role DROP DEFAULT;
-- Drop old check constraint
ALTER TABLE public.staff DROP CONSTRAINT IF EXISTS chk_staff_role;

-- Convert only if it's still text
DO $$ 
DECLARE
    v_type text;
BEGIN
    SELECT data_type INTO v_type 
    FROM information_schema.columns 
    WHERE table_name = 'staff' AND column_name = 'staff_role';
    
    IF v_type = 'text' THEN
        ALTER TABLE public.staff 
            ALTER COLUMN staff_role TYPE public.staff_role_type 
            USING staff_role::public.staff_role_type;
    END IF;
END $$;

-- Set the new default
ALTER TABLE public.staff ALTER COLUMN staff_role SET DEFAULT 'staff'::public.staff_role_type;
