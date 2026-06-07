DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'profile.view_own');

DELETE FROM permissions WHERE code = 'profile.view_own';
