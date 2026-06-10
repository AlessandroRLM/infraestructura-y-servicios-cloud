package section_enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/section_enrollment/section_enrollmentdb"
)

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
func (s *Service) EnrollOwnSection(ctx context.Context, sectionIDStr, programIDStr string) (section_enrollmentdb.SectionEnrollment, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return section_enrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	sectionID, err := parseServiceUUID(sectionIDStr)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}

	programID, err := parseServiceUUID(programIDStr)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}

	return s.repo.EnrollSectionTx(ctx, EnrollSectionParams{
		SectionID: sectionID,
		StudentID: callerID,
		ProgramID: programID,
	}, false)
}

// ListOwnSectionEnrollments returns all live inscriptions for the authenticated student.
// Student identity is derived exclusively from the context.
func (s *Service) ListOwnSectionEnrollments(ctx context.Context) ([]section_enrollmentdb.SectionEnrollment, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}
	return s.repo.ListOwnSectionEnrollments(ctx, callerID)
}

// GetOwnSectionEnrollment fetches an inscription by id and verifies ownership.
// Ownership is checked by confirming the caller's user_id appears in the inscription's
// enrollment. A mismatch returns ErrNotFound — existence is never disclosed.
func (s *Service) GetOwnSectionEnrollment(ctx context.Context, idStr string) (section_enrollmentdb.SectionEnrollment, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return section_enrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	id, err := parseServiceUUID(idStr)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}

	row, err := s.repo.GetOwnSectionEnrollment(ctx, id)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}

	// Ownership check: verify the inscription is in the caller's own-scoped list.
	// Using ListOwnSectionEnrollments to avoid exposing existence to non-owners.
	ownRows, err := s.repo.ListOwnSectionEnrollments(ctx, callerID)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}
	for _, r := range ownRows {
		if r.ID == row.ID {
			return row, nil
		}
	}
	// The inscription exists but does not belong to the caller.
	return section_enrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: inscription does not belong to caller", ErrNotFound)
}

// EnrollSection creates or revives a section inscription for any student (admin path).
// Not window-gated. Can revive a withdrawn inscription (isAdmin=true).
func (s *Service) EnrollSection(ctx context.Context, enrollmentIDStr, sectionIDStr string) (section_enrollmentdb.SectionEnrollment, error) {
	enrollmentID, err := parseServiceUUID(enrollmentIDStr)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}
	sectionID, err := parseServiceUUID(sectionIDStr)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}

	return s.repo.EnrollSectionTx(ctx, EnrollSectionParams{
		SectionID:    sectionID,
		EnrollmentID: enrollmentID,
	}, true)
}

// WithdrawSection transitions an in_progress inscription to withdrawn (admin-only).
func (s *Service) WithdrawSection(ctx context.Context, idStr string) (section_enrollmentdb.SectionEnrollment, error) {
	id, err := parseServiceUUID(idStr)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}
	return s.repo.WithdrawSection(ctx, id)
}

// GetSectionEnrollment retrieves a live inscription by id (admin path).
func (s *Service) GetSectionEnrollment(ctx context.Context, idStr string) (section_enrollmentdb.SectionEnrollment, error) {
	id, err := parseServiceUUID(idStr)
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, err
	}
	return s.repo.GetSectionEnrollment(ctx, id)
}

// ListSectionEnrollments returns live inscriptions with optional filters (admin path).
func (s *Service) ListSectionEnrollments(ctx context.Context, f ListSectionEnrollmentsFilter) ([]section_enrollmentdb.SectionEnrollment, error) {
	return s.repo.ListSectionEnrollments(ctx, f)
}

// SetSectionEnrollmentOutcomeTx transitions a section_enrollment to passed or failed
// within a caller-owned transaction, and persists the rounded final grade alongside.
// Delegates to the repository so the grades slice can compose this within RecordGradeTx.
// Accepts only "passed" or "failed" as outcome. withdrawn source → ErrInvalidTransition.
func (s *Service) SetSectionEnrollmentOutcomeTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, outcome string, finalGrade pgtype.Numeric) (section_enrollmentdb.SectionEnrollment, error) {
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
