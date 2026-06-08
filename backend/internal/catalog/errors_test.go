package catalog_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog"
)

func TestTranslatePgError_UniqueViolation(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: "23505"}
	err := catalog.TranslatePgError(pgErr)
	if !errors.Is(err, catalog.ErrAlreadyExists) {
		t.Errorf("translatePgError(23505): got %v, want ErrAlreadyExists", err)
	}
}

func TestTranslatePgError_FKViolation(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: "23503"}
	err := catalog.TranslatePgError(pgErr)
	if !errors.Is(err, catalog.ErrInvalidInput) {
		t.Errorf("translatePgError(23503): got %v, want ErrInvalidInput", err)
	}
}

func TestTranslatePgError_NoRows(t *testing.T) {
	t.Parallel()

	err := catalog.TranslatePgError(pgx.ErrNoRows)
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("translatePgError(pgx.ErrNoRows): got %v, want ErrNotFound", err)
	}
}

func TestTranslatePgError_OtherError(t *testing.T) {
	t.Parallel()

	other := errors.New("some other database error")
	err := catalog.TranslatePgError(other)
	// Must not wrap into any known sentinel.
	if errors.Is(err, catalog.ErrAlreadyExists) || errors.Is(err, catalog.ErrInvalidInput) || errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("translatePgError(other): got sentinel %v, want passthrough", err)
	}
	// Must still propagate the original error.
	if !errors.Is(err, other) {
		t.Errorf("translatePgError(other): original error not wrapped, got %v", err)
	}
}

func TestTranslatePgError_WrappedUniqueViolation(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: "23505"}
	wrapped := errors.Join(errors.New("outer"), pgErr)
	err := catalog.TranslatePgError(wrapped)
	if !errors.Is(err, catalog.ErrAlreadyExists) {
		t.Errorf("translatePgError(wrapped 23505): got %v, want ErrAlreadyExists", err)
	}
}
