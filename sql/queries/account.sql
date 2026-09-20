-- name: InsertAccount :one
INSERT INTO account (first_name, last_name, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: InsertFarmer :exec
-- Links an account to the farmer subtype table. The account's role is derived
-- from this membership; see GetAccountByEmail.
INSERT INTO farmer (account_id, postal_code) VALUES ($1, $2);

-- name: InsertCustomer :exec
-- Links an account to the customer subtype table.
INSERT INTO customer (account_id, postal_code) VALUES ($1, $2);

-- name: InsertAdmin :exec
-- Links an account to the admin subtype table.
INSERT INTO admin (account_id, role) VALUES ($1, $2);

-- name: GetAccountByEmail :one
-- Role is not stored on account; it is implied by which subtype table the
-- account joins to. admin.role is an admin-internal tier, not the account role.
-- postal_code lives on whichever subtype table matches (farmer/customer); an
-- admin has neither, hence the 0 default -- mirrors the frontend's own
-- "0 means unknown" convention for postalCode.
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
    END AS role,
    COALESCE(f.postal_code, c.postal_code, 0)::int AS postal_code
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
    END AS role,
    COALESCE(f.postal_code, c.postal_code, 0)::int AS postal_code
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

-- name: ListAccounts :many
-- One page of the admin account list, newest first.
--
-- Role is derived exactly the way GetAccountByEmail derives it -- by subtype
-- membership -- so there is one definition of "role" in the codebase. A CASE
-- alias cannot be referenced from WHERE, hence the CTE: the filter then applies
-- to the derived column instead of to a second copy of the expression that
-- could drift from the first.
--
-- total_count is how many rows match the filter before LIMIT, taken in the same
-- query so the count and the page come from one snapshot -- the same
-- single-round-trip rule the statistics queries follow. A page past the end
-- returns no rows, and therefore no count either.
WITH listed AS (
    SELECT
        a.id,
        a.first_name,
        a.last_name,
        a.email,
        a.created_at,
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
)
SELECT
    listed.id,
    listed.first_name,
    listed.last_name,
    listed.email,
    listed.created_at,
    listed.role,
    (COUNT(*) OVER ())::bigint AS total_count
FROM listed
-- An empty argument means "no filter", so one query serves every combination
-- of them. The service has already rejected a role that is not one of the
-- three, so an empty role here is always "any", never "unmatchable".
WHERE (sqlc.arg(role_filter)::text = '' OR listed.role = sqlc.arg(role_filter)::text)
  AND (
      sqlc.arg(search)::text = ''
      OR listed.email ILIKE '%' || sqlc.arg(search)::text || '%'
      OR listed.first_name ILIKE '%' || sqlc.arg(search)::text || '%'
      OR listed.last_name ILIKE '%' || sqlc.arg(search)::text || '%'
  )
-- created_at is not unique, so id breaks the tie. Without it two accounts
-- registered in the same transaction can swap places between page 1 and page 2
-- and one of them is never shown.
ORDER BY listed.created_at DESC, listed.id
LIMIT sqlc.arg(result_limit) OFFSET sqlc.arg(result_offset);
