-- migrate:up
-- Opt-out, not opt-in: broadcasts, announcements, and ripeness notices are
-- all mailed to customers unconditionally today, so defaulting to true
-- preserves current behavior; this column only lets a customer turn it off.
ALTER TABLE customer ADD COLUMN notify_messages_by_email BOOLEAN NOT NULL DEFAULT true;

-- migrate:down
ALTER TABLE customer DROP COLUMN notify_messages_by_email;
