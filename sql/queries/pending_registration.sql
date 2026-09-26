-- name: UpsertPendingRegistration :one
-- Re-registering the same email while a pending (unverified) registration
-- exists refreshes it -- new data, new expiry, new id -- rather than
-- erroring, so a user who lost the first email or mistyped a field is not
-- stuck. A verified account's email is handled separately by the caller
-- (GetAccountByEmail / ErrEmailTaken) before this is ever reached.
INSERT INTO pending_registration (
    first_name, last_name, email, password_hash, role,
    farm_name, address, description, postal_code, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (email) DO UPDATE SET
    first_name    = EXCLUDED.first_name,
    last_name     = EXCLUDED.last_name,
    password_hash = EXCLUDED.password_hash,
    role          = EXCLUDED.role,
    farm_name     = EXCLUDED.farm_name,
    address       = EXCLUDED.address,
    description   = EXCLUDED.description,
    postal_code   = EXCLUDED.postal_code,
    created_at    = CURRENT_TIMESTAMP,
    expires_at    = EXCLUDED.expires_at
RETURNING id;

-- name: GetPendingRegistrationByID :one
-- Expiry is checked here, not by the caller, so "expired" and "never
-- existed" collapse into the same pgx.ErrNoRows the repository already
-- turns into ErrNotFound.
SELECT id, first_name, last_name, email, password_hash, role,
       farm_name, address, description, postal_code
FROM pending_registration
WHERE id = $1 AND expires_at > CURRENT_TIMESTAMP;

-- name: DeletePendingRegistration :exec
DELETE FROM pending_registration WHERE id = $1;
