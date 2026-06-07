package integration_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/authdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
)

// buildAuthServer constructs a dedicated HTTP server wired with the auth service
// using the provided config. It uses the shared pgxPool and testRedisClient so it
// shares the same database and Redis instance as the other integration tests.
// The server is started on a random port by the caller via serveOnRandomPort.
func buildAuthServer(cfg config.Config) *http.Server {
	queries := authdb.New(pgxPool)
	repo := auth.NewPostgresRepository(queries)
	redisStore := session.NewRedisStore(testRedisClient)
	roleLoader := auth.NoopRoleLoader{}
	interceptor := auth.NewSessionInterceptor(redisStore, roleLoader, cfg)
	svc := auth.NewService(repo, redisStore, roleLoader, cfg)
	handler := auth.NewHandler(svc, cfg)
	opts := server.Chain(interceptor)

	mux := http.NewServeMux()
	auth.Register(mux, handler, opts...)

	return &http.Server{ //nolint:gosec // G112: inline test server; Slowloris not a concern here.
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// TestCookieSecure_TrueAddsSecureAttribute verifies (COOKIE_SECURE=true) that
// when the server is configured with CookieSecure=true, the Set-Cookie header on a
// successful Login response includes the Secure attribute.
func TestCookieSecure_TrueAddsSecureAttribute(t *testing.T) {
	const email = "cookie-secure-true@test.local"
	const password = "secure-test-password"
	seedUser(t, email, password, false)

	cfg := sharedCfg
	cfg.CookieSecure = true
	srv := buildAuthServer(cfg)
	targetURL := serveOnRandomPort(t, srv)

	setCookie, err := loginCapturingSetCookie(t, targetURL, email, password)
	if err != nil {
		t.Fatalf("Login on CookieSecure=true server: %v", err)
	}
	if setCookie == "" {
		t.Fatal("no Set-Cookie header returned")
	}

	// The Secure attribute must be present when COOKIE_SECURE=true.
	if !strings.Contains(setCookie, "Secure") {
		t.Errorf("Set-Cookie missing Secure attribute when COOKIE_SECURE=true: %q", setCookie)
	}
}

// TestCookieSecure_FalseOmitsSecureAttribute verifies (COOKIE_SECURE=false) that
// when the server is configured with CookieSecure=false, the Set-Cookie header on a
// successful Login response does NOT include the Secure attribute.
func TestCookieSecure_FalseOmitsSecureAttribute(t *testing.T) {
	const email = "cookie-secure-false@test.local"
	const password = "insecure-test-password"
	seedUser(t, email, password, false)

	cfg := sharedCfg
	cfg.CookieSecure = false
	srv := buildAuthServer(cfg)
	targetURL := serveOnRandomPort(t, srv)

	setCookie, err := loginCapturingSetCookie(t, targetURL, email, password)
	if err != nil {
		t.Fatalf("Login on CookieSecure=false server: %v", err)
	}
	if setCookie == "" {
		t.Fatal("no Set-Cookie header returned")
	}

	// The Secure attribute must NOT be present when COOKIE_SECURE=false.
	if strings.Contains(setCookie, "Secure") {
		t.Errorf("Set-Cookie contains Secure attribute when COOKIE_SECURE=false: %q", setCookie)
	}
}
