package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

// TestMapError_InternalDoesNotLeakRawError verifies that unexpected errors produce
// CodeInternal with the generic "internal error" message, not the raw error string.
func TestMapError_InternalDoesNotLeakRawError(t *testing.T) {
	rawMsg := "pq: relation \"sessions\" does not exist"
	err := mapError(fmt.Errorf("auth: Login: %s", rawMsg))

	connectErr, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeInternal {
		t.Errorf("code = %v, want CodeInternal", connectErr.Code())
	}
	if strings.Contains(connectErr.Message(), rawMsg) {
		t.Errorf("client-visible message must not contain raw error; got: %q", connectErr.Message())
	}
	if connectErr.Message() != "internal error" {
		t.Errorf("message = %q, want %q", connectErr.Message(), "internal error")
	}
}

// TestMapError_UserNotFound_MapsToUnauthenticated verifies that ErrUserNotFound maps
// to CodeUnauthenticated (deleted-user mid-session forces re-login).
func TestMapError_UserNotFound_MapsToUnauthenticated(t *testing.T) {
	err := mapError(ErrUserNotFound)

	connectErr, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want CodeUnauthenticated", connectErr.Code())
	}
}

// buildGetSessionServer wires a real Service with stub deps through the Connect handler
// and returns the test server URL and a Connect client.
func buildGetSessionServer(
	t *testing.T,
	repo Repository,
	loader RoleLoader,
	ctxSeed func(context.Context) context.Context,
) (authv1connect.AuthServiceClient, string) {
	t.Helper()
	svc := NewService(repo, &stubStore{}, loader, config.Config{BcryptCost: 4})
	h := NewHandler(svc, config.Config{})

	path, handler := authv1connect.NewAuthServiceHandler(h)
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if ctxSeed != nil {
			r = r.WithContext(ctxSeed(r.Context()))
		}
		handler.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, srv.URL)
	return client, srv.URL
}

// TestHandlerGetSession_CacheControlHeader verifies the Cache-Control: no-store header
// is set on every GetSession response.
func TestHandlerGetSession_CacheControlHeader(t *testing.T) {
	userID := uuid.New()
	repo := &stubRepo{getUserByIDUser: User{ID: userID, Email: "a@b.com"}}
	loader := &fakeRoleLoader{loadRolesResult: []string{}}

	perms := authz.NewPermissionSet(nil)
	client, _ := buildGetSessionServer(t, repo, loader, func(ctx context.Context) context.Context {
		ctx = WithUserID(ctx, userID)
		return authz.WithPermissions(ctx, perms)
	})

	resp, err := client.GetSession(context.Background(), connect.NewRequest(&authv1.GetSessionRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := resp.Header().Get("Cache-Control")
	if got != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store")
	}
}

// TestHandlerGetSession_ProtoMapping verifies that SessionResult fields are mapped
// correctly onto the proto Session message.
func TestHandlerGetSession_ProtoMapping(t *testing.T) {
	userID := uuid.New()
	email := "test@example.com"
	repo := &stubRepo{getUserByIDUser: User{ID: userID, Email: email}}
	loader := &fakeRoleLoader{loadRolesResult: []string{"admin"}}

	perms := authz.NewPermissionSet([]authz.Permission{"users.manage"})
	client, _ := buildGetSessionServer(t, repo, loader, func(ctx context.Context) context.Context {
		ctx = WithUserID(ctx, userID)
		return authz.WithPermissions(ctx, perms)
	})

	resp, err := client.GetSession(context.Background(), connect.NewRequest(&authv1.GetSessionRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := resp.Msg
	if msg.GetUserId() != userID.String() {
		t.Errorf("user_id = %q, want %q", msg.GetUserId(), userID.String())
	}
	if msg.GetEmail() != email {
		t.Errorf("email = %q, want %q", msg.GetEmail(), email)
	}
	if len(msg.GetRoles()) != 1 || msg.GetRoles()[0] != "admin" {
		t.Errorf("roles = %v, want [admin]", msg.GetRoles())
	}
	if len(msg.GetPermissions()) != 1 || msg.GetPermissions()[0] != "users.manage" {
		t.Errorf("permissions = %v, want [users.manage]", msg.GetPermissions())
	}
}

// TestMapError_NonSentinelMapsToInternal verifies that a plain (non-sentinel) error —
// such as the one returned when user ID is missing from context — maps to CodeInternal.
func TestMapError_NonSentinelMapsToInternal(t *testing.T) {
	err := mapError(fmt.Errorf("auth: GetSession: user ID missing from context"))

	connectErr, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeInternal {
		t.Errorf("code = %v, want CodeInternal", connectErr.Code())
	}
}
