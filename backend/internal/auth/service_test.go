package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

// --- stub implementations ---

type stubRepo struct {
	user             User
	err              error
	getUserByIDUser  User
	getUserByIDErr   error
	getUserByIDCalled bool
}

func (r *stubRepo) GetUserByEmail(_ context.Context, _ string) (User, error) {
	return r.user, r.err
}

func (r *stubRepo) UpdatePasswordHash(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (r *stubRepo) GetUserByID(_ context.Context, _ uuid.UUID) (User, error) {
	r.getUserByIDCalled = true
	return r.getUserByIDUser, r.getUserByIDErr
}

// fakeRoleLoader is a test double for RoleLoader with explicit called sentinels.
type fakeRoleLoader struct {
	loadResult      authz.PermissionSet
	loadErr         error
	loadRolesResult []string
	loadRolesErr    error
	loadRolesCalled bool
}

func (f *fakeRoleLoader) Load(_ context.Context, _ uuid.UUID) (authz.PermissionSet, error) {
	return f.loadResult, f.loadErr
}

func (f *fakeRoleLoader) LoadRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	f.loadRolesCalled = true
	return f.loadRolesResult, f.loadRolesErr
}

type stubStore struct {
	createErr    error
	getDelErr    error
	getDelUserID uuid.UUID
}

func (s *stubStore) Create(_ context.Context, _ string, _ session.Session, _ time.Duration) error {
	return s.createErr
}

func (s *stubStore) Touch(_ context.Context, _ string, _ time.Duration) (session.Session, error) {
	return session.Session{}, nil
}

func (s *stubStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *stubStore) SetReset(_ context.Context, _ string, _ uuid.UUID, _ time.Duration) error {
	return nil
}

func (s *stubStore) GetDelReset(_ context.Context, _ string) (uuid.UUID, error) {
	return s.getDelUserID, s.getDelErr
}

// --- helpers ---

func testConfig() config.Config {
	return config.Config{
		BcryptCost:    4, // bcrypt.MinCost — fast for tests
		SessionTTL:    time.Hour,
		ResetTokenTTL: 15 * time.Minute,
		AppEnv:        "test",
	}
}

// TestConfirmPasswordReset_InvalidToken verifies that an invalid/expired/consumed
// reset token returns ErrInvalidToken (not ErrInvalidCredentials).
func TestConfirmPasswordReset_InvalidToken(t *testing.T) {
	store := &stubStore{getDelErr: session.ErrNotFound}
	svc := NewService(&stubRepo{}, store, NoopRoleLoader{}, testConfig())

	err := svc.ConfirmPasswordReset(context.Background(), "bad-token", "newpass123")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
	// Must NOT be ErrInvalidCredentials (Login sentinel must stay separate).
	if errors.Is(err, ErrInvalidCredentials) {
		t.Error("must not return ErrInvalidCredentials for reset-token-not-found")
	}
}

// TestMapError_InvalidToken verifies ErrInvalidToken maps to CodeInvalidArgument.
func TestMapError_InvalidToken(t *testing.T) {
	err := mapError(ErrInvalidToken)
	connectErr, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connectErr.Code())
	}
}

// TestMapError_InvalidCredentials verifies Login errors still map to CodeUnauthenticated.
func TestMapError_InvalidCredentials(t *testing.T) {
	err := mapError(ErrInvalidCredentials)
	connectErr, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeUnauthenticated {
		t.Errorf("expected CodeUnauthenticated, got %v", connectErr.Code())
	}
}

// TestLogin_UserNotFound_TimingGuard verifies that the dummy hash compare uses the
// configured bcrypt cost so that NewService fails fast if precomputation is absent.
// The actual timing equality is a property of the production cost; at cost 4 the
// call completes in under 100ms and we simply verify it doesn't panic or error out
// in a way that leaks enumeration information.
func TestLogin_UserNotFound_TimingGuard(t *testing.T) {
	repo := &stubRepo{err: ErrUserNotFound}
	svc := NewService(repo, &stubStore{}, NoopRoleLoader{}, testConfig())

	_, err := svc.Login(context.Background(), "missing@example.com", "anypassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for missing user, got %v", err)
	}
}

func seedCtxWithUser(userID uuid.UUID) context.Context {
	return WithUserID(context.Background(), userID)
}

func seedCtxWithUserAndPerms(userID uuid.UUID, perms authz.PermissionSet) context.Context {
	ctx := WithUserID(context.Background(), userID)
	return authz.WithPermissions(ctx, perms)
}

// TestGetSession_HappyPath verifies the full happy-path: repo returns a User,
// loader returns roles, context carries a non-empty PermissionSet; result is
// fully populated with sorted roles and permissions.
func TestGetSession_HappyPath(t *testing.T) {
	userID := uuid.New()
	perms := authz.NewPermissionSet([]authz.Permission{"users.manage", "catalog.manage"})
	ctx := seedCtxWithUserAndPerms(userID, perms)

	repo := &stubRepo{getUserByIDUser: User{ID: userID, Email: "test@example.com"}}
	loader := &fakeRoleLoader{loadRolesResult: []string{"editor", "admin"}}
	svc := NewService(repo, &stubStore{}, loader, testConfig())

	result, err := svc.GetSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.getUserByIDCalled {
		t.Error("expected GetUserByID to be called")
	}
	if !loader.loadRolesCalled {
		t.Error("expected LoadRoles to be called")
	}
	if result.UserID != userID.String() {
		t.Errorf("UserID = %q, want %q", result.UserID, userID.String())
	}
	if result.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", result.Email, "test@example.com")
	}
	wantRoles := []string{"admin", "editor"}
	if len(result.Roles) != len(wantRoles) {
		t.Fatalf("Roles len = %d, want %d", len(result.Roles), len(wantRoles))
	}
	for i, r := range result.Roles {
		if r != wantRoles[i] {
			t.Errorf("Roles[%d] = %q, want %q", i, r, wantRoles[i])
		}
	}
	wantPerms := []string{"catalog.manage", "users.manage"}
	if len(result.Permissions) != len(wantPerms) {
		t.Fatalf("Permissions len = %d, want %d", len(result.Permissions), len(wantPerms))
	}
	for i, p := range result.Permissions {
		if p != wantPerms[i] {
			t.Errorf("Permissions[%d] = %q, want %q", i, p, wantPerms[i])
		}
	}
}

// TestGetSession_UserNotFound verifies that when GetUserByID returns ErrUserNotFound
// the service propagates it so the handler can map it to CodeUnauthenticated.
func TestGetSession_UserNotFound(t *testing.T) {
	userID := uuid.New()
	ctx := seedCtxWithUserAndPerms(userID, authz.PermissionSet{})

	repo := &stubRepo{getUserByIDErr: ErrUserNotFound}
	loader := &fakeRoleLoader{}
	svc := NewService(repo, &stubStore{}, loader, testConfig())

	_, err := svc.GetSession(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// TestGetSession_RoleLess verifies that a user with no roles and no permissions
// receives empty (non-nil) slices and no error.
func TestGetSession_RoleLess(t *testing.T) {
	userID := uuid.New()
	ctx := seedCtxWithUserAndPerms(userID, authz.NewPermissionSet(nil))

	repo := &stubRepo{getUserByIDUser: User{ID: userID, Email: "noadmin@example.com"}}
	loader := &fakeRoleLoader{loadRolesResult: []string{}}
	svc := NewService(repo, &stubStore{}, loader, testConfig())

	result, err := svc.GetSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Roles == nil {
		t.Error("Roles must not be nil for role-less user")
	}
	if len(result.Roles) != 0 {
		t.Errorf("Roles = %v, want []", result.Roles)
	}
	if result.Permissions == nil {
		t.Error("Permissions must not be nil for role-less user")
	}
	if len(result.Permissions) != 0 {
		t.Errorf("Permissions = %v, want []", result.Permissions)
	}
}

// TestGetSession_MissingUserIDInContext verifies that when the context does not carry
// a user ID the service returns a non-nil error (defensive invariant violation).
func TestGetSession_MissingUserIDInContext(t *testing.T) {
	ctx := context.Background()
	svc := NewService(&stubRepo{}, &stubStore{}, &fakeRoleLoader{}, testConfig())

	_, err := svc.GetSession(ctx)
	if err == nil {
		t.Fatal("expected error when userID missing from context, got nil")
	}
}
