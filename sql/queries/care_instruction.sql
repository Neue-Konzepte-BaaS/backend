-- name: InsertCareInstruction :one
-- farm is NULL for a step of the default guide, and the farm's id for a step
-- of that farm's own guide — whose farm_care_guide marker must exist first.
INSERT INTO care_instruction (crop, farm, week, title, body)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, crop, farm, week, title, body, created_at, updated_at;

-- name: UpdateCareInstruction :one
-- Edits every field but the crop and the farm: moving an instruction to
-- another crop or guide is a different guide, not an edit, so it is a delete
-- plus an insert.
UPDATE care_instruction
SET week = $2, title = $3, body = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, crop, farm, week, title, body, created_at, updated_at;

-- name: DeleteCareInstruction :execrows
-- :execrows, unlike DeleteCrop's :exec, so the repository can tell "deleted"
-- from "there was nothing with that id" and answer 404 rather than 204.
DELETE FROM care_instruction WHERE id = $1;

-- name: GetCareInstructionByID :one
SELECT id, crop, farm, week, title, body, created_at, updated_at
FROM care_instruction
WHERE id = $1;

-- name: GetFarmCopyOfCareInstruction :one
-- The farm's copy of one default instruction, found through the id the farmer
-- was shown before the farm took the guide over.
SELECT id, crop, farm, week, title, body, created_at, updated_at
FROM care_instruction
WHERE farm = sqlc.arg(farm) AND based_on = sqlc.arg(based_on);

-- name: GetDefaultCareInstructionsByCrop :many
-- The default guide for one crop, the one an admin maintains.
SELECT id, crop, farm, week, title, body, created_at, updated_at
FROM care_instruction
WHERE crop = $1 AND farm IS NULL
ORDER BY week, created_at;

-- name: InsertFarmCareGuide :execrows
-- Marks the crop's guide as the farm's own. :execrows, so the caller copies
-- the default guide only when this call is the one that took it over.
INSERT INTO farm_care_guide (farm, crop)
VALUES (sqlc.arg(farm), sqlc.arg(crop))
ON CONFLICT DO NOTHING;

-- name: CopyDefaultCareInstructionsToFarm :exec
-- The farm's starting point is the default guide as it stands. created_at is
-- copied too, so steps sharing a week keep the order an admin gave them.
INSERT INTO care_instruction (crop, farm, week, title, body, created_at, updated_at, based_on)
SELECT d.crop, sqlc.arg(farm)::uuid, d.week, d.title, d.body, d.created_at, d.updated_at, d.id
FROM care_instruction d
WHERE d.crop = sqlc.arg(crop)::uuid AND d.farm IS NULL;

-- name: DeleteFarmCareGuide :execrows
-- Resets the crop's guide to the default for this farm: the marker's foreign
-- key takes the farm's instructions with it.
DELETE FROM farm_care_guide WHERE farm = sqlc.arg(farm) AND crop = sqlc.arg(crop);

-- name: GetEffectiveCareInstructions :many
-- The guide each (crop, farm) pair's tenants read, in one round trip: the
-- farm's own when its marker exists, the default otherwise. The scalar
-- subquery yields the farm's id or NULL, and IS NOT DISTINCT FROM matches
-- either the farm's steps or the default's NULL farm accordingly. Several
-- instructions may share a week (a week is a list of tasks, not one task), so
-- created_at breaks the tie and keeps the order they were entered in. The two
-- unnest calls in one select list zip the arrays pairwise.
SELECT
    pair.crop::uuid AS for_crop,
    pair.farm::uuid AS for_farm,
    ci.id, ci.crop, ci.farm, ci.week, ci.title, ci.body, ci.created_at, ci.updated_at
FROM (
    SELECT unnest(sqlc.arg(crops)::uuid[]) AS crop, unnest(sqlc.arg(farms)::uuid[]) AS farm
) AS pair
JOIN care_instruction ci ON ci.crop = pair.crop
WHERE ci.farm IS NOT DISTINCT FROM (
    SELECT g.farm FROM farm_care_guide g WHERE g.farm = pair.farm AND g.crop = pair.crop
)
ORDER BY ci.week, ci.created_at;

-- name: HasFarmCareGuide :one
SELECT EXISTS (
    SELECT 1 FROM farm_care_guide WHERE farm = sqlc.arg(farm) AND crop = sqlc.arg(crop)
);
