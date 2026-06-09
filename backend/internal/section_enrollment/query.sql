-- Cross-domain read strategy: queries here read enrollment, sections, academic_periods,
-- and program_courses schema directly. Coupling is to the shared migration schema, not
-- to Go service layers. The single transaction boundary is what makes paid+program+seat
-- checks atomic. This is the canonical approach for this codebase.

-- name: GetSectionForUpdateWithWindow :one
-- Locks the section row and fetches window columns from the joined academic period.
-- Used as lock step #1 in EnrollSectionTx; lock order is section → key row.
SELECT
    s.capacity,
    s.course_id,
    ap.enrollment_starts_at,
    ap.enrollment_ends_at
FROM sections s
JOIN academic_periods ap ON ap.id = s.academic_period_id
WHERE s.id = $1 AND s.deleted_at IS NULL
FOR UPDATE OF s;

-- name: CountActiveSeats :one
-- Non-locking snapshot count of active inscriptions for pre-check fast-fail.
-- Uses the partial index section_enrollments_active_seat_idx.
-- NOT authoritative — the under-lock count is the source of truth.
SELECT count(*)
FROM section_enrollments
WHERE section_id = $1
  AND status <> 'withdrawn'
  AND deleted_at IS NULL;

-- name: ResolvePaidEnrollmentForProgram :one
-- Resolves the paid enrollment for a student in a given program.
-- Returns ErrNoRows when no paid enrollment exists (pending/cancelled/missing).
SELECT e.id, e.student_id, e.program_id, e.status
FROM enrollments e
WHERE e.student_id = $1
  AND e.program_id = $2
  AND e.status = 'paid'
  AND e.deleted_at IS NULL
LIMIT 1;

-- name: ResolvePaidEnrollmentByID :one
-- Fetches an enrollment by id and verifies it is paid (used in admin path).
SELECT e.id, e.student_id, e.program_id, e.status
FROM enrollments e
WHERE e.id = $1
  AND e.status = 'paid'
  AND e.deleted_at IS NULL;

-- name: CourseInProgram :one
-- Checks whether a course belongs to a program's course list.
-- Returns a boolean; false (no row) means the course is not in the program.
SELECT EXISTS(
    SELECT 1 FROM program_courses
    WHERE program_id = $1 AND course_id = $2
) AS exists;

-- name: GetSectionEnrollmentByKeyForUpdate :one
-- Fetches the inscription row for (enrollment_id, section_id), including withdrawn rows,
-- with a FOR UPDATE lock for revival detection. Lock order: acquired after section lock.
SELECT * FROM section_enrollments
WHERE enrollment_id = $1 AND section_id = $2
FOR UPDATE;

-- name: InsertSectionEnrollment :one
INSERT INTO section_enrollments (enrollment_id, section_id, status)
VALUES ($1, $2, 'in_progress')
RETURNING *;

-- name: ReviveSectionEnrollment :one
-- Revives a withdrawn inscription: sets status back to in_progress and clears deleted_at.
UPDATE section_enrollments
SET status     = 'in_progress',
    deleted_at = NULL,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: WithdrawSectionEnrollment :one
-- Transitions an in_progress inscription to withdrawn.
-- Sets updated_at; does NOT set deleted_at — withdrawn rows are retained for audit/revival.
UPDATE section_enrollments
SET status     = 'withdrawn',
    updated_at = now()
WHERE id = $1 AND status = 'in_progress' AND deleted_at IS NULL
RETURNING *;

-- name: GetSectionEnrollmentByID :one
SELECT * FROM section_enrollments
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListSectionEnrollments :many
SELECT * FROM section_enrollments
WHERE deleted_at IS NULL
  AND (sqlc.narg('section_id')::uuid IS NULL   OR section_id   = sqlc.narg('section_id')::uuid)
  AND (sqlc.narg('enrollment_id')::uuid IS NULL OR enrollment_id = sqlc.narg('enrollment_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL        OR status        = sqlc.narg('status')::text)
ORDER BY created_at;

-- name: ListOwnSectionEnrollments :many
-- Returns all live inscriptions for a student by joining enrollments on student_id.
SELECT se.*
FROM section_enrollments se
JOIN enrollments e ON e.id = se.enrollment_id
WHERE e.student_id = $1
  AND se.deleted_at IS NULL
ORDER BY se.created_at;
