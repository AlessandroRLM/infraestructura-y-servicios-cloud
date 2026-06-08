#!/usr/bin/env bash
#
# Data-driven gRPC smoke test for the profiles slice.
#
# Drives ProfileService over real gRPC (h2c) with grpcurl and exercises the
# two authorization paths end-to-end against a running server:
#   - management (requires users.manage): admin upserts and reads a profile;
#   - self-read (requires profile.view_own): admin reads their own profile;
#   - a role-less user is denied on both paths (CodePermissionDenied);
#   - a call with no session is rejected (CodeUnauthenticated);
#   - invalid input is rejected (CodeInvalidArgument).
#
# The bootstrap admin already holds the admin role (all permissions). A role-less
# user is seeded directly into Postgres, then given a password via the reset flow.
#
# Usage:  ./run_profiles_flow.sh
# Env:    GRPCURL (default $HOME/go/bin/grpcurl), ADDR (default 127.0.0.1:8080)

set -uo pipefail

GRPCURL="${GRPCURL:-$HOME/go/bin/grpcurl}"
ADDR="${ADDR:-127.0.0.1:8080}"
SVC_AUTH="auth.v1.AuthService"
SVC_PROF="profiles.v1.ProfileService"
ADMIN_EMAIL="admin@dev.local"
ADMIN_ID="a0000000-0000-0000-0000-000000000001"
NOADMIN_EMAIL="noadmin@profiles.smoke"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)/proto"

for bin in "$GRPCURL" jq docker; do
  command -v "$bin" >/dev/null 2>&1 || { echo "FATAL: '$bin' not found" >&2; exit 2; }
done

# grpc {auth|prof} METHOD JSON [extra-header]  -> RESP (stdout+stderr), RC
grpc() {
  local which="$1" method="$2" data="$3" header="${4:-}"
  local svc; case "$which" in auth) svc="$SVC_AUTH" ;; prof) svc="$SVC_PROF" ;; *) svc="$which" ;; esac
  local args=(-plaintext -import-path "$PROTO_ROOT" -proto "auth/v1/auth.proto" -proto "profiles/v1/profiles.proto" -d "$data")
  [ -n "$header" ] && args+=(-H "$header")
  RESP="$("$GRPCURL" "${args[@]}" "$ADDR" "$svc/$method" 2>&1)"; RC=$?
}

# login EMAIL PASSWORD  -> SID (via reset flow to set a known password)
login_with_reset() {
  local email="$1" pw="$2"
  grpc auth RequestPasswordReset "$(jq -nc --arg e "$email" '{email:$e}')"
  local token; token="$(echo "$RESP" | grep -oiP '"dev_?token"\s*:\s*"\K[^"]+' || true)"
  [ -n "$token" ] || { echo "FATAL: no dev_token for $email" >&2; exit 2; }
  grpc auth ConfirmPasswordReset "$(jq -nc --arg t "$token" --arg p "$pw" '{token:$t,new_password:$p}')"
  RESP="$("$GRPCURL" -plaintext -v -import-path "$PROTO_ROOT" -proto "auth/v1/auth.proto" \
    -d "$(jq -nc --arg e "$email" --arg p "$pw" '{email:$e,password:$p}')" "$ADDR" "$SVC_AUTH/Login" 2>&1)"
  echo "$RESP" | grep -oiP 'sid=\K[^;]+' | head -1
}

pass=0; fail=0
ok()  { printf '  \033[32m✓\033[0m %s\n' "$1"; pass=$((pass+1)); }
bad() { printf '  \033[31m✗\033[0m %s\n     %s\n' "$1" "$2"; fail=$((fail+1)); }
expect_code() { echo "$RESP" | grep -q "Code: $1" && ok "$2 (-> $1)" || bad "$2" "want Code: $1, got: $(echo "$RESP" | tr '\n' ' ')"; }
expect_ok()   { [ "$RC" -eq 0 ] && ok "$1" || bad "$1" "want OK, got: $(echo "$RESP" | tr '\n' ' ')"; }

echo "▶ seeding a role-less user in Postgres"
docker compose exec -T postgres psql -U app -d academico -q -c \
  "INSERT INTO users (id, email, password_hash) VALUES (uuidv7(), '$NOADMIN_EMAIL', 'placeholder') ON CONFLICT (email) DO NOTHING;" >/dev/null

echo "▶ logging in admin (has users.manage + profile.view_own)"
ADMIN_SID="$(login_with_reset "$ADMIN_EMAIL" "AdminSmoke123!")"
[ -n "$ADMIN_SID" ] || { echo "FATAL: no admin sid" >&2; exit 2; }

echo "▶ logging in the role-less user"
NOADMIN_SID="$(login_with_reset "$NOADMIN_EMAIL" "NoAdminSmoke123!")"
[ -n "$NOADMIN_SID" ] || { echo "FATAL: no role-less sid" >&2; exit 2; }

UP_BODY="$(jq -nc --arg id "$ADMIN_ID" '{user_id:$id, given_names:"Ada", last_name_paternal:"Lovelace", national_id_type:"RUT", national_id:"11111111-1"}')"

echo "▶ management path (admin, users.manage)"
grpc prof UpsertUserProfile "$UP_BODY" "cookie: sid=$ADMIN_SID"; expect_ok "admin UpsertUserProfile"
grpc prof GetUserProfile "$(jq -nc --arg id "$ADMIN_ID" '{user_id:$id}')" "cookie: sid=$ADMIN_SID"; expect_ok "admin GetUserProfile"

echo "▶ self-read path (admin, profile.view_own)"
grpc prof GetOwnProfile '{}' "cookie: sid=$ADMIN_SID"
if [ "$RC" -eq 0 ] && echo "$RESP" | grep -q "$ADMIN_ID"; then ok "admin GetOwnProfile returns own row"; else bad "admin GetOwnProfile" "$(echo "$RESP" | tr '\n' ' ')"; fi

echo "▶ denials"
grpc prof UpsertUserProfile "$UP_BODY"; expect_code "Unauthenticated" "UpsertUserProfile without session"
grpc prof UpsertUserProfile "$UP_BODY" "cookie: sid=$NOADMIN_SID"; expect_code "PermissionDenied" "role-less UpsertUserProfile"
grpc prof GetOwnProfile '{}' "cookie: sid=$NOADMIN_SID"; expect_code "PermissionDenied" "role-less GetOwnProfile"

echo "▶ validation"
grpc prof UpsertUserProfile "$(jq -nc --arg id "$ADMIN_ID" '{user_id:$id, given_names:"", last_name_paternal:"X", national_id_type:"RUT", national_id:"22222222-2"}')" "cookie: sid=$ADMIN_SID"
expect_code "InvalidArgument" "UpsertUserProfile blank given_names"

echo "──────────────────────────────────────"
printf 'RESULT: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
