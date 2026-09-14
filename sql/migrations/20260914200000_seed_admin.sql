-- migrate:up
-- Seed a default admin account for development / demo purposes.
-- Credentials: admin@baas.dev / admin123
-- The hash was generated with argon2id (m=19456, t=2, p=1) — the same
-- parameters the Go service uses at runtime (see internal/credentials/password.go).
-- Replace this hash before deploying to any non-development environment.
DO $$
DECLARE
    admin_id UUID;
BEGIN
    INSERT INTO account (first_name, last_name, email, password_hash)
    VALUES ('Admin', 'BaaS', 'admin@baas.dev',
            '$argon2id$v=19$m=19456,t=2,p=1$FXAQw782AswhDZB2fHZ9Rw$rMZcKmASY+pM0lqfadYBs78F6z6vfVTsBbLWg8loX2U')
    ON CONFLICT (email) DO NOTHING
    RETURNING id INTO admin_id;

    IF admin_id IS NOT NULL THEN
        INSERT INTO admin (account_id, role) VALUES (admin_id, 1);
    END IF;
END $$;

-- Seed a small crop catalog so the app is usable out of the box.
INSERT INTO crop (name, duration_months) VALUES
    ('Tomaten',    4),
    ('Kartoffeln', 3),
    ('Karotten',   5),
    ('Salat',      2),
    ('Zucchini',   3),
    ('Kürbis',     4),
    ('Bohnen',     3),
    ('Erbsen',     2)
ON CONFLICT (name) DO NOTHING;

-- migrate:down
DELETE FROM account WHERE email = 'admin@baas.dev';
DELETE FROM crop WHERE name IN ('Tomaten','Kartoffeln','Karotten','Salat','Zucchini','Kürbis','Bohnen','Erbsen');
