-- migrate:up
ALTER TABLE crop ADD COLUMN name_de TEXT;
ALTER TABLE crop ADD COLUMN name_en TEXT;

UPDATE crop SET name_de = name, name_en = name;

ALTER TABLE crop ALTER COLUMN name_de SET NOT NULL;
ALTER TABLE crop ALTER COLUMN name_en SET NOT NULL;

ALTER TABLE crop ADD CONSTRAINT crop_name_de_key UNIQUE (name_de);
ALTER TABLE crop ADD CONSTRAINT crop_name_en_key UNIQUE (name_en);

-- The original UNIQUE(name) constraint's name is Postgres's default
-- (<table>_<column>_key), not one chosen explicitly in the original
-- migration, so it is looked up rather than hardcoded here.
DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'crop'::regclass
      AND contype = 'u'
      AND conkey = ARRAY[(SELECT attnum FROM pg_attribute WHERE attrelid = 'crop'::regclass AND attname = 'name')];

    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE crop DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

ALTER TABLE crop DROP COLUMN name;

-- migrate:down
ALTER TABLE crop ADD COLUMN name TEXT;
UPDATE crop SET name = name_de;
ALTER TABLE crop ALTER COLUMN name SET NOT NULL;
ALTER TABLE crop ADD CONSTRAINT crop_name_key UNIQUE (name);

ALTER TABLE crop DROP CONSTRAINT crop_name_de_key;
ALTER TABLE crop DROP CONSTRAINT crop_name_en_key;
ALTER TABLE crop DROP COLUMN name_de;
ALTER TABLE crop DROP COLUMN name_en;
