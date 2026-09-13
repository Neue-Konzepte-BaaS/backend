-- name: InsertField :one
INSERT INTO field (name, farmer, coordinates) VALUES ($1, $2, $3) RETURNING id;

-- name: GetFieldByID :one
SELECT id, name, farmer, coordinates
FROM field
WHERE id = $1
LIMIT 1;

-- name: GetFieldOwner :one
SELECT farmer
FROM field
WHERE id = $1
LIMIT 1;

-- name: GetFieldsByFarmer :many
SELECT id, name, farmer, coordinates
FROM field
WHERE farmer = $1
ORDER BY name;
