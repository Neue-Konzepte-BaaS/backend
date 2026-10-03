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
WHERE a.email = $1 AND a.deleted_at IS NULL
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
WHERE a.id = $1 AND a.deleted_at IS NULL
LIMIT 1;

-- name: GetAllRecipients :many
-- Every farmer and customer, for a platform-wide notification. Membership is
-- tested positively rather than by excluding admins, so an account with no
-- subtype row at all, and therefore an empty derived role, is never mailed.
-- A customer who opted out of email notifications is excluded; farmers have
-- no such preference and are never excluded by it.
SELECT
    a.id,
    a.email,
    a.first_name,
    a.last_name
FROM account a
LEFT JOIN customer c ON c.account_id = a.id
WHERE (
    EXISTS (SELECT 1 FROM farmer f WHERE f.account_id = a.id)
    OR (c.account_id IS NOT NULL AND c.notify_messages_by_email = true)
) AND a.deleted_at IS NULL
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

-- name: SoftDeleteAccount :execrows
-- Scrubs personal data and marks the account deleted. The WHERE guard makes
-- this idempotent -- a second call against an already-deleted id affects no
-- rows, which the repository reports as ErrNotFound, the same "nothing to
-- do" shape UpdateRentalStatus already uses for its own state guard.
UPDATE account
SET first_name = $2, last_name = $3, email = $4, password_hash = $5, deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: HasActiveRentalAsCustomer :one
-- Blocks a customer's self-deletion while they are a live tenant on some
-- plot right now. 'approved' only -- a merely pending request is not a
-- tenancy and should not block deletion; it can simply lapse or be declined.
SELECT EXISTS (
    SELECT 1 FROM rental r
    WHERE r.customer = $1 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
) AS has_active;

-- name: HasActiveRentalAsFarmer :one
-- Blocks a farmer's self-deletion while any plot of theirs is rented right
-- now. Same 'approved'-only reasoning as HasActiveRentalAsCustomer.
SELECT EXISTS (
    SELECT 1 FROM rental r
    JOIN plot p ON p.id = r.plot
    JOIN field f ON f.id = p.field
    JOIN farm ON farm.id = f.farm
    WHERE farm.farmer_id = $1 AND r.period @> CURRENT_TIMESTAMP AND r.status = 'approved'
) AS has_active;
