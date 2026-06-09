ALTER TABLE academic_periods
    DROP COLUMN IF EXISTS enrollment_starts_at,
    DROP COLUMN IF EXISTS enrollment_ends_at;
