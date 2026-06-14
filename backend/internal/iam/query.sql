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
       OR u.email ILIKE '%' || sqlc.narg('query') || '%' ESCAPE '\'
       OR (p.given_names || ' ' || p.last_name_paternal) ILIKE '%' || sqlc.narg('query') || '%' ESCAPE '\')
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

-- name: AssignRole :execrows
-- Inserts a user_roles row for the given user and role name.
-- Idempotent: ON CONFLICT DO NOTHING on the composite PK means a re-assign
-- returns 0 affected rows without error. The service layer writes an audit
-- entry on every call (including the 0-row no-op) per EC-05.
INSERT INTO user_roles (user_id, role_id)
SELECT sqlc.arg('user_id'), r.id FROM roles r WHERE r.name = sqlc.arg('role_name')
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RevokeRole :execrows
-- Hard-deletes the user_roles row for the given user and role name.
-- Returns 0 rows affected when the user does not have the role (→ ErrNotFound).
DELETE FROM user_roles ur USING roles r
WHERE ur.role_id = r.id
  AND ur.user_id = sqlc.arg('user_id')
  AND r.name = sqlc.arg('role_name');

-- name: LockAdminUserRoles :many
-- Locks the admin user_roles rows FOR UPDATE so the last-admin count and the
-- subsequent DELETE are serialized within one transaction (closes the TOCTOU
-- where two concurrent revokes could each see >1 admin and both delete).
SELECT ur.user_id FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE r.name = 'admin'
FOR UPDATE OF ur;

-- name: InsertAuditLog :exec
-- Records a role mutation audit event (role.assign or role.revoke).
-- Mirrors internal/grades/query.sql:InsertAuditLog (co-located per-slice copy).
INSERT INTO audit_logs (actor_id, action, entity, entity_id, detail)
VALUES (sqlc.arg('actor_id'), sqlc.arg('action'), sqlc.arg('entity'), sqlc.arg('entity_id'), sqlc.arg('detail'));
