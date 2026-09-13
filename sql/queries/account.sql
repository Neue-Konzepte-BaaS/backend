-- name: InsertAccount :one
INSERT INTO account (first_name, last_name, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id;

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
