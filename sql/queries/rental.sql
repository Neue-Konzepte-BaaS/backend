-- The period column is never selected as a whole: sqlc maps tstzrange to a
-- pgtype.Range, so the bounds are read out as plain timestamps instead.

-- The period starts at the database's clock rather than one supplied by the
-- caller: the availability filter compares against CURRENT_TIMESTAMP, so a
-- start time from a host whose clock runs ahead would leave the plot looking
-- available for the difference.

-- name: InsertRental :one
INSERT INTO rental (plot, customer, crop, period)
VALUES (
    sqlc.arg(plot),
    sqlc.arg(customer),
    sqlc.arg(crop),
    tstzrange(
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP + make_interval(months => sqlc.arg(duration_months)::int)
    )
)
RETURNING id, lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: GetRentalsByCustomer :many
SELECT
    r.id,
    r.plot,
    r.customer,
    r.crop,
    lower(r.period)::timestamptz AS start_at,
    upper(r.period)::timestamptz AS end_at,
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
