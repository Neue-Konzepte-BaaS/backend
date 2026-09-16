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
