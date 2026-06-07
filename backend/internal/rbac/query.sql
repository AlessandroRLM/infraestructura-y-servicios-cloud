-- name: GetPermissionsForUser :many
SELECT p.code FROM permissions p
  JOIN role_permissions rp ON rp.permission_id = p.id
  JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1;
