-- name: ListAuditLogs :many
-- Keyset pagination over the append-only audit_logs table, scoped to one entity instance.
-- entity + entity_id are required and ride audit_logs_entity_idx. Optional actor_id and
-- created_at range are residual filters. Newest-first (id DESC, UUIDv7 = reverse chrono).
SELECT id, actor_id, action, entity, entity_id, detail, created_at
FROM audit_logs
WHERE entity = $1
  AND entity_id = $2
  AND (sqlc.narg('actor_id')::uuid IS NULL      OR actor_id   = sqlc.narg('actor_id')::uuid)
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz   IS NULL OR created_at <= sqlc.narg('created_to')::timestamptz)
  AND (sqlc.narg('page_token')::uuid IS NULL    OR id          < sqlc.narg('page_token')::uuid)
ORDER BY id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: ListRecentAuditLogs :many
-- Global keyset-paginated feed over the entire audit_logs table, newest-first.
-- No entity/entity_id scoping — all rows are visible (global feed).
-- Optional actor_id and created_at range filters are residual.
-- Keyset cursor uses the composite (created_at DESC, id DESC) ordering:
-- cursor_ts + cursor_id encode the last row seen; next page contains rows strictly
-- older than (cursor_ts, cursor_id) in the DESC ordering.
SELECT id, actor_id, action, entity, entity_id, detail, created_at
FROM audit_logs
WHERE (sqlc.narg('actor_id')::uuid IS NULL OR actor_id = sqlc.narg('actor_id')::uuid)
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz IS NULL   OR created_at <= sqlc.narg('created_to')::timestamptz)
  AND (
    sqlc.narg('cursor_ts')::timestamptz IS NULL
    OR created_at < sqlc.narg('cursor_ts')::timestamptz
    OR (created_at = sqlc.narg('cursor_ts')::timestamptz AND id < sqlc.narg('cursor_id')::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('row_limit')::int;
