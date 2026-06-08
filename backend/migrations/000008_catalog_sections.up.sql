-- Sections and section_teachers: extend the catalog with per-section teaching assignments.

CREATE TABLE sections (
    id                 UUID PRIMARY KEY DEFAULT uuidv7(),
    course_id          UUID NOT NULL REFERENCES courses(id),
    academic_period_id UUID NOT NULL REFERENCES academic_periods(id),
    capacity           INT  NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ,
    created_by         UUID REFERENCES users(id),
    updated_by         UUID REFERENCES users(id)
);

-- Append-only M:N join: a teacher may be assigned to multiple sections.
-- teacher_id references teacher_profiles(user_id) — profiles slice must be applied first.
CREATE TABLE section_teachers (
    section_id UUID NOT NULL REFERENCES sections(id),
    teacher_id UUID NOT NULL REFERENCES teacher_profiles(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (section_id, teacher_id)
);
