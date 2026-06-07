package profiles

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles/profilesdb"
)

// Service orchestrates profile business logic: validation, audit-column population, and repo delegation.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the given Repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// UpsertUserProfile validates the input, populates audit columns from context, and delegates to the repository.
func (s *Service) UpsertUserProfile(ctx context.Context, p UpsertUserProfileParams) (profilesdb.UserProfile, error) {
	if p.GivenNames == "" {
		return profilesdb.UserProfile{}, fmt.Errorf("%w: given_names is required", ErrInvalidInput)
	}
	if p.LastNamePaternal == "" {
		return profilesdb.UserProfile{}, fmt.Errorf("%w: last_name_paternal is required", ErrInvalidInput)
	}
	if p.NationalIDType == "" {
		return profilesdb.UserProfile{}, fmt.Errorf("%w: national_id_type is required", ErrInvalidInput)
	}
	if p.NationalID == "" {
		return profilesdb.UserProfile{}, fmt.Errorf("%w: national_id is required", ErrInvalidInput)
	}
	if p.BirthDate != nil && *p.BirthDate != "" {
		if _, err := time.Parse("2006-01-02", *p.BirthDate); err != nil {
			return profilesdb.UserProfile{}, fmt.Errorf("%w: birth_date must be in YYYY-MM-DD format", ErrInvalidInput)
		}
	}

	actor := actorFromContext(ctx)
	p.CreatedBy = actor
	p.UpdatedBy = actor

	return s.repo.UpsertUserProfile(ctx, p)
}

// GetUserProfile retrieves a user profile by user_id, returning ErrNotFound when absent.
func (s *Service) GetUserProfile(ctx context.Context, userID uuid.UUID) (profilesdb.UserProfile, error) {
	return s.repo.GetUserProfile(ctx, userID)
}

// GetOwnProfile retrieves the caller's own user profile using the user_id from context.
// The caller cannot supply a user_id — self-scope is enforced structurally.
// Returns ErrNotFound when no authenticated user is present in the context.
func (s *Service) GetOwnProfile(ctx context.Context) (profilesdb.UserProfile, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return profilesdb.UserProfile{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}
	return s.repo.GetOwnProfile(ctx, callerID)
}

// UpsertStudentProfile validates the input, populates audit columns, and delegates to the repository.
func (s *Service) UpsertStudentProfile(ctx context.Context, p UpsertStudentProfileParams) (profilesdb.StudentProfile, error) {
	if p.AdmissionYear <= 0 {
		return profilesdb.StudentProfile{}, fmt.Errorf("%w: admission_year must be greater than 0", ErrInvalidInput)
	}

	actor := actorFromContext(ctx)
	p.CreatedBy = actor
	p.UpdatedBy = actor

	return s.repo.UpsertStudentProfile(ctx, p)
}

// GetStudentProfile retrieves a student profile by user_id.
func (s *Service) GetStudentProfile(ctx context.Context, userID uuid.UUID) (profilesdb.StudentProfile, error) {
	return s.repo.GetStudentProfile(ctx, userID)
}

// UpsertTeacherProfile populates audit columns and delegates to the repository.
// No mandatory field validation beyond user_id presence (department and title are optional).
func (s *Service) UpsertTeacherProfile(ctx context.Context, p UpsertTeacherProfileParams) (profilesdb.TeacherProfile, error) {
	actor := actorFromContext(ctx)
	p.CreatedBy = actor
	p.UpdatedBy = actor

	return s.repo.UpsertTeacherProfile(ctx, p)
}

// GetTeacherProfile retrieves a teacher profile by user_id.
func (s *Service) GetTeacherProfile(ctx context.Context, userID uuid.UUID) (profilesdb.TeacherProfile, error) {
	return s.repo.GetTeacherProfile(ctx, userID)
}

// AddTeacherQualification validates the input, populates audit columns, and delegates to the repository.
func (s *Service) AddTeacherQualification(ctx context.Context, p AddTeacherQualificationParams) (profilesdb.TeacherQualification, error) {
	if p.Degree == "" {
		return profilesdb.TeacherQualification{}, fmt.Errorf("%w: degree is required", ErrInvalidInput)
	}
	if p.Year <= 0 {
		return profilesdb.TeacherQualification{}, fmt.Errorf("%w: year must be greater than 0", ErrInvalidInput)
	}

	actor := actorFromContext(ctx)
	p.CreatedBy = actor
	p.UpdatedBy = actor

	return s.repo.AddTeacherQualification(ctx, p)
}

// ListTeacherQualifications returns all non-deleted qualifications for the given teacher.
func (s *Service) ListTeacherQualifications(ctx context.Context, teacherID uuid.UUID) ([]profilesdb.TeacherQualification, error) {
	return s.repo.ListTeacherQualifications(ctx, teacherID)
}

// actorFromContext extracts the authenticated user_id from context and returns a pointer.
// Returns nil when no actor is present (e.g. system or background operations).
func actorFromContext(ctx context.Context) *uuid.UUID {
	id, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil
	}
	return &id
}
