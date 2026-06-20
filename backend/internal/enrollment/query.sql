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

-- name: ListEnrollments :many
SELECT
  e.id, e.student_id, e.program_id, e.year, e.status, e.paid_at,
  e.created_at, e.updated_at, e.deleted_at, e.created_by, e.updated_by,
  COALESCE(NULLIF(TRIM(COALESCE(p.given_names,'')||' '||COALESCE(p.last_name_paternal,'')), ''), u.email, '') AS student_name,
  COALESCE(pr.name, '') AS program_name
FROM enrollments e
LEFT JOIN users u ON u.id = e.student_id AND u.deleted_at IS NULL
LEFT JOIN user_profiles p ON p.user_id = e.student_id AND p.deleted_at IS NULL
LEFT JOIN programs pr ON pr.id = e.program_id AND pr.deleted_at IS NULL
WHERE e.deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR e.id < sqlc.narg('page_token')::uuid)
  AND (sqlc.narg('student_id')::uuid IS NULL OR e.student_id = sqlc.narg('student_id')::uuid)
  AND (sqlc.narg('program_id')::uuid IS NULL OR e.program_id = sqlc.narg('program_id')::uuid)
  AND (sqlc.narg('year')::int IS NULL OR e.year = sqlc.narg('year')::int)
  AND (sqlc.narg('status')::text IS NULL OR e.status = sqlc.narg('status')::text)
  AND (sqlc.narg('query')::text IS NULL
       OR u.email ILIKE '%' || sqlc.narg('query') || '%' ESCAPE '\'
       OR (COALESCE(p.given_names,'')||' '||COALESCE(p.last_name_paternal,'')) ILIKE '%' || sqlc.narg('query') || '%' ESCAPE '\'
       OR pr.code ILIKE '%' || sqlc.narg('query') || '%' ESCAPE '\'
       OR pr.name ILIKE '%' || sqlc.narg('query') || '%' ESCAPE '\')
ORDER BY e.id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: ListOwnEnrollments :many
SELECT
  e.id,
  e.student_id,
  e.program_id,
  e.year,
  e.status,
  e.paid_at,
  e.created_at,
  e.updated_at,
  e.deleted_at,
  e.created_by,
  e.updated_by,
  p.name AS program_name
FROM enrollments e
JOIN programs p ON p.id = e.program_id
WHERE e.student_id = sqlc.arg('student_id')::uuid AND e.deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR e.id < sqlc.narg('page_token')::uuid)
ORDER BY e.id DESC
LIMIT sqlc.arg('row_limit')::int;
