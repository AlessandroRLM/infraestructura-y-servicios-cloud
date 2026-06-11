package sectionenrollment

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestTranslatePgError verifies the mapping from Postgres wire errors and pgx
// sentinel values to the domain error sentinels.
func TestTranslatePgError(t *testing.T) {
	t.Parallel()

	t.Run("ErrNoRows maps to ErrNotFound", func(t *testing.T) {
		t.Parallel()
		result := TranslatePgError(pgx.ErrNoRows)
		if !errors.Is(result, ErrNotFound) {
			t.Errorf("TranslatePgError(ErrNoRows) = %v; want errors.Is(_, ErrNotFound)", result)
		}
	})

	t.Run("23505 maps to ErrAlreadyExists", func(t *testing.T) {
		t.Parallel()
		pgErr := &pgconn.PgError{Code: "23505"}
		result := TranslatePgError(pgErr)
		if !errors.Is(result, ErrAlreadyExists) {
			t.Errorf("TranslatePgError(23505) = %v; want errors.Is(_, ErrAlreadyExists)", result)
		}
	})

	t.Run("23503 maps to ErrInvalidInput", func(t *testing.T) {
		t.Parallel()
		pgErr := &pgconn.PgError{Code: "23503"}
		result := TranslatePgError(pgErr)
		if !errors.Is(result, ErrInvalidInput) {
			t.Errorf("TranslatePgError(23503) = %v; want errors.Is(_, ErrInvalidInput)", result)
		}
	})

	t.Run("55P03 maps to ErrLockTimeout", func(t *testing.T) {
		t.Parallel()
		pgErr := &pgconn.PgError{Code: "55P03"}
		result := TranslatePgError(pgErr)
		if !errors.Is(result, ErrLockTimeout) {
			t.Errorf("TranslatePgError(55P03) = %v; want errors.Is(_, ErrLockTimeout)", result)
		}
	})

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if result := TranslatePgError(nil); result != nil {
			t.Errorf("TranslatePgError(nil) = %v; want nil", result)
		}
	})

	t.Run("unknown error wraps original", func(t *testing.T) {
		t.Parallel()
		original := errors.New("unexpected db error")
		result := TranslatePgError(original)
		if result == nil {
			t.Fatal("TranslatePgError(unknown) = nil; want non-nil")
		}
		if errors.Is(result, ErrNotFound) || errors.Is(result, ErrAlreadyExists) || errors.Is(result, ErrLockTimeout) {
			t.Errorf("TranslatePgError(unknown) mapped to a known sentinel unexpectedly: %v", result)
		}
	})
}
