INSERT INTO permissions (code, description)
VALUES ('profile.view_own', 'View own personal profile')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'profile.view_own'
WHERE r.name IN ('admin', 'teacher', 'student')
ON CONFLICT DO NOTHING;
