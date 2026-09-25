-- The period column is never selected as a whole: sqlc maps tstzrange to a
-- pgtype.Range, so the bounds are read out as plain timestamps instead.

-- name: InsertRentalRequest :one
INSERT INTO rental (plot, customer, crop, period, message, status)
VALUES (
    sqlc.arg(plot),
    sqlc.arg(customer),
    sqlc.arg(crop),
    tstzrange(
        sqlc.arg(start_at)::timestamptz,
        sqlc.arg(start_at)::timestamptz + make_interval(months => sqlc.arg(duration_months)::int)
    ),
    sqlc.arg(message),
    'requested'
)
RETURNING id, status, lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: UpdateRentalStatus :one
-- Only a still-requested rental can be decided: the WHERE guard makes this
-- idempotent-safe, since a second approve/decline on the same row returns no
-- rows instead of silently overwriting an earlier decision.
UPDATE rental
SET status = sqlc.arg(status), decided_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND status = 'requested'
RETURNING id, plot, customer, crop, status, lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at, message, decided_at;

-- name: GetRentalWithFieldByID :one
-- Fetches the rental together with the field its plot belongs to, so the
-- service can check the deciding farmer owns that field before approving or
-- declining.
SELECT r.id, r.plot, r.customer, r.crop, r.status, p.field
FROM rental r
JOIN plot p ON p.id = r.plot
WHERE r.id = $1;

-- name: GetRentalsByCustomer :many
SELECT
    r.id,
    r.plot,
    r.customer,
    r.crop,
    lower(r.period)::timestamptz AS start_at,
    upper(r.period)::timestamptz AS end_at,
    r.status,
    r.message,
    r.decided_at,
    p.name AS plot_name,
    p.field,
    p.coordinates,
    ST_Area(p.coordinates::geography)::float8 AS plot_area_square_meters,
    c.name AS crop_name,
    c.duration_months AS crop_duration_months
FROM rental r
JOIN plot p ON p.id = r.plot
JOIN crop c ON c.id = r.crop
WHERE r.customer = $1
ORDER BY lower(r.period) DESC;

-- name: GetRentalsByFarm :many
-- Every rental on the farm's own plots, active and historic alike: unlike
-- GetCustomersOfFarmer, this is not restricted to r.period @> CURRENT_TIMESTAMP,
-- since a farmer reviewing their rental history wants past bookings too.
SELECT
    r.id,
    r.plot,
    r.customer,
    r.crop,
    lower(r.period)::timestamptz AS start_at,
    upper(r.period)::timestamptz AS end_at,
    r.status,
    r.message,
    r.decided_at,
    p.name AS plot_name,
    p.field,
    p.coordinates,
    ST_Area(p.coordinates::geography)::float8 AS plot_area_square_meters,
    f.name AS field_name,
    a.id AS customer_id,
    a.email AS customer_email,
    a.first_name AS customer_first_name,
    a.last_name AS customer_last_name
FROM rental r
JOIN plot p ON p.id = r.plot
JOIN field f ON f.id = p.field
JOIN account a ON a.id = r.customer
WHERE f.farm = $1
ORDER BY lower(r.period) DESC;

-- name: GetActiveRentalsByCustomer :many
-- The customer's rentals covering right now, each with where today falls
-- inside the rental period. Both week numbers are computed from the database
-- clock, for the same reason InsertRentalRequest takes its bounds from it: a
-- host whose clock runs ahead would otherwise put a tenant a week further into
-- their growing season than the rental they were sold.
--
-- 'approved' is load-bearing, not decoration. Since rental requests, a row
-- exists from the moment a customer *asks* for a plot, and a declined one is
-- never deleted — so filtering on the period alone would hand the care guide
-- to someone who was turned down, or who is still waiting for an answer. This
-- is the same audience rule GetAnnouncementsForCustomer applies to the board.
--
-- current_week counts from 1 (the period contains CURRENT_TIMESTAMP, so the
-- elapsed time is never negative), and total_weeks rounds up, so a 13-week-
-- and-one-day rental has a week 14 rather than a partial week nobody is shown.
SELECT
    r.id,
    r.plot,
    r.customer,
    r.crop,
    lower(r.period)::timestamptz AS start_at,
    upper(r.period)::timestamptz AS end_at,
    p.name AS plot_name,
    p.field,
    f.name AS field_name,
    f.farm,
    c.name AS crop_name,
    c.duration_months AS crop_duration_months,
    (FLOOR(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - lower(r.period))) / 604800) + 1)::int AS current_week,
    CEIL(EXTRACT(EPOCH FROM (upper(r.period) - lower(r.period))) / 604800)::int AS total_weeks
FROM rental r
JOIN plot p ON p.id = r.plot
JOIN field f ON f.id = p.field
JOIN crop c ON c.id = r.crop
WHERE r.customer = $1 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
ORDER BY lower(r.period) DESC;
