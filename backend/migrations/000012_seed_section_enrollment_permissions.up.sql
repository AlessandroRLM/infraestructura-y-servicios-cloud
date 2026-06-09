-- Seed section_enrollment.view_own permission and grant to student role.
-- Also grant the pre-existing sections.enroll permission to the student role
-- (the code was seeded in 000004; only the role assignment is new here).

INSERT INTO permissions (code, description)
VALUES ('section_enrollment.view_own', 'View own section enrollment records')
ON CONFLICT (code) DO NOTHING;

-- Grant section_enrollment.view_own to student role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'section_enrollment.view_own'
WHERE r.name = 'student'
ON CONFLICT DO NOTHING;

-- Grant sections.enroll to student role.
-- The permission code was inserted in 000004; only the student role assignment is new.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'sections.enroll'
WHERE r.name = 'student'
ON CONFLICT DO NOTHING;

-- Admin role picks up new permissions via the CROSS JOIN grant in 000004.
-- Re-run it here to ensure the new code is covered without duplication.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'section_enrollment.view_own'
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;
