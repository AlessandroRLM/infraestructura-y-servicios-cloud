-- name: LockProgramQuotaForYear :one
SELECT capacity FROM program_quotas
WHERE program_id = $1 AND year = $2 AND deleted_at IS NULL
FOR UPDATE;

-- name: CountActiveEnrollments :one
SELECT count(*) FROM enrollments
WHERE program_id = $1 AND year = $2
  AND status <> 'cancelled' AND deleted_at IS NULL;

-- name: GetEnrollmentByKeyForUpdate :one
SELECT * FROM enrollments
WHERE student_id = $1 AND program_id = $2 AND year = $3
FOR UPDATE;

-- name: InsertEnrollment :one
INSERT INTO enrollments (student_id, program_id, year, status, created_by, updated_by)
VALUES ($1, $2, $3, 'pending', $4, $5)
RETURNING *;

-- name: ReviveEnrollment :one
UPDATE enrollments
SET status = 'pending', paid_at = NULL, deleted_at = NULL,
    updated_at = now(), updated_by = $2
WHERE id = $1
RETURNING *;

-- name: MarkEnrollmentPaid :one
UPDATE enrollments
SET status = 'paid', paid_at = now(), updated_at = now(), updated_by = $2
WHERE id = $1 AND status = 'pending' AND deleted_at IS NULL
RETURNING *;

-- name: CancelEnrollment :execrows
UPDATE enrollments
SET status = 'cancelled', updated_at = now(), updated_by = $2
WHERE id = $1 AND status IN ('pending', 'paid') AND deleted_at IS NULL;

-- name: GetEnrollment :one
SELECT * FROM enrollments WHERE id = $1 AND deleted_at IS NULL;

-- name: GetEnrollmentAny :one
SELECT * FROM enrollments WHERE id = $1;

-- name: ListEnrollments :many
SELECT * FROM enrollments
WHERE deleted_at IS NULL
  AND (sqlc.narg('student_id')::uuid IS NULL OR student_id = sqlc.narg('student_id')::uuid)
  AND (sqlc.narg('program_id')::uuid IS NULL OR program_id = sqlc.narg('program_id')::uuid)
  AND (sqlc.narg('year')::int IS NULL OR year = sqlc.narg('year')::int)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at;

-- name: ListOwnEnrollments :many
SELECT * FROM enrollments
WHERE student_id = $1 AND deleted_at IS NULL
ORDER BY created_at;
