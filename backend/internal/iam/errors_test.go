package iam_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam"
)

// --- TranslatePgError ---

func TestTranslatePgError_NoRows(t *testing.T) {
	t.Parallel()

	err := iam.TranslatePgError(pgx.ErrNoRows)
	if !errors.Is(err, iam.ErrNotFound) {
		t.Errorf("TranslatePgError(pgx.ErrNoRows): got %v, want ErrNotFound", err)
	}
}

func TestTranslatePgError_UniqueViolation(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: "23505"}
	err := iam.TranslatePgError(pgErr)
	if !errors.Is(err, iam.ErrAlreadyExists) {
		t.Errorf("TranslatePgError(23505): got %v, want ErrAlreadyExists", err)
	}
}

func TestTranslatePgError_FKViolation(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: "23503"}
	err := iam.TranslatePgError(pgErr)
	if !errors.Is(err, iam.ErrInvalidInput) {
		t.Errorf("TranslatePgError(23503): got %v, want ErrInvalidInput", err)
	}
}

func TestTranslatePgError_OtherError(t *testing.T) {
	t.Parallel()

	other := errors.New("some other database error")
	err := iam.TranslatePgError(other)
	// Must not wrap into any known sentinel.
	if errors.Is(err, iam.ErrAlreadyExists) || errors.Is(err, iam.ErrInvalidInput) || errors.Is(err, iam.ErrNotFound) {
		t.Errorf("TranslatePgError(other): got sentinel %v, want passthrough", err)
	}
	// Must still propagate the original error.
	if !errors.Is(err, other) {
		t.Errorf("TranslatePgError(other): original error not wrapped, got %v", err)
	}
}

func TestTranslatePgError_WrappedUniqueViolation(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: "23505"}
	wrapped := errors.Join(errors.New("outer"), pgErr)
	err := iam.TranslatePgError(wrapped)
	if !errors.Is(err, iam.ErrAlreadyExists) {
		t.Errorf("TranslatePgError(wrapped 23505): got %v, want ErrAlreadyExists", err)
	}
}

func TestTranslatePgError_Nil(t *testing.T) {
	t.Parallel()

	err := iam.TranslatePgError(nil)
	if err != nil {
		t.Errorf("TranslatePgError(nil): got %v, want nil", err)
	}
}

// --- MapError ---

func TestMapError_Sentinels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    error
		wantCode connect.Code
	}{
		{"ErrNotFound", iam.ErrNotFound, connect.CodeNotFound},
		{"ErrInvalidInput", iam.ErrInvalidInput, connect.CodeInvalidArgument},
		{"ErrAlreadyExists", iam.ErrAlreadyExists, connect.CodeAlreadyExists},
		{"wrapped ErrNotFound", fmt.Errorf("wrap: %w", iam.ErrNotFound), connect.CodeNotFound},
		{"wrapped ErrAlreadyExists", fmt.Errorf("wrap: %w", iam.ErrAlreadyExists), connect.CodeAlreadyExists},
		{"unexpected error", errors.New("unexpected"), connect.CodeInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := iam.MapError(context.Background(), tc.input)
			connectErr, ok := errors.AsType[*connect.Error](err)
			if !ok {
				t.Fatalf("MapError(%v): returned non-connect error: %v", tc.input, err)
			}
			if connectErr.Code() != tc.wantCode {
				t.Errorf("MapError(%v): code = %v, want %v", tc.input, connectErr.Code(), tc.wantCode)
			}
		})
	}
}

func TestMapError_InternalDoesNotLeakRawError(t *testing.T) {
	t.Parallel()

	rawMsg := "pq: relation \"users\" does not exist"
	err := iam.MapError(context.Background(), fmt.Errorf("iam: some internal failure: %s", rawMsg))

	connectErr, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeInternal {
		t.Errorf("code = %v, want CodeInternal", connectErr.Code())
	}
	if strings.Contains(connectErr.Message(), rawMsg) {
		t.Errorf("client-visible message must not contain raw error; got: %q", connectErr.Message())
	}
	if connectErr.Message() != "internal error" {
		t.Errorf("message = %q, want %q", connectErr.Message(), "internal error")
	}
}
