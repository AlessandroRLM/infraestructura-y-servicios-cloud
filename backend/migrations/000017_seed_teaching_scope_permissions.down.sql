-- Reverse migration 000017: remove teaching-scope permission grants and codes.

-- Drop the section_teachers teacher_id index added for ListOwnSections.
DROP INDEX IF EXISTS section_teachers_teacher_id_idx;

-- Remove role_permissions grants for teacher and admin roles.
DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name IN ('teacher', 'admin'))
  AND permission_id IN (
      SELECT id FROM permissions
      WHERE code IN (
          'section.view_teaching',
          'section_enrollment.view_teaching',
          'profile.view_names'
      )
  );

-- Remove the permission codes.
DELETE FROM permissions
WHERE code IN (
    'section.view_teaching',
    'section_enrollment.view_teaching',
    'profile.view_names'
);
