-- name: InsertSeason :one
-- farm is NULL for a default season, or a farm id for that farm's own.
INSERT INTO season (farm, name, start_month, start_day, end_month, end_day)
VALUES (sqlc.arg(farm), sqlc.arg(name), sqlc.arg(start_month), sqlc.arg(start_day), sqlc.arg(end_month), sqlc.arg(end_day))
RETURNING id, farm, name, start_month, start_day, end_month, end_day, created_at, updated_at;

-- name: UpdateSeason :one
UPDATE season
SET name = sqlc.arg(name), start_month = sqlc.arg(start_month), start_day = sqlc.arg(start_day),
    end_month = sqlc.arg(end_month), end_day = sqlc.arg(end_day), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, farm, name, start_month, start_day, end_month, end_day, created_at, updated_at;

-- name: DeleteSeason :execrows
DELETE FROM season WHERE id = sqlc.arg(id);

-- name: GetSeasonByID :one
SELECT id, farm, name, start_month, start_day, end_month, end_day, created_at, updated_at
FROM season WHERE id = sqlc.arg(id);

-- name: GetDefaultSeasons :many
SELECT id, farm, name, start_month, start_day, end_month, end_day, created_at, updated_at
FROM season WHERE farm IS NULL ORDER BY start_month, start_day;

-- name: GetFarmSeasons :many
SELECT id, farm, name, start_month, start_day, end_month, end_day, created_at, updated_at
FROM season WHERE farm = sqlc.arg(farm) ORDER BY start_month, start_day;
