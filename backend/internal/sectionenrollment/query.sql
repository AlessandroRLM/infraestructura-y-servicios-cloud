-- Cross-domain read strategy: queries here read enrollment, sections, academic_periods,
-- and program_courses schema directly. Coupling is to the shared migration schema, not
-- to Go service layers. The single transaction boundary is what makes paid+program+seat
-- checks atomic. This is the canonical approach for this codebase.

-- name: GetSectionCapacity :one
-- Non-locking read of section capacity and course_id.
-- Used for the pre-check fast-fail BEFORE BeginTx; avoids acquiring the row lock
-- for sections that are obviously full or missing. Returns ErrNoRows when absent.
SELECT
    s.capacity,
    s.course_id
FROM sections s
WHERE s.id = $1 AND s.deleted_at IS NULL;

-- name: GetSectionForUpdateWithWindow :one
-- Locks the section row FOR UPDATE and fetches capacity, course_id, period_year, and
-- whether now() falls within the academic period's enrollment window (inclusive on both ends).
-- window_open=false when the window is not configured (fail-closed).
-- period_year is used by the caller to enforce that the linked enrollment matches
-- the section's academic year before inserting.
-- Used as lock step #1 in EnrollSectionTx; lock order is section → key row.
SELECT
    s.capacity,
    s.course_id,
    ap.year AS period_year,
    (
        ap.enrollment_starts_at IS NOT NULL
        AND ap.enrollment_ends_at IS NOT NULL
        AND now() BETWEEN ap.enrollment_starts_at AND ap.enrollment_ends_at
    ) AS window_open
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

-- name: ResolveEnrollmentByStudentAndProgram :one
-- Resolves an enrollment for a student in a specific program and year.
-- Returns the full status and deleted_at so the caller can distinguish:
--   not found / soft-deleted → ErrNotFound
--   found but status != 'paid' → ErrNotPaid
-- The year parameter must equal the section's academic period year so that a
-- matrícula from a different year cannot satisfy a section in another year.
SELECT e.id, e.student_id, e.program_id, e.status, e.deleted_at
FROM enrollments e
WHERE e.student_id = $1
  AND e.program_id = $2
  AND e.year = $3
  AND e.deleted_at IS NULL;

-- name: ResolveEnrollmentByID :one
-- Resolves an enrollment by id without filtering on status.
-- Returns year, status, and deleted_at so the caller can distinguish:
--   not found / soft-deleted → ErrNotFound
--   found but status != 'paid' → ErrNotPaid
--   found but year ≠ section period year → ErrEnrollmentYearMismatch
SELECT e.id, e.student_id, e.program_id, e.year, e.status, e.deleted_at
FROM enrollments e
WHERE e.id = $1
  AND e.deleted_at IS NULL;

-- name: CourseInProgram :one
-- Checks whether a course belongs to a program's course list.
-- Returns a boolean; false (no row) means the course is not in the program.
SELECT EXISTS(
    SELECT 1 FROM program_courses
    WHERE program_id = $1 AND course_id = $2
) AS exists;

-- name: GetSectionEnrollmentByKeyForUpdate :one
-- Fetches a LIVE inscription row for (enrollment_id, section_id) with a FOR UPDATE lock
-- for revival detection. Filters deleted_at IS NULL so that a soft-deleted row never
-- triggers AlreadyExists or revival logic. Lock order: acquired after section lock.
SELECT * FROM section_enrollments
WHERE enrollment_id = $1 AND section_id = $2 AND deleted_at IS NULL
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
  AND (sqlc.narg('page_token')::uuid IS NULL    OR id            < sqlc.narg('page_token')::uuid)
  AND (sqlc.narg('section_id')::uuid IS NULL    OR section_id    = sqlc.narg('section_id')::uuid)
  AND (sqlc.narg('enrollment_id')::uuid IS NULL OR enrollment_id = sqlc.narg('enrollment_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL        OR status        = sqlc.narg('status')::text)
ORDER BY id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: ListOwnSectionEnrollments :many
-- Returns live inscriptions for a student by joining enrollments on student_id.
-- Keyset pagination: results ordered by se.id DESC; page_token is the exclusive upper bound.
SELECT se.*
FROM section_enrollments se
JOIN enrollments e ON e.id = se.enrollment_id
WHERE e.student_id = sqlc.arg('student_id')::uuid
  AND se.deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR se.id < sqlc.narg('page_token')::uuid)
ORDER BY se.id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: SetSectionEnrollmentOutcome :one
-- Transitions a section_enrollment status to passed or failed and writes the
-- computed final grade, within a caller-owned transaction.
-- Source states in_progress/passed/failed are all valid (allows passed<->failed flips).
-- withdrawn source is rejected (0 rows returned — treated as ErrInvalidTransition).
-- Target must be passed or failed; in_progress is not a valid target.
UPDATE section_enrollments
SET status      = $2,
    final_grade = $3,
    updated_at  = now()
WHERE id = $1
  AND deleted_at IS NULL
  AND status IN ('in_progress', 'passed', 'failed')
  AND $2 IN ('passed', 'failed')
RETURNING *;
