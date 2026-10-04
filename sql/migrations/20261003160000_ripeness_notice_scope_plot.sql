-- migrate:up
-- Existing rows carry field ids in what is about to become a plot column, so
-- they can never satisfy the new FK; there is no way to recover which plot
-- each one meant, and this is pre-release data with no real notice history
-- to preserve.
DELETE FROM ripeness_notice;

DROP INDEX IF EXISTS idx_ripeness_notice_field_created;

ALTER TABLE ripeness_notice DROP CONSTRAINT fk_ripeness_notice_field;
ALTER TABLE ripeness_notice RENAME COLUMN field TO plot;
ALTER TABLE ripeness_notice ADD CONSTRAINT fk_ripeness_notice_plot
    FOREIGN KEY (plot) REFERENCES plot(id) ON DELETE CASCADE;

CREATE INDEX idx_ripeness_notice_plot_created ON ripeness_notice(plot, created_at DESC);

-- migrate:down
DROP INDEX IF EXISTS idx_ripeness_notice_plot_created;

ALTER TABLE ripeness_notice DROP CONSTRAINT fk_ripeness_notice_plot;
ALTER TABLE ripeness_notice RENAME COLUMN plot TO field;
ALTER TABLE ripeness_notice ADD CONSTRAINT fk_ripeness_notice_field
    FOREIGN KEY (field) REFERENCES field(id) ON DELETE CASCADE;

CREATE INDEX idx_ripeness_notice_field_created ON ripeness_notice(field, created_at DESC);
