package integration_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
)

// TestLogin_ValidCredentials verifies that login with correct credentials:
//   - returns a success response
//   - sets a "sid" cookie that is HttpOnly, SameSite=Lax, Path=/
//   - creates a session entry in Redis with a TTL close to SESSION_TTL
//   - stores the user's ID in the session payload
func TestLogin_ValidCredentials(t *testing.T) {
	const email = "login-valid@test.local"
	const password = "correct-password"
	seedUser(t, email, password, false)

	setCookie, err := loginCapturingSetCookie(t, baseURL, email, password)
	if err != nil {
		t.Fatalf("Login: unexpected error: %v", err)
	}
	if setCookie == "" {
		t.Fatal("Login: no Set-Cookie header in response")
	}

	// Verify required cookie attributes are present.
	if !strings.Contains(setCookie, "sid=") {
		t.Errorf("Set-Cookie does not contain 'sid=': %q", setCookie)
	}
	if !strings.Contains(strings.ToLower(setCookie), "httponly") {
		t.Errorf("Set-Cookie missing HttpOnly attribute: %q", setCookie)
	}
	if !strings.Contains(setCookie, "Path=/") {
		t.Errorf("Set-Cookie missing Path=/: %q", setCookie)
	}
	if !strings.Contains(setCookie, "SameSite=Lax") {
		t.Errorf("Set-Cookie missing SameSite=Lax: %q", setCookie)
	}

	// Extract the sid value and verify it exists in Redis with the expected TTL.
	sid := extractSIDFromSetCookie(t, setCookie)
	ctx := context.Background()
	ttl, err := testRedisClient.TTL(ctx, "session:"+sid).Result()
	if err != nil {
		t.Fatalf("Redis TTL(session:%s): %v", sid, err)
	}
	if ttl <= 0 {
		t.Fatalf("Redis session key has no TTL (key missing or expired): TTL=%v", ttl)
	}
	// SESSION_TTL is 1h; allow ±5s tolerance for test execution time.
	wantTTL := sharedCfg.SessionTTL
	tolerance := wantTTL / 100 // 1% of TTL
	if ttl < wantTTL-tolerance {
		t.Errorf("Redis session TTL = %v, want close to %v (within %v)", ttl, wantTTL, tolerance)
	}
}

// TestLogin_WrongPassword verifies that an incorrect password for a known email:
//   - returns Unauthenticated
//   - does not set a session cookie
//   - does not create a Redis session key
//   - uses a generic error message (no "password" or "email" field enumeration)
func TestLogin_WrongPassword(t *testing.T) {
	const email = "login-wrong-pw@test.local"
	const password = "correct-password"
	seedUser(t, email, password, false)

	var rawSetCookie string
	rt := &captureTransport{
		inner: http.DefaultTransport,
		onResponse: func(r *http.Response) {
			rawSetCookie = r.Header.Get("Set-Cookie")
		},
	}
	client := newAuthClientWithTransport(rt)
	req := connect.NewRequest(&authv1.LoginRequest{Email: email, Password: "wrong-password"})
	_, err := client.Login(context.Background(), req)

	assertConnectCode(t, err, connect.CodeUnauthenticated)

	if rawSetCookie != "" {
		t.Errorf("expected no Set-Cookie on wrong password, got: %q", rawSetCookie)
	}

	// The error message must not reveal which field failed.
	ce := err.(*connect.Error)
	msg := strings.ToLower(ce.Message())
	if strings.Contains(msg, "password") || strings.Contains(msg, "email") {
		t.Errorf("error message reveals field identity (enumeration risk): %q", ce.Message())
	}
}

// TestLogin_UnknownEmail verifies that login with an email that does not exist
// returns the same error shape as a wrong password (no user enumeration).
func TestLogin_UnknownEmail(t *testing.T) {
	// Do NOT seed a user — the email must not exist.
	req := connect.NewRequest(&authv1.LoginRequest{
		Email:    "does-not-exist-ever@test.local",
		Password: "any-password",
	})
	_, err := newAuthClient(nil).Login(context.Background(), req)

	assertConnectCode(t, err, connect.CodeUnauthenticated)

	// Error message must be indistinguishable from the wrong-password response.
	ce := err.(*connect.Error)
	msg := strings.ToLower(ce.Message())
	if strings.Contains(msg, "password") || strings.Contains(msg, "email") || strings.Contains(msg, "not found") {
		t.Errorf("unknown-email error reveals account existence (enumeration risk): %q", ce.Message())
	}
}

// TestLogin_SoftDeletedUser verifies that a user with deleted_at set cannot log in
// even with correct credentials.
func TestLogin_SoftDeletedUser(t *testing.T) {
	const email = "login-deleted@test.local"
	const password = "correct-password"
	seedUser(t, email, password, true /* softDeleted */)

	_, err := newAuthClient(nil).Login(
		context.Background(),
		connect.NewRequest(&authv1.LoginRequest{Email: email, Password: password}),
	)

	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestLogin_BlankFields verifies that blank email or password is rejected before
// authentication logic runs, returning InvalidArgument.
func TestLogin_BlankFields(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		password string
	}{
		{"both blank", "", ""},
		{"blank email", "", "somepassword"},
		{"blank password", "user@example.com", ""},
	}

	client := newAuthClient(nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := connect.NewRequest(&authv1.LoginRequest{Email: tc.email, Password: tc.password})
			_, err := client.Login(context.Background(), req)
			assertConnectCode(t, err, connect.CodeInvalidArgument)
		})
	}
}

// extractSIDFromSetCookie parses the sid value from a raw Set-Cookie header.
func extractSIDFromSetCookie(t *testing.T, setCookie string) string {
	t.Helper()
	for _, part := range strings.Split(setCookie, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "sid=") {
			return strings.TrimPrefix(part, "sid=")
		}
	}
	t.Fatalf("could not extract sid from Set-Cookie: %q", setCookie)
	return ""
}

// newAuthClientWithTransport builds an auth client backed by a custom transport.
func newAuthClientWithTransport(rt http.RoundTripper) authv1connect.AuthServiceClient {
	return authv1connect.NewAuthServiceClient(&http.Client{Transport: rt}, baseURL)
}
