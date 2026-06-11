// Package grades implements the grades vertical slice: evaluation scheme management,
// grade recording with optimistic locking, weighted-final computation, and student
// self-view. passed/failed transitions on section_enrollments are mediated through
// the section_enrollment domain's SetSectionEnrollmentOutcomeTx.
package grades

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Domain error sentinels.

// ErrNotFound is returned when a grade, evaluation, or section_enrollment does not exist,
// has been soft-deleted, or a self-scope ownership check fails (existence is never disclosed).
var ErrNotFound = fmt.Errorf("grades: not found")

// ErrInvalidInput is returned when required fields are missing, out of range, or a
// FK violation (23503) is returned by the database.
var ErrInvalidInput = fmt.Errorf("grades: invalid input")

// ErrAlreadyExists is returned when a scheme for the course already exists (23505 on
// evaluations unique key, or detected by the CreateEvaluationSchemeTx guard).
var ErrAlreadyExists = fmt.Errorf("grades: already exists")

// ErrHasDependents is returned when RecreateEvaluationScheme is attempted but at least
// one grade references the current scheme.
var ErrHasDependents = fmt.Errorf("grades: has dependents")

// ErrConflict is returned when an optimistic-lock version check fails (grade.version
// mismatch). The error wraps the current version so the caller can include it in the
// response detail.
var ErrConflict = fmt.Errorf("grades: version conflict")

// ErrSchemeIncomplete is returned when submitted evaluation weights do not sum to exactly
// 1.0 or the evaluation list is empty.
var ErrSchemeIncomplete = fmt.Errorf("grades: scheme incomplete")

// ErrSelfGrade is returned when the teacher attempts to grade their own section_enrollment.
var ErrSelfGrade = fmt.Errorf("grades: caller is the student being graded")

// ErrNotSectionTeacher is returned when the caller does not belong to section_teachers
// for the section that owns the target section_enrollment.
var ErrNotSectionTeacher = fmt.Errorf("grades: caller is not a teacher for this section")

// ErrInvalidTransition is returned when the section_enrollment transition is rejected
// (withdrawn source, or in_progress target). Re-exported concept from the SE domain.
var ErrInvalidTransition = fmt.Errorf("grades: invalid section_enrollment transition")

// ErrWithdrawnSource is returned when RecordGrade/OverrideGrade is called for a
// section_enrollment whose status is withdrawn.
var ErrWithdrawnSource = fmt.Errorf("grades: section_enrollment is withdrawn")

// ErrCourseMismatch is returned when the evaluation belongs to a different course than
// the section_enrollment's section.
var ErrCourseMismatch = fmt.Errorf("grades: evaluation course does not match section_enrollment course")

// TranslatePgError maps Postgres wire errors and pgx sentinels to domain error sentinels
// so that no *pgconn.PgError or raw DB text crosses the service boundary.
//
// Mapping:
//   - pgx.ErrNoRows              → ErrNotFound
//   - 23505 (unique_violation)   → ErrAlreadyExists
//   - 23503 (fk_violation)       → ErrInvalidInput
//   - anything else               → original error wrapped for context
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
		}
	}
	return fmt.Errorf("grades: %w", err)
}

// MapError converts domain error sentinels to Connect RPC error codes.
// Internal errors are logged once with slog and returned with a generic message
// to prevent leaking implementation details to callers.
func MapError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrSchemeIncomplete), errors.Is(err, ErrCourseMismatch):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, ErrHasDependents), errors.Is(err, ErrInvalidTransition), errors.Is(err, ErrWithdrawnSource):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrSelfGrade), errors.Is(err, ErrNotSectionTeacher):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, ErrConflict):
		return connect.NewError(connect.CodeAborted, err)
	default:
		slog.ErrorContext(ctx, "grades: internal error", "err", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}
