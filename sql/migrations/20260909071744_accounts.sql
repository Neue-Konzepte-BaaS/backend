-- migrate:up
CREATE TABLE account (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_account_email ON account(email);

CREATE TABLE farmer (
    account_id UUID PRIMARY KEY,
    -- Specific columns for farmers belong here
    farm_name TEXT NOT NULL,
    postal_code INT NOT NULL,
    CONSTRAINT fk_farmer_account
        FOREIGN KEY (account_id)
        REFERENCES account(id)
        ON DELETE CASCADE
);

CREATE TABLE customer (
    account_id UUID PRIMARY KEY,
    -- Specific columns for customers belong here
    postal_code INT NOT NULl,
    CONSTRAINT fk_farmer_account
        FOREIGN KEY (account_id)
        REFERENCES account(id)
        ON DELETE CASCADE
);

CREATE TABLE admin (
    account_id UUID PRIMARY KEY,
    -- Specific columns for admins belong here
    role INT NOT NULL,
    CONSTRAINT fk_farmer_account
        FOREIGN KEY (account_id)
        REFERENCES account(id)
        ON DELETE CASCADE
);

-- migrate:down
DROP INDEX IF EXISTS idx_account_email;
DROP TABLE admin;
DROP TABLE customer;
DROP TABLE farmer;
DROP TABLE account;
