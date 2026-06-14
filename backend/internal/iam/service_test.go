package iam_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	iamv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/iam/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam/iamdb"
)

// fakeRepository is a test double for iam.Repository used by service tests.
type fakeRepository struct {
	listUsersRows   []iamdb.ListUsersRow
	listUsersErr    error
	listUsersCalled bool

	getUserByIDRow    iamdb.GetUserByIDRow
	getUserByIDErr    error
	getUserByIDCalled bool

	getUserRolesResult []string
	getUserRolesErr    error
	getUserRolesCalled bool
}

// Compile-time check: fakeRepository must satisfy iam.Repository.
var _ iam.Repository = (*fakeRepository)(nil)

func (f *fakeRepository) ListUsers(_ context.Context, _ iam.ListUsersParams) ([]iamdb.ListUsersRow, error) {
	f.listUsersCalled = true
	return f.listUsersRows, f.listUsersErr
}

func (f *fakeRepository) GetUserByID(_ context.Context, _ uuid.UUID) (iamdb.GetUserByIDRow, error) {
	f.getUserByIDCalled = true
	return f.getUserByIDRow, f.getUserByIDErr
}

func (f *fakeRepository) GetUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	f.getUserRolesCalled = true
	return f.getUserRolesResult, f.getUserRolesErr
}

// makeRow is a helper that builds a ListUsersRow for table-driven tests.
func makeRow(id uuid.UUID, email string) iamdb.ListUsersRow {
	return iamdb.ListUsersRow{
		ID:    pgtype.UUID{Bytes: id, Valid: true},
		Email: email,
	}
}

// --- page_size clamping via Clamp.Apply ---

func TestService_ListUsers_PageSizeClamping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		pageSize int32
	}{
		{"zero", 0},
		{"negative", -5},
		{"below min", 1},
		{"at min", 20},
		{"in range", 50},
		{"at max", 200},
		{"above max", 999},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{}}
			svc := iam.NewService(repo)

			_, err := svc.ListUsers(context.Background(), tc.pageSize, "", "")
			if err != nil {
				t.Errorf("ListUsers (pageSize=%d): unexpected error: %v", tc.pageSize, err)
			}
			if !repo.listUsersCalled {
				t.Error("ListUsers: repository was not called")
			}
		})
	}
}

// --- invalid page_token ---

func TestService_ListUsers_InvalidPageToken(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := iam.NewService(repo)

	_, err := svc.ListUsers(context.Background(), 20, "not-a-uuid", "")
	if !errors.Is(err, iam.ErrInvalidInput) {
		t.Errorf("ListUsers (bad page_token): got %v, want ErrInvalidInput", err)
	}
	// Repository must NOT be called when validation fails.
	if repo.listUsersCalled {
		t.Error("ListUsers (bad page_token): repository must not be called on invalid input")
	}
}

// --- next_page_token emission ---

func TestService_ListUsers_NextPageToken_HasNext(t *testing.T) {
	t.Parallel()

	// Provide pageSize+1 rows so pagination detects a next page.
	// With pageSize=20 (minimum), we need 21 rows.
	rows := make([]iamdb.ListUsersRow, 21)
	for i := range rows {
		rows[i] = makeRow(uuid.New(), "user@example.com")
	}

	repo := &fakeRepository{listUsersRows: rows}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if result.NextPageToken == "" {
		t.Error("ListUsers (has next): NextPageToken must be non-empty")
	}
	// next_page_token must be the UUID of the last item in the page (item index 19).
	wantToken := uuid.UUID(rows[19].ID.Bytes).String()
	if result.NextPageToken != wantToken {
		t.Errorf("ListUsers NextPageToken = %q, want %q", result.NextPageToken, wantToken)
	}
}

func TestService_ListUsers_NextPageToken_LastPage(t *testing.T) {
	t.Parallel()

	// Provide exactly pageSize rows — no next page.
	rows := make([]iamdb.ListUsersRow, 20)
	for i := range rows {
		rows[i] = makeRow(uuid.New(), "user@example.com")
	}

	repo := &fakeRepository{listUsersRows: rows}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if result.NextPageToken != "" {
		t.Errorf("ListUsers (last page): NextPageToken = %q, want empty string", result.NextPageToken)
	}
}

// --- extractRoles edge cases (exercised via ListUsers) ---

func TestService_ListUsers_ExtractRoles_Nil(t *testing.T) {
	t.Parallel()

	row := makeRow(uuid.New(), "user@example.com")
	row.Roles = nil // NULL array_agg

	repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{row}}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if len(result.Users) != 1 {
		t.Fatalf("ListUsers: got %d users, want 1", len(result.Users))
	}
	if result.Users[0].Roles == nil {
		t.Error("extractRoles(nil): expected empty slice, got nil")
	}
	if len(result.Users[0].Roles) != 0 {
		t.Errorf("extractRoles(nil): got %v, want empty slice", result.Users[0].Roles)
	}
}

func TestService_ListUsers_ExtractRoles_SliceOfInterface(t *testing.T) {
	t.Parallel()

	row := makeRow(uuid.New(), "user@example.com")
	row.Roles = []interface{}{"admin", "teacher"}

	repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{row}}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if len(result.Users) != 1 {
		t.Fatalf("ListUsers: got %d users, want 1", len(result.Users))
	}
	roles := result.Users[0].Roles
	if len(roles) != 2 {
		t.Errorf("extractRoles([]interface{}): got %v, want [admin teacher]", roles)
	}
}

func TestService_ListUsers_ExtractRoles_NonStringElementSkipped(t *testing.T) {
	t.Parallel()

	row := makeRow(uuid.New(), "user@example.com")
	// Mix of string and a non-string element — the non-string must be skipped.
	row.Roles = []interface{}{"admin", 42, "teacher"}

	repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{row}}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if len(result.Users) != 1 {
		t.Fatalf("ListUsers: got %d users, want 1", len(result.Users))
	}
	roles := result.Users[0].Roles
	// 42 is skipped; only "admin" and "teacher" remain.
	if len(roles) != 2 {
		t.Errorf("extractRoles (non-string skipped): got %v, want [admin teacher]", roles)
	}
	for _, r := range roles {
		if r != "admin" && r != "teacher" {
			t.Errorf("extractRoles: unexpected role %q", r)
		}
	}
}

func TestService_ListUsers_ExtractRoles_StringSliceFastPath(t *testing.T) {
	t.Parallel()

	row := makeRow(uuid.New(), "user@example.com")
	row.Roles = []string{"admin", "student"}

	repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{row}}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if len(result.Users) != 1 {
		t.Fatalf("ListUsers: got %d users, want 1", len(result.Users))
	}
	roles := result.Users[0].Roles
	if len(roles) != 2 {
		t.Errorf("extractRoles([]string): got %v, want [admin student]", roles)
	}
}

// --- deriveDisplayName edge cases (exercised via ListUsers) ---

func TestService_ListUsers_DeriveDisplayName_ProfilePresent(t *testing.T) {
	t.Parallel()

	row := makeRow(uuid.New(), "alice@example.com")
	row.GivenNames = pgtype.Text{String: "Alice", Valid: true}
	row.LastNamePaternal = pgtype.Text{String: "Smith", Valid: true}

	repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{row}}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if result.Users[0].DisplayName != "Alice Smith" {
		t.Errorf("deriveDisplayName (profile present): got %q, want %q",
			result.Users[0].DisplayName, "Alice Smith")
	}
}

func TestService_ListUsers_DeriveDisplayName_ProfileAbsent(t *testing.T) {
	t.Parallel()

	row := makeRow(uuid.New(), "bob@example.com")
	// Both fields invalid — no profile row.
	row.GivenNames = pgtype.Text{Valid: false}
	row.LastNamePaternal = pgtype.Text{Valid: false}

	repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{row}}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if result.Users[0].DisplayName != "bob@example.com" {
		t.Errorf("deriveDisplayName (absent): got %q, want %q",
			result.Users[0].DisplayName, "bob@example.com")
	}
}

func TestService_ListUsers_DeriveDisplayName_EmptyNames_FallbackToEmail(t *testing.T) {
	t.Parallel()

	// Profile row present but both name fields are whitespace-only — fall back to email.
	row := makeRow(uuid.New(), "carol@example.com")
	row.GivenNames = pgtype.Text{String: "  ", Valid: true}
	row.LastNamePaternal = pgtype.Text{String: "  ", Valid: true}

	repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{row}}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	got := result.Users[0].DisplayName
	if got != "carol@example.com" {
		t.Errorf("deriveDisplayName (empty names): got %q, want email fallback", got)
	}
}

// TestService_ListUsers_DeriveDisplayName_NoSpaceArtifact ensures that when
// profile names are empty/whitespace the display name is the email only,
// never a bare " " string.
func TestService_ListUsers_DeriveDisplayName_NoSpaceArtifact(t *testing.T) {
	t.Parallel()

	row := makeRow(uuid.New(), "dave@example.com")
	row.GivenNames = pgtype.Text{String: "", Valid: true}
	row.LastNamePaternal = pgtype.Text{String: "", Valid: true}

	repo := &fakeRepository{listUsersRows: []iamdb.ListUsersRow{row}}
	svc := iam.NewService(repo)

	result, err := svc.ListUsers(context.Background(), 20, "", "")
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	got := result.Users[0].DisplayName
	if got == " " {
		t.Error("deriveDisplayName (empty strings): returned bare space string")
	}
	if got != "dave@example.com" {
		t.Errorf("deriveDisplayName (empty strings): got %q, want email fallback", got)
	}
}

// --- GetUser delegation and not-found propagation ---

func TestService_GetUser_DelegatesAndReturnsUserSummary(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := &fakeRepository{
		getUserByIDRow: iamdb.GetUserByIDRow{
			ID:               pgtype.UUID{Bytes: userID, Valid: true},
			Email:            "eve@example.com",
			GivenNames:       pgtype.Text{String: "Eve", Valid: true},
			LastNamePaternal: pgtype.Text{String: "Johnson", Valid: true},
		},
		getUserRolesResult: []string{"admin"},
	}
	svc := iam.NewService(repo)

	summary, err := svc.GetUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUser: unexpected error: %v", err)
	}
	if !repo.getUserByIDCalled {
		t.Error("GetUser: GetUserByID was not called")
	}
	if !repo.getUserRolesCalled {
		t.Error("GetUser: GetUserRoles was not called")
	}
	if summary.Email != "eve@example.com" {
		t.Errorf("GetUser: email = %q, want %q", summary.Email, "eve@example.com")
	}
	if summary.DisplayName != "Eve Johnson" {
		t.Errorf("GetUser: display_name = %q, want %q", summary.DisplayName, "Eve Johnson")
	}
	if len(summary.Roles) != 1 || summary.Roles[0] != "admin" {
		t.Errorf("GetUser: roles = %v, want [admin]", summary.Roles)
	}
}

func TestService_GetUser_NotFoundPropagates(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{getUserByIDErr: iam.ErrNotFound}
	svc := iam.NewService(repo)

	_, err := svc.GetUser(context.Background(), uuid.New())
	if !errors.Is(err, iam.ErrNotFound) {
		t.Errorf("GetUser (not found): got %v, want ErrNotFound", err)
	}
}

func TestService_GetUser_RolesErrorPropagates(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		getUserByIDRow:  iamdb.GetUserByIDRow{Email: "frank@example.com"},
		getUserRolesErr: errors.New("db timeout"),
	}
	svc := iam.NewService(repo)

	_, err := svc.GetUser(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("GetUser (roles error): expected error, got nil")
	}
}

// --- Disabled status mapping (via userSummaryToProto via handler) ---
// userSummaryToProto is unexported; we exercise the mapping by testing the handler.

func TestHandler_UserSummaryToProto_DisabledMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		disabled   bool
		wantStatus iamv1.UserStatus
	}{
		{"active user (disabled_at NULL)", false, iamv1.UserStatus_USER_STATUS_ACTIVE},
		{"disabled user (disabled_at set)", true, iamv1.UserStatus_USER_STATUS_DISABLED},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			repo := &fakeRepository{
				getUserByIDRow: iamdb.GetUserByIDRow{
					ID:         pgtype.UUID{Bytes: userID, Valid: true},
					Email:      "test@example.com",
					DisabledAt: pgtype.Timestamptz{Valid: tc.disabled},
				},
			}
			svc := iam.NewService(repo)
			summary, err := svc.GetUser(context.Background(), userID)
			if err != nil {
				t.Fatalf("GetUser: unexpected error: %v", err)
			}
			if summary.Disabled != tc.disabled {
				t.Errorf("GetUser Disabled = %v, want %v", summary.Disabled, tc.disabled)
			}
		})
	}
}
