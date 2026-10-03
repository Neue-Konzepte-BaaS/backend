-- migrate:up
-- NULL means the account is active. Doubles as both the "is deleted" flag
-- and the audit timestamp -- deletion is one-way, unlike
-- subscription_plan.is_active, which toggles both ways, so a timestamp fits
-- better than a second boolean that could disagree with it.
ALTER TABLE account ADD COLUMN deleted_at TIMESTAMPTZ NULL;

-- migrate:down
ALTER TABLE account DROP COLUMN deleted_at;
