-- Section enrollment: links a paid annual enrollment to a specific section.
-- This is the highest-contention table in the system; index design and constraint
-- choices are load-bearing (see partial index below).

CREATE TABLE section_enrollments (
    id            UUID        PRIMARY KEY DEFAULT uuidv7(),
    enrollment_id UUID        NOT NULL REFERENCES enrollments(id),
    section_id    UUID        NOT NULL REFERENCES sections(id),
    -- status lifecycle: in_progress (active) → withdrawn (admin-only).
    -- passed/failed transitions are owned by the grades slice via a mediated method;
    -- do not write those values from this slice.
    status        TEXT        NOT NULL DEFAULT 'in_progress'
                              CHECK (status IN ('in_progress', 'passed', 'failed', 'withdrawn')),
    -- registered_at is the business inscription timestamp; created_at is row creation.
    -- Both are set at insert and may coincide but are semantically distinct.
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    -- No created_by / updated_by: section_enrollments is not in the "sensitive human
    -- changes" list per the operations matrix.
    UNIQUE (enrollment_id, section_id)
);

-- Partial index on active seats: bounds the per-section COUNT to at most capacity
-- live rows (withdrawn and soft-deleted excluded), keeping the critical section O(small).
-- The WHERE clause must exactly match the seat-count query predicate.
CREATE INDEX section_enrollments_active_seat_idx
    ON section_enrollments (section_id)
    WHERE status <> 'withdrawn' AND deleted_at IS NULL;
