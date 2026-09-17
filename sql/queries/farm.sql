-- name: InsertFarm :one
INSERT INTO farm (farmer_id, name, address, description) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: GetFarmByID :one
-- TotalSquareMeters sums every plot across every field of this farm; a farm
-- with no fields or plots gets 0, not an error.
SELECT
    farm.id,
    farm.farmer_id,
    farm.name,
    farm.address,
    farm.description,
    farm.founded_at,
    COALESCE(SUM(ST_Area(p.coordinates::geography)), 0)::float8 AS total_square_meters
FROM farm
LEFT JOIN field fi ON fi.farm = farm.id
LEFT JOIN plot p ON p.field = fi.id
WHERE farm.id = $1
GROUP BY farm.id;

-- name: GetFarmIDByFarmerID :one
-- Lean lookup for ownership checks: resolves a farmer's own farm id without
-- the area-summing join GetFarmByID does.
SELECT id FROM farm WHERE farmer_id = $1;

-- name: ListFarms :many
-- One page of the admin farm list.
--
-- The per-farm figures are computed with exactly the predicates
-- statistics.sql uses -- same ST_Area over ::geography, same
-- "period @> CURRENT_TIMESTAMP" test for rented right now. That is what makes
-- these rows reconcile with GetPlatformStatistics: summing field_count,
-- plot_count and rented_plot_count across every row of this list reproduces
-- the platform figures exactly. Change a predicate in one place and the two
-- admin screens start disagreeing.
--
-- Each aggregate hangs off its own LATERAL join rather than a GROUP BY over a
-- join of all three: grouping a farm's fields, plots and rentals in one pass
-- would multiply the rows against each other and count each field once per
-- plot. GetFarmByID can group, because it aggregates one thing for one farm.
--
-- An ungrouped aggregate returns one row even over no rows at all, so a farm
-- with nothing is a row of zeros rather than a missing row -- the same promise
-- GetFarmStatistics makes to a farm that owns nothing. The COALESCEs restate
-- that for sqlc, which types a LEFT JOIN's columns as nullable regardless.
--
-- total_count is how many rows match the filter before LIMIT, taken in the same
-- query so the count and the page come from one snapshot. A page past the end
-- returns no rows, and therefore no count either.
WITH listed AS (
    SELECT
        farm.id,
        farm.farmer_id,
        farm.name,
        farm.address,
        fr.postal_code,
        a.first_name,
        a.last_name,
        a.email,
        a.created_at,
        COALESCE(farm_fields.total, 0)::bigint   AS field_count,
        COALESCE(farm_fields.area, 0)::float8    AS field_area_square_meters,
        COALESCE(farm_plots.total, 0)::bigint    AS plot_count,
        COALESCE(farm_plots.rented, 0)::bigint   AS rented_plot_count,
        COALESCE(farm_plots.area, 0)::float8     AS plot_area_square_meters,
        COALESCE(farm_rentals.active, 0)::bigint AS active_rental_count
    FROM farm
    JOIN farmer fr ON fr.account_id = farm.farmer_id
    JOIN account a ON a.id = farm.farmer_id
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*)::bigint AS total,
            COALESCE(SUM(ST_Area(coordinates::geography)::float8), 0)::float8 AS area
        FROM field
        WHERE field.farm = farm.id
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
        WHERE pf.farm = farm.id
    ) farm_plots ON TRUE
    LEFT JOIN LATERAL (
        SELECT (COUNT(*) FILTER (WHERE r.period @> CURRENT_TIMESTAMP))::bigint AS active
        FROM rental r
        JOIN plot rp ON rp.id = r.plot
        JOIN field rf ON rf.id = rp.field
        WHERE rf.farm = farm.id
    ) farm_rentals ON TRUE
)
SELECT
    listed.id,
    listed.farmer_id,
    listed.name,
    listed.address,
    listed.postal_code,
    listed.first_name,
    listed.last_name,
    listed.email,
    listed.created_at,
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
        OR listed.name ILIKE '%' || sqlc.arg(search)::text || '%'
        OR listed.address ILIKE '%' || sqlc.arg(search)::text || '%'
        OR listed.email ILIKE '%' || sqlc.arg(search)::text || '%'
        OR listed.first_name ILIKE '%' || sqlc.arg(search)::text || '%'
        OR listed.last_name ILIKE '%' || sqlc.arg(search)::text || '%'
    )
  AND (sqlc.narg(postal_code)::int IS NULL OR listed.postal_code = sqlc.narg(postal_code)::int)
-- Farm names are not unique, so id breaks the tie and keeps paging stable.
ORDER BY listed.name, listed.id
LIMIT sqlc.arg(result_limit) OFFSET sqlc.arg(result_offset);
