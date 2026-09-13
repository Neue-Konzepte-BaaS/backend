-- name: FindPostalCodeCoordinates :one
SELECT coordinates
FROM postal_code
WHERE zipcode = $1
ORDER BY name
LIMIT 1;

-- name: FindPostalCodeCoordinatesByCity :one
SELECT coordinates
FROM postal_code
WHERE name ILIKE $1
ORDER BY name
LIMIT 1;
