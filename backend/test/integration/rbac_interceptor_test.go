package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac/rbacdb"
)

// seedUserWithRole inserts a user into the database and assigns the given role by name.
// Returns the user UUID. Registers cleanup to remove both the user_roles row and the user.
func seedUserWithRole(t *testing.T, email, roleName string) uuid.UUID {
	t.Helper()

	hash := "$2a$04$placeholder.hash.for.testing.only.not.a.real.password"
	var userID uuid.UUID
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, hash,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seedUserWithRole: insert user: %v", err)
	}

	_, err = pgxPool.Exec(context.Background(), `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, r.id FROM roles r WHERE r.name = $2
	`, userID, roleName)
	if err != nil {
		t.Fatalf("seedUserWithRole: assign role %q: %v", roleName, err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pgxPool.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID)
		_, _ = pgxPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	return userID
}

// seedUserNoRole inserts a user with no role assignment. Returns the user UUID.
func seedUserNoRole(t *testing.T, email string) uuid.UUID {
	t.Helper()

	hash := "$2a$04$placeholder.hash.for.testing.only.not.a.real.password"
	var userID uuid.UUID
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, hash,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("seedUserNoRole: insert user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	return userID
}

// TestRBACInterceptor_PermissionsStoredInContext verifies that after the session
// interceptor authenticates a request, the permissions are retrievable from context.
func TestRBACInterceptor_PermissionsStoredInContext(t *testing.T) {
	ctx := context.Background()

	// Admin user gets all 11 permissions.
	adminID := seedUserWithRole(t, "rbac-admin@test.local", "admin")
	// User with no role gets empty set.
	noRoleID := seedUserNoRole(t, "rbac-norole@test.local")

	loader := rbac.NewPostgresRoleLoader(rbacdb.New(pgxPool))

	t.Run("admin_user_has_users_manage", func(t *testing.T) {
		perms, err := loader.Load(ctx, adminID)
		if err != nil {
			t.Fatalf("Load admin: %v", err)
		}
		if !perms.Has(authz.PermUsersManage) {
			t.Error("admin PermissionSet should contain users.manage")
		}
		if len(perms) != 11 {
			t.Errorf("admin PermissionSet length = %d, want 11", len(perms))
		}
	})

	t.Run("no_role_user_has_empty_set", func(t *testing.T) {
		perms, err := loader.Load(ctx, noRoleID)
		if err != nil {
			t.Fatalf("Load no-role user: %v", err)
		}
		if len(perms) != 0 {
			t.Errorf("no-role PermissionSet length = %d, want 0", len(perms))
		}
		if perms.Has(authz.PermUsersManage) {
			t.Error("no-role user should not have users.manage")
		}
	})
}

// TestRBACInterceptor_SessionInterceptorStoresPermissions verifies that the session
// interceptor stores the loaded PermissionSet in context so subsequent interceptors
// and handlers can read it.
func TestRBACInterceptor_SessionInterceptorStoresPermissions(t *testing.T) {
	// Seed an admin user and create a session for them.
	adminID := seedUserWithRole(t, "rbac-session-admin@test.local", "admin")

	// Build a dedicated test server that uses the real PostgresRoleLoader.
	loader := rbac.NewPostgresRoleLoader(rbacdb.New(pgxPool))
	testCfg := config.Config{
		BcryptCost:   4,
		SessionTTL:   time.Hour,
		AppEnv:       "test",
		CookieSecure: false,
	}
	redisStore := session.NewRedisStore(testRedisClient)
	sessionInterceptor := auth.NewSessionInterceptor(redisStore, loader, testCfg)

	// A capture interceptor runs AFTER sessionInterceptor and records the PermissionSet.
	var capturedSet authz.PermissionSet
	captureInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			capturedSet, _ = authz.PermissionsFromContext(ctx)
			return next(ctx, req)
		}
	})

	// Wire a minimal handler.
	handler := &noopTestAuthHandler{}
	path, h := authv1connect.NewAuthServiceHandler(handler,
		connect.WithInterceptors(sessionInterceptor, captureInterceptor),
	)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Seed a session for the admin user.
	sid := seedSessionInRedis(t, adminID, time.Hour)

	req := connect.NewRequest(&authv1.LogoutRequest{})
	req.Header().Set("Cookie", "sid="+sid)

	client := authv1connect.NewAuthServiceClient(http.DefaultClient, srv.URL)
	_, err := client.Logout(context.Background(), req)
	if err != nil {
		t.Fatalf("Logout with valid session: %v", err)
	}

	if capturedSet == nil {
		t.Fatal("PermissionsFromContext returned nil set — session interceptor did not store permissions")
	}
	if !capturedSet.Has(authz.PermUsersManage) {
		t.Error("captured PermissionSet for admin should contain users.manage")
	}
}

// TestRBACInterceptor_AuthzChain verifies the full session+authz interceptor chain:
// allow, deny, no-session, and unmapped procedure scenarios.
func TestRBACInterceptor_AuthzChain(t *testing.T) {
	adminID := seedUserWithRole(t, "rbac-chain-admin@test.local", "admin")
	studentID := seedUserWithRole(t, "rbac-chain-student@test.local", "student")

	loader := rbac.NewPostgresRoleLoader(rbacdb.New(pgxPool))
	testCfg := config.Config{
		BcryptCost:   4,
		SessionTTL:   time.Hour,
		AppEnv:       "test",
		CookieSecure: false,
	}
	redisStore := session.NewRedisStore(testRedisClient)
	sessionInterceptor := auth.NewSessionInterceptor(redisStore, loader, testCfg)

	// Protect Logout with users.manage to create a synthetic admin-only route.
	authzInterceptor := auth.NewAuthzInterceptor(
		map[string]authz.Permission{
			authv1connect.AuthServiceLogoutProcedure: authz.PermUsersManage,
		},
		authz.PermissionPolicy{},
	)

	handler := &noopTestAuthHandler{}
	path, h := authv1connect.NewAuthServiceHandler(handler,
		connect.WithInterceptors(sessionInterceptor, authzInterceptor),
	)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := authv1connect.NewAuthServiceClient(http.DefaultClient, srv.URL)

	t.Run("admin_user_is_allowed", func(t *testing.T) {
		sid := seedSessionInRedis(t, adminID, time.Hour)
		req := connect.NewRequest(&authv1.LogoutRequest{})
		req.Header().Set("Cookie", "sid="+sid)
		_, err := client.Logout(context.Background(), req)
		if err != nil {
			t.Errorf("admin should be allowed, got: %v", err)
		}
	})

	t.Run("student_user_is_denied", func(t *testing.T) {
		sid := seedSessionInRedis(t, studentID, time.Hour)
		req := connect.NewRequest(&authv1.LogoutRequest{})
		req.Header().Set("Cookie", "sid="+sid)
		_, err := client.Logout(context.Background(), req)
		assertConnectCode(t, err, connect.CodePermissionDenied)
	})

	t.Run("no_session_returns_unauthenticated", func(t *testing.T) {
		req := connect.NewRequest(&authv1.LogoutRequest{})
		// no Cookie header
		_, err := client.Logout(context.Background(), req)
		assertConnectCode(t, err, connect.CodeUnauthenticated)
	})

	t.Run("unmapped_procedure_passes_through", func(t *testing.T) {
		// Login is public and not in the required map — the authz interceptor must pass
		// the call through to the handler. The test handler always returns nil, so a
		// non-nil error here means the interceptor blocked the call (or wiring is broken).
		req := connect.NewRequest(&authv1.LoginRequest{Email: "x@x.com", Password: "pw"})
		_, err := client.Login(context.Background(), req)
		if err != nil {
			t.Errorf("unmapped procedure Login should reach the handler and succeed, got: %v", err)
		}
	})
}

// noopTestAuthHandler is a minimal stub for integration test servers.
type noopTestAuthHandler struct{}

func (h *noopTestAuthHandler) Login(_ context.Context, _ *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	return connect.NewResponse(&authv1.LoginResponse{}), nil
}

func (h *noopTestAuthHandler) Logout(_ context.Context, _ *connect.Request[authv1.LogoutRequest]) (*connect.Response[authv1.LogoutResponse], error) {
	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}

func (h *noopTestAuthHandler) RequestPasswordReset(_ context.Context, _ *connect.Request[authv1.RequestPasswordResetRequest]) (*connect.Response[authv1.RequestPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.RequestPasswordResetResponse{}), nil
}

func (h *noopTestAuthHandler) ConfirmPasswordReset(_ context.Context, _ *connect.Request[authv1.ConfirmPasswordResetRequest]) (*connect.Response[authv1.ConfirmPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.ConfirmPasswordResetResponse{}), nil
}
