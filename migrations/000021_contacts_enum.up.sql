-- Migration: Add Contact Type Enum
-- Replaces text constraints on contacts table to use an ENUM

-- 1. Create the new ENUM
DO $$ BEGIN
    CREATE TYPE public.contact_type AS ENUM ('customer', 'supplier', 'both', 'lead');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- 2. Drop the old TEXT constraint
ALTER TABLE public.contacts DROP CONSTRAINT IF EXISTS chk_contact_type;

-- 3. Convert the column
ALTER TABLE public.contacts
    ALTER COLUMN contact_type DROP DEFAULT,
    ALTER COLUMN contact_type TYPE public.contact_type
        USING contact_type::public.contact_type,
    ALTER COLUMN contact_type SET DEFAULT 'customer'::public.contact_type;
