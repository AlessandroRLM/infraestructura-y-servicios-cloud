-- name: GetUserByEmail :one
SELECT id, email, password_hash, created_at, updated_at, created_by, updated_by, deleted_at
FROM users
WHERE email = $1
  AND deleted_at IS NULL;

-- name: UpdatePasswordHash :exec
UPDATE users
SET password_hash = $2,
    updated_at    = now()
WHERE id = $1;
