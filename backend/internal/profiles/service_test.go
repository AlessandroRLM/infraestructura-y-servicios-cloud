package profiles_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles/profilesdb"
)

// fakeRepository is a test double for profiles.Repository.
type fakeRepository struct {
	upsertUserProfileErr    error
	upsertUserProfileCalled bool
	upsertUserProfileParams profiles.UpsertUserProfileParams

	getOwnProfileErr    error
	getOwnProfileResult profilesdb.UserProfile
	getOwnProfileID     uuid.UUID

	upsertStudentProfileErr    error
	upsertStudentProfileCalled bool
	upsertStudentProfileParams profiles.UpsertStudentProfileParams

	upsertTeacherProfileErr    error
	upsertTeacherProfileCalled bool

	addTeacherQualificationErr    error
	addTeacherQualificationCalled bool

	listTeacherQualificationsResult []profilesdb.TeacherQualification
	listTeacherQualificationsErr    error
}

func (f *fakeRepository) UpsertUserProfile(_ context.Context, p profiles.UpsertUserProfileParams) (profilesdb.UserProfile, error) {
	f.upsertUserProfileCalled = true
	f.upsertUserProfileParams = p
	return profilesdb.UserProfile{UserID: pgtype.UUID{Bytes: p.UserID, Valid: true}}, f.upsertUserProfileErr
}

func (f *fakeRepository) GetUserProfile(_ context.Context, _ uuid.UUID) (profilesdb.UserProfile, error) {
	return profilesdb.UserProfile{}, nil
}

func (f *fakeRepository) GetOwnProfile(_ context.Context, callerID uuid.UUID) (profilesdb.UserProfile, error) {
	f.getOwnProfileID = callerID
	return f.getOwnProfileResult, f.getOwnProfileErr
}

func (f *fakeRepository) UpsertStudentProfile(_ context.Context, p profiles.UpsertStudentProfileParams) (profilesdb.StudentProfile, error) {
	f.upsertStudentProfileCalled = true
	f.upsertStudentProfileParams = p
	return profilesdb.StudentProfile{}, f.upsertStudentProfileErr
}

func (f *fakeRepository) GetStudentProfile(_ context.Context, _ uuid.UUID) (profilesdb.StudentProfile, error) {
	return profilesdb.StudentProfile{}, nil
}

func (f *fakeRepository) UpsertTeacherProfile(_ context.Context, _ profiles.UpsertTeacherProfileParams) (profilesdb.TeacherProfile, error) {
	f.upsertTeacherProfileCalled = true
	return profilesdb.TeacherProfile{}, f.upsertTeacherProfileErr
}

func (f *fakeRepository) GetTeacherProfile(_ context.Context, _ uuid.UUID) (profilesdb.TeacherProfile, error) {
	return profilesdb.TeacherProfile{}, nil
}

func (f *fakeRepository) AddTeacherQualification(_ context.Context, _ profiles.AddTeacherQualificationParams) (profilesdb.TeacherQualification, error) {
	f.addTeacherQualificationCalled = true
	return profilesdb.TeacherQualification{}, f.addTeacherQualificationErr
}

func (f *fakeRepository) ListTeacherQualifications(_ context.Context, _ uuid.UUID) ([]profilesdb.TeacherQualification, error) {
	return f.listTeacherQualificationsResult, f.listTeacherQualificationsErr
}

// Compile-time check.
var _ profiles.Repository = (*fakeRepository)(nil)

// --- Validation tests ---

func TestService_UpsertUserProfile_Validation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params profiles.UpsertUserProfileParams
	}{
		{
			name:   "empty given_names",
			params: profiles.UpsertUserProfileParams{GivenNames: "", LastNamePaternal: "P", NationalIDType: "RUT", NationalID: "1"},
		},
		{
			name:   "empty last_name_paternal",
			params: profiles.UpsertUserProfileParams{GivenNames: "G", LastNamePaternal: "", NationalIDType: "RUT", NationalID: "1"},
		},
		{
			name:   "empty national_id_type",
			params: profiles.UpsertUserProfileParams{GivenNames: "G", LastNamePaternal: "P", NationalIDType: "", NationalID: "1"},
		},
		{
			name:   "empty national_id",
			params: profiles.UpsertUserProfileParams{GivenNames: "G", LastNamePaternal: "P", NationalIDType: "RUT", NationalID: ""},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := profiles.NewService(repo)

			_, err := svc.UpsertUserProfile(context.Background(), tc.params)

			if !errors.Is(err, profiles.ErrInvalidInput) {
				t.Errorf("UpsertUserProfile(%s): got %v, want ErrInvalidInput", tc.name, err)
			}
			if repo.upsertUserProfileCalled {
				t.Error("repo was called despite invalid input")
			}
		})
	}
}

func TestService_UpsertUserProfile_BirthDateValidation(t *testing.T) {
	t.Parallel()

	malformed := "31-13-1990" // wrong format
	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	params := profiles.UpsertUserProfileParams{
		GivenNames:       "Test",
		LastNamePaternal: "User",
		NationalIDType:   "RUT",
		NationalID:       "12345678-9",
		BirthDate:        &malformed,
	}

	_, err := svc.UpsertUserProfile(context.Background(), params)

	if !errors.Is(err, profiles.ErrInvalidInput) {
		t.Errorf("UpsertUserProfile (malformed birth_date): got %v, want ErrInvalidInput", err)
	}
	if repo.upsertUserProfileCalled {
		t.Error("repo was called despite malformed birth_date")
	}
}

func TestService_UpsertStudentProfile_Validation(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	params := profiles.UpsertStudentProfileParams{
		UserID:        uuid.New(),
		AdmissionYear: 0,
	}
	_, err := svc.UpsertStudentProfile(context.Background(), params)
	if !errors.Is(err, profiles.ErrInvalidInput) {
		t.Errorf("UpsertStudentProfile(year=0): got %v, want ErrInvalidInput", err)
	}
	if repo.upsertStudentProfileCalled {
		t.Error("repo was called despite invalid admission_year")
	}
}

func TestService_AddTeacherQualification_Validation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params profiles.AddTeacherQualificationParams
	}{
		{
			name:   "empty degree",
			params: profiles.AddTeacherQualificationParams{TeacherID: uuid.New(), Degree: "", Year: 2020},
		},
		{
			name:   "year zero",
			params: profiles.AddTeacherQualificationParams{TeacherID: uuid.New(), Degree: "MSc", Year: 0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := profiles.NewService(repo)

			_, err := svc.AddTeacherQualification(context.Background(), tc.params)
			if !errors.Is(err, profiles.ErrInvalidInput) {
				t.Errorf("AddTeacherQualification(%s): got %v, want ErrInvalidInput", tc.name, err)
			}
			if repo.addTeacherQualificationCalled {
				t.Error("repo was called despite invalid input")
			}
		})
	}
}

// --- Audit tests ---

func TestService_UpsertUserProfile_AuditFromContext(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)

	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	params := profiles.UpsertUserProfileParams{
		UserID:           uuid.New(),
		GivenNames:       "Test",
		LastNamePaternal: "User",
		NationalIDType:   "RUT",
		NationalID:       "12345678-9",
	}

	_, err := svc.UpsertUserProfile(ctx, params)
	if err != nil {
		t.Fatalf("UpsertUserProfile: unexpected error: %v", err)
	}

	got := repo.upsertUserProfileParams
	if got.CreatedBy == nil || *got.CreatedBy != actorID {
		t.Errorf("created_by = %v, want %v", got.CreatedBy, actorID)
	}
	if got.UpdatedBy == nil || *got.UpdatedBy != actorID {
		t.Errorf("updated_by = %v, want %v", got.UpdatedBy, actorID)
	}
}

// --- Self-scope tests ---

func TestService_GetOwnProfile_NoContextUser_ReturnsErrNotFound(t *testing.T) {
	t.Parallel()

	// Context has NO authenticated user — GetOwnProfile must fail closed without
	// ever calling the repository.
	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	_, err := svc.GetOwnProfile(context.Background())

	if !errors.Is(err, profiles.ErrNotFound) {
		t.Errorf("GetOwnProfile (no context user): got %v, want ErrNotFound", err)
	}
	// repo must NOT have been called.
	if repo.getOwnProfileID != (uuid.UUID{}) {
		t.Error("repo.GetOwnProfile was called despite missing context user — zero UUID would have been queried")
	}
}

func TestService_GetOwnProfile_UsesContextCallerID(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := auth.WithUserID(context.Background(), callerID)

	expectedProfile := profilesdb.UserProfile{
		UserID: pgtype.UUID{Bytes: callerID, Valid: true},
	}
	repo := &fakeRepository{getOwnProfileResult: expectedProfile}
	svc := profiles.NewService(repo)

	_, err := svc.GetOwnProfile(ctx)
	if err != nil {
		t.Fatalf("GetOwnProfile: unexpected error: %v", err)
	}

	if repo.getOwnProfileID != callerID {
		t.Errorf("GetOwnProfile queried user_id = %v, want %v", repo.getOwnProfileID, callerID)
	}
}

func TestService_GetOwnProfile_PropagatesNotFound(t *testing.T) {
	t.Parallel()

	ctx := auth.WithUserID(context.Background(), uuid.New())
	repo := &fakeRepository{getOwnProfileErr: profiles.ErrNotFound}
	svc := profiles.NewService(repo)

	_, err := svc.GetOwnProfile(ctx)
	if !errors.Is(err, profiles.ErrNotFound) {
		t.Errorf("GetOwnProfile: got %v, want ErrNotFound", err)
	}
}

// --- Decoupling assertion (no AssignRole method on fakeRepository) ---
// The fakeRepository type intentionally has no AssignRole method.
// If service methods call it, it will not compile. This test verifies the
// service contract does not touch role assignment.
func TestService_NoRoleAssignment(t *testing.T) {
	t.Parallel()

	ctx := auth.WithUserID(context.Background(), uuid.New())
	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	params := profiles.UpsertStudentProfileParams{
		UserID:        uuid.New(),
		AdmissionYear: 2024,
	}
	_, err := svc.UpsertStudentProfile(ctx, params)
	if err != nil {
		t.Fatalf("UpsertStudentProfile: unexpected error: %v", err)
	}
	// If we get here, no AssignRole was called (no such method exists on fakeRepository).
}
