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
	getOwnProfileCalled bool

	upsertOwnProfileErr    error
	upsertOwnProfileResult profilesdb.UserProfile
	upsertOwnProfileCalled bool
	upsertOwnProfileParams profiles.UpsertOwnProfileParams

	upsertStudentProfileErr    error
	upsertStudentProfileCalled bool
	upsertStudentProfileParams profiles.UpsertStudentProfileParams

	upsertTeacherProfileErr    error
	upsertTeacherProfileCalled bool

	addTeacherQualificationErr    error
	addTeacherQualificationCalled bool

	listTeacherQualificationsResult []profilesdb.TeacherQualification
	listTeacherQualificationsErr    error
	listTeacherQualificationsCalled bool
	listTeacherQualificationsParams profiles.ListTeacherQualificationsRepoParams
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
	f.getOwnProfileCalled = true
	f.getOwnProfileID = callerID
	return f.getOwnProfileResult, f.getOwnProfileErr
}

func (f *fakeRepository) UpsertOwnProfile(_ context.Context, p profiles.UpsertOwnProfileParams) (profilesdb.UserProfile, error) {
	f.upsertOwnProfileCalled = true
	f.upsertOwnProfileParams = p
	return f.upsertOwnProfileResult, f.upsertOwnProfileErr
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

func (f *fakeRepository) ListTeacherQualifications(_ context.Context, p profiles.ListTeacherQualificationsRepoParams) ([]profilesdb.TeacherQualification, error) {
	f.listTeacherQualificationsCalled = true
	f.listTeacherQualificationsParams = p
	return f.listTeacherQualificationsResult, f.listTeacherQualificationsErr
}

func (f *fakeRepository) ListDisplayNamesByIDs(_ context.Context, _ []uuid.UUID) ([]profilesdb.ListDisplayNamesByIDsRow, error) {
	return nil, nil
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

// TestService_UpsertOwnProfile_TriStateMapping verifies that the service passes
// *string fields to the repository with the correct tri-state semantics:
//   - nil (absent) is passed as nil (COALESCE will preserve the existing column value)
//   - &"" (present-empty) is passed as &"" (COALESCE will set column to empty string)
//   - &"value" (present) is passed as &"value" (COALESCE will set column to the value)
func TestService_UpsertOwnProfile_TriStateMapping(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := auth.WithUserID(context.Background(), callerID)

	emptyStr := ""
	valueStr := "555-0100"

	cases := []struct {
		name           string
		phone          *string
		wantPhone      *string
		wantRepoCalled bool
	}{
		{
			name:           "nil (absent) — must remain nil through to repo",
			phone:          nil,
			wantPhone:      nil,
			wantRepoCalled: true,
		},
		{
			name:           "present-empty — must remain &\"\" through to repo (do not collapse to nil)",
			phone:          &emptyStr,
			wantPhone:      &emptyStr,
			wantRepoCalled: true,
		},
		{
			name:           "present-non-empty — must be passed as-is",
			phone:          &valueStr,
			wantPhone:      &valueStr,
			wantRepoCalled: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepository{}
			svc := profiles.NewService(repo)

			params := profiles.UpsertOwnProfileParams{
				Phone: tc.phone,
			}

			_, _ = svc.UpsertOwnProfile(ctx, params)

			if repo.upsertOwnProfileCalled != tc.wantRepoCalled {
				t.Errorf("repo.upsertOwnProfileCalled = %v, want %v", repo.upsertOwnProfileCalled, tc.wantRepoCalled)
			}
			if !tc.wantRepoCalled {
				return
			}

			gotPhone := repo.upsertOwnProfileParams.Phone
			if tc.wantPhone == nil {
				if gotPhone != nil {
					t.Errorf("phone: got non-nil %q, want nil (skip sentinel)", *gotPhone)
				}
			} else {
				if gotPhone == nil {
					t.Errorf("phone: got nil, want %q (must not collapse to nil)", *tc.wantPhone)
				} else if *gotPhone != *tc.wantPhone {
					t.Errorf("phone: got %q, want %q", *gotPhone, *tc.wantPhone)
				}
			}

			// UserID and UpdatedBy must be set to callerID by the service.
			if repo.upsertOwnProfileParams.UserID != callerID {
				t.Errorf("UserID = %v, want %v", repo.upsertOwnProfileParams.UserID, callerID)
			}
			if repo.upsertOwnProfileParams.UpdatedBy == nil || *repo.upsertOwnProfileParams.UpdatedBy != callerID {
				t.Errorf("UpdatedBy = %v, want %v", repo.upsertOwnProfileParams.UpdatedBy, callerID)
			}
		})
	}
}

// TestService_UpsertOwnProfile_NoContext verifies that missing context user returns ErrNotFound.
func TestService_UpsertOwnProfile_NoContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	_, err := svc.UpsertOwnProfile(context.Background(), profiles.UpsertOwnProfileParams{})
	if !errors.Is(err, profiles.ErrNotFound) {
		t.Errorf("UpsertOwnProfile without context user: got %v, want ErrNotFound", err)
	}
	if repo.upsertOwnProfileCalled {
		t.Error("repo was called despite no authenticated user in context")
	}
}

// TestService_UpsertOwnProfile_InvalidBirthDate verifies birth_date format validation.
func TestService_UpsertOwnProfile_InvalidBirthDate(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := auth.WithUserID(context.Background(), callerID)

	malformed := "31-13-1990"
	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	_, err := svc.UpsertOwnProfile(ctx, profiles.UpsertOwnProfileParams{
		BirthDate: &malformed,
	})
	if !errors.Is(err, profiles.ErrInvalidInput) {
		t.Errorf("UpsertOwnProfile malformed birth_date: got %v, want ErrInvalidInput", err)
	}
	if repo.upsertOwnProfileCalled {
		t.Error("repo was called despite malformed birth_date")
	}
}

// TestService_UpsertOwnProfile_EmptyBirthDateSkipsValidation verifies that present-empty
// birth_date clears the field without triggering format validation.
func TestService_UpsertOwnProfile_EmptyBirthDateSkipsValidation(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := auth.WithUserID(context.Background(), callerID)

	emptyDate := ""
	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	_, err := svc.UpsertOwnProfile(ctx, profiles.UpsertOwnProfileParams{
		BirthDate: &emptyDate,
	})
	// Should NOT return ErrInvalidInput (validation skipped for empty string).
	if errors.Is(err, profiles.ErrInvalidInput) {
		t.Errorf("UpsertOwnProfile empty birth_date: unexpected ErrInvalidInput (should skip validation)")
	}
	if !repo.upsertOwnProfileCalled {
		t.Error("repo was not called for empty birth_date (should be delegated)")
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
	// repo must NOT have been called (a bool sentinel, so a call with the zero
	// UUID is still detected — the zero id alone cannot distinguish not-called).
	if repo.getOwnProfileCalled {
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

// --- ListTeacherQualifications pagination unit tests ---

// TestService_ListTeacherQualifications_ClampMin verifies page_size=0 is clamped to 20.
func TestService_ListTeacherQualifications_ClampMin(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{listTeacherQualificationsResult: nil}
	svc := profiles.NewService(repo)

	_, err := svc.ListTeacherQualifications(context.Background(), uuid.New(), 0, "")
	if err != nil {
		t.Fatalf("ListTeacherQualifications(page_size=0): unexpected error: %v", err)
	}
	if !repo.listTeacherQualificationsCalled {
		t.Fatal("repo not called")
	}
	// RowLimit must be clamped min (20) + 1 = 21.
	if repo.listTeacherQualificationsParams.RowLimit != 21 {
		t.Errorf("RowLimit = %d, want 21 (clamped 20 + 1)", repo.listTeacherQualificationsParams.RowLimit)
	}
}

// TestService_ListTeacherQualifications_ClampMax verifies page_size=999 is clamped to 200.
func TestService_ListTeacherQualifications_ClampMax(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{listTeacherQualificationsResult: nil}
	svc := profiles.NewService(repo)

	_, err := svc.ListTeacherQualifications(context.Background(), uuid.New(), 999, "")
	if err != nil {
		t.Fatalf("ListTeacherQualifications(page_size=999): unexpected error: %v", err)
	}
	// RowLimit must be clamped max (200) + 1 = 201.
	if repo.listTeacherQualificationsParams.RowLimit != 201 {
		t.Errorf("RowLimit = %d, want 201 (clamped 200 + 1)", repo.listTeacherQualificationsParams.RowLimit)
	}
}

// TestService_ListTeacherQualifications_ValidToken verifies a valid UUID page_token is
// forwarded to the repo as a *uuid.UUID.
func TestService_ListTeacherQualifications_ValidToken(t *testing.T) {
	t.Parallel()

	tokenID := uuid.New()
	repo := &fakeRepository{listTeacherQualificationsResult: nil}
	svc := profiles.NewService(repo)

	_, err := svc.ListTeacherQualifications(context.Background(), uuid.New(), 20, tokenID.String())
	if err != nil {
		t.Fatalf("ListTeacherQualifications(valid token): unexpected error: %v", err)
	}
	if repo.listTeacherQualificationsParams.PageToken == nil {
		t.Fatal("PageToken should be non-nil for valid token")
	}
	if *repo.listTeacherQualificationsParams.PageToken != tokenID {
		t.Errorf("PageToken = %v, want %v", *repo.listTeacherQualificationsParams.PageToken, tokenID)
	}
}

// TestService_ListTeacherQualifications_EmptyToken verifies an empty page_token results
// in a nil PageToken (first page).
func TestService_ListTeacherQualifications_EmptyToken(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{listTeacherQualificationsResult: nil}
	svc := profiles.NewService(repo)

	_, err := svc.ListTeacherQualifications(context.Background(), uuid.New(), 20, "")
	if err != nil {
		t.Fatalf("ListTeacherQualifications(empty token): unexpected error: %v", err)
	}
	if repo.listTeacherQualificationsParams.PageToken != nil {
		t.Errorf("PageToken should be nil for empty token, got %v",
			*repo.listTeacherQualificationsParams.PageToken)
	}
}

// TestService_ListTeacherQualifications_InvalidToken verifies a malformed page_token
// returns ErrInvalidInput without calling the repo.
func TestService_ListTeacherQualifications_InvalidToken(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := profiles.NewService(repo)

	_, err := svc.ListTeacherQualifications(context.Background(), uuid.New(), 20, "not-a-uuid")
	if !errors.Is(err, profiles.ErrInvalidInput) {
		t.Errorf("malformed token: got %v, want ErrInvalidInput", err)
	}
	if repo.listTeacherQualificationsCalled {
		t.Error("repo was called despite invalid page_token")
	}
}

// TestService_ListTeacherQualifications_LastPage verifies that when the repo returns
// exactly clampedSize rows, HasNext is false and NextPageToken is empty.
func TestService_ListTeacherQualifications_LastPage(t *testing.T) {
	t.Parallel()

	// Return exactly 20 rows — no HasNext.
	rows := make([]profilesdb.TeacherQualification, 20)
	for i := range rows {
		rows[i].ID.Bytes = uuid.New()
		rows[i].ID.Valid = true
	}

	repo := &fakeRepository{listTeacherQualificationsResult: rows}
	svc := profiles.NewService(repo)

	result, err := svc.ListTeacherQualifications(context.Background(), uuid.New(), 20, "")
	if err != nil {
		t.Fatalf("ListTeacherQualifications(last page): unexpected error: %v", err)
	}
	if len(result.Qualifications) != 20 {
		t.Errorf("got %d qualifications, want 20", len(result.Qualifications))
	}
	if result.NextPageToken != "" {
		t.Errorf("NextPageToken = %q, want \"\" on last page", result.NextPageToken)
	}
}

// TestService_ListTeacherQualifications_HasNext verifies that when the repo returns
// clampedSize+1 rows, HasNext is true and NextPageToken is non-empty.
func TestService_ListTeacherQualifications_HasNext(t *testing.T) {
	t.Parallel()

	// Return 21 rows — triggers HasNext (clamped=20, fetched=21).
	rows := make([]profilesdb.TeacherQualification, 21)
	for i := range rows {
		rows[i].ID.Bytes = uuid.New()
		rows[i].ID.Valid = true
	}

	repo := &fakeRepository{listTeacherQualificationsResult: rows}
	svc := profiles.NewService(repo)

	result, err := svc.ListTeacherQualifications(context.Background(), uuid.New(), 20, "")
	if err != nil {
		t.Fatalf("ListTeacherQualifications(has-next): unexpected error: %v", err)
	}
	// Must return only 20 items (over-fetch trimmed).
	if len(result.Qualifications) != 20 {
		t.Errorf("got %d qualifications, want 20 (over-fetch trimmed)", len(result.Qualifications))
	}
	if result.NextPageToken == "" {
		t.Error("NextPageToken should be non-empty when HasNext is true")
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
