-- migrate:up
-- A plot starts unpriced, same spirit as a plot starting with no offered
-- crops: it only becomes rentable once both this rate and the matching
-- farm_crop_rate row exist (see GetPricedCropOfferingsByPlots).
ALTER TABLE plot
    ADD COLUMN base_price_cents_per_sqm_per_week INTEGER
        CONSTRAINT plot_base_price_positive CHECK (base_price_cents_per_sqm_per_week > 0);

-- One row per crop per farm: a flat rate covering every plot on that farm
-- growing that crop, set once regardless of how many plots offer it. EUR is
-- hard-coded as a Go constant rather than stored here -- this product only
-- serves the German market, so a currency column would be unused
-- optionality.
CREATE TABLE farm_crop_rate (
    farm UUID NOT NULL REFERENCES farm(id),
    crop UUID NOT NULL REFERENCES crop(id),
    price_cents_per_sqm_per_week INTEGER NOT NULL CHECK (price_cents_per_sqm_per_week > 0),
    PRIMARY KEY (farm, crop)
);

-- Tracks a Stripe Checkout Session's lifecycle before a real rental request
-- exists. Payment gates rental creation: a rental (not even a Requested
-- one) is only ever inserted once the linked session's payment is
-- confirmed by the checkout.session.completed webhook, which is the only
-- caller of rentalService.RequestRental for a customer-initiated booking.
-- This deliberately does not touch the rental table's own exclusion
-- constraint or insert path -- it is a queue of payment attempts sitting in
-- front of it, not a replacement for it.
CREATE TABLE rental_checkout (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer UUID NOT NULL REFERENCES customer(account_id),
    plot UUID NOT NULL REFERENCES plot(id),
    crop UUID NOT NULL REFERENCES crop(id),
    -- The rental request's own inputs, captured here so the webhook -- which
    -- only carries the Stripe session id -- has everything it needs to call
    -- RequestRental without asking the customer again.
    start_at TIMESTAMPTZ NOT NULL,
    message TEXT NOT NULL,
    stripe_checkout_session_id TEXT NOT NULL UNIQUE,
    -- pending: session created, awaiting payment.
    -- completed: paid and a rental was created from it (see rental column).
    -- failed: paid, but no rental could be created (e.g. the plot was taken
    --   by a concurrent payment) -- refunded automatically.
    -- expired: the Stripe session expired unpaid.
    -- refunded: paid and a rental was created, but the farmer later
    --   declined it -- refunded, same as failed.
    status TEXT NOT NULL DEFAULT 'pending'
        CONSTRAINT rental_checkout_status_check CHECK (status IN ('pending', 'completed', 'failed', 'expired', 'refunded')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    rental UUID REFERENCES rental(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_rental_checkout_customer ON rental_checkout(customer);
CREATE INDEX idx_rental_checkout_rental ON rental_checkout(rental);

-- migrate:down
DROP TABLE rental_checkout;
DROP TABLE farm_crop_rate;
ALTER TABLE plot DROP COLUMN base_price_cents_per_sqm_per_week;
