#!/usr/bin/env bash
#
# Data-driven gRPC smoke test for the enrollment slice.
#
# Drives EnrollmentService over real gRPC (h2c) with grpcurl against a running server
# and exercises the full enrollment lifecycle plus authorization, quota, and state-machine rules:
#   - admin (enrollment.manage) creates a program + quota (capacity=2), upserts student
#     profiles, then creates enrollments for student A and student B;
#   - state machine: pending->paid (ok), paid->paid (FailedPrecondition), cancel (ok),
#     cancel->cancel (FailedPrecondition), revive (create again, quota re-check);
#   - oversell: with 2 active seats a 3rd CreateEnrollment -> FailedPrecondition (quota full);
#   - quota missing: CreateEnrollment for program with no quota row -> FailedPrecondition;
#   - validation: bad UUID -> InvalidArgument; zero year -> InvalidArgument;
#   - duplicate active enrollment for same (student,program,year) -> AlreadyExists;
#   - own-scope (enrollment.view_own): student sees only their own enrollments;
#     GetOwnEnrollment for another student's id -> NotFound (existence not disclosed);
#   - denials: no session -> Unauthenticated; role-less user -> PermissionDenied.
#
# Usage:  ./run_enrollment_flow.sh
# Env:    GRPCURL (default $HOME/go/bin/grpcurl), ADDR (default 127.0.0.1:8080)

set -uo pipefail

GRPCURL="${GRPCURL:-$HOME/go/bin/grpcurl}"
ADDR="${ADDR:-127.0.0.1:8080}"
SVC_AUTH="auth.v1.AuthService"
SVC_ENR="enrollment.v1.EnrollmentService"
SVC_CAT="catalog.v1.CatalogService"
SVC_PROF="profiles.v1.ProfileService"
ADMIN_EMAIL="admin@dev.local"
NOADMIN_EMAIL="noadmin@enrollment.smoke"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)/proto"

for bin in "$GRPCURL" jq docker; do
  command -v "$bin" >/dev/null 2>&1 || { echo "FATAL: '$bin' not found" >&2; exit 2; }
done

# grpc {auth|enr|cat|prof} METHOD JSON [cookie-sid]  -> RESP (stdout+stderr), RC
grpc() {
  local which="$1" method="$2" data="$3" sid="${4:-}"
  local svc
  case "$which" in
    auth) svc="$SVC_AUTH" ;;
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

echo "▶ seeding a role-less user"
docker compose exec -T postgres psql -U app -d academico -q -c \
  "INSERT INTO users (id, email, password_hash) VALUES (uuidv7(), '$NOADMIN_EMAIL', 'placeholder') ON CONFLICT (email) DO NOTHING;" >/dev/null

echo "▶ login admin (enrollment.manage + catalog.manage + users.manage)"
ADMIN_SID="$(login_with_reset "$ADMIN_EMAIL" "EnrollmentSmoke123!")"
[ -n "$ADMIN_SID" ] || { echo "FATAL: no admin sid" >&2; exit 2; }

# Unique suffix so re-runs against a persistent DB do not collide on unique keys.
SFX="$(date +%s | tail -c 6)"

# Seed two student users and their student_profiles.
STUDENT_A_EMAIL="student.a.$SFX@enrollment.smoke"
STUDENT_B_EMAIL="student.b.$SFX@enrollment.smoke"
STUDENT_C_EMAIL="student.c.$SFX@enrollment.smoke"

echo "▶ seeding student users in Postgres"
docker compose exec -T postgres psql -U app -d academico -q -c \
  "INSERT INTO users (id, email, password_hash) VALUES
     (uuidv7(), '$STUDENT_A_EMAIL', 'placeholder'),
     (uuidv7(), '$STUDENT_B_EMAIL', 'placeholder'),
     (uuidv7(), '$STUDENT_C_EMAIL', 'placeholder')
   ON CONFLICT (email) DO NOTHING;
   -- Assign the student role (grants enrollment.view_own) to the three test students.
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

echo "▶ catalog prerequisites (program + quota, capacity=2)"
grpc cat CreateProgram \
  "$(jq -nc --arg c "PROG-$SFX" '{code:$c, name:"Test Program"}')" "$ADMIN_SID"
expect_ok "CreateProgram"; PROG_ID="$(respid)"

ENROLL_YEAR=2026
grpc cat CreateProgramQuota \
  "$(jq -nc --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" '{program_id:$p, year:$y, admission_quota:2}')" "$ADMIN_SID"
expect_ok "CreateProgramQuota (capacity=2)"

# A second program with no quota — used to test ErrQuotaNotFound.
grpc cat CreateProgram \
  "$(jq -nc --arg c "NOQUOTA-$SFX" '{code:$c, name:"No-Quota Program"}')" "$ADMIN_SID"
expect_ok "CreateProgram (no-quota)"; NOQUOTA_PROG_ID="$(respid)"

echo "▶ happy path: CreateEnrollment + MarkEnrollmentPaid (admin, enrollment.manage)"
grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_A_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_ok "CreateEnrollment student A -> pending"
ENROLL_A_ID="$(respid)"
[ -n "$ENROLL_A_ID" ] && echo "$RESP" | jq -e '.status == "pending"' >/dev/null 2>&1 && ok "status is pending" || \
  bad "CreateEnrollment A status" "expected status=pending in response"

grpc enr MarkEnrollmentPaid \
  "$(jq -nc --arg i "$ENROLL_A_ID" '{id:$i}')" "$ADMIN_SID"
expect_ok "MarkEnrollmentPaid student A -> paid"
echo "$RESP" | jq -e '.status == "paid"' >/dev/null 2>&1 && ok "status is paid" || \
  bad "MarkEnrollmentPaid status" "expected status=paid in response"

grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_B_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_ok "CreateEnrollment student B -> pending"
ENROLL_B_ID="$(respid)"

echo "▶ state machine: invalid transitions (FailedPrecondition)"
grpc enr MarkEnrollmentPaid \
  "$(jq -nc --arg i "$ENROLL_A_ID" '{id:$i}')" "$ADMIN_SID"
expect_code "FailedPrecondition" "MarkEnrollmentPaid on already-paid enrollment"

grpc enr CancelEnrollment \
  "$(jq -nc --arg i "$ENROLL_A_ID" '{id:$i}')" "$ADMIN_SID"
expect_ok "CancelEnrollment (paid -> cancelled) frees a seat"

grpc enr CancelEnrollment \
  "$(jq -nc --arg i "$ENROLL_A_ID" '{id:$i}')" "$ADMIN_SID"
expect_code "FailedPrecondition" "CancelEnrollment on already-cancelled enrollment"

echo "▶ revive: CreateEnrollment for cancelled student A (seat freed)"
grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_A_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_ok "CreateEnrollment revive student A -> pending"
ENROLL_A_REVIVED_ID="$(respid)"

echo "▶ oversell: with 2 active seats, 3rd CreateEnrollment -> FailedPrecondition"
grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_C_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_code "FailedPrecondition" "CreateEnrollment oversell (quota full)"

echo "▶ quota missing: CreateEnrollment for program with no quota row -> FailedPrecondition"
grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_A_ID" --arg p "$NOQUOTA_PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_code "FailedPrecondition" "CreateEnrollment quota not found"

echo "▶ duplicate active enrollment -> AlreadyExists"
grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_B_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$ADMIN_SID"
expect_code "AlreadyExists" "CreateEnrollment duplicate active enrollment"

echo "▶ validation: bad UUID / zero year -> InvalidArgument"
grpc enr CreateEnrollment \
  "$(jq -nc --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:"not-a-uuid", program_id:$p, year:$y}')" "$ADMIN_SID"
expect_code "InvalidArgument" "CreateEnrollment bad student_id UUID"

grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_A_ID" --arg p "$PROG_ID" \
       '{student_id:$s, program_id:$p, year:0}')" "$ADMIN_SID"
expect_code "InvalidArgument" "CreateEnrollment year=0"

grpc enr MarkEnrollmentPaid \
  "$(jq -nc '{id:"not-a-uuid"}')" "$ADMIN_SID"
expect_code "InvalidArgument" "MarkEnrollmentPaid bad id UUID"

grpc enr CancelEnrollment \
  "$(jq -nc '{id:"not-a-uuid"}')" "$ADMIN_SID"
expect_code "InvalidArgument" "CancelEnrollment bad id UUID"

echo "▶ own-scope (student A, enrollment.view_own)"
STUDENT_A_SID="$(login_with_reset "$STUDENT_A_EMAIL" "StudentASmoke123!")"
[ -n "$STUDENT_A_SID" ] || { echo "FATAL: no student A sid" >&2; exit 2; }

grpc enr ListOwnEnrollments '{}' "$STUDENT_A_SID"
expect_ok "ListOwnEnrollments student A"
if echo "$RESP" | jq -e '.enrollments | length > 0' >/dev/null 2>&1; then
  ok "ListOwnEnrollments returns non-empty list"
  # Verify only student A's enrollments are returned (JSON field is camelCase: studentId).
  if echo "$RESP" | jq -e --arg s "$STUDENT_A_ID" '[.enrollments[].studentId] | all(. == $s)' >/dev/null 2>&1; then
    ok "ListOwnEnrollments contains only student A enrollments"
  else
    bad "ListOwnEnrollments data isolation" "found non-student-A entries: $(echo "$RESP" | tr '\n' ' ')"
  fi
else
  bad "ListOwnEnrollments non-empty" "expected at least one enrollment for student A"
fi

grpc enr GetOwnEnrollment \
  "$(jq -nc --arg i "$ENROLL_A_REVIVED_ID" '{id:$i}')" "$STUDENT_A_SID"
expect_ok "GetOwnEnrollment student A (own enrollment)"

# Attempt to fetch student B's enrollment as student A -> NotFound (existence not disclosed).
grpc enr GetOwnEnrollment \
  "$(jq -nc --arg i "$ENROLL_B_ID" '{id:$i}')" "$STUDENT_A_SID"
expect_code "NotFound" "GetOwnEnrollment other student's id -> NotFound"

echo "▶ denials"
grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_A_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')"
expect_code "Unauthenticated" "CreateEnrollment without session"

grpc enr ListOwnEnrollments '{}'; expect_code "Unauthenticated" "ListOwnEnrollments without session"

NOADMIN_SID="$(login_with_reset "$NOADMIN_EMAIL" "NoAdminSmoke123!")"
[ -n "$NOADMIN_SID" ] || { echo "FATAL: no role-less sid" >&2; exit 2; }

grpc enr CreateEnrollment \
  "$(jq -nc --arg s "$STUDENT_A_ID" --arg p "$PROG_ID" --argjson y "$ENROLL_YEAR" \
       '{student_id:$s, program_id:$p, year:$y}')" "$NOADMIN_SID"
expect_code "PermissionDenied" "role-less CreateEnrollment"

grpc enr ListOwnEnrollments '{}' "$NOADMIN_SID"
expect_code "PermissionDenied" "role-less ListOwnEnrollments"

echo "──────────────────────────────────────"
printf 'RESULT: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
