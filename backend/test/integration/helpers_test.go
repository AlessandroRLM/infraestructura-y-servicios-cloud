package integration_test

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/crypto/bcrypt"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
)

// seedUser inserts a test user with a bcrypt cost-4 hash into the shared test database.
// Pass a unique email per test so rows do not collide between tests that run in the
// same database. If softDeleted is true, deleted_at is set to NOW() so the auth service
// treats the user as inactive.
// A Cleanup function is registered to remove the row after the test finishes.
func seedUser(t *testing.T, email, plainPassword string, softDeleted bool) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), 4)
	if err != nil {
		t.Fatalf("seedUser: bcrypt: %v", err)
	}

	ctx := context.Background()
	var sql string
	var args []any
	if softDeleted {
		sql = `INSERT INTO users (email, password_hash, deleted_at) VALUES ($1, $2, NOW())`
		args = []any{email, string(hash)}
	} else {
		sql = `INSERT INTO users (email, password_hash) VALUES ($1, $2)`
		args = []any{email, string(hash)}
	}
	if _, err := pgxPool.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("seedUser: insert %q: %v", email, err)
	}

	t.Cleanup(func() {
		if _, err := pgxPool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email); err != nil {
			t.Logf("seedUser cleanup: delete %q: %v", email, err)
		}
	})
}

// newCookieJar returns a minimal in-memory CookieJar for use in integration tests.
func newCookieJar() *testCookieJar {
	return &testCookieJar{}
}

// testCookieJar is a minimal http.CookieJar. It stores cookies by name and replays
// them on every request, which is sufficient for single-host auth test flows.
type testCookieJar struct {
	cookies []*http.Cookie
}

func (j *testCookieJar) SetCookies(_ *url.URL, cookies []*http.Cookie) {
	for _, c := range cookies {
		replaced := false
		for i, existing := range j.cookies {
			if existing.Name == c.Name {
				j.cookies[i] = c
				replaced = true
				break
			}
		}
		if !replaced {
			j.cookies = append(j.cookies, c)
		}
	}
}

func (j *testCookieJar) Cookies(_ *url.URL) []*http.Cookie {
	return j.cookies
}

// cookieValue returns the value of the first cookie with the given name, or "".
func (j *testCookieJar) cookieValue(name string) string {
	for _, c := range j.cookies {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// newAuthClient returns a Connect AuthService client targeting the shared test server.
// Pass a non-nil CookieJar to capture and replay cookies automatically across calls.
func newAuthClient(jar http.CookieJar) authv1connect.AuthServiceClient {
	return authv1connect.NewAuthServiceClient(&http.Client{Jar: jar}, baseURL)
}

// assertConnectCode asserts that err is a *connect.Error with the expected code.
func assertConnectCode(t *testing.T, err error, wantCode connect.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected Connect error with code %v, got nil", wantCode)
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != wantCode {
		t.Fatalf("Connect error code = %v, want %v (message: %s)", ce.Code(), wantCode, ce.Message())
	}
}

// captureTransport wraps an http.RoundTripper and calls onResponse for each
// completed HTTP response. Used to read raw response headers (e.g., Set-Cookie)
// that are not exposed through the Connect client API.
type captureTransport struct {
	inner      http.RoundTripper
	onResponse func(*http.Response)
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.inner.RoundTrip(req)
	if err == nil && t.onResponse != nil {
		t.onResponse(resp)
	}
	return resp, err
}

// loginCapturingSetCookie calls Login on targetURL and returns the raw Set-Cookie
// header from the underlying HTTP response. This allows tests to assert cookie
// attributes (Secure, HttpOnly, SameSite) that are not surfaced by the Connect layer.
func loginCapturingSetCookie(t *testing.T, targetURL, email, password string) (string, error) {
	t.Helper()
	var rawSetCookie string
	rt := &captureTransport{
		inner: http.DefaultTransport,
		onResponse: func(r *http.Response) {
			rawSetCookie = r.Header.Get("Set-Cookie")
		},
	}
	client := authv1connect.NewAuthServiceClient(&http.Client{Transport: rt}, targetURL)
	req := connect.NewRequest(&authv1.LoginRequest{Email: email, Password: password})
	_, err := client.Login(context.Background(), req)
	return rawSetCookie, err
}

// serveOnRandomPort starts srv on an OS-assigned TCP port, waits until the port
// accepts connections, and registers t.Cleanup to shut it down. Returns the base URL.
// Used by tests that need a dedicated server with custom configuration (e.g., cookie_secure).
func serveOnRandomPort(t *testing.T, srv *http.Server) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("serveOnRandomPort: listen: %v", err)
	}
	srv.Addr = ln.Addr().String()
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			t.Logf("serveOnRandomPort: %v", err)
		}
	}()
	waitForServer(srv.Addr, 3*time.Second)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Logf("serveOnRandomPort Shutdown: %v", err)
		}
	})
	return "http://" + srv.Addr
}
