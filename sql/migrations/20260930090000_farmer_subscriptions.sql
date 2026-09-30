-- migrate:up

-- One row per subscription tier on offer. Prices are admin-editable for
-- NEW subscriptions only: Stripe Prices are immutable, so "editing a
-- plan's price" means creating a new Stripe Price and repointing
-- stripe_price_id at it. A farmer who already subscribed keeps paying
-- whatever Stripe Price their subscription was created against.
CREATE TABLE subscription_plan (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE
        CONSTRAINT subscription_plan_code_check CHECK (code IN ('cheap', 'modest', 'expensive')),
    display_name TEXT NOT NULL,
    -- Max plots a farm on this plan may simultaneously offer. NULL means
    -- unlimited (the top tier).
    max_plots INTEGER
        CONSTRAINT subscription_plan_max_plots_positive CHECK (max_plots IS NULL OR max_plots > 0),
    price_cents INTEGER NOT NULL CHECK (price_cents > 0),
    stripe_price_id TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One row per farmer's subscription lifecycle. Mirrors rental_checkout's
-- relationship to rental: this is the payment ledger; Stripe's own
-- subscription object is the source of truth for billing, kept in sync via
-- webhook.
CREATE TABLE farmer_subscription (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer UUID NOT NULL REFERENCES farmer(account_id),
    plan UUID NOT NULL REFERENCES subscription_plan(id),
    stripe_customer_id TEXT NOT NULL,
    stripe_subscription_id TEXT UNIQUE,
    stripe_checkout_session_id TEXT UNIQUE,
    -- pending: checkout session created, awaiting first payment.
    -- active: payment succeeded, Stripe subscription is live.
    -- past_due: a renewal invoice failed; Stripe is retrying.
    -- canceled: subscription ended, or its checkout session expired unpaid.
    status TEXT NOT NULL DEFAULT 'pending'
        CONSTRAINT farmer_subscription_status_check
        CHECK (status IN ('pending', 'active', 'past_due', 'canceled')),
    current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- At most one non-terminal subscription per farmer; canceled rows are
-- exempt so a lapsed farmer can start a fresh checkout.
CREATE UNIQUE INDEX idx_farmer_subscription_active_one_per_farmer
    ON farmer_subscription (farmer)
    WHERE status IN ('pending', 'active', 'past_due');

CREATE INDEX idx_farmer_subscription_farmer ON farmer_subscription(farmer);

-- migrate:down
DROP TABLE farmer_subscription;
DROP TABLE subscription_plan;
