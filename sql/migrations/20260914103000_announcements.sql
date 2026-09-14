-- migrate:up
CREATE TABLE announcement (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer UUID NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Unlike field, this cascades: an announcement is a statement by a farmer
    -- and has no meaning once that farmer is gone, and the farmer row itself
    -- already cascades from account.
    CONSTRAINT fk_announcement_farmer
        FOREIGN KEY (farmer)
        REFERENCES farmer(account_id)
        ON DELETE CASCADE
);

-- Both reads are "the newest announcements for one farmer", so the index
-- carries the sort order rather than leaving it to a sort node.
CREATE INDEX idx_announcement_farmer_created ON announcement(farmer, created_at DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_announcement_farmer_created;
DROP TABLE announcement;
