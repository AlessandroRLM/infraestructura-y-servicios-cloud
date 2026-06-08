// Package catalog implements the academic catalog vertical slice: programs, courses,
// academic periods, program quotas, and their associations.
package catalog

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Domain error sentinels returned by repository methods and propagated through the service.

// ErrNotFound is returned when a requested row does not exist or has been soft-deleted.
var ErrNotFound = fmt.Errorf("catalog: not found")

// ErrInvalidInput is returned when required fields are missing or invalid,
// and also when the repository translates a FK violation (23503).
var ErrInvalidInput = fmt.Errorf("catalog: invalid input")

// ErrAlreadyExists is returned when a unique constraint is violated (23505).
var ErrAlreadyExists = fmt.Errorf("catalog: already exists")

// ErrHasDependents is returned when a soft-delete is blocked by live dependents.
var ErrHasDependents = fmt.Errorf("catalog: has dependents")

// TranslatePgError maps Postgres wire errors to domain sentinels so that no
// *pgconn.PgError or raw DB text crosses the service boundary.
//
// Mapping:
//   - 23505 (unique_violation) → ErrAlreadyExists
//   - 23503 (foreign_key_violation) → ErrInvalidInput
//   - pgx.ErrNoRows → ErrNotFound
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
	return fmt.Errorf("catalog: %w", err)
}
