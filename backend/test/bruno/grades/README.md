# grades/

Bruno smoke collection for `grades.v1.GradesService`.

## Prerequisites

Run `seed.sh` before executing this collection. The script seeds:
- Students A, B, C with the `student` role
- A teacher user (`teacher.${SFX}@gr.smoke`) with the `teacher` role
- Academic periods with enrollment windows

Export the printed variables and pass them to `bru run` via `--env-var` flags or by
sourcing the output. Required vars: `BRUNO_STUDENT_A_ID`, `BRUNO_STUDENT_B_ID`,
`BRUNO_TEACHER_ID`, `BRUNO_TEACHER_EMAIL`, `BRUNO_STUDENT_A_EMAIL`, `BRUNO_STUDENT_B_EMAIL`.

## Flow

| Range | Block |
|-------|-------|
| 01–28 | Setup (admin login, catalog provisioning, section enrollments, teacher/student login) |
| 29–33 | Scheme (CreateEvaluationScheme happy path, duplicate, permission denied, bad weights, single-eval) |
| 34–46 | Recording (insert, correction, stale version, out-of-range, partial grading, outcome checks) |
| 47–52 | Denials (out-of-scope teacher, student cannot record, empty scoped list, admin override, student own list, unauthenticated) |
| 53–54 | Recreate (blocked by grades, succeeds on course with no grades) |

## Grade computations embedded in the flow

| Step | State | Formula | Result | Status |
|------|-------|---------|--------|--------|
| After 39 | eval1=5.5, eval2=4.5, eval3=5.0 | 5.5×0.4+4.5×0.3+5.0×0.3 = 5.05 | 5.1 (round half-up) | passed |
| After 43 | eval1=5.5, eval2=1.0, eval3=5.0 | 5.5×0.4+1.0×0.3+5.0×0.3 = 4.0  | 4.0 (exact boundary) | passed |
| After 45 | eval1=5.5, eval2=1.0, eval3=1.0 | 5.5×0.4+1.0×0.3+1.0×0.3 = 2.8  | 2.8 | failed |
