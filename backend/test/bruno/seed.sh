#!/usr/bin/env bash
#
# seed.sh — one-time DB seeding for the backend-smoke Bruno collection.
#
# Run this script ONCE against a running dev stack before executing `bru run`.
# It performs the SQL inserts that the API cannot do (direct enrollment-window
# manipulation and user/role seeding without an onboarding RPC).
#
# Usage:
#   ./seed.sh
#
# Requirements:
#   - docker compose up (postgres container must be reachable)
#   - The same SFX (suffix) used by request 01 is generated here and exported
#     as environment variables. You must source this file or pass the variables
#     to `bru run` so the Bruno pre-request script and the DB rows stay in sync.
#
# NOTE: The Bruno pre-request script in request 01 generates its own SFX at
#       runtime using Date.now(). For full consistency, run seed.sh immediately
#       before `bru run` and use the same SFX by either:
#         a) Having seed.sh print export statements and eval them in CI, OR
#         b) Setting BRUNO_SFX to a fixed value and reading it in both places.
#       The recommended CI pattern is option (a): see the exported vars below.

set -euo pipefail

PSQL_CMD="docker compose exec -T postgres psql -U app -d academico"
ENROLL_YEAR=2026
SFX="$(date +%s | tail -c 6)"
OPEN_TERM=$((100 + (SFX % 800)))
CLOSED_TERM=$((900 + (SFX % 99)))

STUDENT_A_EMAIL="student.a.${SFX}@se.smoke"
STUDENT_B_EMAIL="student.b.${SFX}@se.smoke"
STUDENT_C_EMAIL="student.c.${SFX}@se.smoke"
NOADMIN_EMAIL="noadmin@se.smoke"

echo "▶ seeding users (students A/B/C + role-less noadmin)"
$PSQL_CMD -q -c "
  INSERT INTO users (id, email, password_hash) VALUES
    (uuidv7(), '${STUDENT_A_EMAIL}', 'placeholder'),
    (uuidv7(), '${STUDENT_B_EMAIL}', 'placeholder'),
    (uuidv7(), '${STUDENT_C_EMAIL}', 'placeholder'),
    (uuidv7(), '${NOADMIN_EMAIL}',   'placeholder')
  ON CONFLICT (email) DO NOTHING;

  INSERT INTO user_roles (user_id, role_id)
    SELECT u.id, r.id
    FROM users u, roles r
    WHERE u.email IN ('${STUDENT_A_EMAIL}', '${STUDENT_B_EMAIL}', '${STUDENT_C_EMAIL}')
      AND r.name = 'student'
  ON CONFLICT DO NOTHING;
"

echo "▶ fetching seeded user IDs"
STUDENT_A_ID="$($PSQL_CMD -tAq -c "SELECT id FROM users WHERE email='${STUDENT_A_EMAIL}';")"
STUDENT_B_ID="$($PSQL_CMD -tAq -c "SELECT id FROM users WHERE email='${STUDENT_B_EMAIL}';")"
STUDENT_C_ID="$($PSQL_CMD -tAq -c "SELECT id FROM users WHERE email='${STUDENT_C_EMAIL}';")"

[ -n "$STUDENT_A_ID" ] && [ -n "$STUDENT_B_ID" ] && [ -n "$STUDENT_C_ID" ] || \
  { echo "FATAL: could not fetch student IDs" >&2; exit 2; }

echo "▶ seeding academic_periods with enrollment windows"
# Open window: started 1 hour ago, ends 1 hour from now.
OPEN_PERIOD_ID="$($PSQL_CMD -tAq -c "
  INSERT INTO academic_periods (id, year, term, start_date, end_date, enrollment_starts_at, enrollment_ends_at)
  VALUES (uuidv7(), ${ENROLL_YEAR}, ${OPEN_TERM},
          '${ENROLL_YEAR}-03-01', '${ENROLL_YEAR}-07-15',
          now() - interval '1 hour', now() + interval '1 hour')
  RETURNING id;")"

# Closed window: starts 24 hours in the future — never open right now.
CLOSED_PERIOD_ID="$($PSQL_CMD -tAq -c "
  INSERT INTO academic_periods (id, year, term, start_date, end_date, enrollment_starts_at, enrollment_ends_at)
  VALUES (uuidv7(), ${ENROLL_YEAR}, ${CLOSED_TERM},
          '${ENROLL_YEAR}-08-01', '${ENROLL_YEAR}-12-15',
          now() + interval '24 hours', now() + interval '48 hours')
  RETURNING id;")"

[ -n "$OPEN_PERIOD_ID" ] && [ -n "$CLOSED_PERIOD_ID" ] || \
  { echo "FATAL: could not create academic periods" >&2; exit 2; }

echo "▶ done — exporting variables for bru run"
echo ""
echo "# Source these exports or pass them as --env-var flags to bru run:"
echo "export BRUNO_SFX='${SFX}'"
echo "export BRUNO_STUDENT_A_ID='${STUDENT_A_ID}'"
echo "export BRUNO_STUDENT_B_ID='${STUDENT_B_ID}'"
echo "export BRUNO_STUDENT_C_ID='${STUDENT_C_ID}'"
echo "export BRUNO_OPEN_PERIOD_ID='${OPEN_PERIOD_ID}'"
echo "export BRUNO_CLOSED_PERIOD_ID='${CLOSED_PERIOD_ID}'"
echo "export BRUNO_OPEN_TERM='${OPEN_TERM}'"
echo "export BRUNO_CLOSED_TERM='${CLOSED_TERM}'"
echo "export BRUNO_STUDENT_A_EMAIL='${STUDENT_A_EMAIL}'"
echo "export BRUNO_STUDENT_B_EMAIL='${STUDENT_B_EMAIL}'"
echo "export BRUNO_STUDENT_C_EMAIL='${STUDENT_C_EMAIL}'"
echo ""
echo "  SFX=${SFX}"
echo "  student_a_id=${STUDENT_A_ID}"
echo "  student_b_id=${STUDENT_B_ID}"
echo "  student_c_id=${STUDENT_C_ID}"
echo "  open_period_id=${OPEN_PERIOD_ID}"
echo "  closed_period_id=${CLOSED_PERIOD_ID}"
echo "  open_term=${OPEN_TERM} / closed_term=${CLOSED_TERM}"
