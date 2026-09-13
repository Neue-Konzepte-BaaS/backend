-- name: InsertPlot :one
INSERT INTO plot (name, field, coordinates) VALUES ($1, $2, $3) RETURNING id;

-- name: GetPlotByID :one
SELECT id, name, field, coordinates
FROM plot
WHERE id = $1
LIMIT 1;

-- name: GetPlotsByFields :many
SELECT id, name, field, coordinates
FROM plot
WHERE field = ANY($1::uuid[])
ORDER BY name;
