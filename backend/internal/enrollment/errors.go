// Package enrollment implements the annual enrollment vertical slice:
// quota-enforced program admission, status lifecycle, and student self-view.
package enrollment

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Domain error sentinels returned by repository methods and propagated through the service.

// ErrNotFound is returned when a requested enrollment does not exist or has been soft-deleted.
var ErrNotFound = fmt.Errorf("enrollment: not found")

// ErrInvalidInput is returned when required fields are missing or invalid,
// and also when the repository translates a FK violation (23503).
var ErrInvalidInput = fmt.Errorf("enrollment: invalid input")

// ErrAlreadyExists is returned when a live enrollment already exists for the
// (student_id, program_id, year) key (23505 or detected by row-level check).
var ErrAlreadyExists = fmt.Errorf("enrollment: already exists")

// ErrQuotaFull is returned when the active seat count equals the program's
// admission capacity for the given year.
var ErrQuotaFull = fmt.Errorf("enrollment: quota full")

// ErrQuotaNotFound is returned when no live program_quotas row exists for
// the (program_id, year) pair. Missing quota is a configuration error;
// there is no implicit unlimited capacity.
var ErrQuotaNotFound = fmt.Errorf("enrollment: quota not found")

// ErrInvalidTransition is returned when an operation requests a status change
// that is not permitted by the state machine (e.g. marking a cancelled
// enrollment as paid, or cancelling an already-cancelled enrollment).
var ErrInvalidTransition = fmt.Errorf("enrollment: invalid state transition")

// TranslatePgError maps Postgres wire errors to domain sentinels so that no
// *pgconn.PgError or raw DB text crosses the service boundary.
//
// Mapping:
//   - pgx.ErrNoRows     → ErrNotFound
//   - 23505 (unique_violation) → ErrAlreadyExists
//   - 23503 (foreign_key_violation) → ErrInvalidInput
//   - anything else → original error wrapped for stack context
func TranslatePgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w", ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w", ErrAlreadyExists)
		case "23503":
			return fmt.Errorf("%w", ErrInvalidInput)
		}
	}
	return fmt.Errorf("enrollment: %w", err)
}
