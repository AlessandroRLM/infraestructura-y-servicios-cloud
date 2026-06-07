-- name: UpsertUserProfile :one
INSERT INTO user_profiles (
    user_id,
    given_names,
    last_name_paternal,
    last_name_maternal,
    national_id_type,
    national_id,
    birth_date,
    phone,
    personal_email,
    address_street,
    commune,
    region,
    country,
    postal_code,
    sex,
    nationality,
    photo_url,
    emergency_contact_name,
    emergency_contact_phone,
    created_by,
    updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
ON CONFLICT (user_id) DO UPDATE SET
    given_names              = EXCLUDED.given_names,
    last_name_paternal       = EXCLUDED.last_name_paternal,
    last_name_maternal       = EXCLUDED.last_name_maternal,
    national_id_type         = EXCLUDED.national_id_type,
    national_id              = EXCLUDED.national_id,
    birth_date               = EXCLUDED.birth_date,
    phone                    = EXCLUDED.phone,
    personal_email           = EXCLUDED.personal_email,
    address_street           = EXCLUDED.address_street,
    commune                  = EXCLUDED.commune,
    region                   = EXCLUDED.region,
    country                  = EXCLUDED.country,
    postal_code              = EXCLUDED.postal_code,
    sex                      = EXCLUDED.sex,
    nationality              = EXCLUDED.nationality,
    photo_url                = EXCLUDED.photo_url,
    emergency_contact_name   = EXCLUDED.emergency_contact_name,
    emergency_contact_phone  = EXCLUDED.emergency_contact_phone,
    updated_at               = now(),
    updated_by               = EXCLUDED.updated_by
RETURNING *;

-- name: GetUserProfile :one
SELECT *
FROM user_profiles
WHERE user_id = $1
  AND deleted_at IS NULL;

-- name: GetOwnProfile :one
SELECT *
FROM user_profiles
WHERE user_id = $1
  AND deleted_at IS NULL;

-- name: UpsertStudentProfile :one
INSERT INTO student_profiles (
    user_id,
    admission_year,
    created_by,
    updated_by
) VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    admission_year = EXCLUDED.admission_year,
    updated_at     = now(),
    updated_by     = EXCLUDED.updated_by
RETURNING *;

-- name: GetStudentProfile :one
SELECT *
FROM student_profiles
WHERE user_id = $1
  AND deleted_at IS NULL;

-- name: UpsertTeacherProfile :one
INSERT INTO teacher_profiles (
    user_id,
    department,
    title,
    created_by,
    updated_by
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET
    department = EXCLUDED.department,
    title      = EXCLUDED.title,
    updated_at = now(),
    updated_by = EXCLUDED.updated_by
RETURNING *;

-- name: GetTeacherProfile :one
SELECT *
FROM teacher_profiles
WHERE user_id = $1
  AND deleted_at IS NULL;

-- name: AddTeacherQualification :one
INSERT INTO teacher_qualifications (
    teacher_id,
    degree,
    year,
    created_by,
    updated_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListTeacherQualifications :many
SELECT *
FROM teacher_qualifications
WHERE teacher_id = $1
  AND deleted_at IS NULL
ORDER BY year, created_at;
