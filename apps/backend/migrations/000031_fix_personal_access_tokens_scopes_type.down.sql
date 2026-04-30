-- Best-effort reversal: convert scopes back to text[] when it is currently
-- JSONB. Loses any non-array JSON values; in practice the column only ever
-- contains arrays of scope strings.
--
-- The DROP DEFAULT step mirrors the up migration: jsonb default cannot be
-- auto-cast to text[].
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

    IF current_type = 'jsonb' THEN
        ALTER TABLE personal_access_tokens
            ALTER COLUMN scopes DROP DEFAULT;
        ALTER TABLE personal_access_tokens
            ALTER COLUMN scopes TYPE text[]
                USING ARRAY(SELECT jsonb_array_elements_text(scopes));
        ALTER TABLE personal_access_tokens
            ALTER COLUMN scopes SET DEFAULT ARRAY[]::text[];
    END IF;
END $$;
