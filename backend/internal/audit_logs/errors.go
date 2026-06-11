// Package audit_logs implements the audit_logs vertical slice: a single read-only
// Connect RPC (ListAuditLogs) over the append-only audit_logs table.
package audit_logs

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

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = fmt.Errorf("audit_logs: not found")

// ErrInvalidInput is returned when required fields are missing, malformed, or invalid.
var ErrInvalidInput = fmt.Errorf("audit_logs: invalid input")

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
	return fmt.Errorf("audit_logs: %w", err)
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
	default:
		slog.ErrorContext(ctx, "audit_logs: internal error", "err", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}
