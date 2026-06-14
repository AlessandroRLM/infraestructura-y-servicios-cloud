// Package iam implements the IAM vertical slice: user listing, lookup, and role management.
package iam

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
var ErrNotFound = fmt.Errorf("iam: not found")

// ErrInvalidInput is returned when required fields are missing, malformed, or invalid.
var ErrInvalidInput = fmt.Errorf("iam: invalid input")

// ErrAlreadyExists is returned when a unique constraint would be violated.
var ErrAlreadyExists = fmt.Errorf("iam: already exists")

// TranslatePgError maps Postgres wire errors and pgx sentinels to domain error sentinels
// so that no *pgconn.PgError or raw DB text crosses the service boundary.
//
// Mapping:
//   - pgx.ErrNoRows        → ErrNotFound
//   - 23505 (unique)       → ErrAlreadyExists
//   - 23503 (fk_violation) → ErrInvalidInput
//   - anything else        → wrapped error for context
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
	return fmt.Errorf("iam: %w", err)
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
	case errors.Is(err, ErrAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	default:
		slog.ErrorContext(ctx, "iam: internal error", "err", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}
