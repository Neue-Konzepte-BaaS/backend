-- migrate:up
-- The statistics queries filter fields by owner and join plots to their field.
-- Postgres indexes only the referenced side of a foreign key, never the
-- referencing side, so both of these were sequential scans over every field
-- and plot on the platform rather than just the calling farmer's.
--
-- rental.plot deliberately gets no index: the rental_no_overlap exclusion
-- constraint's GiST index already leads with that column.
CREATE INDEX idx_field_farmer ON field (farmer);
CREATE INDEX idx_plot_field ON plot (field);

-- migrate:down
DROP INDEX IF EXISTS idx_plot_field;
DROP INDEX IF EXISTS idx_field_farmer;
