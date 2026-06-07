-- Remove seed data in FK-safe order.
-- Delete ALL user_roles rows referencing the seeded roles before touching role_permissions or roles.
-- Scoping to the seeded role IDs (not a specific user) is necessary because any user — including
-- test fixtures — may hold one of these roles; a narrower delete would leave dangling FK rows.
DELETE FROM user_roles
WHERE role_id IN (SELECT id FROM roles WHERE name IN ('admin', 'teacher', 'student'));

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
