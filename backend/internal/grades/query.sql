-- Cross-domain read strategy: queries here read sections and section_teachers directly
-- as the canonical cross-domain join pattern for this codebase. The single transaction
-- boundary makes all checks atomic.

-- name: InsertEvaluation :one
-- Inserts one evaluation for a course scheme with an explicit position.
INSERT INTO evaluations (course_id, weight, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListEvaluationsForCourse :many
-- Returns all live evaluations for a course ordered by position.
SELECT * FROM evaluations
WHERE course_id = $1 AND deleted_at IS NULL
ORDER BY position;

-- name: CountLiveEvaluationsForCourse :one
-- Counts live evaluations for a course. Used to check scheme completeness.
SELECT COUNT(*) FROM evaluations
WHERE course_id = $1 AND deleted_at IS NULL;

-- name: LockEvaluationsForCourse :many
-- Acquires FOR UPDATE row locks on all live evaluation rows for a course.
-- Must run before CountGradesForEvaluations inside RecreateEvaluationSchemeTx so that
-- a concurrent RecordGradeTx (which holds FOR KEY SHARE on the evaluation via the FK)
-- either blocks this recreate until it commits (and is then counted) or is blocked by
-- this lock until the recreate commits — preventing a TOCTOU race under READ COMMITTED.
SELECT id FROM evaluations
WHERE course_id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: SoftDeleteEvaluationsForCourse :exec
-- Soft-deletes all live evaluations for a course (RecreateEvaluationScheme path).
UPDATE evaluations
SET deleted_at = now(), updated_at = now()
WHERE course_id = $1 AND deleted_at IS NULL;

-- name: CountGradesForEvaluations :one
-- Counts grades referencing any live evaluation of a course.
-- Used to gate RecreateEvaluationScheme: non-zero means the scheme is in use.
SELECT COUNT(*) FROM grades g
JOIN evaluations e ON e.id = g.evaluation_id
WHERE e.course_id = $1 AND e.deleted_at IS NULL;

-- name: InsertGrade :one
-- Inserts a new grade. ON CONFLICT DO NOTHING so that the caller can detect
-- whether an existing grade was hit (0 rows returned) or a new one was created.
INSERT INTO grades (evaluation_id, section_enrollment_id, graded_by, value, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (evaluation_id, section_enrollment_id) DO NOTHING
RETURNING *;

-- name: UpdateGradeByVersion :one
-- Applies an optimistic-lock update: increments version and sets updated fields.
-- Returns 0 rows when the version does not match (conflict).
UPDATE grades
SET value      = $3,
    version    = version + 1,
    graded_by  = $4,
    evaluated_at = now(),
    updated_at = now(),
    updated_by = $5
WHERE id = $1
  AND version = $2
RETURNING *;

-- name: GetGradeByKey :one
-- Fetches a grade by its unique business key (used to read current version on conflict).
SELECT * FROM grades
WHERE evaluation_id = $1
  AND section_enrollment_id = $2
  AND deleted_at IS NULL;

-- name: GetGradeByID :one
-- Fetches a grade by primary key.
SELECT * FROM grades
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListGradesBySectionEnrollment :many
-- Lists all grades for a section_enrollment (used for weighted-final computation).
SELECT * FROM grades
WHERE section_enrollment_id = $1 AND deleted_at IS NULL;

-- name: GetSectionEnrollmentForGrade :one
-- Locks the section_enrollment row FOR UPDATE and resolves section_id and course_id.
-- This is the serialization point for all concurrent grade writes targeting the same SE.
SELECT
    se.id,
    se.enrollment_id,
    se.section_id,
    se.status,
    se.final_grade,
    s.course_id,
    e.student_id
FROM section_enrollments se
JOIN sections s ON s.id = se.section_id
JOIN enrollments e ON e.id = se.enrollment_id
WHERE se.id = $1 AND se.deleted_at IS NULL
FOR UPDATE OF se;

-- name: ListGradesForSection :many
-- Lists grades for all section_enrollments in a section.
-- Caller must scope by their section_teachers membership in the service layer.
SELECT g.*
FROM grades g
JOIN section_enrollments se ON se.id = g.section_enrollment_id
WHERE se.section_id = $1
  AND g.deleted_at IS NULL
  AND se.deleted_at IS NULL;

-- name: ListGradesForSectionByTeacher :many
-- Lists grades for all section_enrollments in a section, scoped to a teacher.
-- Returns empty if the teacher is not in section_teachers for the section.
SELECT g.*
FROM grades g
JOIN section_enrollments se ON se.id = g.section_enrollment_id
WHERE se.section_id = $1
  AND g.deleted_at IS NULL
  AND se.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM section_teachers st
    WHERE st.section_id = $1 AND st.teacher_id = $2
  );

-- name: ListGradesForSectionPaged :many
-- Keyset-paginated list of grades for a section (admin path, no teacher scope).
-- Ordered by g.id DESC. page_token is the exclusive upper bound on g.id (NULL = start).
SELECT g.*
FROM grades g
JOIN section_enrollments se ON se.id = g.section_enrollment_id
WHERE se.section_id = sqlc.arg('section_id')
  AND g.deleted_at IS NULL
  AND se.deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR g.id < sqlc.narg('page_token')::uuid)
ORDER BY g.id DESC
LIMIT sqlc.arg('row_limit');

-- name: ListGradesForSectionByTeacherPaged :many
-- Keyset-paginated list of grades for a section, scoped to a teacher.
-- Returns empty if the teacher is not in section_teachers for the section.
-- Ordered by g.id DESC. page_token is the exclusive upper bound on g.id (NULL = start).
SELECT g.*
FROM grades g
JOIN section_enrollments se ON se.id = g.section_enrollment_id
WHERE se.section_id = sqlc.arg('section_id')
  AND g.deleted_at IS NULL
  AND se.deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR g.id < sqlc.narg('page_token')::uuid)
  AND EXISTS (
    SELECT 1 FROM section_teachers st
    WHERE st.section_id = sqlc.arg('section_id') AND st.teacher_id = sqlc.arg('teacher_id')
  )
ORDER BY g.id DESC
LIMIT sqlc.arg('row_limit');

-- name: ListOwnGradesPaged :many
-- Keyset-paginated list of grades for a student with enriched label columns.
-- Joins evaluations, sections, courses, academic_periods for display labels.
-- program_id and program_name are sourced from the student's enrollment program
-- (enrollments.program_id → programs), not from the catalog M:N program_courses table.
-- Optional academic_period_id and program_id predicates narrow results without fan-out.
-- Ordered by g.id DESC. page_token is the exclusive upper bound on g.id (NULL = start).
SELECT
    g.id,
    g.evaluation_id,
    g.section_enrollment_id,
    g.graded_by,
    g.value,
    g.evaluated_at,
    g.version,
    g.created_at,
    g.updated_at,
    g.created_by,
    g.updated_by,
    g.deleted_at,
    ev.position                                   AS evaluation_position,
    ev.weight                                     AS evaluation_weight,
    c.code                                        AS course_code,
    c.name                                        AS course_name,
    se.final_grade                                AS section_final_grade,
    se.status                                     AS section_status,
    ap.year                                       AS academic_period_year,
    ap.term                                       AS academic_period_term,
    en.program_id                                 AS program_id,
    p.name                                        AS program_name
FROM grades g
JOIN section_enrollments se ON se.id = g.section_enrollment_id
JOIN enrollments en ON en.id = se.enrollment_id
JOIN programs p ON p.id = en.program_id
JOIN evaluations ev ON ev.id = g.evaluation_id
JOIN sections s ON s.id = se.section_id
JOIN courses c ON c.id = s.course_id
JOIN academic_periods ap ON ap.id = s.academic_period_id
WHERE en.student_id = sqlc.arg('student_id')::uuid
  AND g.deleted_at IS NULL
  AND se.deleted_at IS NULL
  AND en.deleted_at IS NULL
  AND (sqlc.narg('page_token')::uuid IS NULL OR g.id < sqlc.narg('page_token')::uuid)
  AND (sqlc.narg('academic_period_id')::uuid IS NULL OR s.academic_period_id = sqlc.narg('academic_period_id')::uuid)
  AND (sqlc.narg('program_id')::uuid IS NULL OR en.program_id = sqlc.narg('program_id')::uuid)
ORDER BY g.id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: GetGradeByIDForTeacher :one
-- Fetches a grade by primary key only if the caller is in section_teachers for the grade's section.
-- Returns no rows if the grade does not exist or the caller is not in that section.
SELECT g.*
FROM grades g
JOIN section_enrollments se ON se.id = g.section_enrollment_id
WHERE g.id = $1
  AND g.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM section_teachers st
    WHERE st.section_id = se.section_id AND st.teacher_id = $2
  );

-- name: ListOwnGrades :many
-- Lists all grades for a student by joining through enrollments.
SELECT g.*
FROM grades g
JOIN section_enrollments se ON se.id = g.section_enrollment_id
JOIN enrollments e ON e.id = se.enrollment_id
WHERE e.student_id = $1
  AND g.deleted_at IS NULL
  AND se.deleted_at IS NULL
  AND e.deleted_at IS NULL;

-- name: ListOwnGradePeriods :many
-- Returns the distinct academic periods in which the student has grades.
-- Ordered by year DESC, term DESC for most-recent-first display.
SELECT DISTINCT
    ap.id   AS academic_period_id,
    ap.year AS year,
    ap.term AS term
FROM grades g
JOIN section_enrollments se ON se.id = g.section_enrollment_id
JOIN enrollments e ON e.id = se.enrollment_id
JOIN sections s ON s.id = se.section_id
JOIN academic_periods ap ON ap.id = s.academic_period_id
WHERE e.student_id = sqlc.arg('student_id')::uuid
  AND g.deleted_at IS NULL
  AND se.deleted_at IS NULL
  AND e.deleted_at IS NULL
ORDER BY ap.year DESC, ap.term DESC;

-- name: IsTeacherForSection :one
-- Checks whether a user is in section_teachers for the given section.
SELECT EXISTS(
    SELECT 1 FROM section_teachers
    WHERE section_id = $1 AND teacher_id = $2
) AS exists;

-- name: GetEvaluationByID :one
-- Fetches an evaluation by primary key (any state, for existence checks).
SELECT * FROM evaluations WHERE id = $1;

-- name: InsertAuditLog :exec
-- Records an audit event for a grade value change (old value → new value).
-- actor_id may be NULL for system-initiated changes.
INSERT INTO audit_logs (actor_id, action, entity, entity_id, detail)
VALUES ($1, $2, $3, $4, $5);

