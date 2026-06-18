-- Seed teaching-scope permission codes and grant them to teacher and admin roles.
-- These permissions gate the new teacher read-tier RPCs:
--   section.view_teaching             → ListOwnSections
--   section_enrollment.view_teaching  → ListSectionRosterForTeacher
--   profile.view_names                → ListDisplayNamesByIDs

INSERT INTO permissions (code, description)
VALUES ('section.view_teaching', 'View sections the authenticated user teaches')
ON CONFLICT (code) DO NOTHING;

INSERT INTO permissions (code, description)
VALUES ('section_enrollment.view_teaching', 'View the student roster of sections the authenticated user teaches')
ON CONFLICT (code) DO NOTHING;

INSERT INTO permissions (code, description)
VALUES ('profile.view_names', 'Read display names (given_names + last_name_paternal) by user id')
ON CONFLICT (code) DO NOTHING;

-- Grant section.view_teaching to teacher role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'section.view_teaching'
WHERE r.name = 'teacher'
ON CONFLICT DO NOTHING;

-- Grant section_enrollment.view_teaching to teacher role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'section_enrollment.view_teaching'
WHERE r.name = 'teacher'
ON CONFLICT DO NOTHING;

-- Grant profile.view_names to teacher role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'profile.view_names'
WHERE r.name = 'teacher'
ON CONFLICT DO NOTHING;

-- Grant section.view_teaching to admin role (explicit re-grant for idempotency).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'section.view_teaching'
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Grant section_enrollment.view_teaching to admin role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'section_enrollment.view_teaching'
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Grant profile.view_names to admin role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'profile.view_names'
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Index for ListOwnSections: the composite PK (section_id, teacher_id) does not
-- serve a leading teacher_id scan; this index enables efficient teacher→sections lookup.
CREATE INDEX IF NOT EXISTS section_teachers_teacher_id_idx ON section_teachers (teacher_id);
