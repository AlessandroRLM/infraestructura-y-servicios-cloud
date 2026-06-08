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
ORDER BY created_at;

-- name: SoftDeleteProgram :exec
UPDATE programs
SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL;

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
ORDER BY created_at;

-- name: SoftDeleteCourse :exec
UPDATE courses
SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- Program courses (M:N append-only)

-- name: InsertProgramCourse :one
INSERT INTO program_courses (program_id, course_id)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteProgramCourse :exec
DELETE FROM program_courses
WHERE program_id = $1 AND course_id = $2;

-- name: ListProgramCourses :many
SELECT * FROM program_courses
WHERE program_id = $1
ORDER BY created_at;

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

-- name: SoftDeleteAcademicPeriod :exec
UPDATE academic_periods
SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- Program quotas

-- name: InsertProgramQuota :one
INSERT INTO program_quotas (program_id, year, capacity, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5)
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

-- name: SoftDeleteProgramQuota :exec
UPDATE program_quotas
SET deleted_at = now(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL;
