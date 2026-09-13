-- name: InsertCrop :one
INSERT INTO crop (name, duration_months) VALUES ($1, $2) RETURNING id;

-- name: GetAllCrops :many
SELECT id, name, duration_months
FROM crop
ORDER BY name;

-- name: GetCropByID :one
SELECT id, name, duration_months
FROM crop
WHERE id = $1
LIMIT 1;

-- name: DeleteFieldCrops :exec
DELETE FROM field_crop WHERE field = $1;

-- name: InsertFieldCrop :exec
INSERT INTO field_crop (field, crop) VALUES ($1, $2);

-- name: GetCropsByField :many
SELECT c.id, c.name, c.duration_months
FROM field_crop fc
JOIN crop c ON c.id = fc.crop
WHERE fc.field = $1
ORDER BY c.name;

-- name: GetCropsByFields :many
SELECT fc.field, c.id, c.name, c.duration_months
FROM field_crop fc
JOIN crop c ON c.id = fc.crop
WHERE fc.field = ANY(sqlc.arg(fields)::uuid[])
ORDER BY c.name;
