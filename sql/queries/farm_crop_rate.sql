-- name: GetFarmCropRates :many
SELECT crop, price_cents_per_sqm_per_week
FROM farm_crop_rate
WHERE farm = $1
ORDER BY crop;

-- name: GetFarmCropRate :one
SELECT price_cents_per_sqm_per_week
FROM farm_crop_rate
WHERE farm = $1 AND crop = $2
LIMIT 1;

-- name: DeleteFarmCropRates :exec
DELETE FROM farm_crop_rate WHERE farm = $1;

-- name: InsertFarmCropRate :exec
INSERT INTO farm_crop_rate (farm, crop, price_cents_per_sqm_per_week)
VALUES ($1, $2, $3);
