-- Remove role grants for section_enrollment.view_own and sections.enroll (student only).
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'student')
  AND permission_id IN (
      SELECT id FROM permissions
      WHERE code IN ('section_enrollment.view_own', 'sections.enroll')
  );

-- Remove admin grant for section_enrollment.view_own.
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'admin')
  AND permission_id = (SELECT id FROM permissions WHERE code = 'section_enrollment.view_own');

-- Remove permission code.
DELETE FROM permissions WHERE code = 'section_enrollment.view_own';
