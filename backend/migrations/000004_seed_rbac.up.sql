-- Roles
INSERT INTO roles (name) VALUES
    ('admin'),
    ('teacher'),
    ('student')
ON CONFLICT (name) DO NOTHING;

-- Permissions — initial codes matching the operations matrix (additional codes added in later migrations)
INSERT INTO permissions (code, description) VALUES
    ('users.manage',        'Manage users, roles, and permissions'),
    ('catalog.manage',      'Manage catalog (programs, courses, sections)'),
    ('enrollment.manage',   'Manage annual enrollment'),
    ('sections.enroll',     'Enroll or drop sections'),
    ('enrollment.view_own', 'View own enrollment and sections'),
    ('grades.write',        'Register or edit grades'),
    ('grades.read',         'View grades for a section'),
    ('grades.view_own',     'View own grades'),
    ('reports.read',        'Generate reports'),
    ('audit.read',          'View audit logs'),
    ('grades.override',     'Override any grade (admin only; audited)')
ON CONFLICT (code) DO NOTHING;

-- Role → permission mappings
-- admin gets all permissions via CROSS JOIN (count grows as migrations add more codes)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- teacher gets: grades.write, grades.read, reports.read
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('grades.write', 'grades.read', 'reports.read')
WHERE r.name = 'teacher'
ON CONFLICT DO NOTHING;

-- student gets: enrollment.view_own, grades.view_own
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('enrollment.view_own', 'grades.view_own')
WHERE r.name = 'student'
ON CONFLICT DO NOTHING;

-- Bootstrap admin user gets the admin role
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM (VALUES ('a0000000-0000-0000-0000-000000000001'::uuid)) AS u(id)
JOIN roles r ON r.name = 'admin'
ON CONFLICT DO NOTHING;
