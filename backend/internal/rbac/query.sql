-- name: GetPermissionsForUser :many
-- DISTINCT is required: a user with multiple roles that share a permission would otherwise
-- produce duplicate rows, one per (role, permission) combination.
SELECT DISTINCT p.code FROM permissions p
  JOIN role_permissions rp ON rp.permission_id = p.id
  JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1;

-- name: GetRolesForUser :many
SELECT r.name FROM roles r
  JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.name;
