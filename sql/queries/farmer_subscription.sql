-- name: InsertFarmerSubscription :one
INSERT INTO farmer_subscription (farmer, plan, stripe_customer_id, stripe_checkout_session_id)
VALUES (
    sqlc.arg(farmer),
    sqlc.arg(plan),
    sqlc.arg(stripe_customer_id),
    sqlc.arg(stripe_checkout_session_id)
)
RETURNING id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at;

-- name: GetFarmerSubscriptionBySessionID :one
SELECT id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at
FROM farmer_subscription
WHERE stripe_checkout_session_id = $1
LIMIT 1;

-- name: GetFarmerSubscriptionByStripeSubscriptionID :one
SELECT id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at
FROM farmer_subscription
WHERE stripe_subscription_id = $1
LIMIT 1;

-- name: GetActiveSubscriptionByFarmer :one
-- Used by the RequireActiveSubscription middleware and the plot-count cap
-- check. Returns no rows (mapped to ErrNotFound) if the farmer has never
-- subscribed, or their only subscription is pending/canceled.
SELECT id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at
FROM farmer_subscription
WHERE farmer = sqlc.arg(farmer) AND status IN ('active', 'past_due')
LIMIT 1;

-- name: GetSubscriptionByFarmer :one
-- Unlike GetActiveSubscriptionByFarmer, this returns the farmer's current
-- non-terminal subscription regardless of status (including pending), so a
-- farmer mid-checkout can poll their own status.
SELECT id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at
FROM farmer_subscription
WHERE farmer = sqlc.arg(farmer) AND status IN ('pending', 'active', 'past_due')
LIMIT 1;

-- name: ActivateFarmerSubscription :one
-- The WHERE guard makes this idempotent-safe: Stripe retries webhook
-- delivery, so a second checkout.session.completed for the same session
-- must not re-run this.
UPDATE farmer_subscription
SET status = 'active', stripe_subscription_id = sqlc.arg(stripe_subscription_id), current_period_end = sqlc.arg(current_period_end), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND status = 'pending'
RETURNING id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at;

-- name: ExpireFarmerSubscription :one
-- Same idempotency guard as ActivateFarmerSubscription.
UPDATE farmer_subscription
SET status = 'canceled', updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND status = 'pending'
RETURNING id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at;

-- name: MarkFarmerSubscriptionPastDue :one
-- Keyed by Stripe subscription id, since invoice.payment_failed carries
-- that rather than a checkout session id. Guarded to only affect a
-- currently-active subscription.
UPDATE farmer_subscription
SET status = 'past_due', updated_at = CURRENT_TIMESTAMP
WHERE stripe_subscription_id = sqlc.arg(stripe_subscription_id) AND status = 'active'
RETURNING id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at;

-- name: ReactivateFarmerSubscription :one
-- Fired by invoice.payment_succeeded: covers both a routine renewal (already
-- active, this just refreshes current_period_end) and recovery from
-- past_due, so it is not guarded on a specific prior status the way the
-- other transitions are.
UPDATE farmer_subscription
SET status = 'active', current_period_end = sqlc.arg(current_period_end), updated_at = CURRENT_TIMESTAMP
WHERE stripe_subscription_id = sqlc.arg(stripe_subscription_id) AND status IN ('active', 'past_due')
RETURNING id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at;

-- name: CancelFarmerSubscription :one
-- Fired by customer.subscription.deleted. Guarded against a subscription
-- that is already canceled (e.g. a retried webhook delivery).
UPDATE farmer_subscription
SET status = 'canceled', updated_at = CURRENT_TIMESTAMP
WHERE stripe_subscription_id = sqlc.arg(stripe_subscription_id) AND status IN ('pending', 'active', 'past_due')
RETURNING id, farmer, plan, stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id, status, current_period_end, created_at, updated_at;
