-- Rollback: Remove Role Enum and revert trigger

-- 1. Revert the trigger back to its original (no role handling)
CREATE OR REPLACE FUNCTION public.handle_new_auth_user()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO public.profiles (
        id,
        email,
        created_at,
        updated_at,
        tenant_id,
        full_name,
        avatar_url
    )
    VALUES (
        NEW.id,
        NEW.email,
        now(),
        now(),
        NULLIF(NEW.raw_user_meta_data->>'tenant_id', '')::UUID,
        NEW.raw_user_meta_data->>'full_name',
        NEW.raw_user_meta_data->>'avatar_url'
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;
CREATE TRIGGER on_auth_user_created
AFTER INSERT ON auth.users
FOR EACH ROW
EXECUTE FUNCTION public.handle_new_auth_user();

-- 2. Revert the role column back to TEXT
ALTER TABLE public.profiles
    ALTER COLUMN role DROP DEFAULT,
    ALTER COLUMN role TYPE TEXT
        USING role::TEXT,
    ALTER COLUMN role SET DEFAULT 'member',
    ALTER COLUMN role SET NOT NULL;

-- 3. Drop the ENUM type
DROP TYPE IF EXISTS public.profile_role;
