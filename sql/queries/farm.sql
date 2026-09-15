-- name: InsertFarm :one
INSERT INTO farm (farmer_id, name, address, description) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: GetFarmByID :one
-- TotalSquareMeters sums every plot across every field of this farm; a farm
-- with no fields or plots gets 0, not an error.
SELECT
    farm.id,
    farm.farmer_id,
    farm.name,
    farm.address,
    farm.description,
    farm.founded_at,
    COALESCE(SUM(ST_Area(p.coordinates::geography)), 0)::float8 AS total_square_meters
FROM farm
LEFT JOIN field fi ON fi.farm = farm.id
LEFT JOIN plot p ON p.field = fi.id
WHERE farm.id = $1
GROUP BY farm.id;

-- name: GetFarmIDByFarmerID :one
-- Lean lookup for ownership checks: resolves a farmer's own farm id without
-- the area-summing join GetFarmByID does.
SELECT id FROM farm WHERE farmer_id = $1;
