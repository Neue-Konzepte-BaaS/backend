-- name: InsertCrop :one
INSERT INTO crop (name_de, name_en, duration_months) VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateCrop :one
UPDATE crop SET name_de = $2, name_en = $3, duration_months = $4
WHERE id = $1
RETURNING id, name_de, name_en, duration_months;

-- name: DeleteCrop :exec
DELETE FROM crop WHERE id = $1;

-- name: GetAllCrops :many
SELECT id, name_de, name_en, duration_months
FROM crop
WHERE NOT is_placeholder
ORDER BY name_de;

-- name: GetCropByID :one
SELECT id, name_de, name_en, duration_months
FROM crop
WHERE id = $1
LIMIT 1;

-- name: DeletePlotCrops :exec
DELETE FROM plot_crop WHERE plot = $1;

-- name: InsertPlotCrop :exec
INSERT INTO plot_crop (plot, crop) VALUES ($1, $2);

-- name: GetCropsByPlot :many
SELECT c.id, c.name_de, c.name_en, c.duration_months
FROM plot_crop pc
JOIN crop c ON c.id = pc.crop
WHERE pc.plot = $1
ORDER BY c.name_de;

-- name: GetCropsByPlots :many
SELECT pc.plot, c.id, c.name_de, c.name_en, c.duration_months
FROM plot_crop pc
JOIN crop c ON c.id = pc.crop
WHERE pc.plot = ANY(sqlc.arg(plots)::uuid[])
ORDER BY c.name_de;

-- name: GetPricedCropOfferingsByPlots :many
-- Only rows where both halves of the price are set: the plot's own base
-- rate, and the farm's rate for that crop. A crop missing either is not a
-- real offer from a customer's perspective (see PlotSearchService), so it
-- must not appear in the results at all -- not with a null or zero price.
SELECT
    pc.plot,
    c.id, c.name_de, c.name_en, c.duration_months,
    p.base_price_cents_per_sqm_per_week,
    fcr.price_cents_per_sqm_per_week AS farm_crop_rate_cents_per_sqm_per_week,
    ST_Area(p.coordinates::geography)::float8 AS area_square_meters
FROM plot_crop pc
JOIN plot p ON p.id = pc.plot
JOIN crop c ON c.id = pc.crop
JOIN field f ON f.id = p.field
JOIN farm_crop_rate fcr ON fcr.farm = f.farm AND fcr.crop = pc.crop
WHERE pc.plot = ANY(sqlc.arg(plots)::uuid[])
  AND p.base_price_cents_per_sqm_per_week IS NOT NULL
ORDER BY c.name_de;
