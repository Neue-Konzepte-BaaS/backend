-- name: InsertCropSeasonRule :one
-- farm is NULL for the default rule, or a farm id for that farm's own.
-- season must belong to that same farm (or be a default, for a default
-- rule) - the caller validates this before inserting, since a plain FK
-- cannot express it.
INSERT INTO season_crop (season_id, crop_id, farm_id)
VALUES (sqlc.arg(season_id), sqlc.arg(crop_id), sqlc.arg(farm_id))
RETURNING id, season_id, crop_id, farm_id, created_at, updated_at;

-- name: UpdateCropSeasonRule :one
-- Only the season a rule points at can change; the crop and farm it belongs
-- to are fixed at creation.
UPDATE season_crop
SET season_id = sqlc.arg(season_id), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, season_id, crop_id, farm_id, created_at, updated_at;

-- name: DeleteCropSeasonRule :execrows
DELETE FROM season_crop WHERE id = sqlc.arg(id);

-- name: GetCropSeasonRuleByID :one
SELECT id, season_id, crop_id, farm_id, created_at, updated_at
FROM season_crop WHERE id = sqlc.arg(id);

-- name: GetCropSeasonRuleForCrop :one
-- The rule for this crop in exactly one set: the default rule when farm is
-- NULL, that farm's own rule otherwise. Unlike GetEffectiveSeasonForCrop,
-- this never falls back from a farm's set to the defaults - it is how a
-- write finds the exact row it is repointing or removing.
SELECT id, season_id, crop_id, farm_id, created_at, updated_at
FROM season_crop
WHERE crop_id = sqlc.arg(crop) AND farm_id IS NOT DISTINCT FROM sqlc.arg(farm);

-- name: GetEffectiveSeasonForCrop :one
-- The season a crop is checked against for a given farm: that farm's own
-- rule if it has one, the default rule otherwise. Returns no rows if
-- neither exists, meaning the crop is unrestricted for that farm.
SELECT s.id, s.farm, s.name, s.start_month, s.start_day, s.end_month, s.end_day, s.created_at, s.updated_at
FROM season_crop sc
JOIN season s ON s.id = sc.season_id
WHERE sc.crop_id = sqlc.arg(crop) AND sc.farm_id = sqlc.arg(farm)
UNION ALL
SELECT s.id, s.farm, s.name, s.start_month, s.start_day, s.end_month, s.end_day, s.created_at, s.updated_at
FROM season_crop sc
JOIN season s ON s.id = sc.season_id
WHERE sc.crop_id = sqlc.arg(crop) AND sc.farm_id IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM season_crop other WHERE other.crop_id = sqlc.arg(crop) AND other.farm_id = sqlc.arg(farm)
  )
LIMIT 1;

-- name: GetEffectiveSeasonsForCrops :many
-- The batched form of GetEffectiveSeasonForCrop: the season each (crop,
-- farm) pair is checked against, in one round trip - a plot listing with
-- several crops reads all of them at once instead of once per crop. A pair
-- with no rule at all, default or farm-owned, is absent from the result
-- rather than a row with nulls.
SELECT
    pair.crop::uuid AS for_crop,
    pair.farm::uuid AS for_farm,
    s.id, s.farm, s.name, s.start_month, s.start_day, s.end_month, s.end_day, s.created_at, s.updated_at
FROM (
    SELECT unnest(sqlc.arg(crops)::uuid[]) AS crop, unnest(sqlc.arg(farms)::uuid[]) AS farm
) AS pair
JOIN season_crop sc ON sc.crop_id = pair.crop
    AND sc.farm_id IS NOT DISTINCT FROM (
        SELECT farm_id FROM season_crop WHERE crop_id = pair.crop AND farm_id = pair.farm
        UNION ALL
        SELECT NULL WHERE NOT EXISTS (
            SELECT 1 FROM season_crop WHERE crop_id = pair.crop AND farm_id = pair.farm
        )
        LIMIT 1
    )
JOIN season s ON s.id = sc.season_id;
