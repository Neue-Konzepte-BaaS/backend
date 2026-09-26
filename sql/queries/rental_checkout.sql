-- name: InsertRentalCheckout :one
INSERT INTO rental_checkout (customer, plot, crop, start_at, message, stripe_checkout_session_id, amount_cents)
VALUES (
    sqlc.arg(customer),
    sqlc.arg(plot),
    sqlc.arg(crop),
    sqlc.arg(start_at),
    sqlc.arg(message),
    sqlc.arg(stripe_checkout_session_id),
    sqlc.arg(amount_cents)
)
RETURNING id, customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents, rental, created_at, updated_at;

-- name: GetRentalCheckoutBySessionID :one
SELECT id, customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents, rental, created_at, updated_at
FROM rental_checkout
WHERE stripe_checkout_session_id = $1
LIMIT 1;

-- name: CompleteRentalCheckout :one
-- The WHERE guard makes this idempotent-safe: Stripe retries webhook
-- delivery, so a second checkout.session.completed for the same session
-- must not re-run this and must not overwrite which rental it produced.
UPDATE rental_checkout
SET status = 'completed', rental = sqlc.arg(rental), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND status = 'pending'
RETURNING id, customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents, rental, created_at, updated_at;

-- name: FailRentalCheckout :one
-- Same idempotency guard as CompleteRentalCheckout.
UPDATE rental_checkout
SET status = 'failed', updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND status = 'pending'
RETURNING id, customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents, rental, created_at, updated_at;

-- name: ExpireRentalCheckout :one
-- Same idempotency guard as CompleteRentalCheckout.
UPDATE rental_checkout
SET status = 'expired', updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND status = 'pending'
RETURNING id, customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents, rental, created_at, updated_at;

-- name: GetCompletedRentalCheckoutByRental :one
-- Used when a farmer declines a rental, to find the payment that must now
-- be refunded. Returns no rows (mapped to ErrNotFound) if the rental was
-- never paid for through Stripe.
SELECT id, customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents, rental, created_at, updated_at
FROM rental_checkout
WHERE rental = sqlc.arg(rental) AND status = 'completed'
LIMIT 1;

-- name: MarkRentalCheckoutRefunded :one
-- Same idempotency guard as CompleteRentalCheckout.
UPDATE rental_checkout
SET status = 'refunded', updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND status = 'completed'
RETURNING id, customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents, rental, created_at, updated_at;
