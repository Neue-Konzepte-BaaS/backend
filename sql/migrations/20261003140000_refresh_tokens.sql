-- migrate:up
-- One row per issued refresh token. token_hash is SHA-256 of the JWT's jti
-- claim, never the raw token -- a DB leak must not hand out usable sessions.
-- revoked_at follows account.deleted_at's convention: NULL means active,
-- doubles as the audit timestamp of when it was blocked (rotated, logged
-- out, or account deleted). Revocation is one-way, so a timestamp fits
-- better than a boolean.
CREATE TABLE refresh_token (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES account(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_refresh_token_account_id ON refresh_token(account_id);

-- migrate:down
DROP TABLE refresh_token;
