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
