package enrollment

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
)

// TestMapError_Table verifies that every domain sentinel maps to the expected Connect code
// and that unmapped errors produce CodeInternal with a generic message (no leakage).
func TestMapError_Table(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("original internal details — must not leak")

	cases := []struct {
		name     string
		err      error
		wantCode connect.Code
	}{
		{
			name:     "ErrInvalidInput",
			err:      ErrInvalidInput,
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:     "ErrNotFound",
			err:      ErrNotFound,
			wantCode: connect.CodeNotFound,
		},
		{
			name:     "ErrAlreadyExists",
			err:      ErrAlreadyExists,
			wantCode: connect.CodeAlreadyExists,
		},
		{
			name:     "ErrQuotaFull",
			err:      ErrQuotaFull,
			wantCode: connect.CodeFailedPrecondition,
		},
		{
			name:     "ErrQuotaNotFound",
			err:      ErrQuotaNotFound,
			wantCode: connect.CodeFailedPrecondition,
		},
		{
			name:     "ErrInvalidTransition",
			err:      ErrInvalidTransition,
			wantCode: connect.CodeFailedPrecondition,
		},
		{
			name:     "unmapped_error_CodeInternal",
			err:      sentinel,
			wantCode: connect.CodeInternal,
		},
		{
			name:     "wrapped_ErrNotFound",
			err:      errors.Join(ErrNotFound, errors.New("extra")),
			wantCode: connect.CodeNotFound,
		},
		{
			name:     "wrapped_ErrInvalidInput",
			err:      errors.Join(ErrInvalidInput, errors.New("field")),
			wantCode: connect.CodeInvalidArgument,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := MapError(tc.err)
			ce, ok := got.(*connect.Error)
			if !ok {
				t.Fatalf("MapError(%v) = %T, want *connect.Error", tc.err, got)
			}
			if ce.Code() != tc.wantCode {
				t.Errorf("MapError(%v).Code() = %v, want %v", tc.err, ce.Code(), tc.wantCode)
			}
			// Unmapped errors must not expose original error text.
			if tc.wantCode == connect.CodeInternal {
				if ce.Message() == sentinel.Error() {
					t.Errorf("MapError: CodeInternal leaked original error text %q", sentinel.Error())
				}
			}
		})
	}
}
