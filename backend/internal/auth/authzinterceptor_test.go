package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
)

// fakePolicy is a test double for authz.Policy that always returns a fixed decision.
type fakePolicy struct{ allowed bool }

func (f fakePolicy) Evaluate(_ context.Context, _ authz.Permission) authz.Decision {
	return authz.Decision{Allowed: f.allowed}
}

// buildAuthzServer wires a Connect server with the authz interceptor applied to the
// auth service handler. The procedure protected is authv1connect.AuthServiceLogoutProcedure.
func buildAuthzServer(t *testing.T, requiredPerms map[string]authz.Permission, policy authz.Policy) (string, *httptest.Server) {
	t.Helper()

	interceptor := auth.NewAuthzInterceptor(requiredPerms, policy)

	// Minimal auth service stub — we only need the handler to exist; the interceptor
	// runs before the handler and may reject the request without invoking it.
	handler := &noopAuthHandler{}
	path, h := authv1connect.NewAuthServiceHandler(handler,
		connect.WithInterceptors(interceptor),
	)

	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL, srv
}

// noopAuthHandler is a minimal stub that satisfies authv1connect.AuthServiceHandler.
// It returns success for every call so the interceptor under test controls outcomes.
type noopAuthHandler struct{}

func (h *noopAuthHandler) Login(_ context.Context, _ *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	return connect.NewResponse(&authv1.LoginResponse{}), nil
}

func (h *noopAuthHandler) Logout(_ context.Context, _ *connect.Request[authv1.LogoutRequest]) (*connect.Response[authv1.LogoutResponse], error) {
	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}

func (h *noopAuthHandler) RequestPasswordReset(_ context.Context, _ *connect.Request[authv1.RequestPasswordResetRequest]) (*connect.Response[authv1.RequestPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.RequestPasswordResetResponse{}), nil
}

func (h *noopAuthHandler) ConfirmPasswordReset(_ context.Context, _ *connect.Request[authv1.ConfirmPasswordResetRequest]) (*connect.Response[authv1.ConfirmPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.ConfirmPasswordResetResponse{}), nil
}

// withPermissionsTransport injects a PermissionSet into the outgoing request context
// by wrapping the transport. The authz interceptor reads it on the server side via
// authz.PermissionsFromContext — but since the interceptor runs on the server, and
// we want to test the server-side interceptor behavior, we use a server-side approach:
// the permissions are stored in context BEFORE the authz interceptor runs.
//
// For these unit tests we use a dedicated header to signal permissions, handled by a
// wrapper interceptor that pre-populates the context before the authz interceptor runs.
// This mirrors the production chain where the session interceptor runs first.
func makePermissionsInterceptor(set authz.PermissionSet) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ctx = authz.WithPermissions(ctx, set)
			return next(ctx, req)
		}
	}
}

func TestAuthzInterceptor_EmptyMapPassesThrough(t *testing.T) {
	t.Parallel()

	// Empty map: every procedure passes through regardless of permission set.
	url, _ := buildAuthzServer(t, map[string]authz.Permission{}, fakePolicy{allowed: false})
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, url)

	req := connect.NewRequest(&authv1.LogoutRequest{})
	_, err := client.Logout(context.Background(), req)
	if err != nil {
		t.Errorf("empty-map interceptor should pass through, got: %v", err)
	}
}

func TestAuthzInterceptor_MappedProcedureAllowed(t *testing.T) {
	t.Parallel()

	// Logout procedure is in the required map; fakePolicy always allows.
	permSet := authz.NewPermissionSet([]authz.Permission{authz.PermUsersManage})
	permsInterceptor := makePermissionsInterceptor(permSet)

	handler := &noopAuthHandler{}
	path, h := authv1connect.NewAuthServiceHandler(handler,
		connect.WithInterceptors(
			permsInterceptor,
			auth.NewAuthzInterceptor(
				map[string]authz.Permission{
					authv1connect.AuthServiceLogoutProcedure: authz.PermUsersManage,
				},
				fakePolicy{allowed: true},
			),
		),
	)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := authv1connect.NewAuthServiceClient(http.DefaultClient, srv.URL)
	_, err := client.Logout(context.Background(), connect.NewRequest(&authv1.LogoutRequest{}))
	if err != nil {
		t.Errorf("caller with required permission should be allowed, got: %v", err)
	}
}

func TestAuthzInterceptor_MappedProcedureDenied(t *testing.T) {
	t.Parallel()

	// Logout procedure is in the required map; fakePolicy always denies.
	permSet := authz.NewPermissionSet([]authz.Permission{authz.PermUsersManage})
	permsInterceptor := makePermissionsInterceptor(permSet)

	handler := &noopAuthHandler{}
	path, h := authv1connect.NewAuthServiceHandler(handler,
		connect.WithInterceptors(
			permsInterceptor,
			auth.NewAuthzInterceptor(
				map[string]authz.Permission{
					authv1connect.AuthServiceLogoutProcedure: authz.PermUsersManage,
				},
				fakePolicy{allowed: false},
			),
		),
	)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := authv1connect.NewAuthServiceClient(http.DefaultClient, srv.URL)
	_, err := client.Logout(context.Background(), connect.NewRequest(&authv1.LogoutRequest{}))
	if err == nil {
		t.Fatal("expected PermissionDenied error, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodePermissionDenied {
		t.Errorf("error code = %v, want PermissionDenied", ce.Code())
	}
}

func TestAuthzInterceptor_MappedProcedureNoPermissionsInContext(t *testing.T) {
	t.Parallel()

	// Logout is in the map; context has no PermissionSet; fakePolicy denies (empty set).
	handler := &noopAuthHandler{}
	path, h := authv1connect.NewAuthServiceHandler(handler,
		connect.WithInterceptors(
			auth.NewAuthzInterceptor(
				map[string]authz.Permission{
					authv1connect.AuthServiceLogoutProcedure: authz.PermUsersManage,
				},
				fakePolicy{allowed: false},
			),
		),
	)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := authv1connect.NewAuthServiceClient(http.DefaultClient, srv.URL)
	_, err := client.Logout(context.Background(), connect.NewRequest(&authv1.LogoutRequest{}))
	if err == nil {
		t.Fatal("expected PermissionDenied, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodePermissionDenied {
		t.Errorf("error code = %v, want PermissionDenied", ce.Code())
	}
}
