-- name: InsertRipenessNotice :one
-- Farm, field and crop names are joined back in so the notice is immediately
-- returnable in the shape both the response and the inbox need, without a
-- second round trip.
WITH inserted AS (
    INSERT INTO ripeness_notice (farmer, field, crop)
    VALUES ($1, $2, $3)
    RETURNING id, farmer, field, crop, created_at
)
SELECT i.id, i.farmer, i.field, i.crop, i.created_at,
       farm.name AS farm_name, fi.name AS field_name, c.name AS crop_name
FROM inserted i
JOIN farm ON farm.farmer_id = i.farmer
JOIN field fi ON fi.id = i.field
JOIN crop c ON c.id = i.crop;

-- name: GetRipenessNoticesForCustomer :many
-- Notices for fields the customer currently rents a plot on, growing exactly
-- the notice's crop — the same audience the notice was mailed to in the
-- first place. DISTINCT because renting several matching plots on the same
-- field must not repeat the notice.
SELECT DISTINCT rn.id, rn.farmer, rn.field, rn.crop, rn.created_at,
       farm.name AS farm_name, fi.name AS field_name, cr.name AS crop_name
FROM ripeness_notice rn
JOIN farm ON farm.farmer_id = rn.farmer
JOIN field fi ON fi.id = rn.field
JOIN crop cr ON cr.id = rn.crop
JOIN plot p ON p.field = fi.id
JOIN rental r ON r.plot = p.id AND r.crop = rn.crop
WHERE r.customer = $1 AND r.period @> CURRENT_TIMESTAMP
ORDER BY rn.created_at DESC;

-- name: GetCustomersOfFarmerForFieldAndCrop :many
-- Everyone to notify about ripeness: customers with an active rental on a
-- plot of this field, growing exactly this crop. DISTINCT — a customer
-- renting several matching plots is mailed once.
SELECT DISTINCT a.id, a.email, a.first_name, a.last_name
FROM account a
JOIN customer c ON c.account_id = a.id
JOIN rental r ON r.customer = c.account_id
JOIN plot p ON p.id = r.plot
WHERE p.field = $1 AND r.crop = $2 AND r.period @> CURRENT_TIMESTAMP
ORDER BY a.email;
