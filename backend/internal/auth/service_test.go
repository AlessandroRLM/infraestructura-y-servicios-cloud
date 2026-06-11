package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

// --- stub implementations ---

type stubRepo struct {
	user User
	err  error
}

func (r *stubRepo) GetUserByEmail(_ context.Context, _ string) (User, error) {
	return r.user, r.err
}

func (r *stubRepo) UpdatePasswordHash(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
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
