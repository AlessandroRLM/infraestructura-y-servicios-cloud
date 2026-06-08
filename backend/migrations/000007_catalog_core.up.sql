-- Core catalog taxonomy: programs, courses, program_courses, academic_periods, program_quotas.
-- Sections and section_teachers are in the next migration.

CREATE TABLE programs (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    code       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);

CREATE TABLE courses (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    code       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    credits    INT  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);

-- Append-only M:N join: a course may belong to multiple programs.
CREATE TABLE program_courses (
    program_id UUID NOT NULL REFERENCES programs(id),
    course_id  UUID NOT NULL REFERENCES courses(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (program_id, course_id)
);

-- Academic periods have no created_by/updated_by per the audit matrix.
CREATE TABLE academic_periods (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    year       INT  NOT NULL,
    term       INT  NOT NULL,
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (year, term)
);

-- Program quotas: fully audited and soft-deletable per the audit matrix.
CREATE TABLE program_quotas (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    program_id UUID NOT NULL REFERENCES programs(id),
    year       INT  NOT NULL,
    capacity   INT  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    UNIQUE (program_id, year)
);
