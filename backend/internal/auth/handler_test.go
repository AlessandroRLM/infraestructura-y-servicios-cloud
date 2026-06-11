package auth

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
	rawMsg := "pq: relation \"sessions\" does not exist"
	err := mapError(fmt.Errorf("auth: Login: %s", rawMsg))

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
