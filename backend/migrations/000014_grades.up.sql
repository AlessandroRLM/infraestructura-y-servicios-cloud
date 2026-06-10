-- Evaluation schemes: one immutable scheme per course (admin-managed, course-level).
-- Evaluations identify by server-assigned position (1..N from submitted order).
-- deleted_at present for the RecreateEvaluationScheme path (soft-delete + insert).
-- No created_by/updated_by per §10.1: evaluations is not in the sensitive-human-changes set.

CREATE TABLE evaluations (
    id         UUID         PRIMARY KEY DEFAULT uuidv7(),
    course_id  UUID         NOT NULL REFERENCES courses(id),
    weight     NUMERIC(4,3) NOT NULL CHECK (weight > 0 AND weight <= 1),
    position   INT          NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (course_id, position)
);

CREATE INDEX evaluations_course_idx ON evaluations (course_id) WHERE deleted_at IS NULL;

-- Grades: one per (evaluation, section_enrollment). Undeletable; corrections update value.
-- version enables optimistic-locking (the only entity with this column per §10.1).
-- deleted_at present for §10.1 schema conformance; NEVER written (no delete RPC exists).

CREATE TABLE grades (
    id                    UUID         PRIMARY KEY DEFAULT uuidv7(),
    evaluation_id         UUID         NOT NULL REFERENCES evaluations(id),
    section_enrollment_id UUID         NOT NULL REFERENCES section_enrollments(id),
    graded_by             UUID         NOT NULL REFERENCES users(id),
    value                 NUMERIC(3,1) NOT NULL CHECK (value >= 1.0 AND value <= 7.0),
    evaluated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    version               INT          NOT NULL DEFAULT 1,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by            UUID         REFERENCES users(id),
    updated_by            UUID         REFERENCES users(id),
    deleted_at            TIMESTAMPTZ,
    UNIQUE (evaluation_id, section_enrollment_id)
);

CREATE INDEX grades_section_enrollment_idx ON grades (section_enrollment_id);

-- Add the computed final grade column to section_enrollments.
-- Written atomically with the outcome by the mediated SetSectionEnrollmentOutcome call.
-- NULL while in_progress; set to the rounded (half-up, 1 decimal) weighted final on completion.
ALTER TABLE section_enrollments
    ADD COLUMN final_grade NUMERIC(3,1) NULL;
