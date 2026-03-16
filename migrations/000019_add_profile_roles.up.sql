-- Migration: Add Role Enum and Fix Profile Trigger
-- Replaces the plain TEXT 'role' column with a strongly-typed ENUM
-- and fixes handle_new_auth_user to properly read the role from user metadata.

-- 1. Create the role ENUM type
DO $$ BEGIN
    CREATE TYPE public.profile_role AS ENUM ('owner', 'admin', 'staff', 'member');
EXCEPTION
    WHEN duplicate_object THEN NULL; -- idempotent: ignore if already exists
END $$;

-- 2. Alter the profiles table to use the new ENUM
--    We cast the existing TEXT values to profile_role; any invalid value will error here.
ALTER TABLE public.profiles
    ALTER COLUMN role DROP DEFAULT,
    ALTER COLUMN role TYPE public.profile_role
        USING role::public.profile_role,
    ALTER COLUMN role SET DEFAULT 'member'::public.profile_role,
    ALTER COLUMN role SET NOT NULL;

-- 3. Fix the trigger: now reads 'role' from raw_user_meta_data
--    Falls back to 'member' if the metadata key is absent or empty.
CREATE OR REPLACE FUNCTION public.handle_new_auth_user()
RETURNS TRIGGER AS $$
DECLARE
    v_role public.profile_role;
BEGIN
    -- Safely parse the role from metadata; fall back to 'member' if invalid or absent.
    BEGIN
        v_role := COALESCE(
            NULLIF(NEW.raw_user_meta_data->>'role', ''),
            'member'
        )::public.profile_role;
    EXCEPTION WHEN invalid_text_representation THEN
        v_role := 'member'::public.profile_role;
    END;

    INSERT INTO public.profiles (
        id,
        email,
        created_at,
        updated_at,
        tenant_id,
        full_name,
        avatar_url,
        role
    )
    VALUES (
        NEW.id,
        NEW.email,
        now(),
        now(),
        NULLIF(NEW.raw_user_meta_data->>'tenant_id', '')::UUID,
        NEW.raw_user_meta_data->>'full_name',
        NEW.raw_user_meta_data->>'avatar_url',
        v_role
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Re-attach trigger (it was already created in 000001, but the function is now updated)
DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;
CREATE TRIGGER on_auth_user_created
AFTER INSERT ON auth.users
FOR EACH ROW
EXECUTE FUNCTION public.handle_new_auth_user();
