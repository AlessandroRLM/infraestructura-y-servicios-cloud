-- seed_demo.sql — Hand-written idempotent demo-data seed.
--
-- Purpose : Load a minimal but realistic demo dataset for recordings and demos.
-- Safety  : Idempotent — safe to run multiple times; no duplicates or errors on re-run.
-- Scope   : Sets the admin password to a known value and inserts demo users, catalog,
--           enrollments, evaluations, and grades.
--
-- WARNING : This script sets the admin@dev.local password and inserts demo data.
--           For use in demo/recording environments only. Do NOT run in production
--           without understanding the consequences.
--
-- Credentials:
--   Email               | Password     | Role    | Access
--   admin@dev.local     | Admin1234!   | admin   | Full access
--   teacher@dev.local   | Teacher1234! | teacher | Grades, reports
--   student1@dev.local  | Student1234! | student | Own grades and enrollments
--   student2@dev.local  | Student1234! | student | Own grades and enrollments
--   student3@dev.local  | Student1234! | student | Own grades and enrollments
--
-- UUID scheme (deterministic, hex-only):
--   a0000000-0000-0000-0000-0000000000xx  bootstrap admin (from migration 000002)
--   b0000000-0000-0000-0000-0000000000xx  seed users  (01=teacher, 02-04=students)
--   c0000000-0000-0000-0000-0000000000xx  catalog     (01=program, 02-03=courses,
--                                                      04=period, 05=quota,
--                                                      06-07=sections)
--   d0000000-0000-0000-0000-0000000000xx  enrollments (01-03=program enrollments,
--                                                      11-15=section enrollments)
--   e0000000-0000-0000-0000-0000000000xx  evaluations (01-03=PROG1, 04-05=BD1)
--   f0000000-0000-0000-0000-0000000000xx  grades

BEGIN;

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------

-- Reset the bootstrap admin password to a known demo hash (Admin1234!, $2a cost 10).
UPDATE users
SET    password_hash = '$2a$10$eHpDhX3KPlVAQ.nHBi8M8O0T4.dKTMUNQFuOxRXaup80JrjteUEsG',
       updated_at    = now()
WHERE  email = 'admin@dev.local';

INSERT INTO users (id, email, password_hash, created_at, updated_at, created_by, updated_by)
VALUES ('b0000000-0000-0000-0000-000000000001', 'teacher@dev.local',
        '$2a$10$jlxWRiqdUP56N3lAsYDZmOTIeO4s6rA2m4RQcwqQy3oZHPk7szn/6',
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, email, password_hash, created_at, updated_at, created_by, updated_by)
VALUES ('b0000000-0000-0000-0000-000000000002', 'student1@dev.local',
        '$2a$10$OOAJ3En2v/XizFZo1WPPMu6zsZDovgAUzyeypnCPddl/aZa2HC2oW',
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, email, password_hash, created_at, updated_at, created_by, updated_by)
VALUES ('b0000000-0000-0000-0000-000000000003', 'student2@dev.local',
        '$2a$10$OOAJ3En2v/XizFZo1WPPMu6zsZDovgAUzyeypnCPddl/aZa2HC2oW',
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, email, password_hash, created_at, updated_at, created_by, updated_by)
VALUES ('b0000000-0000-0000-0000-000000000004', 'student3@dev.local',
        '$2a$10$OOAJ3En2v/XizFZo1WPPMu6zsZDovgAUzyeypnCPddl/aZa2HC2oW',
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (email) DO NOTHING;

-- ---------------------------------------------------------------------------
-- User roles
-- ---------------------------------------------------------------------------

INSERT INTO user_roles (user_id, role_id, created_at)
VALUES ('b0000000-0000-0000-0000-000000000001',
        (SELECT id FROM roles WHERE name = 'teacher'),
        '2025-01-01 00:00:00+00')
ON CONFLICT (user_id, role_id) DO NOTHING;

INSERT INTO user_roles (user_id, role_id, created_at)
VALUES ('b0000000-0000-0000-0000-000000000002',
        (SELECT id FROM roles WHERE name = 'student'),
        '2025-01-01 00:00:00+00')
ON CONFLICT (user_id, role_id) DO NOTHING;

INSERT INTO user_roles (user_id, role_id, created_at)
VALUES ('b0000000-0000-0000-0000-000000000003',
        (SELECT id FROM roles WHERE name = 'student'),
        '2025-01-01 00:00:00+00')
ON CONFLICT (user_id, role_id) DO NOTHING;

INSERT INTO user_roles (user_id, role_id, created_at)
VALUES ('b0000000-0000-0000-0000-000000000004',
        (SELECT id FROM roles WHERE name = 'student'),
        '2025-01-01 00:00:00+00')
ON CONFLICT (user_id, role_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- User profiles (all 5 users including admin)
-- ---------------------------------------------------------------------------

INSERT INTO user_profiles (
    user_id, given_names, last_name_paternal, last_name_maternal,
    national_id_type, national_id, sex, nationality,
    created_at, updated_at, created_by, updated_by
) VALUES (
    'a0000000-0000-0000-0000-000000000001', 'Administrador', 'Sistema', NULL,
    'RUT', '11111111-1', 'M', 'Chilena',
    '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
    'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001'
) ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_profiles (
    user_id, given_names, last_name_paternal, last_name_maternal,
    national_id_type, national_id, sex, nationality,
    created_at, updated_at, created_by, updated_by
) VALUES (
    'b0000000-0000-0000-0000-000000000001', 'Ana', 'González', 'Muñoz',
    'RUT', '12345678-9', 'F', 'Chilena',
    '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
    'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001'
) ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_profiles (
    user_id, given_names, last_name_paternal, last_name_maternal,
    national_id_type, national_id, sex, nationality,
    created_at, updated_at, created_by, updated_by
) VALUES (
    'b0000000-0000-0000-0000-000000000002', 'Luis', 'Pérez', 'Torres',
    'RUT', '23456789-0', 'M', 'Chilena',
    '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
    'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001'
) ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_profiles (
    user_id, given_names, last_name_paternal, last_name_maternal,
    national_id_type, national_id, sex, nationality,
    created_at, updated_at, created_by, updated_by
) VALUES (
    'b0000000-0000-0000-0000-000000000003', 'Sofía', 'Ramírez', 'Castro',
    'RUT', '34567890-1', 'F', 'Chilena',
    '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
    'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001'
) ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_profiles (
    user_id, given_names, last_name_paternal, last_name_maternal,
    national_id_type, national_id, sex, nationality,
    created_at, updated_at, created_by, updated_by
) VALUES (
    'b0000000-0000-0000-0000-000000000004', 'Carlos', 'López', 'Fuentes',
    'RUT', '45678901-2', 'M', 'Chilena',
    '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
    'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001'
) ON CONFLICT (user_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Teacher and student profiles
-- ---------------------------------------------------------------------------

INSERT INTO teacher_profiles (user_id, department, title, created_at, updated_at, created_by, updated_by)
VALUES ('b0000000-0000-0000-0000-000000000001', 'Informática', 'Profesora Asistente',
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO student_profiles (user_id, admission_year, created_at, updated_at, created_by, updated_by)
VALUES ('b0000000-0000-0000-0000-000000000002', 2025,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO student_profiles (user_id, admission_year, created_at, updated_at, created_by, updated_by)
VALUES ('b0000000-0000-0000-0000-000000000003', 2025,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO student_profiles (user_id, admission_year, created_at, updated_at, created_by, updated_by)
VALUES ('b0000000-0000-0000-0000-000000000004', 2025,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (user_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Programs
-- ---------------------------------------------------------------------------

INSERT INTO programs (id, code, name, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-000000000001', 'ICOMP', 'Ingeniería en Computación',
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (code) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Courses
-- ---------------------------------------------------------------------------

-- PROG1: evaluations with weights 0.400 + 0.300 + 0.300 = 1.000
INSERT INTO courses (id, code, name, credits, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-000000000002', 'PROG1', 'Programación I', 5,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (code) DO NOTHING;

-- BD1: evaluations with weights 0.500 + 0.500 = 1.000
INSERT INTO courses (id, code, name, credits, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-000000000003', 'BD1', 'Base de Datos', 4,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (code) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Program → course links
-- ---------------------------------------------------------------------------

INSERT INTO program_courses (program_id, course_id, created_at)
VALUES ('c0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002',
        '2025-01-01 00:00:00+00')
ON CONFLICT (program_id, course_id) DO NOTHING;

INSERT INTO program_courses (program_id, course_id, created_at)
VALUES ('c0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000003',
        '2025-01-01 00:00:00+00')
ON CONFLICT (program_id, course_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Academic periods
-- ---------------------------------------------------------------------------

-- Period 2025-1: historical, closed enrollment window (for graded section enrollments).
INSERT INTO academic_periods (
    id, year, term, start_date, end_date,
    enrollment_starts_at, enrollment_ends_at,
    created_at, updated_at
) VALUES (
    'c0000000-0000-0000-0000-000000000004', 2025, 1, '2025-03-01', '2025-07-31',
    '2025-02-01 00:00:00+00', '2025-03-15 00:00:00+00',
    '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00'
) ON CONFLICT (id) DO UPDATE
    SET year = excluded.year,
        term = excluded.term,
        start_date = excluded.start_date,
        end_date = excluded.end_date,
        enrollment_starts_at = excluded.enrollment_starts_at,
        enrollment_ends_at = excluded.enrollment_ends_at,
        updated_at = excluded.updated_at;

-- Period 2026-1: enrollment window is ALWAYS open relative to run time (now - 7d / now + 30d).
-- Used for in_progress section enrollments so the demo shows an active enrollment period.
INSERT INTO academic_periods (
    id, year, term, start_date, end_date,
    enrollment_starts_at, enrollment_ends_at,
    created_at, updated_at
) VALUES (
    'c0000000-0000-0000-0000-000000000008', 2026, 1, '2026-03-01', '2026-07-31',
    now() - interval '7 days', now() + interval '30 days',
    now(), now()
) ON CONFLICT (id) DO UPDATE
    SET year = excluded.year,
        term = excluded.term,
        start_date = excluded.start_date,
        end_date = excluded.end_date,
        enrollment_starts_at = excluded.enrollment_starts_at,
        enrollment_ends_at = excluded.enrollment_ends_at,
        updated_at = excluded.updated_at;

-- ---------------------------------------------------------------------------
-- Program quotas (required for enrollment to succeed)
-- ---------------------------------------------------------------------------

INSERT INTO program_quotas (id, program_id, year, capacity, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000001', 2025, 50,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

INSERT INTO program_quotas (id, program_id, year, capacity, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000001', 2026, 50,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Sections (period 2025-1: graded; period 2026-1: in_progress)
-- ---------------------------------------------------------------------------

-- 2025-1 sections
INSERT INTO sections (id, course_id, academic_period_id, capacity, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-000000000006', 'c0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000004', 30,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

INSERT INTO sections (id, course_id, academic_period_id, capacity, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-000000000007', 'c0000000-0000-0000-0000-000000000003',
        'c0000000-0000-0000-0000-000000000004', 30,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

-- 2026-1 sections (active enrollment window)
INSERT INTO sections (id, course_id, academic_period_id, capacity, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-00000000000a', 'c0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000008', 30,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

INSERT INTO sections (id, course_id, academic_period_id, capacity, created_at, updated_at, created_by, updated_by)
VALUES ('c0000000-0000-0000-0000-00000000000b', 'c0000000-0000-0000-0000-000000000003',
        'c0000000-0000-0000-0000-000000000008', 30,
        '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Section teachers
-- ---------------------------------------------------------------------------

INSERT INTO section_teachers (section_id, teacher_id, created_at)
VALUES ('c0000000-0000-0000-0000-000000000006', 'b0000000-0000-0000-0000-000000000001',
        '2025-01-01 00:00:00+00')
ON CONFLICT (section_id, teacher_id) DO NOTHING;

INSERT INTO section_teachers (section_id, teacher_id, created_at)
VALUES ('c0000000-0000-0000-0000-000000000007', 'b0000000-0000-0000-0000-000000000001',
        '2025-01-01 00:00:00+00')
ON CONFLICT (section_id, teacher_id) DO NOTHING;

INSERT INTO section_teachers (section_id, teacher_id, created_at)
VALUES ('c0000000-0000-0000-0000-00000000000a', 'b0000000-0000-0000-0000-000000000001',
        '2025-01-01 00:00:00+00')
ON CONFLICT (section_id, teacher_id) DO NOTHING;

INSERT INTO section_teachers (section_id, teacher_id, created_at)
VALUES ('c0000000-0000-0000-0000-00000000000b', 'b0000000-0000-0000-0000-000000000001',
        '2025-01-01 00:00:00+00')
ON CONFLICT (section_id, teacher_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Enrollments (program-level, all paid)
-- year=2025 for the graded period; year=2026 for the active period
-- ---------------------------------------------------------------------------

-- 2025 enrollments (students 1, 2, 3) — historical graded period
INSERT INTO enrollments (id, student_id, program_id, year, status, paid_at, created_at, updated_at, created_by, updated_by)
VALUES ('d0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000001', 2025, 'paid',
        '2025-01-15 00:00:00+00', '2025-01-15 00:00:00+00', '2025-01-15 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (student_id, program_id, year) DO NOTHING;

INSERT INTO enrollments (id, student_id, program_id, year, status, paid_at, created_at, updated_at, created_by, updated_by)
VALUES ('d0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000003',
        'c0000000-0000-0000-0000-000000000001', 2025, 'paid',
        '2025-01-15 00:00:00+00', '2025-01-15 00:00:00+00', '2025-01-15 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (student_id, program_id, year) DO NOTHING;

INSERT INTO enrollments (id, student_id, program_id, year, status, paid_at, created_at, updated_at, created_by, updated_by)
VALUES ('d0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000004',
        'c0000000-0000-0000-0000-000000000001', 2025, 'paid',
        '2025-01-15 00:00:00+00', '2025-01-15 00:00:00+00', '2025-01-15 00:00:00+00',
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (student_id, program_id, year) DO NOTHING;

-- 2026 enrollments (students 1, 2, 3) — active period
INSERT INTO enrollments (id, student_id, program_id, year, status, paid_at, created_at, updated_at, created_by, updated_by)
VALUES ('d0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000001', 2026, 'paid',
        now(), now(), now(),
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (student_id, program_id, year) DO NOTHING;

INSERT INTO enrollments (id, student_id, program_id, year, status, paid_at, created_at, updated_at, created_by, updated_by)
VALUES ('d0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000003',
        'c0000000-0000-0000-0000-000000000001', 2026, 'paid',
        now(), now(), now(),
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (student_id, program_id, year) DO NOTHING;

INSERT INTO enrollments (id, student_id, program_id, year, status, paid_at, created_at, updated_at, created_by, updated_by)
VALUES ('d0000000-0000-0000-0000-000000000006', 'b0000000-0000-0000-0000-000000000004',
        'c0000000-0000-0000-0000-000000000001', 2026, 'paid',
        now(), now(), now(),
        'a0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001')
ON CONFLICT (student_id, program_id, year) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Evaluations
-- Course PROG1 (c0000000-2): weights 0.400 + 0.300 + 0.300 = 1.000
-- Course BD1   (c0000000-3): weights 0.500 + 0.500       = 1.000
-- ---------------------------------------------------------------------------

INSERT INTO evaluations (id, course_id, weight, position, created_at, updated_at)
VALUES ('e0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002',
        0.400, 1, '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

INSERT INTO evaluations (id, course_id, weight, position, created_at, updated_at)
VALUES ('e0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000002',
        0.300, 2, '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

INSERT INTO evaluations (id, course_id, weight, position, created_at, updated_at)
VALUES ('e0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000002',
        0.300, 3, '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

INSERT INTO evaluations (id, course_id, weight, position, created_at, updated_at)
VALUES ('e0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000003',
        0.500, 1, '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

INSERT INTO evaluations (id, course_id, weight, position, created_at, updated_at)
VALUES ('e0000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000003',
        0.500, 2, '2025-01-01 00:00:00+00', '2025-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Section enrollments (2025-1: fully graded; 2026-1: in_progress)
--
-- Graded section enrollments — final_grade arithmetic:
--
--   d0000000-11 (student1, PROG1, passed):
--     6.5×0.400 + 5.0×0.300 + 4.5×0.300 = 2.600 + 1.500 + 1.350 = 5.450 → 5.5
--
--   d0000000-12 (student2, PROG1, failed):
--     3.0×0.400 + 3.5×0.300 + 2.5×0.300 = 1.200 + 1.050 + 0.750 = 3.000 → 3.0
--
--   d0000000-14 (student1, BD1, passed):
--     6.0×0.500 + 5.0×0.500 = 3.000 + 2.500 = 5.500 → 5.5
--
-- In_progress (no final_grade):
--   d0000000-13 (student3, PROG1), d0000000-15 (student2, BD1), d0000000-16 (student3, BD1)
--   d0000000-21..26 (all 2026-1 enrollments)
-- ---------------------------------------------------------------------------

-- 2025-1, PROG1: student1 passed
INSERT INTO section_enrollments (id, enrollment_id, section_id, status, final_grade, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000011', 'd0000000-0000-0000-0000-000000000001',
        'c0000000-0000-0000-0000-000000000006', 'passed', 5.5,
        '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00')
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

-- 2025-1, PROG1: student2 failed
INSERT INTO section_enrollments (id, enrollment_id, section_id, status, final_grade, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000012', 'd0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000006', 'failed', 3.0,
        '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00')
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

-- 2025-1, PROG1: student3 in_progress (no final_grade)
INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000013', 'd0000000-0000-0000-0000-000000000003',
        'c0000000-0000-0000-0000-000000000006', 'in_progress',
        '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00')
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

-- 2025-1, BD1: student1 passed
INSERT INTO section_enrollments (id, enrollment_id, section_id, status, final_grade, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000014', 'd0000000-0000-0000-0000-000000000001',
        'c0000000-0000-0000-0000-000000000007', 'passed', 5.5,
        '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00')
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

-- 2025-1, BD1: student2 in_progress (partial grades only)
INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000015', 'd0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000007', 'in_progress',
        '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00')
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

-- 2025-1, BD1: student3 in_progress (no grades)
INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000016', 'd0000000-0000-0000-0000-000000000003',
        'c0000000-0000-0000-0000-000000000007', 'in_progress',
        '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00', '2025-02-10 00:00:00+00')
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

-- 2026-1, PROG1: all 3 students in_progress
INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000021', 'd0000000-0000-0000-0000-000000000004',
        'c0000000-0000-0000-0000-00000000000a', 'in_progress',
        now(), now(), now())
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000022', 'd0000000-0000-0000-0000-000000000005',
        'c0000000-0000-0000-0000-00000000000a', 'in_progress',
        now(), now(), now())
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000023', 'd0000000-0000-0000-0000-000000000006',
        'c0000000-0000-0000-0000-00000000000a', 'in_progress',
        now(), now(), now())
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

-- 2026-1, BD1: all 3 students in_progress
INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000024', 'd0000000-0000-0000-0000-000000000004',
        'c0000000-0000-0000-0000-00000000000b', 'in_progress',
        now(), now(), now())
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000025', 'd0000000-0000-0000-0000-000000000005',
        'c0000000-0000-0000-0000-00000000000b', 'in_progress',
        now(), now(), now())
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO section_enrollments (id, enrollment_id, section_id, status, registered_at, created_at, updated_at)
VALUES ('d0000000-0000-0000-0000-000000000026', 'd0000000-0000-0000-0000-000000000006',
        'c0000000-0000-0000-0000-00000000000b', 'in_progress',
        now(), now(), now())
ON CONFLICT (enrollment_id, section_id) WHERE deleted_at IS NULL DO NOTHING;

-- ---------------------------------------------------------------------------
-- Grades (only for fully-graded section enrollments)
--
-- Grades for d0000000-11 (student1, PROG1, 2025-1 → passed, final=5.5):
--   eval1 (w=0.400): 6.5   eval2 (w=0.300): 5.0   eval3 (w=0.300): 4.5
--   weighted sum: 6.5×0.4 + 5.0×0.3 + 4.5×0.3 = 2.600+1.500+1.350 = 5.450 → 5.5 ✓
--
-- Grades for d0000000-12 (student2, PROG1, 2025-1 → failed, final=3.0):
--   eval1 (w=0.400): 3.0   eval2 (w=0.300): 3.5   eval3 (w=0.300): 2.5
--   weighted sum: 3.0×0.4 + 3.5×0.3 + 2.5×0.3 = 1.200+1.050+0.750 = 3.000 → 3.0 ✓
--
-- Grades for d0000000-14 (student1, BD1, 2025-1 → passed, final=5.5):
--   eval4 (w=0.500): 6.0   eval5 (w=0.500): 5.0
--   weighted sum: 6.0×0.5 + 5.0×0.5 = 3.000+2.500 = 5.500 → 5.5 ✓
--
-- Partial grade for d0000000-15 (student2, BD1, 2025-1 → in_progress):
--   eval4 (w=0.500): 5.5   (eval5 not yet graded)
-- ---------------------------------------------------------------------------

-- student1, PROG1 (se d0000000-11)
INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000001',
        'e0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000011',
        'b0000000-0000-0000-0000-000000000001',
        6.5, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000002',
        'e0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000011',
        'b0000000-0000-0000-0000-000000000001',
        5.0, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000003',
        'e0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000011',
        'b0000000-0000-0000-0000-000000000001',
        4.5, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

-- student2, PROG1 (se d0000000-12)
INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000004',
        'e0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000012',
        'b0000000-0000-0000-0000-000000000001',
        3.0, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000005',
        'e0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000012',
        'b0000000-0000-0000-0000-000000000001',
        3.5, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000006',
        'e0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000012',
        'b0000000-0000-0000-0000-000000000001',
        2.5, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

-- student1, BD1 (se d0000000-14)
INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000007',
        'e0000000-0000-0000-0000-000000000004', 'd0000000-0000-0000-0000-000000000014',
        'b0000000-0000-0000-0000-000000000001',
        6.0, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000008',
        'e0000000-0000-0000-0000-000000000005', 'd0000000-0000-0000-0000-000000000014',
        'b0000000-0000-0000-0000-000000000001',
        5.0, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

-- student2, BD1 partial grade (se d0000000-15, in_progress; only eval4 graded)
INSERT INTO grades (id, evaluation_id, section_enrollment_id, graded_by, value, evaluated_at, version, created_at, updated_at, created_by, updated_by)
VALUES ('f0000000-0000-0000-0000-000000000009',
        'e0000000-0000-0000-0000-000000000004', 'd0000000-0000-0000-0000-000000000015',
        'b0000000-0000-0000-0000-000000000001',
        5.5, '2025-07-15 00:00:00+00', 1,
        '2025-07-15 00:00:00+00', '2025-07-15 00:00:00+00',
        'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001')
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING;

COMMIT;
