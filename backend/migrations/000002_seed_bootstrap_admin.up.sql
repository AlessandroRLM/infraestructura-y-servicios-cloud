-- WARNING: DEV-ONLY PASSWORD. Production MUST rotate this hash before first login,
-- e.g. UPDATE users SET password_hash = '<bcrypt>' WHERE email = 'admin@dev.local'.
INSERT INTO users (id, email, password_hash, created_at, updated_at)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'admin@dev.local',
    '$2a$12$QFhfwzWhGAuMZPMq7srv.u95W0IdqVhblOhATUaurUcRc/0mexPeG',
    now(),
    now()
)
ON CONFLICT (email) DO NOTHING;
