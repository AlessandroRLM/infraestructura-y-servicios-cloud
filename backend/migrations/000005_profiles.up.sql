CREATE TABLE user_profiles (
    user_id                  UUID PRIMARY KEY REFERENCES users(id),
    given_names              TEXT NOT NULL,
    last_name_paternal       TEXT NOT NULL,
    last_name_maternal       TEXT,
    national_id_type         TEXT NOT NULL,
    national_id              TEXT NOT NULL UNIQUE,
    birth_date               DATE,
    phone                    TEXT,
    personal_email           TEXT,
    address_street           TEXT,
    commune                  TEXT,
    region                   TEXT,
    country                  TEXT,
    postal_code              TEXT,
    sex                      TEXT,
    nationality              TEXT,
    photo_url                TEXT,
    emergency_contact_name   TEXT,
    emergency_contact_phone  TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ,
    created_by               UUID REFERENCES users(id),
    updated_by               UUID REFERENCES users(id)
);

CREATE TABLE student_profiles (
    user_id        UUID PRIMARY KEY REFERENCES users(id),
    admission_year INT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    created_by     UUID REFERENCES users(id),
    updated_by     UUID REFERENCES users(id)
);

CREATE TABLE teacher_profiles (
    user_id    UUID PRIMARY KEY REFERENCES users(id),
    department TEXT,
    title      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);

CREATE TABLE teacher_qualifications (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    teacher_id UUID NOT NULL REFERENCES teacher_profiles(user_id),
    degree     TEXT NOT NULL,
    year       INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);
