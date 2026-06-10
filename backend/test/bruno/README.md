# backend-smoke Bruno Collection

HTTP smoke test collection for the backend using the Connect protocol over plain HTTP.
Replaces the bash gRPC harness in `test/grpc/`.

## Why Connect-over-HTTP instead of native gRPC

`bru run` (CLI) skips native gRPC requests (usebruno/bruno issues #5928, #6067, #6068).
The connect-go server speaks the Connect protocol on the same handlers:
each RPC is `POST http://<host>/<package>.v1.<Service>/<Method>` with
`Content-Type: application/json` and a JSON body using proto field names in
lowerCamelCase (connect-go uses protojson defaults; snake_case is also accepted).

Error responses are non-2xx JSON: `{"code":"<connect_code>","message":"..."}`.

## Prerequisites

- Docker Compose stack up: `docker compose up -d`
- Server running with these env vars:

```
DATABASE_URL=postgres://app:app@localhost:5432/academico?sslmode=disable
REDIS_URL=redis://localhost:6379
SESSION_TTL=30m
RESET_TOKEN_TTL=15m
APP_ENV=dev
COOKIE_SECURE=false
BCRYPT_COST=10
HTTP_ADDR=:8080
```

## One-time seed

Run `seed.sh` once per fresh database to insert:
- Student users A, B, C (with the `student` role) and a role-less noadmin user
- Academic periods with the correct enrollment windows (open and closed)
  — the API's `CreateAcademicPeriod` does not accept `enrollment_starts_at` /
  `enrollment_ends_at`, so these must be inserted directly via psql.

```bash
chmod +x seed.sh
eval "$(./seed.sh | grep '^export')"
```

The script prints export statements. Source them so the variables are available
to `bru run` as `--env-var` flags (see below).

## Install npm dependencies

```bash
npm install
```

## Run the collection

Two CLI flags are mandatory:

- `--sandbox developer` — the default `safe` sandbox blocks `require("@faker-js/faker")` in pre-request scripts.
- `--disable-cookies` — the collection manages session cookies explicitly via captured variables; the automatic cookie jar would leak the admin session into the unauthenticated-denial requests.

```bash
eval "$(bash seed.sh | grep '^export')"
bru run --env local --sandbox developer --disable-cookies \
  --env-var student_a_id="$BRUNO_STUDENT_A_ID" \
  --env-var student_b_id="$BRUNO_STUDENT_B_ID" \
  --env-var student_c_id="$BRUNO_STUDENT_C_ID" \
  --env-var student_a_email="$BRUNO_STUDENT_A_EMAIL" \
  --env-var student_b_email="$BRUNO_STUDENT_B_EMAIL" \
  --env-var student_c_email="$BRUNO_STUDENT_C_EMAIL" \
  --env-var open_period_id="$BRUNO_OPEN_PERIOD_ID" \
  --env-var closed_period_id="$BRUNO_CLOSED_PERIOD_ID" \
  --env-var open_term="$BRUNO_OPEN_TERM" \
  --env-var closed_term="$BRUNO_CLOSED_TERM" \
  --output results.json
```

A healthy run reports all requests passed AND a non-zero assertions count
(e.g. `Assertions 77/77`). A run with `Assertions 0/0` means the YAML schema
is not being parsed — never trust the Status line alone.

To run a single folder (health needs no seed variables):

```bash
bru run health/ --env local
```

## Collection structure

```
.
├── opencollection.yml          collection manifest (HTTP mode)
├── environments/
│   └── local.yml               host, seeded admin credentials
├── package.json                @faker-js/faker dependency
├── seed.sh                     SQL seeding (run once before bru run)
├── health/
│   └── 01-ping.yml
├── auth/
│   ├── 01-admin-request-reset.yml
│   ├── 02-admin-confirm-reset.yml
│   ├── 03-admin-login.yml
│   └── 04..11-login-table-cases-and-session-lifecycle.yml
├── section_enrollment/
│   ├── 01..27-setup-chain.yml  (admin login, catalog, enrollment, student logins)
│   ├── 28-happy-path-*.yml
│   ├── 29-30-idempotency-*.yml
│   ├── 31-admin-enroll-*.yml
│   ├── 32-window-closed-*.yml
│   ├── 33-not-paid-*.yml
│   ├── 34-year-mismatch-*.yml
│   ├── 35-course-not-in-program-*.yml
│   ├── 36-37-oversell-*.yml
│   ├── 38-withdraw-*.yml
│   ├── 39-revival-*.yml
│   ├── 40-41-self-revive-*.yml
│   ├── 42-restore-*.yml
│   ├── 43-45-self-scope-*.yml
│   └── 46-57-denials-*.yml
└── grades/
    ├── 01..18-setup-*.yml      admin auth chain, catalog entities, teacher + student
    │                           profiles (FK prerequisites), enrollment, student A login
    ├── 19-happy-path-list-evaluations.yml
    ├── 20..22-happy-path-override-grade-*.yml   3 OverrideGrade calls (eval 1/2/3)
    ├── 23..25-happy-path-get-list-*.yml         GetGrade, ListGradesForSection, ListOwnGrades
    ├── 26-happy-path-get-section-enrollment-passed.yml  final_grade=4.3 assertion
    └── 27..32-denial-*.yml     6 negative cases (permission, validation, conflict, etc.)
```

## Notes on seeded variables

`student_a_id`, `student_b_id`, `student_c_id`, `open_period_id`, and
`closed_period_id` are produced by `seed.sh` and must be injected into the
Bruno environment before `bru run`. The recommended approach in CI:

```yaml
- name: Seed DB
  run: eval "$(./test/bruno/seed.sh | grep '^export')"
- name: Patch Bruno env
  run: |
    # Inject seeded IDs into environments/local.yml using yq or sd
    sd 'student_a_id: ""' "student_a_id: $BRUNO_STUDENT_A_ID" test/bruno/environments/local.yml
    # ... repeat for other IDs
- name: Run Bruno
  run: bru run --env local test/bruno/section_enrollment/
```

## Parse validation

Run a parse-level check (no live server needed — expect connection errors, not parse errors):

```bash
bru run health/ --env local 2>&1 | grep -v "Error: "
```

If Bruno exits with a parse error the collection YAML is malformed.
A connection-refused exit is expected and correct.
