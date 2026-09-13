-- migrate:up
-- argon2id encodes its salt inside the PHC hash string in password_hash,
-- so a separate salt column is redundant.
ALTER TABLE account DROP COLUMN salt;

-- migrate:down
ALTER TABLE account ADD COLUMN salt TEXT NOT NULL DEFAULT '';
