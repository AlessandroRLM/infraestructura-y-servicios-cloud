-- Add enrollment window columns to academic_periods.
-- These define the institutional inscription window: students may self-enroll
-- only when now() falls within [enrollment_starts_at, enrollment_ends_at].
-- NULL/unset is treated as "window not configured" → fail-closed for self-enrollment.
ALTER TABLE academic_periods
    ADD COLUMN enrollment_starts_at TIMESTAMPTZ,
    ADD COLUMN enrollment_ends_at   TIMESTAMPTZ;
