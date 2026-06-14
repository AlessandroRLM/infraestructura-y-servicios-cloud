package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/enrollment/enrollmentdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/pagination"
)

const (
	// enrollmentPageSizeMin is the minimum effective page size for enrollment list operations.
	enrollmentPageSizeMin = 20
	// enrollmentPageSizeMax is the maximum effective page size for enrollment list operations.
	enrollmentPageSizeMax = 200
)

// enrollmentClamp is the shared page-size clamp for enrollment list operations.
var enrollmentClamp = pagination.Clamp{Min: enrollmentPageSizeMin, Max: enrollmentPageSizeMax}

// ListEnrollmentsResult holds the paginated result for ListEnrollments.
type ListEnrollmentsResult struct {
	Enrollments   []enrollmentdb.Enrollment
	NextPageToken string
}

// ListEnrollmentsFilter holds optional filter parameters for ListEnrollments.
// A nil pointer means the filter is not applied.
type ListEnrollmentsFilter struct {
	StudentID *uuid.UUID
	ProgramID *uuid.UUID
	Year      *int32
	Status    *string
}

// Service orchestrates enrollment business logic: validation, audit-column population,
// self-scope enforcement, and delegation to the Repository.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the given Repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateEnrollment validates input and delegates the quota-checked create/revive to the
// repository. year must be positive; student_id and program_id must be valid UUIDs.
func (s *Service) CreateEnrollment(ctx context.Context, studentIDStr, programIDStr string, year int32) (enrollmentdb.Enrollment, error) {
	if year <= 0 {
		return enrollmentdb.Enrollment{}, fmt.Errorf("%w: year must be greater than 0", ErrInvalidInput)
	}

	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		return enrollmentdb.Enrollment{}, fmt.Errorf("%w: invalid student_id", ErrInvalidInput)
	}

	programID, err := uuid.Parse(programIDStr)
	if err != nil {
		return enrollmentdb.Enrollment{}, fmt.Errorf("%w: invalid program_id", ErrInvalidInput)
	}

	return s.repo.CreateEnrollmentTx(ctx, CreateEnrollmentParams{
		StudentID: studentID,
		ProgramID: programID,
		Year:      year,
	}, actorFromContext(ctx))
}

// MarkEnrollmentPaid validates the id and delegates the pending→paid transition.
func (s *Service) MarkEnrollmentPaid(ctx context.Context, idStr string) (enrollmentdb.Enrollment, error) {
	id, err := parseServiceUUID(idStr)
	if err != nil {
		return enrollmentdb.Enrollment{}, err
	}
	return s.repo.MarkEnrollmentPaid(ctx, id, actorFromContext(ctx))
}

// CancelEnrollment validates the id and delegates the pending|paid→cancelled transition.
func (s *Service) CancelEnrollment(ctx context.Context, idStr string) error {
	id, err := parseServiceUUID(idStr)
	if err != nil {
		return err
	}
	return s.repo.CancelEnrollment(ctx, id, actorFromContext(ctx))
}

// GetEnrollment retrieves a live enrollment by id.
func (s *Service) GetEnrollment(ctx context.Context, idStr string) (enrollmentdb.Enrollment, error) {
	id, err := parseServiceUUID(idStr)
	if err != nil {
		return enrollmentdb.Enrollment{}, err
	}
	return s.repo.GetEnrollment(ctx, id)
}

// ListEnrollments returns a paginated page of live enrollments ordered by id DESC.
// pageSize is clamped to [20, 200]. pageToken must be a valid UUID string or empty.
// Returns ErrInvalidInput when the token cannot be parsed as a UUID.
// All optional filters (student_id, program_id, year, status) are preserved alongside
// the keyset cursor.
func (s *Service) ListEnrollments(ctx context.Context, f ListEnrollmentsFilter, pageSize int32, pageToken string) (ListEnrollmentsResult, error) {
	clamped := enrollmentClamp.Apply(pageSize)

	var tokenUUID *uuid.UUID
	if pageToken != "" {
		id, err := uuid.Parse(pageToken)
		if err != nil {
			return ListEnrollmentsResult{}, fmt.Errorf("%w: page_token is not a valid UUID: %q", ErrInvalidInput, pageToken)
		}
		tokenUUID = &id
	}

	rows, err := s.repo.ListEnrollments(ctx, ListEnrollmentsRepoParams{
		PageToken: tokenUUID,
		StudentID: f.StudentID,
		ProgramID: f.ProgramID,
		Year:      f.Year,
		Status:    f.Status,
		RowLimit:  int32(clamped + 1),
	})
	if err != nil {
		return ListEnrollmentsResult{}, err
	}

	page := pagination.Paginate(rows, clamped)
	nextToken := pagination.TokenOf(page, func(r enrollmentdb.Enrollment) uuid.UUID {
		return uuid.UUID(r.ID.Bytes)
	})

	return ListEnrollmentsResult{
		Enrollments:   page.Items,
		NextPageToken: nextToken,
	}, nil
}

// ListOwnEnrollments returns a paginated page of live enrollments for the authenticated
// student. The student identity is injected from context; pageSize and pageToken follow
// the same AIP-158 keyset rules as ListEnrollments.
// Returns ErrNotFound when no authenticated user is present (fail-closed).
func (s *Service) ListOwnEnrollments(ctx context.Context, pageSize int32, pageToken string) (ListEnrollmentsResult, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return ListEnrollmentsResult{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	clamped := enrollmentClamp.Apply(pageSize)

	var tokenUUID *uuid.UUID
	if pageToken != "" {
		id, err := uuid.Parse(pageToken)
		if err != nil {
			return ListEnrollmentsResult{}, fmt.Errorf("%w: page_token is not a valid UUID: %q", ErrInvalidInput, pageToken)
		}
		tokenUUID = &id
	}

	rows, err := s.repo.ListOwnEnrollments(ctx, ListOwnEnrollmentsRepoParams{
		StudentID: userID,
		PageToken: tokenUUID,
		RowLimit:  int32(clamped + 1),
	})
	if err != nil {
		return ListEnrollmentsResult{}, err
	}

	page := pagination.Paginate(rows, clamped)
	nextToken := pagination.TokenOf(page, func(r enrollmentdb.Enrollment) uuid.UUID {
		return uuid.UUID(r.ID.Bytes)
	})

	return ListEnrollmentsResult{
		Enrollments:   page.Items,
		NextPageToken: nextToken,
	}, nil
}

// GetOwnEnrollment fetches the enrollment by id and verifies that it belongs to the
// calling user. An ownership mismatch returns ErrNotFound — existence is never disclosed.
func (s *Service) GetOwnEnrollment(ctx context.Context, idStr string) (enrollmentdb.Enrollment, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return enrollmentdb.Enrollment{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	id, err := parseServiceUUID(idStr)
	if err != nil {
		return enrollmentdb.Enrollment{}, err
	}

	row, err := s.repo.GetEnrollment(ctx, id)
	if err != nil {
		return enrollmentdb.Enrollment{}, err
	}

	// Ownership check: student_id must equal the calling user's id.
	if row.StudentID.Bytes != userID {
		// Return ErrNotFound — never leak existence to a caller who does not own the row.
		return enrollmentdb.Enrollment{}, fmt.Errorf("%w", ErrNotFound)
	}

	return row, nil
}

// --- Helpers ---

// actorFromContext extracts the authenticated user_id from context and returns a pointer.
// Returns nil when no actor is present (system or background operations).
func actorFromContext(ctx context.Context) *uuid.UUID {
	id, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil
	}
	return &id
}

// parseServiceUUID parses a string UUID and returns ErrInvalidInput on failure.
// Used by service methods to validate incoming id strings before any DB call.
func parseServiceUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%w: invalid id %q", ErrInvalidInput, s)
	}
	return id, nil
}
