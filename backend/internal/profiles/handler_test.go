package profiles

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
)

// TestMapError_InternalDoesNotLeakRawError verifies that unexpected errors produce
// CodeInternal with the generic "internal error" message, not the raw error string.
func TestMapError_InternalDoesNotLeakRawError(t *testing.T) {
	rawMsg := "pq: relation \"users\" does not exist"
	err := mapError(fmt.Errorf("profiles: GetUserProfile: %s", rawMsg))

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

// TestMapError_KnownSentinels verifies that ErrNotFound and ErrInvalidInput still
// map to their expected codes even after the internal-error fix.
func TestMapError_KnownSentinels(t *testing.T) {
	cases := []struct {
		name     string
		in       error
		wantCode connect.Code
	}{
		{"ErrInvalidInput", ErrInvalidInput, connect.CodeInvalidArgument},
		{"ErrNotFound", ErrNotFound, connect.CodeNotFound},
		{"wrapped ErrNotFound", fmt.Errorf("wrap: %w", ErrNotFound), connect.CodeNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapError(tc.in)
			ce, ok := errors.AsType[*connect.Error](got)
			if !ok {
				t.Fatalf("expected *connect.Error, got %T", got)
			}
			if ce.Code() != tc.wantCode {
				t.Errorf("code = %v, want %v", ce.Code(), tc.wantCode)
			}
		})
	}
}
