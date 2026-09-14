-- name: InsertCrop :one
INSERT INTO crop (name, duration_months) VALUES ($1, $2) RETURNING id;

-- name: DeleteCrop :exec
DELETE FROM crop WHERE id = $1;

-- name: GetAllCrops :many
SELECT id, name, duration_months
FROM crop
ORDER BY name;

-- name: GetCropByID :one
SELECT id, name, duration_months
FROM crop
WHERE id = $1
LIMIT 1;

-- name: DeletePlotCrops :exec
DELETE FROM plot_crop WHERE plot = $1;

-- name: InsertPlotCrop :exec
INSERT INTO plot_crop (plot, crop) VALUES ($1, $2);

-- name: GetCropsByPlot :many
SELECT c.id, c.name, c.duration_months
FROM plot_crop pc
JOIN crop c ON c.id = pc.crop
WHERE pc.plot = $1
ORDER BY c.name;

-- name: GetCropsByPlots :many
SELECT pc.plot, c.id, c.name, c.duration_months
FROM plot_crop pc
JOIN crop c ON c.id = pc.crop
WHERE pc.plot = ANY(sqlc.arg(plots)::uuid[])
ORDER BY c.name;
