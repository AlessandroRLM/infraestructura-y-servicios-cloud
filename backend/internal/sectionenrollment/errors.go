// Package sectionenrollment implements the section enrollment vertical slice:
// window-gated self-enrollment, capacity-enforced seat accounting under a FOR UPDATE lock,
// and an admission control interceptor to protect the connection pool under stampede load.
package sectionenrollment

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Domain error sentinels returned by repository methods and propagated through the service.

// ErrNotFound is returned when a requested inscription does not exist, has been
// soft-deleted, or a self-scope ownership check fails (existence is never disclosed).
var ErrNotFound = fmt.Errorf("section_enrollment: not found")

// ErrInvalidInput is returned when required fields are missing or invalid,
// and also when the repository translates a FK violation (23503).
var ErrInvalidInput = fmt.Errorf("section_enrollment: invalid input")

// ErrAlreadyExists is returned when a live inscription already exists for the
// (enrollment_id, section_id) key (23505 or detected by row-level check).
var ErrAlreadyExists = fmt.Errorf("section_enrollment: already exists")

// ErrSectionFull is returned when the active seat count equals the section's capacity
// at the time of the under-lock check. Maps to CodeFailedPrecondition (NOT ResourceExhausted).
var ErrSectionFull = fmt.Errorf("section_enrollment: section full")

// ErrWindowClosed is returned when a self-enrollment attempt is made outside the
// academic period's enrollment window, or when the window is not configured (fail-closed).
var ErrWindowClosed = fmt.Errorf("section_enrollment: enrollment window closed")

// ErrNotPaid is returned when the linked enrollment does not have status='paid'.
var ErrNotPaid = fmt.Errorf("section_enrollment: enrollment is not paid")

// ErrCourseNotInProgram is returned when the section's course is not in the
// program_courses list for the enrollment's program.
var ErrCourseNotInProgram = fmt.Errorf("section_enrollment: course not in program")

// ErrEnrollmentYearMismatch is returned when the supplied enrollment's year does not
// match the academic period year of the target section. A matrícula is annual — it
// can only be used to enroll in sections belonging to the same academic year.
// Maps to CodeFailedPrecondition.
var ErrEnrollmentYearMismatch = fmt.Errorf("section_enrollment: enrollment year does not match the section's academic period year")

// ErrInvalidTransition is returned for any state transition that is not permitted
// by the state machine (e.g. withdrawing an already-withdrawn inscription).
var ErrInvalidTransition = fmt.Errorf("section_enrollment: invalid state transition")

// ErrWithdrawnNotRevivable is returned when a student attempts to self-enroll a section
// for which a withdrawn inscription already exists. Revival is admin-only.
var ErrWithdrawnNotRevivable = fmt.Errorf("section_enrollment: withdrawn inscription cannot be self-revived; contact an administrator")

// ErrLockTimeout is returned when the FOR UPDATE lock acquisition on the section row
// exceeds the configured lock_timeout budget (Postgres error 55P03).
// Maps to CodeUnavailable — transient contention.
var ErrLockTimeout = fmt.Errorf("section_enrollment: lock timeout")

// ErrAdmissionSaturated is returned by the concurrency limiter when the configured
// in-flight cap for inscription transactions is reached.
// Maps to CodeResourceExhausted — distinct from SectionFull (FailedPrecondition)
// and LockTimeout (Unavailable).
var ErrAdmissionSaturated = fmt.Errorf("section_enrollment: too many concurrent inscription requests; admission control saturated")

// TranslatePgError maps Postgres wire errors and pgx sentinels to domain error sentinels
// so that no *pgconn.PgError or raw DB text crosses the service boundary.
//
// Mapping:
//   - pgx.ErrNoRows          → ErrNotFound
//   - 23505 (unique_violation) → ErrAlreadyExists
//   - 23503 (fk_violation)   → ErrInvalidInput
//   - 55P03 (lock_not_available) → ErrLockTimeout
//   - anything else           → original error wrapped for stack context
func TranslatePgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w", ErrNotFound)
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w", ErrAlreadyExists)
		case "23503":
			return fmt.Errorf("%w", ErrInvalidInput)
		case "55P03":
			return fmt.Errorf("%w", ErrLockTimeout)
		}
	}
	return fmt.Errorf("section_enrollment: %w", err)
}
