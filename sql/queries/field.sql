-- name: InsertField :one
INSERT INTO field (name, farm, coordinates) VALUES ($1, $2, $3) RETURNING id;

-- name: GetFieldByID :one
SELECT id, name, farm, coordinates
FROM field
WHERE id = $1
LIMIT 1;

-- name: GetFieldFarm :one
SELECT farm
FROM field
WHERE id = $1
LIMIT 1;

-- name: GetFieldsByFarm :many
SELECT id, name, farm, coordinates
FROM field
WHERE farm = $1
ORDER BY name;

-- name: GetFieldsByFarmWithAvailablePlotStats :many
-- A farm's fields for a customer browsing plots, each with how many plots
-- are available right now and their combined area. "Available" here uses
-- the exact same predicate as GetNearestPlots (plot.sql) -- not the
-- status = 'approved' test statistics.sql/ListFarms use for "rented" --
-- because this count must agree with what /api/plots/nearest actually
-- returns for the field: a still-undecided rental request already hides a
-- plot there, so it must not be counted as available here either.
--
-- LEFT JOIN LATERAL, not a GROUP BY: a field with zero currently-available
-- plots still gets its own row (COALESCEd to zero) instead of disappearing,
-- so a customer sees the field listed with "0 available" rather than the
-- field silently vanishing.
SELECT
    field.id,
    field.name,
    field.farm,
    field.coordinates,
    COALESCE(available.total, 0)::bigint AS plot_count,
    COALESCE(available.area, 0)::float8 AS area_square_meters
FROM field
LEFT JOIN LATERAL (
    SELECT
        COUNT(*)::bigint AS total,
        COALESCE(SUM(ST_Area(plot.coordinates::geography)::float8), 0)::float8 AS area
    FROM plot
    WHERE plot.field = field.id
      AND NOT EXISTS (
          SELECT 1 FROM rental r
          WHERE r.plot = plot.id AND r.period @> CURRENT_TIMESTAMP AND r.status <> 'declined'
      )
) available ON TRUE
WHERE field.farm = $1
ORDER BY field.name;
