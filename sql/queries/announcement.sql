-- name: InsertAnnouncement :one
-- The farm name is joined back in so a stored announcement is immediately
-- returnable in the same shape the board reads, without a second round trip.
-- field/plot are optional (sqlc.narg): NULL/NULL means every current renter,
-- same as before scoping existed.
WITH inserted AS (
    INSERT INTO announcement (farmer, subject, body, field, plot)
    VALUES ($1, $2, $3, sqlc.narg(field), sqlc.narg(plot))
    RETURNING id, farmer, subject, body, created_at, field, plot
)
SELECT i.id, i.farmer, i.subject, i.body, i.created_at, i.field, i.plot, farm.name AS farm_name
FROM inserted i
JOIN farm ON farm.farmer_id = i.farmer;

-- name: GetAnnouncementsByFarmer :many
-- The farmer's own board: what he has posted, newest first.
SELECT a.id, a.farmer, a.subject, a.body, a.created_at, a.field, a.plot, farm.name AS farm_name
FROM announcement a
JOIN farm ON farm.farmer_id = a.farmer
WHERE a.farmer = $1
ORDER BY a.created_at DESC;

-- name: GetAnnouncementsForCustomer :many
-- The customer's board: notices from every farmer he is currently renting
-- from. The rental gates which *farmers* he reads, not which notices, so the
-- board is deliberately not a record of what he was mailed: a new renter reads
-- everything that farmer has ever posted, including notices from before his
-- rental began, and an ended rental takes the whole board with it. Renting
-- several plots from the same farmer must not repeat that farmer's notices,
-- hence DISTINCT.
--
-- A scoped announcement (field or plot set) only shows up if the matched
-- rental actually covers that field/plot — fi and p are already the specific
-- field/plot the joined rental sits on, so this ties the scope to the rental
-- that qualifies the customer rather than to the farmer's holdings at large.
SELECT DISTINCT a.id, a.farmer, a.subject, a.body, a.created_at, a.field, a.plot, farm.name AS farm_name
FROM announcement a
JOIN farm ON farm.farmer_id = a.farmer
JOIN field fi ON fi.farm = farm.id
JOIN plot p ON p.field = fi.id
JOIN rental r ON r.plot = p.id
WHERE r.customer = $1 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
  AND ((a.field IS NULL AND a.plot IS NULL) OR a.field = fi.id OR a.plot = p.id)
ORDER BY a.created_at DESC;

-- name: GetCustomersOfFarmer :many
-- Everyone a farmer may address: the customers currently renting one of his
-- plots. DISTINCT is load-bearing — a customer renting three plots from the
-- same farmer is one person and must be mailed once.
SELECT DISTINCT a.id, a.email, a.first_name, a.last_name
FROM account a
JOIN customer c ON c.account_id = a.id
JOIN rental r ON r.customer = c.account_id
JOIN plot p ON p.id = r.plot
JOIN field f ON f.id = p.field
JOIN farm ON farm.id = f.farm
WHERE farm.farmer_id = $1 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
ORDER BY a.email;

-- name: GetCustomersOfFarmerForField :many
-- The audience for an announcement scoped to one field: customers with an
-- active, approved rental on a plot of that field. 'approved' for the same
-- reason as GetCustomersOfFarmer: a pending or declined request keeps a row
-- whose period covers now, and its customer is not a tenant.
SELECT DISTINCT a.id, a.email, a.first_name, a.last_name
FROM account a
JOIN customer c ON c.account_id = a.id
JOIN rental r ON r.customer = c.account_id
JOIN plot p ON p.id = r.plot
WHERE p.field = $1 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
ORDER BY a.email;

-- name: GetCustomersOfFarmerForPlot :many
-- The audience for an announcement scoped to one plot: customers with an
-- active, approved rental on that plot (at most one at a time, but a farmer
-- can still re-post after a rental ends).
SELECT DISTINCT a.id, a.email, a.first_name, a.last_name
FROM account a
JOIN customer c ON c.account_id = a.id
JOIN rental r ON r.customer = c.account_id
WHERE r.plot = $1 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
ORDER BY a.email;
