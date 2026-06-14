-- Seed profile.edit_own permission and grant to admin, teacher, and student roles.

INSERT INTO permissions (code, description)
VALUES ('profile.edit_own', 'Edit own contact/personal profile fields')
ON CONFLICT (code) DO NOTHING;

-- Grant profile.edit_own to admin role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'profile.edit_own'
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Grant profile.edit_own to teacher role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'profile.edit_own'
WHERE r.name = 'teacher'
ON CONFLICT DO NOTHING;

-- Grant profile.edit_own to student role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'profile.edit_own'
WHERE r.name = 'student'
ON CONFLICT DO NOTHING;
