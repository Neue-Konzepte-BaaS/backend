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
-- Only plots that are free right now; a rental that has run out stops
-- hiding its plot. A still-undecided request hides the plot too, same as
-- the rental_no_overlap exclusion constraint -- only a declined request
-- frees it.
WHERE NOT EXISTS (
    SELECT 1 FROM rental r
    WHERE r.plot = plot.id AND r.period @> CURRENT_TIMESTAMP AND r.status <> 'declined'
)
ORDER BY plot.coordinates <-> ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)
LIMIT sqlc.arg(result_limit);
