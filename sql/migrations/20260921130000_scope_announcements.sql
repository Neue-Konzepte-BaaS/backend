-- migrate:up
-- Both nullable: a post with neither set still reaches every current
-- renter, exactly like before this migration.
ALTER TABLE announcement ADD COLUMN field UUID REFERENCES field(id) ON DELETE CASCADE;
ALTER TABLE announcement ADD COLUMN plot UUID REFERENCES plot(id) ON DELETE CASCADE;

-- A post targets a whole field or a single plot, never both at once.
ALTER TABLE announcement ADD CONSTRAINT announcement_scope_not_both
    CHECK (NOT (field IS NOT NULL AND plot IS NOT NULL));

CREATE INDEX idx_announcement_field ON announcement(field) WHERE field IS NOT NULL;
CREATE INDEX idx_announcement_plot ON announcement(plot) WHERE plot IS NOT NULL;

-- migrate:down
DROP INDEX IF EXISTS idx_announcement_field;
DROP INDEX IF EXISTS idx_announcement_plot;
ALTER TABLE announcement DROP CONSTRAINT announcement_scope_not_both;
ALTER TABLE announcement DROP COLUMN field;
ALTER TABLE announcement DROP COLUMN plot;
