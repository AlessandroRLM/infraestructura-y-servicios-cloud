#!/usr/bin/env bash
#
# Data-driven gRPC smoke test for the catalog slice.
#
# Drives CatalogService over real gRPC (h2c) with grpcurl against a running server
# and exercises the full admin taxonomy lifecycle plus the authorization and
# integrity rules:
#   - admin (catalog.manage) creates programs, courses, periods, quotas, sections;
#   - M:N course<->program association; section<->teacher assignment;
#   - dependent-blocking soft-delete (course/period/section blocked by live children);
#   - quota upsert; duplicate code -> AlreadyExists; bad FK -> InvalidArgument;
#   - validation (empty code, non-positive capacity) -> InvalidArgument;
#   - a role-less user -> PermissionDenied; no session -> Unauthenticated.
#
# Catalog teacher assignment needs a teacher_profile, so the admin upserts one for
# itself via ProfileService (it holds users.manage) and assigns itself to a section.
#
# Usage:  ./run_catalog_flow.sh
# Env:    GRPCURL (default $HOME/go/bin/grpcurl), ADDR (default 127.0.0.1:8080)

set -uo pipefail

GRPCURL="${GRPCURL:-$HOME/go/bin/grpcurl}"
ADDR="${ADDR:-127.0.0.1:8080}"
SVC_AUTH="auth.v1.AuthService"
SVC_CAT="catalog.v1.CatalogService"
SVC_PROF="profiles.v1.ProfileService"
ADMIN_EMAIL="admin@dev.local"
ADMIN_ID="a0000000-0000-0000-0000-000000000001"
NOADMIN_EMAIL="noadmin@catalog.smoke"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)/proto"

for bin in "$GRPCURL" jq docker; do
  command -v "$bin" >/dev/null 2>&1 || { echo "FATAL: '$bin' not found" >&2; exit 2; }
done

# grpc {auth|cat|prof} METHOD JSON [cookie-sid]  -> RESP (stdout+stderr), RC
grpc() {
  local which="$1" method="$2" data="$3" sid="${4:-}"
  local svc; case "$which" in auth) svc="$SVC_AUTH" ;; cat) svc="$SVC_CAT" ;; prof) svc="$SVC_PROF" ;; *) svc="$which" ;; esac
  local args=(-plaintext -import-path "$PROTO_ROOT"
    -proto auth/v1/auth.proto -proto catalog/v1/catalog.proto -proto profiles/v1/profiles.proto
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
ok()  { printf '  \033[32m✓\033[0m %s\n' "$1"; pass=$((pass+1)); }
bad() { printf '  \033[31m✗\033[0m %s\n     %s\n' "$1" "$2"; fail=$((fail+1)); }
expect_ok()   { [ "$RC" -eq 0 ] && ok "$1" || bad "$1" "want OK, got: $(echo "$RESP" | tr '\n' ' ')"; }
expect_code() { echo "$RESP" | grep -q "Code: $1" && ok "$2 (-> $1)" || bad "$2" "want $1, got: $(echo "$RESP" | tr '\n' ' ')"; }

echo "▶ seeding a role-less user"
docker compose exec -T postgres psql -U app -d academico -q -c \
  "INSERT INTO users (id, email, password_hash) VALUES (uuidv7(), '$NOADMIN_EMAIL', 'placeholder') ON CONFLICT (email) DO NOTHING;" >/dev/null

echo "▶ login admin (catalog.manage + users.manage)"
ADMIN_SID="$(login_with_reset "$ADMIN_EMAIL" "CatalogSmoke123!")"
[ -n "$ADMIN_SID" ] || { echo "FATAL: no admin sid" >&2; exit 2; }

# Unique code suffix so re-runs against a persistent DB do not collide.
SFX="$(date +%s | tail -c 6)"

echo "▶ taxonomy happy path (admin)"
grpc cat CreateProgram "$(jq -nc --arg c "INFO-$SFX" '{code:$c, name:"Informatica"}')" "$ADMIN_SID"; expect_ok "CreateProgram"; PROG_ID="$(respid)"
grpc cat CreateCourse "$(jq -nc --arg c "MAT-$SFX" '{code:$c, name:"Calculo I", credits:6}')" "$ADMIN_SID"; expect_ok "CreateCourse"; COURSE_ID="$(respid)"
grpc cat AddCourseToProgram "$(jq -nc --arg p "$PROG_ID" --arg c "$COURSE_ID" '{program_id:$p, course_id:$c}')" "$ADMIN_SID"; expect_ok "AddCourseToProgram"
grpc cat CreateAcademicPeriod "$(jq -nc '{year:2025, term:1, start_date:"2025-03-01", end_date:"2025-07-15"}')" "$ADMIN_SID"; expect_ok "CreateAcademicPeriod"; PERIOD_ID="$(respid)"
grpc cat CreateProgramQuota "$(jq -nc --arg p "$PROG_ID" '{program_id:$p, year:2025, admission_quota:40}')" "$ADMIN_SID"; expect_ok "CreateProgramQuota"
grpc cat CreateProgramQuota "$(jq -nc --arg p "$PROG_ID" '{program_id:$p, year:2025, admission_quota:45}')" "$ADMIN_SID"; expect_ok "CreateProgramQuota again (upsert, no AlreadyExists)"
grpc cat CreateSection "$(jq -nc --arg c "$COURSE_ID" --arg a "$PERIOD_ID" '{course_id:$c, academic_period_id:$a, seat_capacity:30}')" "$ADMIN_SID"; expect_ok "CreateSection"; SECTION_ID="$(respid)"

echo "▶ section_teachers (admin upserts own teacher profile, then assigns)"
grpc prof UpsertTeacherProfile "$(jq -nc --arg u "$ADMIN_ID" '{user_id:$u, department:"Matematica", title:"Profesor"}')" "$ADMIN_SID"; expect_ok "UpsertTeacherProfile (admin)"
grpc cat AssignTeacherToSection "$(jq -nc --arg s "$SECTION_ID" --arg t "$ADMIN_ID" '{section_id:$s, teacher_id:$t}')" "$ADMIN_SID"; expect_ok "AssignTeacherToSection"

echo "▶ dependent-blocking soft-delete (FailedPrecondition)"
grpc cat DeleteSection "$(jq -nc --arg i "$SECTION_ID" '{id:$i}')" "$ADMIN_SID"; expect_code "FailedPrecondition" "DeleteSection blocked by teacher"
grpc cat DeleteAcademicPeriod "$(jq -nc --arg i "$PERIOD_ID" '{id:$i}')" "$ADMIN_SID"; expect_code "FailedPrecondition" "DeleteAcademicPeriod blocked by section"
grpc cat DeleteCourse "$(jq -nc --arg i "$COURSE_ID" '{id:$i}')" "$ADMIN_SID"; expect_code "FailedPrecondition" "DeleteCourse blocked by program/section"

echo "▶ validation + conflicts"
grpc cat CreateProgram "$(jq -nc '{code:"", name:"x"}')" "$ADMIN_SID"; expect_code "InvalidArgument" "CreateProgram empty code"
grpc cat CreateSection "$(jq -nc --arg c "$COURSE_ID" --arg a "$PERIOD_ID" '{course_id:$c, academic_period_id:$a, seat_capacity:0}')" "$ADMIN_SID"; expect_code "InvalidArgument" "CreateSection zero capacity"
grpc cat CreateProgram "$(jq -nc --arg c "INFO-$SFX" '{code:$c, name:"dup"}')" "$ADMIN_SID"; expect_code "AlreadyExists" "CreateProgram duplicate code"
grpc cat CreateSection "$(jq -nc --arg a "$PERIOD_ID" '{course_id:"00000000-0000-7000-8000-000000000000", academic_period_id:$a, seat_capacity:10}')" "$ADMIN_SID"; expect_code "InvalidArgument" "CreateSection bad course FK"

echo "▶ denials"
grpc cat CreateProgram "$(jq -nc '{code:"NOPE", name:"x"}')"; expect_code "Unauthenticated" "CreateProgram without session"
NOADMIN_SID="$(login_with_reset "$NOADMIN_EMAIL" "NoAdminSmoke123!")"
grpc cat CreateProgram "$(jq -nc '{code:"NOPE2", name:"x"}')" "$NOADMIN_SID"; expect_code "PermissionDenied" "role-less CreateProgram"

echo "──────────────────────────────────────"
printf 'RESULT: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
