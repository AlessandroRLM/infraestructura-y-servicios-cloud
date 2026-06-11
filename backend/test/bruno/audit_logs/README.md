# audit_logs/

Bruno smoke tests for the `AuditLogsService` Connect-RPC slice.

## Flow overview

The folder is self-contained. It builds the full program/course/section/enrollment/grade
chain from scratch, performs a grade correction (the only write path that produces an
`audit_logs` row), and then verifies the audit log response shape.

### Setup chain (01–20)

| # | Step | Captures |
|---|------|----------|
| 01–02 | Admin password reset | `audit_admin_reset_token` |
| 03 | Admin login | `audit_admin_sid` |
| 04 | CreateProgram | `audit_prog_id` |
| 05 | CreateCourse | `audit_course_id` |
| 06 | AddCourseToProgram | — |
| 07 | CreateProgramQuota (year=2026) | — |
| 08 | CreateSection (open window period) | `audit_section_id` |
| 09 | UpsertTeacherProfile (bootstrap admin) | — |
| 10 | AssignTeacherToSection (bootstrap admin) | — |
| 11 | CreateEvaluationScheme (3 evals: 0.4/0.3/0.3) | `audit_eval_1_id` |
| 12 | UpsertStudentProfile A | — |
| 13–14 | CreateEnrollment + MarkEnrollmentPaid | `audit_enroll_a_id` |
| 15–17 | Student A password reset + login | `audit_student_a_sid` |
| 18 | EnrollSection (admin enrolls student A) | `audit_se_a_id` |
| 19 | OverrideGrade eval_1 → 5.0 (initial insert) | `audit_grade_id`, `audit_grade_version` |
| 20 | OverrideGrade eval_1 → 6.5 (correction, expectedVersion=1) | — |

Step 20 is the only write path that triggers `RecordGradeTx` update logic. When
`isUpdate=true` and `oldValue != ""`, it inserts an `audit_logs` row:

```
action   = "grade.update"
entity   = "grades"
entity_id = audit_grade_id
actor_id = a0000000-0000-0000-0000-000000000001  (bootstrap admin)
detail   = {"old_value":"5.0","new_value":"6.5"}
```

### Happy path (21–22)

| # | Step | Key assertions |
|---|------|----------------|
| 21 | ListAuditLogs (entity=grades, entity_id=audit_grade_id) | HTTP 200; logs ≥ 1; row shape (id, action, entity, entityId, createdAt present; actorId = admin UUID); detail contains old_value/new_value/5.0/6.5 |
| 22 | ListAuditLogs pagination contract (same entity, page_size=20) | HTTP 200; `(nextPageToken ?? '') === ''` (only page) |

### Negative cases (23–26)

| # | Scenario | Expected code |
|---|----------|---------------|
| 23 | No session cookie | `unauthenticated` |
| 24 | Student session (lacks `audit.read`) | `permission_denied` |
| 25 | entity field absent | `invalid_argument` |
| 26 | entity_id not a UUID | `invalid_argument` |

## Assertion gotchas

- `detail` is a raw JSON string, not a parsed object. Assertions use
  `res.body.logs[0].detail.includes('"old_value"')` rather than object property access.
- protojson omits empty scalar fields: `nextPageToken` is **absent** from the response
  when empty. The check in step 22 uses `(res.body.nextPageToken ?? '') === ''`.
- All captured vars use the `audit_` prefix to avoid collision with the `grades/` folder
  vars when running the full collection.

## Prerequisites

- Stack running: `docker compose up -d` from the repo root.
- `seed.sh` executed and its `export` statements sourced (provides `student_a_id`,
  `student_a_email`, and `open_period_id`).

## Run

```bash
eval "$(bash seed.sh | grep '^export')"
npx --yes @usebruno/cli@3.4.2 run audit_logs/ --env local --sandbox developer \
  --disable-cookies \
  --env-var student_a_id="$BRUNO_STUDENT_A_ID" \
  --env-var student_a_email="$BRUNO_STUDENT_A_EMAIL" \
  --env-var open_period_id="$BRUNO_OPEN_PERIOD_ID"
```

A healthy run reports all 26 requests passed with a non-zero assertion count.

## Variable prefix convention

All runtime vars captured by this folder use the `audit_` prefix
(e.g. `audit_admin_sid`, `audit_grade_id`, `audit_student_a_sid`).
This prevents collision when the full collection runs all folders in sequence.

## Permission model

| Permission | Role | RPC |
|---|---|---|
| `audit.read` | admin | ListAuditLogs |
| `grades.override` | admin | OverrideGrade (produces audit rows) |
