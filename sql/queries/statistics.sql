-- Every figure is cast to the type Go should see (::bigint, ::float8). sqlc has
-- no PostGIS catalog, so the cast is what pins the column type -- the same
-- trick plot.sql uses for ST_Distance.
--
-- A cast pins the TYPE only; it does not make a NULL non-NULL. SUM over an
-- empty set is NULL and sqlc will have generated a plain float64, so every such
-- aggregate is wrapped in COALESCE -- otherwise a farmer with no fields yet
-- fails to scan at runtime. That is a correctness requirement, not a typing
-- nicety.
--
-- One CTE per statistic group; the final SELECT is nothing but aliased column
-- references. Adding a statistic means adding a CTE and a line below. Keep
-- every cross-joined CTE free of GROUP BY: a CTE returning no rows would empty
-- the whole result.

-- name: GetFarmStatistics :one
-- Aggregates one farmer's own fields, plots and rentals. Every CTE is scoped by
-- farmer; a farmer who owns nothing gets zeros rather than no row.
WITH farm_plot_rows AS (
    -- Same "rented right now" test as plot.sql: containment of the current
    -- instant, so an expired rental stops counting with no cleanup job.
    SELECT
        p.coordinates,
        EXISTS (
            SELECT 1 FROM rental r
            WHERE r.plot = p.id AND r.period @> CURRENT_TIMESTAMP
        ) AS is_rented
    FROM plot p
    JOIN field f ON f.id = p.field
    WHERE f.farmer = sqlc.arg(farmer)
),
farm_fields AS (
    SELECT
        COUNT(*)::bigint AS total,
        COALESCE(SUM(ST_Area(coordinates::geography)::float8), 0)::float8 AS area
    FROM field
    WHERE farmer = sqlc.arg(farmer)
),
farm_plots AS (
    SELECT
        COUNT(*)::bigint AS total,
        (COUNT(*) FILTER (WHERE is_rented))::bigint AS rented,
        COALESCE(SUM(ST_Area(coordinates::geography)::float8), 0)::float8 AS area
    FROM farm_plot_rows
),
farm_rentals AS (
    SELECT
        COUNT(*)::bigint AS total,
        (COUNT(*) FILTER (WHERE r.period @> CURRENT_TIMESTAMP))::bigint AS active,
        (COUNT(*) FILTER (WHERE r.created_at >= CURRENT_TIMESTAMP - INTERVAL '30 days'))::bigint AS last_30_days
    FROM rental r
    JOIN plot p ON p.id = r.plot
    JOIN field f ON f.id = p.field
    WHERE f.farmer = sqlc.arg(farmer)
)
SELECT
    CURRENT_TIMESTAMP::timestamptz AS generated_at,
    farm_fields.total         AS field_count,
    farm_fields.area          AS field_area_square_meters,
    farm_plots.total          AS plot_count,
    farm_plots.rented         AS rented_plot_count,
    farm_plots.area           AS plot_area_square_meters,
    farm_rentals.total        AS rental_count,
    farm_rentals.active       AS active_rental_count,
    farm_rentals.last_30_days AS rentals_last_30_days
FROM farm_fields, farm_plots, farm_rentals;

-- name: GetPlatformStatistics :one
-- The same figures across every farmer, plus the account counts only an admin
-- sees.
WITH platform_plot_rows AS (
    SELECT
        p.coordinates,
        EXISTS (
            SELECT 1 FROM rental r
            WHERE r.plot = p.id AND r.period @> CURRENT_TIMESTAMP
        ) AS is_rented
    FROM plot p
),
platform_fields AS (
    SELECT
        COUNT(*)::bigint AS total,
        COALESCE(SUM(ST_Area(coordinates::geography)::float8), 0)::float8 AS area
    FROM field
),
platform_plots AS (
    SELECT
        COUNT(*)::bigint AS total,
        (COUNT(*) FILTER (WHERE is_rented))::bigint AS rented,
        COALESCE(SUM(ST_Area(coordinates::geography)::float8), 0)::float8 AS area
    FROM platform_plot_rows
),
platform_rentals AS (
    SELECT
        COUNT(*)::bigint AS total,
        (COUNT(*) FILTER (WHERE period @> CURRENT_TIMESTAMP))::bigint AS active,
        (COUNT(*) FILTER (WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '30 days'))::bigint AS last_30_days
    FROM rental
),
platform_accounts AS (
    SELECT
        COUNT(*)::bigint AS total,
        (COUNT(*) FILTER (WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '30 days'))::bigint AS last_30_days
    FROM account
),
-- Role is subtype membership (see account.sql), so the per-role counts are
-- simply the sizes of the subtype tables.
platform_farmers AS (
    SELECT COUNT(*)::bigint AS total FROM farmer
),
platform_customers AS (
    SELECT COUNT(*)::bigint AS total FROM customer
)
SELECT
    CURRENT_TIMESTAMP::timestamptz AS generated_at,
    platform_fields.total         AS field_count,
    platform_fields.area          AS field_area_square_meters,
    platform_plots.total          AS plot_count,
    platform_plots.rented         AS rented_plot_count,
    platform_plots.area           AS plot_area_square_meters,
    platform_rentals.total        AS rental_count,
    platform_rentals.active       AS active_rental_count,
    platform_rentals.last_30_days AS rentals_last_30_days,
    platform_accounts.total        AS account_count,
    platform_farmers.total         AS farmer_count,
    platform_customers.total       AS customer_count,
    platform_accounts.last_30_days AS accounts_last_30_days
FROM platform_fields, platform_plots, platform_rentals,
    platform_accounts, platform_farmers, platform_customers;
