package sectionenrollment

import (
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment/sectionenrollmentdb"
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
		{"ErrUnauthenticated → Unauthenticated", ErrUnauthenticated, connect.CodeUnauthenticated},
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

// --- ownSectionEnrollmentRowToProto field-mapping tests ---

// newEnrichedRow constructs a ListOwnSectionEnrollmentsEnrichedRow with controllable
// course/period display label values, suitable for proto-conversion assertions.
func newEnrichedRow(seID, enrollmentID, sectionID uuid.UUID, courseName, courseCode string, periodYear, periodTerm int32) sectionenrollmentdb.ListOwnSectionEnrollmentsEnrichedRow {
	now := time.Now()
	return sectionenrollmentdb.ListOwnSectionEnrollmentsEnrichedRow{
		ID:           pgtype.UUID{Bytes: seID, Valid: true},
		EnrollmentID: pgtype.UUID{Bytes: enrollmentID, Valid: true},
		SectionID:    pgtype.UUID{Bytes: sectionID, Valid: true},
		Status:       "in_progress",
		RegisteredAt: pgtype.Timestamptz{Time: now, Valid: true},
		CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		CourseName:   courseName,
		CourseCode:   courseCode,
		PeriodYear:   periodYear,
		PeriodTerm:   periodTerm,
	}
}

// TestOwnSectionEnrollmentRowToProto_EnrichedFields verifies that ownSectionEnrollmentRowToProto
// maps course_name, course_code, period_year, and period_term from the enriched row into
// the corresponding proto fields (11–14). These fields are populated only by
// ListOwnSectionEnrollments; zero values on all other RPCs (backward-compatible).
func TestOwnSectionEnrollmentRowToProto_EnrichedFields(t *testing.T) {
	t.Parallel()

	seID := uuid.New()
	enrollmentID := uuid.New()
	sectionID := uuid.New()

	row := newEnrichedRow(seID, enrollmentID, sectionID, "Calculus I", "MATH-101", 2024, 2)
	proto := ownSectionEnrollmentRowToProto(row)

	if proto.GetCourseName() != "Calculus I" {
		t.Errorf("course_name = %q, want %q", proto.GetCourseName(), "Calculus I")
	}
	if proto.GetCourseCode() != "MATH-101" {
		t.Errorf("course_code = %q, want %q", proto.GetCourseCode(), "MATH-101")
	}
	if proto.GetPeriodYear() != 2024 {
		t.Errorf("period_year = %d, want 2024", proto.GetPeriodYear())
	}
	if proto.GetPeriodTerm() != 2 {
		t.Errorf("period_term = %d, want 2", proto.GetPeriodTerm())
	}
}

// TestOwnSectionEnrollmentRowToProto_CoreFields verifies that the core section_enrollment
// fields (id, enrollment_id, section_id, status) are mapped correctly alongside
// the enriched fields.
func TestOwnSectionEnrollmentRowToProto_CoreFields(t *testing.T) {
	t.Parallel()

	seID := uuid.New()
	enrollmentID := uuid.New()
	sectionID := uuid.New()

	row := newEnrichedRow(seID, enrollmentID, sectionID, "Physics II", "PHYS-202", 2025, 1)
	proto := ownSectionEnrollmentRowToProto(row)

	if proto.GetId() != seID.String() {
		t.Errorf("id = %q, want %q", proto.GetId(), seID.String())
	}
	if proto.GetEnrollmentId() != enrollmentID.String() {
		t.Errorf("enrollment_id = %q, want %q", proto.GetEnrollmentId(), enrollmentID.String())
	}
	if proto.GetSectionId() != sectionID.String() {
		t.Errorf("section_id = %q, want %q", proto.GetSectionId(), sectionID.String())
	}
	if proto.GetStatus() != "in_progress" {
		t.Errorf("status = %q, want in_progress", proto.GetStatus())
	}
}

// TestSectionEnrollmentToProto_EnrichedFieldsAreZero verifies that sectionEnrollmentToProto
// (used by all other RPCs except ListOwnSectionEnrollments) leaves course_name, course_code,
// period_year, and period_term at their zero values — backward-compatible, no leakage.
func TestSectionEnrollmentToProto_EnrichedFieldsAreZero(t *testing.T) {
	t.Parallel()

	row := newInsertedRow(uuid.New(), uuid.New(), uuid.New())
	proto := sectionEnrollmentToProto(row)

	if proto.GetCourseName() != "" {
		t.Errorf("sectionEnrollmentToProto: course_name = %q, want empty (other RPCs must leave it zero)", proto.GetCourseName())
	}
	if proto.GetCourseCode() != "" {
		t.Errorf("sectionEnrollmentToProto: course_code = %q, want empty", proto.GetCourseCode())
	}
	if proto.GetPeriodYear() != 0 {
		t.Errorf("sectionEnrollmentToProto: period_year = %d, want 0", proto.GetPeriodYear())
	}
	if proto.GetPeriodTerm() != 0 {
		t.Errorf("sectionEnrollmentToProto: period_term = %d, want 0", proto.GetPeriodTerm())
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
