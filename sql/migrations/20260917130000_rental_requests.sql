-- migrate:up
ALTER TABLE rental
    ADD COLUMN status TEXT NOT NULL DEFAULT 'requested'
        CONSTRAINT rental_status_check CHECK (status IN ('requested', 'approved', 'declined')),
    ADD COLUMN message TEXT NOT NULL DEFAULT '',
    ADD COLUMN decided_at TIMESTAMPTZ;

ALTER TABLE rental ALTER COLUMN status DROP DEFAULT;
ALTER TABLE rental ALTER COLUMN message DROP DEFAULT;

-- A declined request must stop blocking the plot, so the constraint is
-- rebuilt as partial: requested and approved rows still exclude on overlap,
-- declined ones no longer occupy the period they asked for.
ALTER TABLE rental DROP CONSTRAINT rental_no_overlap;
ALTER TABLE rental
    ADD CONSTRAINT rental_no_overlap
        EXCLUDE USING gist (plot WITH =, period WITH &&) WHERE (status <> 'declined');

-- migrate:down
ALTER TABLE rental DROP CONSTRAINT rental_no_overlap;
ALTER TABLE rental
    ADD CONSTRAINT rental_no_overlap
        EXCLUDE USING gist (plot WITH =, period WITH &&);

ALTER TABLE rental
    DROP COLUMN status,
    DROP COLUMN message,
    DROP COLUMN decided_at;
