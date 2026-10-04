-- name: InsertRipenessNotice :one
-- Farm, plot and crop names are joined back in so the notice is immediately
-- returnable in the shape both the response and the inbox need, without a
-- second round trip.
WITH inserted AS (
    INSERT INTO ripeness_notice (farmer, plot, crop)
    VALUES ($1, $2, $3)
    RETURNING id, farmer, plot, crop, created_at
)
SELECT i.id, i.farmer, i.plot, i.crop, i.created_at,
       farm.name AS farm_name, pl.name AS plot_name, c.name_de AS crop_name
FROM inserted i
JOIN farm ON farm.farmer_id = i.farmer
JOIN plot pl ON pl.id = i.plot
JOIN crop c ON c.id = i.crop;

-- name: GetRipenessNoticesForCustomer :many
-- Notices for plots the customer currently rents, growing exactly the
-- notice's crop — the same audience the notice was mailed to in the first
-- place, so it also requires an approved rental: a pending or declined
-- request keeps a row whose period covers now.
SELECT DISTINCT rn.id, rn.farmer, rn.plot, rn.crop, rn.created_at,
       farm.name AS farm_name, pl.name AS plot_name, cr.name_de AS crop_name
FROM ripeness_notice rn
JOIN farm ON farm.farmer_id = rn.farmer
JOIN plot pl ON pl.id = rn.plot
JOIN crop cr ON cr.id = rn.crop
JOIN rental r ON r.plot = rn.plot AND r.crop = rn.crop
WHERE r.customer = $1 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
ORDER BY rn.created_at DESC;

-- name: GetCustomersOfFarmerForPlotAndCrop :many
-- Everyone to notify about ripeness: customers with an active, approved
-- rental on this exact plot, growing exactly this crop, who have not opted
-- out of email notifications.
SELECT DISTINCT a.id, a.email, a.first_name, a.last_name
FROM account a
JOIN customer c ON c.account_id = a.id
JOIN rental r ON r.customer = c.account_id
WHERE r.plot = $1 AND r.crop = $2 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
  AND a.deleted_at IS NULL
  AND c.notify_messages_by_email = true
ORDER BY a.email;
