package sectionenrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/pagination"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment/sectionenrollmentdb"
)

const (
	// sectionEnrollmentPageSizeMin is the minimum effective page size for list operations.
	sectionEnrollmentPageSizeMin = 20
	// sectionEnrollmentPageSizeMax is the maximum effective page size for list operations.
	sectionEnrollmentPageSizeMax = 200
)

// sectionEnrollmentClamp is the shared page-size clamp for section_enrollment list operations.
var sectionEnrollmentClamp = pagination.Clamp{Min: sectionEnrollmentPageSizeMin, Max: sectionEnrollmentPageSizeMax}

// ListSectionEnrollmentsResult holds the paginated result for ListSectionEnrollments and
// ListOwnSectionEnrollments.
type ListSectionEnrollmentsResult struct {
	SectionEnrollments []sectionenrollmentdb.SectionEnrollment
	NextPageToken      string
}

// Service orchestrates section_enrollment business logic: UUID validation, self-scope
// enforcement, and delegation to the Repository.
//
// passed/failed transitions are owned by the grades slice via SetSectionEnrollmentOutcome;
// do not write status directly from this service.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the given Repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// EnrollOwnSection creates a section inscription for the authenticated student.
// Student identity is derived exclusively from the context; no student_id in the request.
// Window-gated (isAdmin=false). Returns ErrNotFound when no user is in context.
// programIDStr must be a valid UUID identifying which paid enrollment to link —
// this disambiguates students enrolled in multiple programs sharing the same course.
func (s *Service) EnrollOwnSection(ctx context.Context, sectionIDStr, programIDStr string) (sectionenrollmentdb.SectionEnrollment, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return sectionenrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	sectionID, err := parseServiceUUID(sectionIDStr)
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}

	programID, err := parseServiceUUID(programIDStr)
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}

	return s.repo.EnrollSectionTx(ctx, EnrollSectionParams{
		SectionID: sectionID,
		StudentID: callerID,
		ProgramID: programID,
	}, false)
}

// ListOwnSectionEnrollments returns a paginated page of live inscriptions for the authenticated
// student. Student identity is derived exclusively from the context.
// pageSize is clamped to [20, 200]. pageToken must be a valid UUID string or empty.
// Returns ErrNotFound when no authenticated user is present (fail-closed).
func (s *Service) ListOwnSectionEnrollments(ctx context.Context, pageSize int32, pageToken string) (ListSectionEnrollmentsResult, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return ListSectionEnrollmentsResult{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	clamped := sectionEnrollmentClamp.Apply(pageSize)

	var tokenUUID *uuid.UUID
	if pageToken != "" {
		id, err := uuid.Parse(pageToken)
		if err != nil {
			return ListSectionEnrollmentsResult{}, fmt.Errorf("%w: page_token is not a valid UUID: %q", ErrInvalidInput, pageToken)
		}
		tokenUUID = &id
	}

	rows, err := s.repo.ListOwnSectionEnrollments(ctx, ListOwnSectionEnrollmentsRepoParams{
		StudentID: callerID,
		PageToken: tokenUUID,
		RowLimit:  int32(clamped + 1),
	})
	if err != nil {
		return ListSectionEnrollmentsResult{}, err
	}

	page := pagination.Paginate(rows, clamped)
	nextToken := pagination.TokenOf(page, func(r sectionenrollmentdb.SectionEnrollment) uuid.UUID {
		return uuid.UUID(r.ID.Bytes)
	})

	return ListSectionEnrollmentsResult{
		SectionEnrollments: page.Items,
		NextPageToken:      nextToken,
	}, nil
}

// GetOwnSectionEnrollment fetches an inscription by id and verifies ownership.
// Ownership is checked by confirming the caller's user_id appears in the inscription's
// enrollment. A mismatch returns ErrNotFound — existence is never disclosed.
func (s *Service) GetOwnSectionEnrollment(ctx context.Context, idStr string) (sectionenrollmentdb.SectionEnrollment, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return sectionenrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	id, err := parseServiceUUID(idStr)
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}

	row, err := s.repo.GetOwnSectionEnrollment(ctx, id)
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}

	// Ownership check: fetch the first page of the caller's own inscriptions and verify
	// the fetched row appears in the caller's scope. Using the max page size to avoid
	// false negatives on students with many inscriptions; the repo-level call avoids
	// inflating the service result type.
	ownRows, err := s.repo.ListOwnSectionEnrollments(ctx, ListOwnSectionEnrollmentsRepoParams{
		StudentID: callerID,
		RowLimit:  int32(sectionEnrollmentPageSizeMax + 1),
	})
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}
	for _, r := range ownRows {
		if r.ID == row.ID {
			return row, nil
		}
	}
	// The inscription exists but does not belong to the caller.
	return sectionenrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: inscription does not belong to caller", ErrNotFound)
}

// EnrollSection creates or revives a section inscription for any student (admin path).
// Not window-gated. Can revive a withdrawn inscription (isAdmin=true).
func (s *Service) EnrollSection(ctx context.Context, enrollmentIDStr, sectionIDStr string) (sectionenrollmentdb.SectionEnrollment, error) {
	enrollmentID, err := parseServiceUUID(enrollmentIDStr)
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}
	sectionID, err := parseServiceUUID(sectionIDStr)
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}

	return s.repo.EnrollSectionTx(ctx, EnrollSectionParams{
		SectionID:    sectionID,
		EnrollmentID: enrollmentID,
	}, true)
}

// WithdrawSection transitions an in_progress inscription to withdrawn (admin-only).
func (s *Service) WithdrawSection(ctx context.Context, idStr string) (sectionenrollmentdb.SectionEnrollment, error) {
	id, err := parseServiceUUID(idStr)
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}
	return s.repo.WithdrawSection(ctx, id)
}

// GetSectionEnrollment retrieves a live inscription by id (admin path).
func (s *Service) GetSectionEnrollment(ctx context.Context, idStr string) (sectionenrollmentdb.SectionEnrollment, error) {
	id, err := parseServiceUUID(idStr)
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, err
	}
	return s.repo.GetSectionEnrollment(ctx, id)
}

// ListSectionEnrollments returns a paginated page of live inscriptions with optional filters
// (admin path). pageSize is clamped to [20, 200]. pageToken must be a valid UUID string or empty.
// Returns ErrInvalidInput when the token cannot be parsed as a UUID.
// All optional filters (section_id, enrollment_id, status) are preserved alongside the keyset cursor.
func (s *Service) ListSectionEnrollments(ctx context.Context, f ListSectionEnrollmentsFilter, pageSize int32, pageToken string) (ListSectionEnrollmentsResult, error) {
	clamped := sectionEnrollmentClamp.Apply(pageSize)

	var tokenUUID *uuid.UUID
	if pageToken != "" {
		id, err := uuid.Parse(pageToken)
		if err != nil {
			return ListSectionEnrollmentsResult{}, fmt.Errorf("%w: page_token is not a valid UUID: %q", ErrInvalidInput, pageToken)
		}
		tokenUUID = &id
	}

	rows, err := s.repo.ListSectionEnrollments(ctx, ListSectionEnrollmentsRepoParams{
		PageToken:    tokenUUID,
		SectionID:    f.SectionID,
		EnrollmentID: f.EnrollmentID,
		Status:       f.Status,
		RowLimit:     int32(clamped + 1),
	})
	if err != nil {
		return ListSectionEnrollmentsResult{}, err
	}

	page := pagination.Paginate(rows, clamped)
	nextToken := pagination.TokenOf(page, func(r sectionenrollmentdb.SectionEnrollment) uuid.UUID {
		return uuid.UUID(r.ID.Bytes)
	})

	return ListSectionEnrollmentsResult{
		SectionEnrollments: page.Items,
		NextPageToken:      nextToken,
	}, nil
}

// SetSectionEnrollmentOutcomeTx transitions a section_enrollment to passed or failed
// within a caller-owned transaction, and persists the rounded final grade alongside.
// Delegates to the repository so the grades slice can compose this within RecordGradeTx.
// Accepts only "passed" or "failed" as outcome. withdrawn source → ErrInvalidTransition.
func (s *Service) SetSectionEnrollmentOutcomeTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, outcome string, finalGrade pgtype.Numeric) (sectionenrollmentdb.SectionEnrollment, error) {
	return s.repo.SetSectionEnrollmentOutcomeTx(ctx, tx, id, outcome, finalGrade)
}

// --- Helpers ---

// parseServiceUUID parses a string UUID and returns ErrInvalidInput on failure.
func parseServiceUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%w: invalid id %q", ErrInvalidInput, s)
	}
	return id, nil
}
