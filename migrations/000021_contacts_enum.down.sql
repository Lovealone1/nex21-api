-- Rollback: Remove Contact Type Enum
-- Reverts ENUM column back to TEXT and restores constraint

-- 1. Convert public.contact_type ENUM to TEXT
ALTER TABLE public.contacts
    ALTER COLUMN contact_type DROP DEFAULT,
    ALTER COLUMN contact_type TYPE TEXT
        USING contact_type::text,
    ALTER COLUMN contact_type SET DEFAULT 'customer';

-- 2. Re-apply constraint
ALTER TABLE public.contacts
    ADD CONSTRAINT chk_contact_type 
    CHECK (contact_type IN ('customer', 'supplier', 'both', 'lead'));

-- 3. Drop ENUM type
DROP TYPE IF EXISTS public.contact_type;
