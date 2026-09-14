-- name: InsertAnnouncement :one
-- The farm name is joined back in so a stored announcement is immediately
-- returnable in the same shape the board reads, without a second round trip.
WITH inserted AS (
    INSERT INTO announcement (farmer, subject, body)
    VALUES ($1, $2, $3)
    RETURNING id, farmer, subject, body, created_at
)
SELECT i.id, i.farmer, i.subject, i.body, i.created_at, f.farm_name
FROM inserted i
JOIN farmer f ON f.account_id = i.farmer;

-- name: GetAnnouncementsByFarmer :many
-- The farmer's own board: what he has posted, newest first.
SELECT a.id, a.farmer, a.subject, a.body, a.created_at, f.farm_name
FROM announcement a
JOIN farmer f ON f.account_id = a.farmer
WHERE a.farmer = $1
ORDER BY a.created_at DESC;

-- name: GetAnnouncementsForCustomer :many
-- The customer's board: notices from every farmer he is currently renting
-- from. The rental filter matches the one the fan-out uses, so a customer
-- reads exactly the announcements he was also mailed. Renting several plots
-- from the same farmer must not repeat that farmer's notices, hence DISTINCT.
SELECT DISTINCT a.id, a.farmer, a.subject, a.body, a.created_at, f.farm_name
FROM announcement a
JOIN farmer f ON f.account_id = a.farmer
JOIN field fi ON fi.farmer = f.account_id
JOIN plot p ON p.field = fi.id
JOIN rental r ON r.plot = p.id
WHERE r.customer = $1 AND r.period @> CURRENT_TIMESTAMP
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
WHERE f.farmer = $1 AND r.period @> CURRENT_TIMESTAMP
ORDER BY a.email;
