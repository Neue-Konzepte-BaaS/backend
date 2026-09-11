-- name: InsertAccount :one
INSERT INTO account (first_name, last_name, email, password_hash, salt) VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: GetAccountByEmail :one
SELECT id, first_name, last_name, password_hash, salt FROM account WHERE email = $1 LIMIT 1;
