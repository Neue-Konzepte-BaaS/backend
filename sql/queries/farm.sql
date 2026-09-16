-- A farm is a farmer subtype row, not a table of its own, so "listing farms"
-- means listing farmers from the farm side.
--
-- The per-farm figures are computed with exactly the predicates
-- statistics.sql uses -- same ST_Area over ::geography, same
-- "period @> CURRENT_TIMESTAMP" test for rented right now. That is what makes
-- these rows reconcile with GetPlatformStatistics: summing field_count,
-- plot_count and rented_plot_count across every row of this list reproduces
-- the platform figures exactly. Change a predicate in one place and the two
-- admin screens start disagreeing.

-- name: ListFarms :many
-- One page of the admin farm list.
--
-- Each aggregate hangs off its own LATERAL join rather than a GROUP BY over a
-- join of all three: grouping a farm's fields, plots and rentals in one pass
-- would multiply the rows against each other and count each field once per
-- plot. Separate laterals keep every figure independent of the others.
--
-- An ungrouped aggregate returns one row even over no rows at all, so a farm
-- with nothing is a row of zeros rather than a missing row -- the same promise
-- GetFarmStatistics makes to a farmer who owns nothing. The COALESCEs restate
-- that for sqlc, which types a LEFT JOIN's columns as nullable regardless.
WITH listed AS (
    SELECT
        a.id,
        a.first_name,
        a.last_name,
        a.email,
        a.created_at,
        f.farm_name,
        f.postal_code,
        COALESCE(farm_fields.total, 0)::bigint  AS field_count,
        COALESCE(farm_fields.area, 0)::float8   AS field_area_square_meters,
        COALESCE(farm_plots.total, 0)::bigint   AS plot_count,
        COALESCE(farm_plots.rented, 0)::bigint  AS rented_plot_count,
        COALESCE(farm_plots.area, 0)::float8    AS plot_area_square_meters,
        COALESCE(farm_rentals.active, 0)::bigint AS active_rental_count
    FROM farmer f
    JOIN account a ON a.id = f.account_id
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*)::bigint AS total,
            COALESCE(SUM(ST_Area(coordinates::geography)::float8), 0)::float8 AS area
        FROM field
        WHERE field.farmer = f.account_id
    ) farm_fields ON TRUE
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*)::bigint AS total,
            (COUNT(*) FILTER (
                WHERE EXISTS (
                    SELECT 1 FROM rental r
                    WHERE r.plot = p.id AND r.period @> CURRENT_TIMESTAMP
                )
            ))::bigint AS rented,
            COALESCE(SUM(ST_Area(p.coordinates::geography)::float8), 0)::float8 AS area
        FROM plot p
        JOIN field pf ON pf.id = p.field
        WHERE pf.farmer = f.account_id
    ) farm_plots ON TRUE
    LEFT JOIN LATERAL (
        SELECT (COUNT(*) FILTER (WHERE r.period @> CURRENT_TIMESTAMP))::bigint AS active
        FROM rental r
        JOIN plot rp ON rp.id = r.plot
        JOIN field rf ON rf.id = rp.field
        WHERE rf.farmer = f.account_id
    ) farm_rentals ON TRUE
)
SELECT
    listed.id,
    listed.first_name,
    listed.last_name,
    listed.email,
    listed.created_at,
    listed.farm_name,
    listed.postal_code,
    listed.field_count,
    listed.field_area_square_meters,
    listed.plot_count,
    listed.rented_plot_count,
    listed.plot_area_square_meters,
    listed.active_rental_count,
    (COUNT(*) OVER ())::bigint AS total_count
FROM listed
-- An empty search and a NULL postal code each mean "no filter", so one query
-- serves every combination of them.
WHERE (
        sqlc.arg(search)::text = ''
        OR listed.farm_name ILIKE '%' || sqlc.arg(search)::text || '%'
        OR listed.email ILIKE '%' || sqlc.arg(search)::text || '%'
        OR listed.first_name ILIKE '%' || sqlc.arg(search)::text || '%'
        OR listed.last_name ILIKE '%' || sqlc.arg(search)::text || '%'
    )
  AND (sqlc.narg(postal_code)::int IS NULL OR listed.postal_code = sqlc.narg(postal_code)::int)
-- Farm names are not unique, so id breaks the tie and keeps paging stable.
ORDER BY listed.farm_name, listed.id
LIMIT sqlc.arg(result_limit) OFFSET sqlc.arg(result_offset);
