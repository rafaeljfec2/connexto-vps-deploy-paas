-- Align personal_access_tokens.scopes with the type promised by 000029 (JSONB).
--
-- Some production databases were created during a window where the table
-- existed with `scopes text[]` (e.g. provisioned out-of-band before the
-- 000029 migration file landed in main). The Go repository marshals scopes
-- as JSON before the INSERT, so a `text[]` column rejects every write with
-- "invalid input syntax for type text[]" and the API returns HTTP 500.
--
-- This migration is idempotent: it only converts when the current type is
-- ARRAY, so fresh installs (where 000029 already created JSONB) are no-ops.
--
-- The DROP DEFAULT step is required because Postgres cannot auto-cast a
-- `'{}'::text[]` default expression to jsonb during `ALTER COLUMN ... TYPE`
-- (SQLSTATE 42804). We drop the default first, retype the column, then
-- restore the default with the new type.
DO $$
DECLARE
    current_type text;
BEGIN
    SELECT data_type
      INTO current_type
      FROM information_schema.columns
     WHERE table_schema = 'public'
       AND table_name   = 'personal_access_tokens'
       AND column_name  = 'scopes';

    IF current_type = 'ARRAY' THEN
        ALTER TABLE personal_access_tokens
            ALTER COLUMN scopes DROP DEFAULT;
        ALTER TABLE personal_access_tokens
            ALTER COLUMN scopes TYPE JSONB USING to_jsonb(scopes);
        ALTER TABLE personal_access_tokens
            ALTER COLUMN scopes SET DEFAULT '[]'::jsonb;
    END IF;
END $$;
