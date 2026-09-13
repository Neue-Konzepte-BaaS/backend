-- name: InsertPlot :one
INSERT INTO plot (name, field, coordinates) VALUES ($1, $2, $3) RETURNING id;

-- name: GetPlotByID :one
SELECT id, name, field, coordinates
FROM plot
WHERE id = $1
LIMIT 1;
