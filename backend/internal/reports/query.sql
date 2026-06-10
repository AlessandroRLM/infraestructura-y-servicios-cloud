-- Cross-domain read queries — coupled to the shared migration schema, not to any Go domain package.
-- All queries filter deleted_at IS NULL on every soft-deletable table.
-- LIMIT cap+1 pattern: service detects len(rows) > cap and sets truncated=true.

-- name: ActaSectionExists :one
SELECT EXISTS(
    SELECT 1 FROM sections WHERE id = $1 AND deleted_at IS NULL
) AS exists;

-- name: ActaForSectionAdmin :many
SELECT
    se.id                       AS se_id,
    e.student_id                AS student_id,
    up.given_names              AS given_names,
    up.last_name_paternal       AS last_name_paternal,
    up.last_name_maternal       AS last_name_maternal,
    ev.id                       AS evaluation_id,
    ev.position                 AS position,
    g.value                     AS grade_value,
    se.final_grade              AS final_grade,
    se.status                   AS enrollment_status
FROM sections s
JOIN section_enrollments se
    ON se.section_id = s.id
    AND se.status != 'withdrawn'
    AND se.deleted_at IS NULL
JOIN enrollments e
    ON e.id = se.enrollment_id
    AND e.deleted_at IS NULL
JOIN student_profiles sp
    ON sp.user_id = e.student_id
    AND sp.deleted_at IS NULL
JOIN user_profiles up
    ON up.user_id = e.student_id
    AND up.deleted_at IS NULL
LEFT JOIN grades g
    ON g.section_enrollment_id = se.id
    AND g.deleted_at IS NULL
LEFT JOIN evaluations ev
    ON ev.id = g.evaluation_id
    AND ev.deleted_at IS NULL
WHERE s.id = $1
  AND s.deleted_at IS NULL
ORDER BY up.last_name_paternal, up.given_names, ev.position
LIMIT 501;

-- name: ActaForSectionByTeacher :many
SELECT
    se.id                       AS se_id,
    e.student_id                AS student_id,
    up.given_names              AS given_names,
    up.last_name_paternal       AS last_name_paternal,
    up.last_name_maternal       AS last_name_maternal,
    ev.id                       AS evaluation_id,
    ev.position                 AS position,
    g.value                     AS grade_value,
    se.final_grade              AS final_grade,
    se.status                   AS enrollment_status
FROM sections s
JOIN section_enrollments se
    ON se.section_id = s.id
    AND se.status != 'withdrawn'
    AND se.deleted_at IS NULL
JOIN enrollments e
    ON e.id = se.enrollment_id
    AND e.deleted_at IS NULL
JOIN student_profiles sp
    ON sp.user_id = e.student_id
    AND sp.deleted_at IS NULL
JOIN user_profiles up
    ON up.user_id = e.student_id
    AND up.deleted_at IS NULL
LEFT JOIN grades g
    ON g.section_enrollment_id = se.id
    AND g.deleted_at IS NULL
LEFT JOIN evaluations ev
    ON ev.id = g.evaluation_id
    AND ev.deleted_at IS NULL
WHERE s.id = $1
  AND s.deleted_at IS NULL
  AND EXISTS (
      SELECT 1 FROM section_teachers st
      WHERE st.section_id = s.id AND st.teacher_id = $2
  )
ORDER BY up.last_name_paternal, up.given_names, ev.position
LIMIT 501;

-- name: OccupancyPeriodExists :one
SELECT EXISTS(
    SELECT 1 FROM academic_periods WHERE id = $1 AND deleted_at IS NULL
) AS exists;

-- name: OccupancyForPeriod :many
SELECT
    s.id                                    AS section_id,
    s.capacity                              AS capacity,
    c.name                                  AS course_name,
    COUNT(se.id)                            AS active_seat_count
FROM sections s
LEFT JOIN section_enrollments se
    ON se.section_id = s.id
    AND se.status != 'withdrawn'
    AND se.deleted_at IS NULL
LEFT JOIN courses c
    ON c.id = s.course_id
WHERE s.academic_period_id = $1
  AND s.deleted_at IS NULL
GROUP BY s.id, s.capacity, c.name
ORDER BY s.id
LIMIT 1001;

-- name: ProgramExists :one
SELECT EXISTS(
    SELECT 1 FROM programs WHERE id = $1 AND deleted_at IS NULL
) AS exists;

-- name: ProgramSummary :many
SELECT
    pq.id                                                               AS quota_id,
    pq.capacity                                                         AS quota_capacity,
    (SELECT COUNT(*)
     FROM enrollments e
     WHERE e.program_id = $1
       AND e.year = $2
       AND e.status != 'cancelled'
       AND e.deleted_at IS NULL)::int4                                  AS enrolled_count
FROM program_quotas pq
WHERE pq.program_id = $1
  AND pq.year = $2
  AND pq.deleted_at IS NULL
LIMIT 201;

-- name: StudentExists :one
SELECT EXISTS(
    SELECT 1 FROM student_profiles WHERE user_id = $1 AND deleted_at IS NULL
) AS exists;

-- name: FichaForStudent :many
SELECT
    ap.id                                   AS academic_period_id,
    ap.year::text || '-' || ap.term::text   AS academic_period_name,
    s.id                                    AS section_id,
    c.name                                  AS course_name,
    se.status                               AS enrollment_status,
    se.final_grade                          AS final_grade,
    ev.id                                   AS evaluation_id,
    ev.position                             AS position,
    g.value                                 AS grade_value
FROM enrollments e
JOIN section_enrollments se
    ON se.enrollment_id = e.id
    AND se.deleted_at IS NULL
JOIN sections s
    ON s.id = se.section_id
    AND s.deleted_at IS NULL
JOIN courses c
    ON c.id = s.course_id
JOIN academic_periods ap
    ON ap.id = s.academic_period_id
LEFT JOIN grades g
    ON g.section_enrollment_id = se.id
    AND g.deleted_at IS NULL
LEFT JOIN evaluations ev
    ON ev.id = g.evaluation_id
    AND ev.deleted_at IS NULL
WHERE e.student_id = $1
  AND e.deleted_at IS NULL
ORDER BY ap.year, ap.term, c.code, ev.position
LIMIT 1001;
