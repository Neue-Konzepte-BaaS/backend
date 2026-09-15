-- name: InsertAccount :one
INSERT INTO account (first_name, last_name, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: InsertFarmer :exec
-- Links an account to the farmer subtype table. The account's role is derived
-- from this membership; see GetAccountByEmail.
INSERT INTO farmer (account_id, farm_name, postal_code) VALUES ($1, $2, $3);

-- name: InsertCustomer :exec
-- Links an account to the customer subtype table.
INSERT INTO customer (account_id, postal_code) VALUES ($1, $2);

-- name: InsertAdmin :exec
-- Links an account to the admin subtype table.
INSERT INTO admin (account_id, role) VALUES ($1, $2);

-- name: GetAccountByEmail :one
-- Role is not stored on account; it is implied by which subtype table the
-- account joins to. admin.role is an admin-internal tier, not the account role.
SELECT
    a.id,
    a.first_name,
    a.last_name,
    a.password_hash,
    a.email,
    CASE
        WHEN ad.account_id IS NOT NULL THEN 'admin'
        WHEN f.account_id IS NOT NULL THEN 'farmer'
        WHEN c.account_id IS NOT NULL THEN 'customer'
        ELSE ''
    END AS role
FROM account a
LEFT JOIN admin ad ON ad.account_id = a.id
LEFT JOIN farmer f ON f.account_id = a.id
LEFT JOIN customer c ON c.account_id = a.id
WHERE a.email = $1
LIMIT 1;

-- name: GetAccountByID :one
SELECT
    a.id,
    a.first_name,
    a.last_name,
    a.email,
    CASE
        WHEN ad.account_id IS NOT NULL THEN 'admin'
        WHEN f.account_id IS NOT NULL THEN 'farmer'
        WHEN c.account_id IS NOT NULL THEN 'customer'
        ELSE ''
    END AS role
FROM account a
LEFT JOIN admin ad ON ad.account_id = a.id
LEFT JOIN farmer f ON f.account_id = a.id
LEFT JOIN customer c ON c.account_id = a.id
WHERE a.id = $1
LIMIT 1;

-- name: GetAllRecipients :many
-- Every farmer and customer, for a platform-wide notification. Membership is
-- tested positively rather than by excluding admins, so an account with no
-- subtype row at all, and therefore an empty derived role, is never mailed.
SELECT
    a.id,
    a.email,
    a.first_name,
    a.last_name
FROM account a
WHERE EXISTS (SELECT 1 FROM farmer f WHERE f.account_id = a.id)
   OR EXISTS (SELECT 1 FROM customer c WHERE c.account_id = a.id)
ORDER BY a.email;
