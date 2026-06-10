-- audit_logs: append-only event log for sensitive value changes (e.g. grade corrections).
-- Carries only created_at per §10.1 (append-only join table pattern): no updated_at,
-- no deleted_at, no *_by columns. Rows are never updated or deleted.

CREATE TABLE audit_logs (
    id         UUID        PRIMARY KEY DEFAULT uuidv7(),
    actor_id   UUID        REFERENCES users(id),
    action     TEXT        NOT NULL,
    entity     TEXT        NOT NULL,
    entity_id  UUID        NOT NULL,
    detail     JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for efficient lookup of all events for a given entity instance.
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity, entity_id);
