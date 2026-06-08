package catalog_test

import (
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog"
)

func TestMapError_Sentinels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    error
		wantCode connect.Code
	}{
		{"ErrNotFound", catalog.ErrNotFound, connect.CodeNotFound},
		{"ErrInvalidInput", catalog.ErrInvalidInput, connect.CodeInvalidArgument},
		{"ErrAlreadyExists", catalog.ErrAlreadyExists, connect.CodeAlreadyExists},
		{"ErrHasDependents", catalog.ErrHasDependents, connect.CodeFailedPrecondition},
		{"wrapped ErrNotFound", fmt.Errorf("wrap: %w", catalog.ErrNotFound), connect.CodeNotFound},
		{"wrapped ErrAlreadyExists", fmt.Errorf("wrap: %w", catalog.ErrAlreadyExists), connect.CodeAlreadyExists},
		{"unexpected error", errors.New("unexpected"), connect.CodeInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := catalog.MapError(tc.input)
			var connectErr *connect.Error
			if !errors.As(err, &connectErr) {
				t.Fatalf("MapError(%v): returned non-connect error: %v", tc.input, err)
			}
			if connectErr.Code() != tc.wantCode {
				t.Errorf("MapError(%v): code = %v, want %v", tc.input, connectErr.Code(), tc.wantCode)
			}
		})
	}
}
