package profiles

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/pagination"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles/profilesdb"
)

// profilesClamp enforces the global page-size bounds for all profiles lists.
var profilesClamp = pagination.Clamp{Min: 20, Max: 200}

// ListTeacherQualificationsResult carries the paginated result for ListTeacherQualifications.
type ListTeacherQualificationsResult struct {
	Qualifications []profilesdb.TeacherQualification
	NextPageToken  string
}

// DisplayNameEntry carries the resolved display-name fields for a single user.
type DisplayNameEntry struct {
	UserID           string
	GivenNames       string
	LastNamePaternal string
}

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

// UpsertOwnProfile applies PATCH-semantics edits to the caller's own profile.
// Identity is derived from the session context; no user_id field is accepted.
// Returns ErrNotFound when no authenticated user is in context or no profile row exists.
func (s *Service) UpsertOwnProfile(ctx context.Context, p UpsertOwnProfileParams) (profilesdb.UserProfile, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return profilesdb.UserProfile{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	// Validate birth_date only when present and non-empty.
	if p.BirthDate != nil && *p.BirthDate != "" {
		if _, err := time.Parse("2006-01-02", *p.BirthDate); err != nil {
			return profilesdb.UserProfile{}, fmt.Errorf("%w: birth_date must be in YYYY-MM-DD format", ErrInvalidInput)
		}
	}

	p.UserID = callerID
	p.UpdatedBy = &callerID

	return s.repo.UpsertOwnProfile(ctx, p)
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

// ListTeacherQualifications returns a keyset-paginated page of non-deleted qualifications
// for the given teacher. pageSize is clamped to [20, 200]. An empty pageToken fetches
// the first page; a non-empty token must be a valid UUID — malformed tokens return
// ErrInvalidInput.
func (s *Service) ListTeacherQualifications(
	ctx context.Context,
	teacherID uuid.UUID,
	pageSize int32,
	pageToken string,
) (ListTeacherQualificationsResult, error) {
	clamped := profilesClamp.Apply(pageSize)

	var tok *uuid.UUID
	if pageToken != "" {
		parsed, err := uuid.Parse(pageToken)
		if err != nil {
			return ListTeacherQualificationsResult{}, fmt.Errorf("%w: invalid page_token", ErrInvalidInput)
		}
		tok = &parsed
	}

	rows, err := s.repo.ListTeacherQualifications(ctx, ListTeacherQualificationsRepoParams{
		TeacherID: &teacherID,
		PageToken: tok,
		RowLimit:  int32(clamped + 1),
	})
	if err != nil {
		return ListTeacherQualificationsResult{}, err
	}

	page := pagination.Paginate(rows, clamped)
	nextToken := pagination.TokenOf(page, func(r profilesdb.TeacherQualification) uuid.UUID {
		return uuid.UUID(r.ID.Bytes)
	})

	return ListTeacherQualificationsResult{
		Qualifications: page.Items,
		NextPageToken:  nextToken,
	}, nil
}

// ListDisplayNamesByIDs resolves display names for the provided user IDs.
// Empty input returns an empty result without error (S-21).
// Unknown or soft-deleted user IDs are omitted from the result (S-20).
// No cross-context scope check is performed — see ADR-4 trust boundary.
//
// Malformed UUIDs (e.g. "not-a-uuid") cause the entire call to fail with
// ErrInvalidInput (→ CodeInvalidArgument). This is asymmetric with the
// omit-unknown-valid-id behavior: a syntactically invalid ID is a caller
// error; a valid UUID with no matching profile row is silently omitted.
func (s *Service) ListDisplayNamesByIDs(ctx context.Context, userIDStrs []string) ([]DisplayNameEntry, error) {
	if len(userIDStrs) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(userIDStrs))
	for _, raw := range userIDStrs {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid user_id %q", ErrInvalidInput, raw)
		}
		ids = append(ids, parsed)
	}

	rows, err := s.repo.ListDisplayNamesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	entries := make([]DisplayNameEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, DisplayNameEntry{
			UserID:           uuid.UUID(r.UserID.Bytes).String(),
			GivenNames:       r.GivenNames,
			LastNamePaternal: r.LastNamePaternal,
		})
	}
	return entries, nil
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
