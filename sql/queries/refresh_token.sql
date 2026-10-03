-- name: InsertRefreshToken :exec
INSERT INTO refresh_token (account_id, token_hash, expires_at)
VALUES ($1, $2, $3);

-- name: GetActiveRefreshTokenByHash :one
-- Expiry and revocation are both checked here, not by the caller, so
-- "revoked", "expired" and "never existed" collapse into the same
-- pgx.ErrNoRows the repository already turns into ErrNotFound.
SELECT id, account_id, token_hash, expires_at, revoked_at, created_at
FROM refresh_token
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP;

-- name: RevokeRefreshTokenByHash :exec
UPDATE refresh_token SET revoked_at = CURRENT_TIMESTAMP
WHERE token_hash = $1 AND revoked_at IS NULL;
