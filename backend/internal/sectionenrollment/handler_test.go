package sectionenrollment

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
)

// TestMapError_AllMappings verifies that each domain sentinel maps to the correct
// Connect error code, and that the original error text is NOT forwarded on internal errors.
func TestMapError_AllMappings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		err      error
		wantCode connect.Code
	}{
		{"SectionFull → FailedPrecondition", ErrSectionFull, connect.CodeFailedPrecondition},
		{"WindowClosed → FailedPrecondition", ErrWindowClosed, connect.CodeFailedPrecondition},
		{"NotPaid → FailedPrecondition", ErrNotPaid, connect.CodeFailedPrecondition},
		{"CourseNotInProgram → FailedPrecondition", ErrCourseNotInProgram, connect.CodeFailedPrecondition},
		{"EnrollmentYearMismatch → FailedPrecondition", ErrEnrollmentYearMismatch, connect.CodeFailedPrecondition},
		{"InvalidTransition → FailedPrecondition", ErrInvalidTransition, connect.CodeFailedPrecondition},
		{"WithdrawnNotRevivable → FailedPrecondition", ErrWithdrawnNotRevivable, connect.CodeFailedPrecondition},
		{"AdmissionSaturated → ResourceExhausted", ErrAdmissionSaturated, connect.CodeResourceExhausted},
		{"LockTimeout → Unavailable", ErrLockTimeout, connect.CodeUnavailable},
		{"AlreadyExists → AlreadyExists", ErrAlreadyExists, connect.CodeAlreadyExists},
		{"NotFound → NotFound", ErrNotFound, connect.CodeNotFound},
		{"InvalidInput → InvalidArgument", ErrInvalidInput, connect.CodeInvalidArgument},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := MapError(tc.err)
			ce, ok := result.(*connect.Error)
			if !ok {
				t.Fatalf("MapError(%v) returned %T; want *connect.Error", tc.err, result)
			}
			if ce.Code() != tc.wantCode {
				t.Errorf("code = %v, want %v", ce.Code(), tc.wantCode)
			}
		})
	}
}

// TestMapError_Internal_NoLeak verifies that an unmapped (internal) error does NOT
// forward the original error message — only the generic "internal error" string is returned.
func TestMapError_Internal_NoLeak(t *testing.T) {
	t.Parallel()

	secretErr := errors.New("super secret db internals: password=hunter2")
	result := MapError(secretErr)
	ce, ok := result.(*connect.Error)
	if !ok {
		t.Fatalf("MapError(unknown) returned %T; want *connect.Error", result)
	}
	if ce.Code() != connect.CodeInternal {
		t.Errorf("code = %v, want CodeInternal", ce.Code())
	}
	if ce.Message() == secretErr.Error() {
		t.Error("MapError leaked original error message; expected generic 'internal error'")
	}
	if ce.Message() != "internal error" {
		t.Errorf("message = %q, want 'internal error'", ce.Message())
	}
}

// TestMapError_WrappedSentinels verifies that wrapped sentinels (fmt.Errorf("%w", ...))
// are also correctly mapped by errors.Is traversal.
func TestMapError_WrappedSentinels(t *testing.T) {
	t.Parallel()

	wrapped := errors.Join(errors.New("context"), ErrSectionFull)
	result := MapError(wrapped)
	ce, ok := result.(*connect.Error)
	if !ok {
		t.Fatalf("MapError(wrapped SectionFull) returned %T", result)
	}
	if ce.Code() != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want CodeFailedPrecondition", ce.Code())
	}
}

