CREATE TABLE roles (
    id         uuid        PRIMARY KEY DEFAULT uuidv7(),
    name       text        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    code        text        NOT NULL UNIQUE,
    description text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- append-only join table per §10.1: composite PK, created_at only, no updated_at
CREATE TABLE role_permissions (
    role_id       uuid        NOT NULL REFERENCES roles(id),
    permission_id uuid        NOT NULL REFERENCES permissions(id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

-- append-only join table per §10.1: composite PK, created_at only, no updated_at
CREATE TABLE user_roles (
    user_id    uuid        NOT NULL REFERENCES users(id),
    role_id    uuid        NOT NULL REFERENCES roles(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);
