#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# Required env vars (fail fast if missing)
# ---------------------------------------------------------------------------
: "${GCS_BUCKET:?GCS_BUCKET is required}"
: "${S3_BUCKET:?S3_BUCKET is required}"

# ---------------------------------------------------------------------------
# Optional env vars with defaults
# ---------------------------------------------------------------------------
NAMESPACE="${NAMESPACE:-prod}"
KUBECTL="${KUBECTL:-kubectl}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log() { printf '%s  %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
STAMP="$(date -u +%Y%m%d-%H%M)"
FILENAME="academico-${STAMP}.sql.gz"
WORKDIR="$(mktemp -d)"

trap 'log "Cleaning up workdir ${WORKDIR}"; rm -rf "${WORKDIR}"' EXIT

log "Starting backup: ${FILENAME}"
log "Namespace: ${NAMESPACE} | GCS: gs://${GCS_BUCKET}/ | S3: s3://${S3_BUCKET}/"

# Dump from the postgres pod — it has pg_dump and the DB credentials in its env.
# (The api container is distroless: no shell, no pg_dump.)
log "Running pg_dump via kubectl"
"${KUBECTL}" -n "${NAMESPACE}" exec statefulset/postgres -- \
  sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  | gzip > "${WORKDIR}/${FILENAME}"

log "pg_dump complete ($(du -sh "${WORKDIR}/${FILENAME}" | cut -f1))"

# Upload to primary GCS bucket
log "Uploading to GCS: gs://${GCS_BUCKET}/${FILENAME}"
gcloud storage cp "${WORKDIR}/${FILENAME}" "gs://${GCS_BUCKET}/${FILENAME}"
log "GCS upload complete"

# Replicate to AWS S3 (cross-cloud DR copy)
log "Uploading to S3: s3://${S3_BUCKET}/${FILENAME}"
aws s3 cp "${WORKDIR}/${FILENAME}" "s3://${S3_BUCKET}/${FILENAME}"
log "S3 upload complete"

log "Backup finished successfully: ${FILENAME}"
