-- name: AdminPasswordHash :one
SELECT password_hash FROM admins WHERE username = $1;

-- name: SetAdmin :exec
-- Creates the admin, or changes their password if they exist.
INSERT INTO admins (username, password_hash) VALUES ($1, $2)
ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash;
