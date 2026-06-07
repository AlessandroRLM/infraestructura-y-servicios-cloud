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

// makePermissionsInterceptor injects a PermissionSet into the server-side context
// before the authz interceptor runs, mirroring the production session interceptor.
func makePermissionsInterceptor(set authz.PermissionSet) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ctx = authz.WithPermissions(ctx, set)
			return next(ctx, req)
		}
	}
}

// buildAuthzServer builds a test Connect server with the authz interceptor wired.
func buildAuthzServer(
	t *testing.T,
	exempt map[string]struct{},
	policies map[string]authz.PolicyFunc,
) (string, *httptest.Server) {
	t.Helper()

	interceptor := auth.NewAuthzInterceptor(exempt, policies)
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

// noopAuthHandler satisfies authv1connect.AuthServiceHandler, returning success for all calls.
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

// ── Exempt procedures ──────────────────────────────────────────────────────

func TestAuthzInterceptor_ExemptProcedure_PassesThrough(t *testing.T) {
	t.Parallel()

	// Login is in the exempt set; even without any permission, it must pass.
	exempt := map[string]struct{}{
		authv1connect.AuthServiceLoginProcedure: {},
	}
	url, _ := buildAuthzServer(t, exempt, map[string]authz.PolicyFunc{})
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, url)

	_, err := client.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{}))
	if err != nil {
		t.Errorf("exempt procedure should pass through, got: %v", err)
	}
}

// ── Mapped procedure — allow ───────────────────────────────────────────────

func TestAuthzInterceptor_MappedProcedure_Allow(t *testing.T) {
	t.Parallel()

	// Logout is mapped to RequirePermission(PermUsersManage).
	// The caller has that permission in context.
	permSet := authz.NewPermissionSet([]authz.Permission{authz.PermUsersManage})
	permsInterceptor := makePermissionsInterceptor(permSet)

	exempt := map[string]struct{}{}
	policies := map[string]authz.PolicyFunc{
		authv1connect.AuthServiceLogoutProcedure: authz.RequirePermission(authz.PermUsersManage),
	}

	interceptor := auth.NewAuthzInterceptor(exempt, policies)
	handler := &noopAuthHandler{}
	path, h := authv1connect.NewAuthServiceHandler(handler,
		connect.WithInterceptors(permsInterceptor, interceptor),
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

// ── Mapped procedure — deny ────────────────────────────────────────────────

func TestAuthzInterceptor_MappedProcedure_Deny(t *testing.T) {
	t.Parallel()

	// Logout is mapped to RequirePermission(PermUsersManage).
	// The caller only has PermCatalogManage → denied. Reason must surface.
	permSet := authz.NewPermissionSet([]authz.Permission{authz.PermCatalogManage})
	permsInterceptor := makePermissionsInterceptor(permSet)

	exempt := map[string]struct{}{}
	policies := map[string]authz.PolicyFunc{
		authv1connect.AuthServiceLogoutProcedure: authz.RequirePermission(authz.PermUsersManage),
	}

	interceptor := auth.NewAuthzInterceptor(exempt, policies)
	handler := &noopAuthHandler{}
	path, h := authv1connect.NewAuthServiceHandler(handler,
		connect.WithInterceptors(permsInterceptor, interceptor),
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
	if ce.Message() == "" {
		t.Error("denied response should carry the policy reason, got empty message")
	}
}

// ── Fail-closed: unmapped and non-exempt ──────────────────────────────────

func TestAuthzInterceptor_UnmappedAndNonExempt_DeniedFailClosed(t *testing.T) {
	t.Parallel()

	// Logout is neither in exempt nor in policies.
	// The interceptor must deny with CodePermissionDenied — this is the fail-closed guarantee.
	exempt := map[string]struct{}{
		authv1connect.AuthServiceLoginProcedure: {},
	}
	url, _ := buildAuthzServer(t, exempt, map[string]authz.PolicyFunc{})
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, url)

	_, err := client.Logout(context.Background(), connect.NewRequest(&authv1.LogoutRequest{}))
	if err == nil {
		t.Fatal("unmapped non-exempt procedure should be denied (fail-closed), got nil error")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodePermissionDenied {
		t.Errorf("error code = %v, want PermissionDenied (fail-closed)", ce.Code())
	}
}

// ── Mapped procedure — no permissions in context ──────────────────────────

func TestAuthzInterceptor_MappedProcedure_NoPermissionsInContext(t *testing.T) {
	t.Parallel()

	// Logout is mapped but the context has no PermissionSet (unauthenticated or misconfigured chain).
	exempt := map[string]struct{}{}
	policies := map[string]authz.PolicyFunc{
		authv1connect.AuthServiceLogoutProcedure: authz.RequirePermission(authz.PermUsersManage),
	}
	url, _ := buildAuthzServer(t, exempt, policies)
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, url)

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
