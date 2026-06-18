#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# Optional env vars with defaults
# ---------------------------------------------------------------------------
NAMESPACE="${NAMESPACE:-prod}"
KUBECTL="${KUBECTL:-kubectl}"
SQL_FILE="${SQL_FILE:-/opt/backup/seed_demo.sql}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log() { printf '%s  %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
log "Starting seed"
log "Namespace: ${NAMESPACE} | SQL file: ${SQL_FILE}"

if [ ! -f "${SQL_FILE}" ]; then
  printf 'ERROR: SQL file not found: %s\n' "${SQL_FILE}" >&2
  exit 1
fi

# Pipe the SQL into the postgres pod's psql. The api container is distroless
# (no shell, no psql) so psql runs inside the postgres pod, exactly like backup.sh.
"${KUBECTL}" -n "${NAMESPACE}" exec -i statefulset/postgres -- \
  sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < "${SQL_FILE}"

log "Seed finished successfully"
