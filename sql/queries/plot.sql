-- name: InsertPlot :one
INSERT INTO plot (name, field, coordinates) VALUES ($1, $2, $3)
RETURNING id, ST_Area(coordinates::geography)::float8 AS area_square_meters;

-- name: GetPlotByID :one
SELECT id, name, field, coordinates, base_price_cents_per_sqm_per_week, ST_Area(coordinates::geography)::float8 AS area_square_meters
FROM plot
WHERE id = $1
LIMIT 1;

-- name: GetPlotField :one
SELECT field
FROM plot
WHERE id = $1
LIMIT 1;

-- name: GetPlotsByFields :many
SELECT id, name, field, coordinates, base_price_cents_per_sqm_per_week, ST_Area(coordinates::geography)::float8 AS area_square_meters
FROM plot
WHERE field = ANY($1::uuid[])
ORDER BY name;

-- name: UpdatePlotBasePrice :exec
UPDATE plot SET base_price_cents_per_sqm_per_week = sqlc.arg(base_price_cents_per_sqm_per_week) WHERE id = sqlc.arg(id);

-- name: CountPlotsByFarm :one
-- Used by the subscription plot-count cap: how many plots a farm currently
-- offers, regardless of rental status.
SELECT COUNT(*)::bigint
FROM plot
JOIN field ON field.id = plot.field
WHERE field.farm = sqlc.arg(farm);

-- name: GetNearestPlots :many
SELECT
    plot.id,
    plot.name,
    plot.field,
    plot.coordinates,
    ST_Area(plot.coordinates::geography)::float8 AS area_square_meters,
    field.farm AS farm,
    ST_Distance(
        ST_Centroid(plot.coordinates)::geography,
        ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)::geography
    )::float8 AS distance_meters
FROM plot
JOIN field ON field.id = plot.field
-- Joined up to the owning farmer's account so a deleted farmer's plots can
-- be excluded below. Every plot's field has a farm with a farmer account by
-- schema guarantee, so these stay INNER JOINs -- no legitimate plot is lost.
JOIN farm ON farm.id = field.farm
JOIN account farmer_account ON farmer_account.id = farm.farmer_id
-- Only plots that are free right now; a rental that has run out stops
-- hiding its plot. A still-undecided request hides the plot too, same as
-- the rental_no_overlap exclusion constraint -- only a declined request
-- frees it.
WHERE NOT EXISTS (
    SELECT 1 FROM rental r
    WHERE r.plot = plot.id AND r.period @> CURRENT_TIMESTAMP AND r.status <> 'declined'
)
-- A deleted farmer no longer offers any of their plots.
AND farmer_account.deleted_at IS NULL
-- Optional: only this farm's plots (its detail page), instead of whichever
-- farms happen to fill the nearest-N.
AND (sqlc.narg(farm)::uuid IS NULL OR field.farm = sqlc.narg(farm)::uuid)
ORDER BY plot.coordinates <-> ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)
LIMIT sqlc.arg(result_limit);
