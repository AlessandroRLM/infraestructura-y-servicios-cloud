# grades/

Bruno smoke tests for the `GradesService` Connect-RPC slice.

## Flow overview

The folder is self-contained: it sets up its own program, course, section, enrollment,
and evaluation scheme without relying on the `section_enrollment/` folder having run first.

### Setup chain (01–18)

| # | Step | Captures |
|---|------|----------|
| 01–03 | Admin password reset + login | `admin_sid` |
| 04 | CreateProgram | `grades_prog_id` |
| 05 | CreateCourse | `grades_course_id` |
| 06 | AddCourseToProgram | — |
| 07 | CreateProgramQuota (year=2026) | — |
| 08 | CreateSection (open window period) | `grades_section_id` |
| 09 | UpsertTeacherProfile (bootstrap admin) — FK prerequisite for 10 | — |
| 10 | AssignTeacherToSection (bootstrap admin) | — |
| 11 | **CreateEvaluationScheme** (3 evals: 0.4/0.3/0.3) | `eval_1_id`, `eval_2_id`, `eval_3_id` |
| 12 | UpsertStudentProfile A — FK prerequisite for 13 | — |
| 13–14 | CreateEnrollment + MarkEnrollmentPaid (student A) | `grades_enroll_a_id` |
| 15–17 | Student A password reset + login | `grades_student_a_sid` |
| 18 | **EnrollSection** (admin enrolls student A) | `grades_se_a_id`; asserts `final_grade` absent |

FK prerequisites: `enrollments.student_id` references `student_profiles(user_id)` and
`sections.teacher_id` references `teacher_profiles(user_id)`, so both profile upserts
must run before their dependent steps on a fresh database.

### Happy path (19–26)

| # | Step | Key assertions |
|---|------|----------------|
| 19 | ListEvaluations | 3 evals, positions 1–3 |
| 20 | OverrideGrade eval_1 → 5.0 | value=5.0, version=1 |
| 21 | OverrideGrade eval_2 → 4.5 | value=4.5, version=1 |
| 22 | OverrideGrade eval_3 → 3.0 (completes scheme) | value=3.0, version=1 |
| 23 | GetGrade (grade_1) | id, value=5.0, gradedBy present |
| 24 | ListGradesForSection | ≥1 grade, sectionEnrollmentId matches |
| 25 | ListOwnGrades (student A) | value non-empty, no gradedBy field |
| 26 | **GetSectionEnrollment** | status=passed, **final_grade=4.3** |

Weighted final: `5.0×0.4 + 4.5×0.3 + 3.0×0.3 = 4.25` → rounds HALF-UP to `4.3` → `passed`.

### Negative cases (27–32)

| # | Scenario | Expected code |
|---|----------|---------------|
| 27 | Student calls RecordGrade (missing grades.write) | `permission_denied` |
| 28 | Admin overrides with value=8.0 (outside [1.0, 7.0]) | `invalid_argument` |
| 29 | Admin grades against nil evaluation UUID | `not_found` |
| 30 | Admin corrects grade_1 with stale expectedVersion=0 | `aborted` |
| 31 | Admin creates a second scheme for the same course | `already_exists` |
| 32 | OverrideGrade without a session cookie | `unauthenticated` |

### Assertion gotchas

- The Bruno CLI coerces numeric-looking expected values (`"0.400"`, `"5.0"`) to numbers,
  while the server serializes decimals as strings — decimal assertions therefore use a
  JS strict-equality expression (`res.body.grade.value === '5.0'`) asserted against `"true"`.
- protojson omits empty scalar fields: before completion `finalGrade` is **absent** from
  the response, not `""`. The check in 18 uses `(res.body.finalGrade ?? '') === ''`.

## Prerequisites

- Stack running: `docker compose up -d` from the repo root.
- `seed.sh` executed and its `export` statements sourced (provides `student_a_id`,
  `student_a_email`, `open_period_id`, and related vars).

## Run

```bash
eval "$(bash seed.sh | grep '^export')"
bru run grades/ --env local --sandbox developer --disable-cookies \
  --env-var student_a_id="$BRUNO_STUDENT_A_ID" \
  --env-var student_a_email="$BRUNO_STUDENT_A_EMAIL" \
  --env-var open_period_id="$BRUNO_OPEN_PERIOD_ID"
```

A healthy run reports all 32 requests passed with a non-zero assertion count
(66 assertions as of this writing).

## Permission model

| Permission | Role | RPCs |
|---|---|---|
| `grades.override` | admin | CreateEvaluationScheme, RecreateEvaluationScheme, OverrideGrade |
| `grades.read` | admin, teacher | ListEvaluations, GetGrade, ListGradesForSection |
| `grades.write` | teacher | RecordGrade (+ must be in section_teachers) |
| `grades.view_own` | student | ListOwnGrades |

The `OverrideGrade` path (admin) bypasses the `section_teachers` check entirely.
`RecordGrade` (teacher) requires both `grades.write` and `section_teachers` membership
for the section that owns the target section enrollment.
