CREATE TABLE enrollments (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    student_id UUID NOT NULL REFERENCES student_profiles(user_id),
    program_id UUID NOT NULL REFERENCES programs(id),
    year       INT  NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'paid', 'cancelled')),
    paid_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    UNIQUE (student_id, program_id, year)
);
