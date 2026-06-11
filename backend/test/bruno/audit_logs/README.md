# audit_logs/

Bruno smoke tests for the `AuditLogsService` Connect-RPC slice.

## Flow overview

The folder is self-contained and intentionally minimal: it only requires admin login
and a student login — no catalog/section/enrollment setup chain.

### Setup chain (01, 05)

| # | Step | Captures |
|---|------|----------|
| 01 | Admin login | `admin_sid`, `audit_entity_id` |
| 05 | Student A login | `student_a_sid` |

### Happy path (02–03)

| # | Step | Key assertions |
|---|------|----------------|
| 02 | ListAuditLogs first page (no seeded rows) | HTTP 200, empty list |
| 03 | ListAuditLogs second page (token from 02) | HTTP 200, `nextPageToken` absent/empty |

### Negative cases (04, 06–08)

| # | Scenario | Expected code |
|---|----------|---------------|
| 04 | No session cookie | `unauthenticated` |
| 06 | Student calls ListAuditLogs (lacks `audit.read`) | `permission_denied` |
| 07 | entity field absent | `invalid_argument` |
| 08 | entity_id is not a UUID | `invalid_argument` |

## Assertion gotchas

- protojson omits empty scalar fields: `nextPageToken` is **absent** from the response when
  empty, not `""`. The check in step 03 uses `(res.body.nextPageToken ?? '') === ''`.

## Prerequisites

- Stack running: `docker compose up -d` from the repo root.
- `seed.sh` executed and its `export` statements sourced (provides `student_a_email`).

## Run

```bash
eval "$(bash seed.sh | grep '^export')"
npx --yes @usebruno/cli@3.4.2 run audit_logs/ --env local \
  --env-var student_a_email="$BRUNO_STUDENT_A_EMAIL"
```

A healthy run reports all 7 requests passed (steps 01–08 minus step 05 which sets up
student_a — 8 files, the student login step 05 counts as 1 request).

## Permission model

| Permission | Role | RPC |
|---|---|---|
| `audit.read` | admin | ListAuditLogs |
