package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

// errInfra simulates an unexpected infrastructure error that must not reach the client.
var errInfra = errors.New("redis: ERR connection refused")

// failingStore returns errInfra on Touch, simulating a store unavailability.
type failingStore struct{}

func (s *failingStore) Create(_ context.Context, _ string, _ session.Session, _ time.Duration) error {
	return nil
}

func (s *failingStore) Touch(_ context.Context, _ string, _ time.Duration) (session.Session, error) {
	return session.Session{}, errInfra
}

func (s *failingStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *failingStore) SetReset(_ context.Context, _ string, _ uuid.UUID, _ time.Duration) error {
	return nil
}

func (s *failingStore) GetDelReset(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}

// failingLoader returns errInfra on Load, simulating an RBAC-store failure.
type failingLoader struct{}

func (l *failingLoader) Load(_ context.Context, _ uuid.UUID) (authz.PermissionSet, error) {
	return authz.PermissionSet{}, errInfra
}

// successStore returns a valid session so the loader is reached.
type successStore struct {
	userID uuid.UUID
}

func (s *successStore) Create(_ context.Context, _ string, _ session.Session, _ time.Duration) error {
	return nil
}

func (s *successStore) Touch(_ context.Context, _ string, _ time.Duration) (session.Session, error) {
	return session.Session{UserID: s.userID}, nil
}

func (s *successStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *successStore) SetReset(_ context.Context, _ string, _ uuid.UUID, _ time.Duration) error {
	return nil
}

func (s *successStore) GetDelReset(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}

// buildInterceptorServer wires a session interceptor with the given store and loader
// in front of a no-op Logout handler and returns its test URL.
func buildInterceptorServer(t *testing.T, store session.Store, loader RoleLoader) string {
	t.Helper()
	cfg := config.Config{SessionTTL: time.Hour}
	interceptor := NewSessionInterceptor(store, loader, cfg)

	path, h := authv1connect.NewAuthServiceHandler(
		&noopInterceptorHandler{},
		connect.WithInterceptors(interceptor),
	)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

// noopInterceptorHandler satisfies authv1connect.AuthServiceHandler with no-op methods.
type noopInterceptorHandler struct{}

func (h *noopInterceptorHandler) Login(_ context.Context, _ *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	return connect.NewResponse(&authv1.LoginResponse{}), nil
}
func (h *noopInterceptorHandler) Logout(_ context.Context, _ *connect.Request[authv1.LogoutRequest]) (*connect.Response[authv1.LogoutResponse], error) {
	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}
func (h *noopInterceptorHandler) RequestPasswordReset(_ context.Context, _ *connect.Request[authv1.RequestPasswordResetRequest]) (*connect.Response[authv1.RequestPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.RequestPasswordResetResponse{}), nil
}
func (h *noopInterceptorHandler) ConfirmPasswordReset(_ context.Context, _ *connect.Request[authv1.ConfirmPasswordResetRequest]) (*connect.Response[authv1.ConfirmPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.ConfirmPasswordResetResponse{}), nil
}

// TestSessionInterceptor_StoreTouchFailure_DoesNotLeakInternalError verifies that a
// store.Touch infrastructure failure returns CodeInternal with a generic message,
// not the raw infrastructure error string.
func TestSessionInterceptor_StoreTouchFailure_DoesNotLeakInternalError(t *testing.T) {
	baseURL := buildInterceptorServer(t, &failingStore{}, NoopRoleLoader{})
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, baseURL)

	req := connect.NewRequest(&authv1.LogoutRequest{})
	req.Header().Set("Cookie", "sid=any-session-id")

	_, err := client.Logout(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ce, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeInternal {
		t.Errorf("code = %v, want CodeInternal", ce.Code())
	}
	if strings.Contains(ce.Message(), errInfra.Error()) {
		t.Errorf("client-visible message must not contain raw error; got: %q", ce.Message())
	}
	if ce.Message() != "internal error" {
		t.Errorf("message = %q, want %q", ce.Message(), "internal error")
	}
}

// TestSessionInterceptor_LoaderFailure_DoesNotLeakInternalError verifies that a
// loader.Load infrastructure failure returns CodeInternal with a generic message.
func TestSessionInterceptor_LoaderFailure_DoesNotLeakInternalError(t *testing.T) {
	baseURL := buildInterceptorServer(t, &successStore{userID: uuid.New()}, &failingLoader{})
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, baseURL)

	req := connect.NewRequest(&authv1.LogoutRequest{})
	req.Header().Set("Cookie", "sid=any-session-id")

	_, err := client.Logout(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ce, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeInternal {
		t.Errorf("code = %v, want CodeInternal", ce.Code())
	}
	if strings.Contains(ce.Message(), errInfra.Error()) {
		t.Errorf("client-visible message must not contain raw error; got: %q", ce.Message())
	}
	if ce.Message() != "internal error" {
		t.Errorf("message = %q, want %q", ce.Message(), "internal error")
	}
}
