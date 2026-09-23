-- migrate:up
CREATE TABLE ripeness_notice (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer UUID NOT NULL,
    field UUID NOT NULL,
    crop UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Cascades like announcement: a notice is a statement by a farmer about
    -- one of his fields and has no meaning once either is gone.
    CONSTRAINT fk_ripeness_notice_farmer
        FOREIGN KEY (farmer)
        REFERENCES farmer(account_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_ripeness_notice_field
        FOREIGN KEY (field)
        REFERENCES field(id)
        ON DELETE CASCADE,
    -- Crop is a shared catalog entry, not owned by this notice, so it does
    -- not cascade: a crop referenced by a past notice cannot be deleted,
    -- same as a crop referenced by a rental.
    CONSTRAINT fk_ripeness_notice_crop
        FOREIGN KEY (crop)
        REFERENCES crop(id)
);

-- The customer inbox reads "newest notices for the fields I rent on", so the
-- index carries that sort order rather than leaving it to a sort node.
CREATE INDEX idx_ripeness_notice_field_created ON ripeness_notice(field, created_at DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_ripeness_notice_field_created;
DROP TABLE ripeness_notice;
