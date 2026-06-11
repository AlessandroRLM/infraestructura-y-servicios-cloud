// Package reports implements the reports vertical slice: four authorized, Redis-cached
// read RPCs for grade reports, section occupancy, program enrollment summaries, and
// student academic records.
package reports

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Domain error sentinels.

// ErrNotFound is returned when a requested entity does not exist or is soft-deleted,
// and for out-of-scope teacher requests (existence is never disclosed to unauthorized callers).
var ErrNotFound = fmt.Errorf("reports: not found")

// ErrInvalidInput is returned when required fields are missing, malformed, or out of range.
var ErrInvalidInput = fmt.Errorf("reports: invalid input")

// ErrPermissionDenied is returned when a teacher calls an admin-only RPC.
// This is an operation-level denial (not resource-existence anti-leak).
var ErrPermissionDenied = fmt.Errorf("reports: caller does not have permission for this operation")

// TranslatePgError maps Postgres wire errors and pgx sentinels to domain error sentinels
// so that no *pgconn.PgError or raw DB text crosses the service boundary.
//
// Mapping:
//   - pgx.ErrNoRows              → ErrNotFound
//   - 23503 (fk_violation)       → ErrInvalidInput
//   - anything else              → wrapped error for context
func TranslatePgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w", ErrNotFound)
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23503":
			return fmt.Errorf("%w", ErrInvalidInput)
		}
	}
	return fmt.Errorf("reports: %w", err)
}

// MapError converts domain error sentinels to Connect RPC error codes.
// Internal errors are logged once with slog and returned with a generic message
// to prevent leaking implementation details to callers.
func MapError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrPermissionDenied):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		logInternalError(ctx, err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}
