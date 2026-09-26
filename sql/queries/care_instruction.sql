-- name: InsertCareInstruction :one
INSERT INTO care_instruction (crop, week, title, body)
VALUES ($1, $2, $3, $4)
RETURNING id, crop, week, title, body, created_at, updated_at;

-- name: UpdateCareInstruction :one
-- Edits every field but the crop: moving an instruction to another crop is a
-- different guide, not an edit, so it is a delete plus an insert.
UPDATE care_instruction
SET week = $2, title = $3, body = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, crop, week, title, body, created_at, updated_at;

-- name: DeleteCareInstruction :execrows
-- :execrows, unlike DeleteCrop's :exec, so the repository can tell "deleted"
-- from "there was nothing with that id" and answer 404 rather than 204.
DELETE FROM care_instruction WHERE id = $1;

-- name: GetCareInstructionsByCrop :many
SELECT id, crop, week, title, body, created_at, updated_at
FROM care_instruction
WHERE crop = $1
ORDER BY week, created_at;

-- name: GetCareInstructionsByCrops :many
-- The customer-facing read: one round trip for every crop the customer is
-- currently growing, grouped by crop in the repository. Several instructions
-- may share a week (a week is a list of tasks, not one task), so created_at
-- breaks the tie and keeps the order an admin entered them in.
SELECT id, crop, week, title, body, created_at, updated_at
FROM care_instruction
WHERE crop = ANY(sqlc.arg(crops)::uuid[])
ORDER BY week, created_at;
