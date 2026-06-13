DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'profile.edit_own');

DELETE FROM permissions WHERE code = 'profile.edit_own';
