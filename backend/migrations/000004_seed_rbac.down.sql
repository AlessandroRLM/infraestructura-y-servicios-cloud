-- Remove seed data in FK-safe order.
-- Delete user_roles rows for the bootstrap admin first.
DELETE FROM user_roles
WHERE user_id = 'a0000000-0000-0000-0000-000000000001';

-- Remove all role_permissions rows for the three seeded roles.
DELETE FROM role_permissions
WHERE role_id IN (
    SELECT id FROM roles WHERE name IN ('admin', 'teacher', 'student')
);

-- Remove the 11 seeded permissions.
DELETE FROM permissions
WHERE code IN (
    'users.manage',
    'catalog.manage',
    'enrollment.manage',
    'sections.enroll',
    'enrollment.view_own',
    'grades.write',
    'grades.read',
    'grades.view_own',
    'reports.read',
    'audit.read',
    'grades.override'
);

-- Remove the three seeded roles.
DELETE FROM roles
WHERE name IN ('admin', 'teacher', 'student');
