-- name: InsertSubscriptionPlan :exec
-- Used only at startup to seed the three fixed tiers -- see
-- seedSubscriptionPlans in main.go.
INSERT INTO subscription_plan (code, display_name, max_plots, price_cents, stripe_price_id)
VALUES (
    sqlc.arg(code),
    sqlc.arg(display_name),
    sqlc.arg(max_plots),
    sqlc.arg(price_cents),
    sqlc.arg(stripe_price_id)
);

-- name: GetActiveSubscriptionPlans :many
SELECT id, code, display_name, max_plots, price_cents, stripe_price_id, is_active, created_at, updated_at
FROM subscription_plan
WHERE is_active
ORDER BY price_cents;

-- name: GetSubscriptionPlanByID :one
SELECT id, code, display_name, max_plots, price_cents, stripe_price_id, is_active, created_at, updated_at
FROM subscription_plan
WHERE id = $1
LIMIT 1;

-- name: GetSubscriptionPlanByCode :one
SELECT id, code, display_name, max_plots, price_cents, stripe_price_id, is_active, created_at, updated_at
FROM subscription_plan
WHERE code = $1
LIMIT 1;

-- name: ListSubscriptionPlans :many
-- The admin view: every plan, active or retired.
SELECT id, code, display_name, max_plots, price_cents, stripe_price_id, is_active, created_at, updated_at
FROM subscription_plan
ORDER BY price_cents;

-- name: UpdateSubscriptionPlanPrice :one
-- Repoints a plan at a newly created Stripe Price -- see subscription_plan's
-- doc comment for why this is a new Stripe Price rather than an update to
-- the existing one. Only affects future subscriptions to this plan;
-- existing farmer_subscription rows keep referencing whatever Stripe
-- Subscription (and thus Price) they were created against.
UPDATE subscription_plan
SET price_cents = sqlc.arg(price_cents), stripe_price_id = sqlc.arg(stripe_price_id), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, code, display_name, max_plots, price_cents, stripe_price_id, is_active, created_at, updated_at;

-- name: SetSubscriptionPlanActive :exec
UPDATE subscription_plan
SET is_active = sqlc.arg(is_active), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id);
