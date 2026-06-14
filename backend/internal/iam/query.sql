-- name: ListUsers :many
-- Keyset pagination over the users table with optional search and inline roles.
-- Ordered newest-first (id DESC, UUIDv7 = reverse chronological). display_name
-- derived via LEFT JOIN to user_profiles; roles aggregated in-query (no N+1).
SELECT u.id, u.email, u.disabled_at,
       p.given_names, p.last_name_paternal,
       (SELECT array_agg(r.name ORDER BY r.name)
        FROM user_roles ur
        JOIN roles r ON r.id = ur.role_id
        WHERE ur.user_id = u.id) AS roles
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.id AND p.deleted_at IS NULL
WHERE u.deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR u.id < sqlc.narg('page_token')::uuid)
  AND (sqlc.narg('query')::text IS NULL
       OR u.email ILIKE '%' || sqlc.narg('query') || '%'
       OR (p.given_names || ' ' || p.last_name_paternal) ILIKE '%' || sqlc.narg('query') || '%')
ORDER BY u.id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: GetUserByID :one
-- Returns identity and profile columns for a single non-deleted user by UUID.
-- LEFT JOIN user_profiles so users without a profile row still return.
SELECT u.id, u.email, u.disabled_at,
       p.given_names, p.last_name_paternal
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.id AND p.deleted_at IS NULL
WHERE u.id = sqlc.arg('user_id') AND u.deleted_at IS NULL;

-- name: GetUserRoles :many
-- Returns role names for a single user, sorted alphabetically.
SELECT r.name FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = sqlc.arg('user_id')
ORDER BY r.name;
