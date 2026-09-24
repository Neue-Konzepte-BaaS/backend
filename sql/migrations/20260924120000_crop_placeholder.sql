-- migrate:up
-- The 'unknown' crop only exists to backfill rental.crop for rentals created
-- before crops did (see 20260913165138_crops.sql). It must stay referenceable,
-- but never be offered as a real crop.
ALTER TABLE crop ADD COLUMN is_placeholder BOOLEAN NOT NULL DEFAULT false;

UPDATE crop SET is_placeholder = true WHERE name = 'unknown';

-- migrate:down
ALTER TABLE crop DROP COLUMN is_placeholder;
