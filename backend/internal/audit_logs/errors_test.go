package audit_logs

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestMapError verifies that MapError maps domain sentinels to the correct Connect codes.
func TestMapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantCode connect.Code
	}{
		{
			name:     "ErrInvalidInput returns CodeInvalidArgument",
			err:      ErrInvalidInput,
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:     "wrapped ErrInvalidInput returns CodeInvalidArgument",
			err:      fmt.Errorf("wrapped: %w", ErrInvalidInput),
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:     "ErrNotFound returns CodeNotFound",
			err:      ErrNotFound,
			wantCode: connect.CodeNotFound,
		},
		{
			name:     "unknown error returns CodeInternal",
			err:      errors.New("unknown"),
			wantCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MapError(context.Background(), tt.err)
			var connectErr *connect.Error
			if !errors.As(got, &connectErr) {
				t.Fatalf("expected *connect.Error, got %T: %v", got, got)
			}
			if connectErr.Code() != tt.wantCode {
				t.Errorf("MapError code = %v, want %v", connectErr.Code(), tt.wantCode)
			}
		})
	}
}

// TestMapError_Internal_ReturnsGenericMessage verifies that an internal error
// does not leak the original message to the caller.
func TestMapError_Internal_ReturnsGenericMessage(t *testing.T) {
	t.Parallel()
	secretErr := errors.New("secret internal detail")
	got := MapError(context.Background(), secretErr)
	var connectErr *connect.Error
	if !errors.As(got, &connectErr) {
		t.Fatalf("expected *connect.Error, got %T", got)
	}
	for _, detail := range connectErr.Details() {
		if detail != nil {
			t.Logf("detail: %v", detail)
		}
	}
	if connectErr.Message() == secretErr.Error() {
		t.Error("MapError must not leak the original error message for internal errors")
	}
}

// TestTranslatePgError verifies the mapping from pgx/PgError to domain sentinels.
func TestTranslatePgError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name:    "nil returns nil",
			err:     nil,
			wantErr: nil,
		},
		{
			name:    "pgx.ErrNoRows returns ErrNotFound",
			err:     pgx.ErrNoRows,
			wantErr: ErrNotFound,
		},
		{
			name: "23503 FK violation returns ErrInvalidInput",
			err: &pgconn.PgError{
				Code: "23503",
			},
			wantErr: ErrInvalidInput,
		},
		{
			name:    "unknown error is wrapped (not a sentinel)",
			err:     errors.New("some db error"),
			wantErr: nil, // should NOT be ErrNotFound or ErrInvalidInput
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TranslatePgError(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Errorf("TranslatePgError(nil) = %v, want nil", got)
				}
				return
			}
			if tt.wantErr != nil {
				if !errors.Is(got, tt.wantErr) {
					t.Errorf("TranslatePgError(%v) = %v, want errors.Is(%v)", tt.err, got, tt.wantErr)
				}
			} else {
				// unknown error case: must not be a known sentinel
				if errors.Is(got, ErrNotFound) || errors.Is(got, ErrInvalidInput) {
					t.Errorf("TranslatePgError unknown err should not map to a sentinel, got: %v", got)
				}
				if got == nil {
					t.Errorf("TranslatePgError unknown error should return a non-nil wrapped error")
				}
			}
		})
	}
}
