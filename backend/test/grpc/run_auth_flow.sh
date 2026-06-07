#!/usr/bin/env bash
#
# Data-driven gRPC smoke test for the auth slice.
#
# Reads the proto from the repo, drives the AuthService over real gRPC (h2c) with
# grpcurl, runs a table of login payloads from data/login_cases.json, and exercises
# the full session lifecycle (reset -> confirm -> login -> logout -> reject).
#
# Apidog tooling cannot drive gRPC headlessly, so this uses grpcurl directly.
#
# Usage:
#   ./run_auth_flow.sh [iterations]      # default 1
# Env overrides:
#   GRPCURL   path to grpcurl       (default: $HOME/go/bin/grpcurl)
#   ADDR      server host:port      (default: 127.0.0.1:8080)
#   ADMIN     bootstrap admin email (default: admin@dev.local)

set -uo pipefail

GRPCURL="${GRPCURL:-$HOME/go/bin/grpcurl}"
ADDR="${ADDR:-127.0.0.1:8080}"
ADMIN="${ADMIN:-admin@dev.local}"
ITERATIONS="${1:-1}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)/proto"
PROTO_FILE="auth/v1/auth.proto"
SVC="auth.v1.AuthService"
DATA="$SCRIPT_DIR/data/login_cases.json"
KNOWN_PW="LoopPass123!"

for bin in "$GRPCURL" jq; do
  command -v "$bin" >/dev/null 2>&1 || { echo "FATAL: '$bin' not found" >&2; exit 2; }
done
[ -f "$DATA" ] || { echo "FATAL: data file not found: $DATA" >&2; exit 2; }

# call METHOD JSON  -> RESP (stdout+stderr), RC (exit code)
call() {
  RESP="$("$GRPCURL" -plaintext -import-path "$PROTO_ROOT" -proto "$PROTO_FILE" \
    -d "$2" "$ADDR" "$SVC/$1" 2>&1)"
  RC=$?
}
# call with extra header (e.g. cookie metadata)
call_h() {
  RESP="$("$GRPCURL" -plaintext -H "$3" -import-path "$PROTO_ROOT" -proto "$PROTO_FILE" \
    -d "$2" "$ADDR" "$SVC/$1" 2>&1)"
  RC=$?
}

pass=0; fail=0
ok()   { printf '  \033[32m✓\033[0m %s\n' "$1"; pass=$((pass+1)); }
bad()  { printf '  \033[31m✗\033[0m %s\n     %s\n' "$1" "$2"; fail=$((fail+1)); }

echo "▶ target $ADDR · proto $PROTO_FILE · $ITERATIONS iteration(s)"

# Reachability.
call Login '{"email":"","password":""}'
if echo "$RESP" | grep -qiE 'Failed to dial|connection refused|context deadline'; then
  echo "FATAL: server not reachable at $ADDR — is it up?" >&2; exit 2
fi

# Seed a known password through the reset flow (the bootstrap admin hash is unknown).
echo "▶ seeding known password for $ADMIN via reset flow"
call RequestPasswordReset "$(jq -nc --arg e "$ADMIN" '{email:$e}')"
TOKEN="$(echo "$RESP" | grep -oiP '"dev_?token"\s*:\s*"\K[^"]+' || true)"
[ -n "$TOKEN" ] || { echo "FATAL: no dev_token returned (is APP_ENV non-production?)" >&2; exit 2; }
call ConfirmPasswordReset "$(jq -nc --arg t "$TOKEN" --arg p "$KNOWN_PW" '{token:$t,new_password:$p}')"
[ "$RC" -eq 0 ] || { echo "FATAL: ConfirmPasswordReset failed: $RESP" >&2; exit 2; }

# Data-driven login table.
for ((iter=1; iter<=ITERATIONS; iter++)); do
  echo "▶ login cases — iteration $iter/$ITERATIONS"
  # Read each case as a compact JSON object — robust to empty fields, which a
  # tab/whitespace IFS would silently collapse.
  while IFS= read -r row; do
    name="$(jq -r '.name'     <<<"$row")"
    email="$(jq -r '.email'    <<<"$row")"
    pw="$(jq -r '.password' <<<"$row")"
    expect="$(jq -r '.expect'   <<<"$row")"
    [ "$pw" = "__KNOWN__" ] && pw="$KNOWN_PW"
    payload="$(jq -nc --arg e "$email" --arg p "$pw" '{email:$e,password:$p}')"
    call Login "$payload"
    if [ "$expect" = "OK" ]; then
      if [ "$RC" -eq 0 ]; then ok "$name"; else bad "$name" "expected OK, got: $(echo "$RESP" | tr '\n' ' ')"; fi
    else
      if echo "$RESP" | grep -q "Code: $expect"; then ok "$name (-> $expect)";
      else bad "$name" "expected Code: $expect, got: $(echo "$RESP" | tr '\n' ' ')"; fi
    fi
  done < <(jq -c '.[]' "$DATA")
done

# Session lifecycle: login -> capture sid -> protected call -> logout -> reject reuse.
echo "▶ session lifecycle"
RESP="$("$GRPCURL" -plaintext -v -import-path "$PROTO_ROOT" -proto "$PROTO_FILE" \
  -d "$(jq -nc --arg e "$ADMIN" --arg p "$KNOWN_PW" '{email:$e,password:$p}')" \
  "$ADDR" "$SVC/Login" 2>&1)"
SID="$(echo "$RESP" | grep -oiP 'sid=\K[^;]+' | head -1 || true)"
[ -n "$SID" ] && ok "login issued session cookie" || bad "login cookie" "no sid in Set-Cookie metadata"

call_h Logout '{}' "cookie: sid=$SID"
[ "$RC" -eq 0 ] && ok "logout with valid cookie" || bad "logout with valid cookie" "$(echo "$RESP" | tr '\n' ' ')"

call_h Logout '{}' "cookie: sid=$SID"
echo "$RESP" | grep -q "Code: Unauthenticated" && ok "reused cookie rejected after logout" \
  || bad "reused cookie" "expected Unauthenticated, got: $(echo "$RESP" | tr '\n' ' ')"

call Logout '{}'
echo "$RESP" | grep -q "Code: Unauthenticated" && ok "logout without cookie rejected" \
  || bad "logout without cookie" "expected Unauthenticated, got: $(echo "$RESP" | tr '\n' ' ')"

echo "──────────────────────────────────────"
printf 'RESULT: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
