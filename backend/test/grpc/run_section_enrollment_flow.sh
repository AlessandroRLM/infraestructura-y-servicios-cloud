#!/usr/bin/env bash
#
# Data-driven gRPC smoke test for the section_enrollment slice.
#
# Drives SectionEnrollmentService over real gRPC (h2c) with grpcurl against a running server
# and exercises the full section inscription lifecycle plus authorization, capacity, window,
# and state-machine rules:
#   - admin (enrollment.manage + catalog.manage + users.manage) sets up a program + course +
#     open-window academic period + section (capacity=2); creates paid enrollments for two
#     students; also sets up a closed-window period + section for the window-rejection test;
#   - student self-enroll happy path: EnrollOwnSection with open window + paid matrícula -> OK;
#   - admin enroll: EnrollSection (admin, enrollment_id) -> OK;
#   - window closed: EnrollOwnSection into a closed-window section -> FailedPrecondition;
#   - not paid: student with a pending matrícula -> FailedPrecondition;
#   - year mismatch: admin enrolls enrollment whose year != section's period year -> FailedPrecondition;
#   - course not in program: program_id that does not contain the section's course -> FailedPrecondition;
#   - oversell: fill capacity=2 then attempt a 3rd -> FailedPrecondition (section full);
#   - withdraw (admin-only): WithdrawSection on in_progress -> OK (frees a seat);
#   - revival (admin-only): admin EnrollSection on withdrawn (enrollment, section) -> OK;
#   - student cannot self-revive withdrawn inscription -> FailedPrecondition;
#   - idempotency/duplicate: EnrollOwnSection on already in_progress -> AlreadyExists;
#   - self-scope: GetOwnSectionEnrollment / ListOwnSectionEnrollments returns only caller's own;
#     another student's section enrollment id -> NotFound;
#   - denials: management RPC no session -> Unauthenticated; student or role-less -> PermissionDenied.
#
# Usage:  ./run_section_enrollment_flow.sh
# Env:    GRPCURL (default $HOME/go/bin/grpcurl), ADDR (default 127.0.0.1:8080)

set -uo pipefail

GRPCURL="${GRPCURL:-$HOME/go/bin/grpcurl}"
ADDR="${ADDR:-127.0.0.1:8080}"
SVC_AUTH="auth.v1.AuthService"
SVC_SE="section_enrollment.v1.SectionEnrollmentService"
SVC_ENR="enrollment.v1.EnrollmentService"
SVC_CAT="catalog.v1.CatalogService"
SVC_PROF="profiles.v1.ProfileService"
ADMIN_EMAIL="admin@dev.local"
NOADMIN_EMAIL="noadmin@se.smoke"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)/proto"

for bin in "$GRPCURL" jq docker; do
  command -v "$bin" >/dev/null 2>&1 || { echo "FATAL: '$bin' not found" >&2; exit 2; }
done

# grpc {auth|se|enr|cat|prof} METHOD JSON [cookie-sid]  -> RESP (stdout+stderr), RC
grpc() {
  local which="$1" method="$2" data="$3" sid="${4:-}"
  local svc
  case "$which" in
    auth) svc="$SVC_AUTH" ;;
    se)   svc="$SVC_SE"   ;;
    enr)  svc="$SVC_ENR"  ;;
    cat)  svc="$SVC_CAT"  ;;
    prof) svc="$SVC_PROF" ;;
    *)    svc="$which"    ;;
  esac
  local args=(-plaintext -import-path "$PROTO_ROOT"
    -proto auth/v1/auth.proto
    -proto enrollment/v1/enrollment.proto
    -proto catalog/v1/catalog.proto
    -proto profiles/v1/profiles.proto
    -proto section_enrollment/v1/section_enrollment.proto
    -d "$data")
  [ -n "$sid" ] && args+=(-H "cookie: sid=$sid")
  RESP="$("$GRPCURL" "${args[@]}" "$ADDR" "$svc/$method" 2>&1)"; RC=$?
}

login_with_reset() {
  local email="$1" pw="$2"
  grpc auth RequestPasswordReset "$(jq -nc --arg e "$email" '{email:$e}')"
  local token; token="$(echo "$RESP" | grep -oiP '"dev_?token"\s*:\s*"\K[^"]+' || true)"
  [ -n "$token" ] || { echo "FATAL: no dev_token for $email" >&2; exit 2; }
  grpc auth ConfirmPasswordReset "$(jq -nc --arg t "$token" --arg p "$pw" '{token:$t,new_password:$p}')"
  RESP="$("$GRPCURL" -plaintext -v -import-path "$PROTO_ROOT" -proto auth/v1/auth.proto \
    -d "$(jq -nc --arg e "$email" --arg p "$pw" '{email:$e,password:$p}')" "$ADDR" "$SVC_AUTH/Login" 2>&1)"
  echo "$RESP" | grep -oiP 'sid=\K[^;]+' | head -1
}
respid() { echo "$RESP" | jq -r '.id' 2>/dev/null; }

pass=0; fail=0
ok()          { printf '  \033[32m✓\033[0m %s\n' "$1"; pass=$((pass+1)); }
bad()         { printf '  \033[31m✗\033[0m %s\n     %s\n' "$1" "$2"; fail=$((fail+1)); }
expect_ok()   { [ "$RC" -eq 0 ] && ok "$1" || bad "$1" "want OK, got: $(echo "$RESP" | tr '\n' ' ')"; }
expect_code() { echo "$RESP" | grep -q "Code: $1" && ok "$2 (-> $1)" || bad "$2" "want $1, got: $(echo "$RESP" | tr '\n' ' ')"; }

echo "▶ seeding role-less user"
docker compose exec -T postgres psql -U app -d academico -q -c \
  "INSERT INTO users (id, email, password_hash) VALUES (uuidv7(), '$NOADMIN_EMAIL', 'placeholder') ON CONFLICT (email) DO NOTHING;" >/dev/null

echo "▶ login admin (enrollment.manage + catalog.manage + users.manage)"
ADMIN_SID="$(login_with_reset "$ADMIN_EMAIL" "SESmoke123!")"
[ -n "$ADMIN_SID" ] || { echo "FATAL: no admin sid" >&2; exit 2; }

# Unique suffix so re-runs against a persistent DB do not collide on unique keys.
SFX="$(date +%s | tail -c 6)"

# Seed student users.
STUDENT_A_EMAIL="student.a.$SFX@se.smoke"
STUDENT_B_EMAIL="student.b.$SFX@se.smoke"
STUDENT_C_EMAIL="student.c.$SFX@se.smoke"  # used for window-closed + pending-matrícula tests

echo "▶ seeding student users in Postgres"
docker compose exec -T postgres psql -U app -d academico -q -c \
  "INSERT INTO users (id, email, password_hash) VALUES
     (uuidv7(), '$STUDENT_A_EMAIL', 'placeholder'),
     (uuidv7(), '$STUDENT_B_EMAIL', 'placeholder'),
     (uuidv7(), '$STUDENT_C_EMAIL', 'placeholder')
   ON CONFLICT (email) DO NOTHING;
   INSERT INTO user_roles (user_id, role_id)
     SELECT u.id, r.id FROM users u, roles r
     WHERE u.email IN ('$STUDENT_A_EMAIL', '$STUDENT_B_EMAIL', '$STUDENT_C_EMAIL')
       AND r.name = 'student'
   ON CONFLICT DO NOTHING;" >/dev/null

STUDENT_A_ID="$(docker compose exec -T postgres psql -U app -d academico -tAq \
  -c "SELECT id FROM users WHERE email='$STUDENT_A_EMAIL';")"
STUDENT_B_ID="$(docker compose exec -T postgres psql -U app -d academico -tAq \
  -c "SELECT id FROM users WHERE email='$STUDENT_B_EMAIL';")"
STUDENT_C_ID="$(docker compose exec -T postgres psql -U app -d academico -tAq \
  -c "SELECT id FROM users WHERE email='$STUDENT_C_EMAIL';")"

[ -n "$STUDENT_A_ID" ] && [ -n "$STUDENT_B_ID" ] && [ -n "$STUDENT_C_ID" ] || \
  { echo "FATAL: could not fetch student IDs" >&2; exit 2; }

echo "▶ upsert student profiles via ProfileService (admin, users.manage)"
grpc prof UpsertStudentProfile \
  "$(jq -nc --arg u "$STUDENT_A_ID" '{user_id:$u, admission_year:2026}')" "$ADMIN_SID"
expect_ok "UpsertStudentProfile A"

grpc prof UpsertStudentProfile \
  "$(jq -nc --arg u "$STUDENT_B_ID" '{user_id:$u, admission_year:2026}')" "$ADMIN_SID"
expect_ok "UpsertStudentProfile B"

grpc prof UpsertStudentProfile \
  "$(jq -nc --arg u "$STUDENT_C_ID" '{user_id:$u, admission_year:2026}')" "$ADMIN_SID"
expect_ok "UpsertStudentProfile C"

# ---------------------------------------------------------------------------
# Catalog setup: program, course, open-window period, section (capacity=2)
# ---------------------------------------------------------------------------
echo "▶ catalog setup: program + course + open-window period + section (capacity=2)"

ENROLL_YEAR=2026
MISMATCH_YEAR=2025

grpc cat CreateProgram \
  "$(jq -nc --arg c "SEPROG-$SFX" '{code:$c, name:"SE Test Program"}')" "$ADMIN_SID"
expect_ok "CreateProgram"; PROG_ID="$(respid)"

grpc cat CreateCourse \
  "$(jq -nc --arg c "SECRS-$SFX" '{code:$c, name:"SE Test Course", credits:4}')" "$ADMIN_SID"
expect_ok "CreateCourse"; COURSE_ID="$(respid)"

grpc cat AddCourseToProgram \
  "$(jq -nc --arg p "$PROG_ID" --arg c "$COURSE_ID" '{program_id:$p, course_id:$c}')" "$ADMIN_SID"
expect_ok "AddCourseToProgram"

# Open window: starts 1 hour ago, ends 1 hour from now.
# Use psql to insert directly because CreateAcademicPeriod does not accept window columns.
# Use a high term number derived from SFX to avoid collisions on (year,term) unique key.
OPEN_TERM=$((100 + (${SFX} % 800)))
CLOSED_TERM=$((900 + (${SFX} % 99)))

OPEN_PERIOD_ID="$(docker compose exec -T postgres psql -U app -d academico -tAq -c \
  "INSERT INTO academic_periods (id, year, term, start_date, end_date, enrollment_starts_at, enrollment_ends_at)
   VALUES (uuidv7(), $ENROLL_YEAR, $OPEN_TERM, '${ENROLL_YEAR}-03-01', '${ENROLL_YEAR}-07-15',
           now() - interval '1 hour', now() + interval '1 hour')
   RETURNING id;")"
[ -n "$OPEN_PERIOD_ID" ] || { echo "FATAL: could not create open-window period" >&2; exit 2; }
echo "  open_period_id=$OPEN_PERIOD_ID"

grpc cat CreateProgramQuota \
  "$(jq -nc --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" '{program_id:$p, year:$y, admission_quota:50}')" "$ADMIN_SID"
expect_ok "CreateProgramQuota (year=$ENROLL_YEAR)"

grpc cat CreateProgramQuota \
  "$(jq -nc --arg p "$PROG_ID" --argjson y "$MISMATCH_YEAR" '{program_id:$p, year:$y, admission_quota:50}')" "$ADMIN_SID"
expect_ok "CreateProgramQuota (year=$MISMATCH_YEAR)"

grpc cat CreateSection \
  "$(jq -nc --arg c "$COURSE_ID" --arg a "$OPEN_PERIOD_ID" '{course_id:$c, academic_period_id:$a, seat_capacity:2}')" "$ADMIN_SID"
expect_ok "CreateSection (capacity=2, open window)"; SECTION_ID="$(respid)"

# Closed-window period (window starts 24 h in the future → never open right now).
CLOSED_PERIOD_ID="$(docker compose exec -T postgres psql -U app -d academico -tAq -c \
  "INSERT INTO academic_periods (id, year, term, start_date, end_date, enrollment_starts_at, enrollment_ends_at)
   VALUES (uuidv7(), $ENROLL_YEAR, $CLOSED_TERM, '${ENROLL_YEAR}-08-01', '${ENROLL_YEAR}-12-15',
           now() + interval '24 hours', now() + interval '48 hours')
   RETURNING id;")"
[ -n "$CLOSED_PERIOD_ID" ] || { echo "FATAL: could not create closed-window period" >&2; exit 2; }

grpc cat CreateSection \
  "$(jq -nc --arg c "$COURSE_ID" --arg a "$CLOSED_PERIOD_ID" '{course_id:$c, academic_period_id:$a, seat_capacity:10}')" "$ADMIN_SID"
expect_ok "CreateSection (closed window)"; CLOSED_SECTION_ID="$(respid)"

# A second program NOT linked to COURSE_ID — for the course-not-in-program rejection.
grpc cat CreateProgram \
  "$(jq -nc --arg c "SEOTHER-$SFX" '{code:$c, name:"SE Other Program"}')" "$ADMIN_SID"
expect_ok "CreateProgram (other, course not added)"; OTHER_PROG_ID="$(respid)"

# ---------------------------------------------------------------------------
# Enrollment setup: paid matrícula for A (open-period year), paid for B,
# pending for C (for the not-paid test), year-2025 enrollment for a mismatch test.
# ---------------------------------------------------------------------------
echo "▶ enrollment setup: paid + pending matrícula rows"

grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_A_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_ok "CreateEnrollment student A -> pending"; ENROLL_A_ID="$(respid)"

grpc enr MarkEnrollmentPaid \
  "$(jq -nc --arg i "$ENROLL_A_ID" '{id:$i}')" "$ADMIN_SID"
expect_ok "MarkEnrollmentPaid student A -> paid"

grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_B_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_ok "CreateEnrollment student B -> pending"; ENROLL_B_ID="$(respid)"

grpc enr MarkEnrollmentPaid \
  "$(jq -nc --arg i "$ENROLL_B_ID" '{id:$i}')" "$ADMIN_SID"
expect_ok "MarkEnrollmentPaid student B -> paid"

# Student C: pending matrícula only (for ErrNotPaid test).
grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_C_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_ok "CreateEnrollment student C -> pending (intentionally not paid)"; ENROLL_C_ID="$(respid)"

# Year-2025 enrollment for student A in the same program (for year-mismatch test).
grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_A_ID" --arg p "$PROG_ID" --argjson y "$MISMATCH_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_ok "CreateEnrollment student A year=2025 (for mismatch test)"; ENROLL_A_2025_ID="$(respid)"

grpc enr MarkEnrollmentPaid \
  "$(jq -nc --arg i "$ENROLL_A_2025_ID" '{id:$i}')" "$ADMIN_SID"
expect_ok "MarkEnrollmentPaid student A 2025 -> paid"

# Login student A and C for self-service tests.
STUDENT_A_SID="$(login_with_reset "$STUDENT_A_EMAIL" "StudentASmoke123!")"
[ -n "$STUDENT_A_SID" ] || { echo "FATAL: no student A sid" >&2; exit 2; }

STUDENT_C_SID="$(login_with_reset "$STUDENT_C_EMAIL" "StudentCSmoke123!")"
[ -n "$STUDENT_C_SID" ] || { echo "FATAL: no student C sid" >&2; exit 2; }

# ---------------------------------------------------------------------------
# Happy path: student A self-enrolls into SECTION_ID (open window, paid matrícula)
# ---------------------------------------------------------------------------
echo "▶ happy path: EnrollOwnSection (window open, paid matrícula)"
grpc se EnrollOwnSection \
  "$(jq -nc --arg s "$SECTION_ID" --arg p "$PROG_ID" '{section_id:$s, program_id:$p}')" "$STUDENT_A_SID"
expect_ok "EnrollOwnSection student A -> in_progress"
SE_A_ID="$(respid)"
if echo "$RESP" | jq -e '.status == "in_progress"' >/dev/null 2>&1; then
  ok "EnrollOwnSection status is in_progress"
else
  bad "EnrollOwnSection status" "expected status=in_progress, got: $(echo "$RESP" | tr '\n' ' ')"
fi

# ---------------------------------------------------------------------------
# Idempotency / duplicate: enrolling an already in_progress (enrollment, section) -> AlreadyExists
# Run immediately after student A's first enroll — only 1 seat taken, so the
# section is not full and the pre-check passes; the key uniqueness check fires.
# ---------------------------------------------------------------------------
echo "▶ idempotency/duplicate: EnrollSection on already in_progress -> AlreadyExists"
grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_A_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')" "$ADMIN_SID"
expect_code "AlreadyExists" "EnrollSection duplicate in_progress (admin)"

echo "▶ idempotency/duplicate: EnrollOwnSection on already in_progress -> AlreadyExists"
grpc se EnrollOwnSection \
  "$(jq -nc --arg s "$SECTION_ID" --arg p "$PROG_ID" '{section_id:$s, program_id:$p}')" "$STUDENT_A_SID"
expect_code "AlreadyExists" "EnrollOwnSection duplicate in_progress (student)"

# ---------------------------------------------------------------------------
# Admin enroll: admin enrolls student B via enrollment_id
# ---------------------------------------------------------------------------
echo "▶ admin enroll: EnrollSection (admin, enrollment_id)"
grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_B_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')" "$ADMIN_SID"
expect_ok "EnrollSection admin enroll student B -> in_progress"
SE_B_ID="$(respid)"

# ---------------------------------------------------------------------------
# Window closed: student C tries to self-enroll into closed-window section
# ---------------------------------------------------------------------------
echo "▶ window closed: EnrollOwnSection into closed-window section -> FailedPrecondition"
grpc se EnrollOwnSection \
  "$(jq -nc --arg s "$CLOSED_SECTION_ID" --arg p "$PROG_ID" '{section_id:$s, program_id:$p}')" "$STUDENT_C_SID"
expect_code "FailedPrecondition" "EnrollOwnSection window closed"

# ---------------------------------------------------------------------------
# Not paid: student C (pending matrícula) tries open-window section -> FailedPrecondition
# ---------------------------------------------------------------------------
echo "▶ not paid: EnrollOwnSection with pending matrícula -> FailedPrecondition"
grpc se EnrollOwnSection \
  "$(jq -nc --arg s "$SECTION_ID" --arg p "$PROG_ID" '{section_id:$s, program_id:$p}')" "$STUDENT_C_SID"
expect_code "FailedPrecondition" "EnrollOwnSection not paid"

# ---------------------------------------------------------------------------
# Year mismatch: admin sends enrollment whose year=2025 but section is in 2026 period
# ---------------------------------------------------------------------------
echo "▶ year mismatch: EnrollSection with enrollment year != section period year -> FailedPrecondition"
grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_A_2025_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')" "$ADMIN_SID"
expect_code "FailedPrecondition" "EnrollSection year mismatch"

# ---------------------------------------------------------------------------
# Course not in program: student A uses a program that doesn't contain the section's course
# ---------------------------------------------------------------------------
echo "▶ course not in program: EnrollOwnSection with unrelated program_id -> FailedPrecondition"
grpc se EnrollOwnSection \
  "$(jq -nc --arg s "$SECTION_ID" --arg p "$OTHER_PROG_ID" '{section_id:$s, program_id:$p}')" "$STUDENT_A_SID"
expect_code "FailedPrecondition" "EnrollOwnSection course not in program"

# ---------------------------------------------------------------------------
# Oversell: capacity=2, A and B already enrolled — 3rd enroll -> FailedPrecondition
# We use admin EnrollSection with student C's enrollment_id after paying C.
# First mark C paid so the capacity check is the limiting factor.
# ---------------------------------------------------------------------------
echo "▶ oversell: fill capacity=2, then 3rd -> FailedPrecondition (section full)"
grpc enr MarkEnrollmentPaid \
  "$(jq -nc --arg i "$ENROLL_C_ID" '{id:$i}')" "$ADMIN_SID"
expect_ok "MarkEnrollmentPaid student C -> paid (for oversell test)"

grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_C_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')" "$ADMIN_SID"
expect_code "FailedPrecondition" "EnrollSection oversell (section full)"

# ---------------------------------------------------------------------------
# Withdraw (admin-only): WithdrawSection on student A's in_progress -> OK
# ---------------------------------------------------------------------------
echo "▶ withdraw (admin-only): WithdrawSection -> OK (withdrawn)"
grpc se WithdrawSection \
  "$(jq -nc --arg i "$SE_A_ID" '{id:$i}')" "$ADMIN_SID"
expect_ok "WithdrawSection student A -> withdrawn"

# Now capacity has a free seat again (A's slot freed).

# ---------------------------------------------------------------------------
# Revival (admin-only): admin re-enrolls the withdrawn (ENROLL_A, SECTION) pair -> OK
# ---------------------------------------------------------------------------
echo "▶ revival (admin-only): EnrollSection on withdrawn inscription -> OK (in_progress)"
grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_A_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')" "$ADMIN_SID"
expect_ok "EnrollSection revive student A -> in_progress"
SE_A_REVIVED_ID="$(respid)"
if echo "$RESP" | jq -e '.status == "in_progress"' >/dev/null 2>&1; then
  ok "EnrollSection revived status is in_progress"
else
  bad "EnrollSection revived status" "expected status=in_progress, got: $(echo "$RESP" | tr '\n' ' ')"
fi

# ---------------------------------------------------------------------------
# Student cannot self-revive a withdrawn inscription.
# Withdraw student A again so the key is in 'withdrawn' state, then student A
# tries EnrollOwnSection for the same (section, program) -> ErrWithdrawnNotRevivable.
# ---------------------------------------------------------------------------
echo "▶ student cannot self-revive withdrawn inscription -> FailedPrecondition"
grpc se WithdrawSection \
  "$(jq -nc --arg i "$SE_A_REVIVED_ID" '{id:$i}')" "$ADMIN_SID"
expect_ok "WithdrawSection student A again (setup for self-revive test)"

grpc se EnrollOwnSection \
  "$(jq -nc --arg s "$SECTION_ID" --arg p "$PROG_ID" '{section_id:$s, program_id:$p}')" "$STUDENT_A_SID"
expect_code "FailedPrecondition" "EnrollOwnSection student self-revive withdrawn -> FailedPrecondition"

# Restore: admin re-enrolls A (was withdrawn) so self-scope tests can access a live inscription.
grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_A_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')" "$ADMIN_SID"
expect_ok "EnrollSection revive student A (restore for self-scope test)"
SE_A_FINAL_ID="$(respid)"

# ---------------------------------------------------------------------------
# Self-scope: student A sees own enrollments, not another student's
# ---------------------------------------------------------------------------
echo "▶ self-scope: GetOwnSectionEnrollment / ListOwnSectionEnrollments"
grpc se GetOwnSectionEnrollment \
  "$(jq -nc --arg i "$SE_A_FINAL_ID" '{id:$i}')" "$STUDENT_A_SID"
expect_ok "GetOwnSectionEnrollment student A (own id)"

# Fetch student B's id as student A -> NotFound (existence not disclosed).
grpc se GetOwnSectionEnrollment \
  "$(jq -nc --arg i "$SE_B_ID" '{id:$i}')" "$STUDENT_A_SID"
expect_code "NotFound" "GetOwnSectionEnrollment other student's id -> NotFound"

grpc se ListOwnSectionEnrollments '{}' "$STUDENT_A_SID"
expect_ok "ListOwnSectionEnrollments student A"
if echo "$RESP" | jq -e '.sectionEnrollments | length > 0' >/dev/null 2>&1; then
  ok "ListOwnSectionEnrollments returns non-empty list"
else
  bad "ListOwnSectionEnrollments non-empty" "expected at least one section enrollment for student A"
fi

# ---------------------------------------------------------------------------
# Denials
# ---------------------------------------------------------------------------
echo "▶ denials"

# No session -> Unauthenticated.
grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_A_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')"
expect_code "Unauthenticated" "EnrollSection without session"

grpc se EnrollOwnSection \
  "$(jq -nc --arg s "$SECTION_ID" --arg p "$PROG_ID" '{section_id:$s, program_id:$p}')"
expect_code "Unauthenticated" "EnrollOwnSection without session"

grpc se WithdrawSection "$(jq -nc --arg i "$SE_B_ID" '{id:$i}')"
expect_code "Unauthenticated" "WithdrawSection without session"

grpc se ListOwnSectionEnrollments '{}'
expect_code "Unauthenticated" "ListOwnSectionEnrollments without session"

# Role-less user -> PermissionDenied for management RPCs.
NOADMIN_SID="$(login_with_reset "$NOADMIN_EMAIL" "NoAdminSESmoke123!")"
[ -n "$NOADMIN_SID" ] || { echo "FATAL: no role-less sid" >&2; exit 2; }

grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_A_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')" "$NOADMIN_SID"
expect_code "PermissionDenied" "role-less EnrollSection"

grpc se WithdrawSection "$(jq -nc --arg i "$SE_B_ID" '{id:$i}')" "$NOADMIN_SID"
expect_code "PermissionDenied" "role-less WithdrawSection"

grpc se ListOwnSectionEnrollments '{}' "$NOADMIN_SID"
expect_code "PermissionDenied" "role-less ListOwnSectionEnrollments"

# Student using a management RPC -> PermissionDenied.
grpc se EnrollSection \
  "$(jq -nc --arg e "$ENROLL_A_ID" --arg s "$SECTION_ID" '{enrollment_id:$e, section_id:$s}')" "$STUDENT_A_SID"
expect_code "PermissionDenied" "student EnrollSection (management RPC)"

grpc se WithdrawSection "$(jq -nc --arg i "$SE_B_ID" '{id:$i}')" "$STUDENT_A_SID"
expect_code "PermissionDenied" "student WithdrawSection (management RPC)"

echo "──────────────────────────────────────"
printf 'RESULT: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
