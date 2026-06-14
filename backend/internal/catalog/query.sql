-- Programs

-- name: InsertProgram :one
INSERT INTO programs (code, name, created_by, updated_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateProgram :one
UPDATE programs
SET code = $2, name = $3, updated_at = now(), updated_by = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetProgram :one
SELECT * FROM programs
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPrograms :many
SELECT * FROM programs
WHERE deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR id < sqlc.narg('page_token')::uuid)
ORDER BY id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: SoftDeleteProgram :execrows
UPDATE programs
SET deleted_at = now(), updated_at = now(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetProgramForUpdate :one
SELECT * FROM programs
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: CountProgramCourses :one
SELECT count(*) FROM program_courses
WHERE program_id = $1;

-- name: CountLiveProgramQuotas :one
SELECT count(*) FROM program_quotas
WHERE program_id = $1 AND deleted_at IS NULL;

-- Courses

-- name: InsertCourse :one
INSERT INTO courses (code, name, credits, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateCourse :one
UPDATE courses
SET code = $2, name = $3, credits = $4, updated_at = now(), updated_by = $5
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetCourse :one
SELECT * FROM courses
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCourses :many
SELECT * FROM courses
WHERE deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR id < sqlc.narg('page_token')::uuid)
ORDER BY id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: GetCourseForUpdate :one
SELECT * FROM courses
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: CountCourseProgramAssociations :one
SELECT count(*) FROM program_courses
WHERE course_id = $1;

-- name: SoftDeleteCourse :execrows
UPDATE courses
SET deleted_at = now(), updated_at = now(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- Program courses (M:N append-only)

-- name: InsertProgramCourse :one
INSERT INTO program_courses (program_id, course_id)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteProgramCourse :execrows
DELETE FROM program_courses
WHERE program_id = $1 AND course_id = $2;

-- name: ListProgramCoursesWithCourse :many
SELECT
    pc.program_id,
    pc.course_id,
    pc.created_at        AS association_created_at,
    c.id                 AS c_id,
    c.code               AS c_code,
    c.name               AS c_name,
    c.credits            AS c_credits,
    c.created_at         AS c_created_at,
    c.updated_at         AS c_updated_at,
    c.deleted_at         AS c_deleted_at,
    c.created_by         AS c_created_by,
    c.updated_by         AS c_updated_by
FROM program_courses pc
JOIN courses c ON c.id = pc.course_id AND c.deleted_at IS NULL
WHERE pc.program_id = $1
ORDER BY pc.created_at;

-- Academic periods

-- name: InsertAcademicPeriod :one
INSERT INTO academic_periods (year, term, start_date, end_date)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAcademicPeriod :one
UPDATE academic_periods
SET year = $2, term = $3, start_date = $4, end_date = $5, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetAcademicPeriod :one
SELECT * FROM academic_periods
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAcademicPeriods :many
SELECT * FROM academic_periods
WHERE deleted_at IS NULL
ORDER BY year, term;

-- name: GetAcademicPeriodForUpdate :one
SELECT * FROM academic_periods
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: SoftDeleteAcademicPeriod :execrows
UPDATE academic_periods
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- Program quotas

-- name: UpsertProgramQuota :one
INSERT INTO program_quotas (program_id, year, capacity, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (program_id, year) DO UPDATE
  SET capacity   = EXCLUDED.capacity,
      updated_at = now(),
      updated_by = EXCLUDED.updated_by,
      deleted_at = NULL
RETURNING *;

-- name: UpdateProgramQuota :one
UPDATE program_quotas
SET year = $2, capacity = $3, updated_at = now(), updated_by = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetProgramQuota :one
SELECT * FROM program_quotas
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProgramQuotas :many
SELECT * FROM program_quotas
WHERE program_id = $1 AND deleted_at IS NULL
ORDER BY year;

-- name: SoftDeleteProgramQuota :execrows
UPDATE program_quotas
SET deleted_at = now(), updated_at = now(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- Sections

-- name: InsertSection :one
INSERT INTO sections (course_id, academic_period_id, capacity, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateSection :one
UPDATE sections
SET capacity = $2, updated_at = now(), updated_by = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetSection :one
SELECT * FROM sections
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListSections :many
SELECT * FROM sections
WHERE deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR id < sqlc.narg('page_token')::uuid)
  AND (sqlc.narg('course_id')::uuid IS NULL OR course_id = sqlc.narg('course_id')::uuid)
  AND (sqlc.narg('academic_period_id')::uuid IS NULL OR academic_period_id = sqlc.narg('academic_period_id')::uuid)
ORDER BY id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: GetSectionForUpdate :one
SELECT * FROM sections
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: SoftDeleteSection :execrows
UPDATE sections
SET deleted_at = now(), updated_at = now(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: CountLiveSectionsByCourse :one
SELECT count(*) FROM sections
WHERE course_id = $1 AND deleted_at IS NULL;

-- name: CountLiveSectionsByAcademicPeriod :one
SELECT count(*) FROM sections
WHERE academic_period_id = $1 AND deleted_at IS NULL;

-- Section teachers (M:N append-only)

-- name: InsertSectionTeacher :one
INSERT INTO section_teachers (section_id, teacher_id)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteSectionTeacher :execrows
DELETE FROM section_teachers
WHERE section_id = $1 AND teacher_id = $2;

-- name: ListSectionTeachers :many
SELECT * FROM section_teachers
WHERE section_id = $1
ORDER BY created_at;

-- name: CountSectionTeachers :one
SELECT count(*) FROM section_teachers
WHERE section_id = $1;
