#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# Required env vars (fail fast if missing)
# ---------------------------------------------------------------------------
: "${S3_BUCKET:?S3_BUCKET is required}"

# ---------------------------------------------------------------------------
# Optional env vars with defaults
# ---------------------------------------------------------------------------
BACKUP_OBJECT="${BACKUP_OBJECT:-latest}"
TARGET_NAMESPACE="${TARGET_NAMESPACE:-test}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log() { printf '%s  %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

# ---------------------------------------------------------------------------
# Resolve "latest" to an actual S3 key
# ---------------------------------------------------------------------------
if [ "${BACKUP_OBJECT}" = "latest" ]; then
  log "Resolving latest object from s3://${S3_BUCKET}/"
  BACKUP_OBJECT="$(aws s3 ls "s3://${S3_BUCKET}/" \
    | grep 'academico-.*\.sql\.gz' \
    | sort \
    | tail -n 1 \
    | awk '{print $4}')"
  if [ -z "${BACKUP_OBJECT}" ]; then
    printf 'ERROR: No backup objects found in s3://%s/\n' "${S3_BUCKET}" >&2
    exit 1
  fi
  log "Resolved to: ${BACKUP_OBJECT}"
fi

WORKDIR="$(mktemp -d)"
trap 'log "Cleaning up workdir ${WORKDIR}"; rm -rf "${WORKDIR}"' EXIT

log "Restore test starting"
log "Source:    s3://${S3_BUCKET}/${BACKUP_OBJECT}"
log "Target NS: ${TARGET_NAMESPACE}"

# Download from S3 (cross-cloud restore — proves the DR copy is usable)
log "Downloading from S3"
aws s3 cp "s3://${S3_BUCKET}/${BACKUP_OBJECT}" "${WORKDIR}/${BACKUP_OBJECT}"
log "Download complete ($(du -sh "${WORKDIR}/${BACKUP_OBJECT}" | cut -f1))"

# Restore into the target namespace's postgres — DATABASE_URL expands in-pod
log "Restoring into ${TARGET_NAMESPACE}/statefulset/postgres"
gunzip -c "${WORKDIR}/${BACKUP_OBJECT}" | \
  kubectl -n "${TARGET_NAMESPACE}" exec -i statefulset/postgres -- \
    sh -c 'psql "$DATABASE_URL"'
log "Restore complete"

# Validation query — counts enrollments as documented restore-test evidence (RNF-5)
log "Running validation query"
kubectl -n "${TARGET_NAMESPACE}" exec statefulset/postgres -- \
  sh -c 'psql "$DATABASE_URL" -c "SELECT count(*) AS enrollment_count FROM enrollments;"'

log "Restore test finished successfully"
