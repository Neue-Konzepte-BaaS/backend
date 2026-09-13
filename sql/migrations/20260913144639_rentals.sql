-- migrate:up
-- btree_gist lets a GiST exclusion constraint mix equality on a plain UUID
-- column with range overlap, which the no-overlap constraint below needs.
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE rental (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plot UUID NOT NULL,
    customer UUID NOT NULL,
    period TSTZRANGE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_rental_plot
        FOREIGN KEY (plot)
        REFERENCES plot(id),
    CONSTRAINT fk_rental_customer
        FOREIGN KEY (customer)
        REFERENCES customer(account_id),
    -- Enforcing this in the database rather than by checking before insert
    -- keeps two concurrent bookings of the same plot from both succeeding.
    CONSTRAINT rental_no_overlap
        EXCLUDE USING gist (plot WITH =, period WITH &&)
);

CREATE INDEX idx_rental_customer ON rental(customer);

-- migrate:down
DROP INDEX IF EXISTS idx_rental_customer;
DROP TABLE rental;
