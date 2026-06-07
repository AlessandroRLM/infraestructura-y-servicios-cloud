package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
)

// TestLogout_SessionDeletedAndCookieInvalidated verifies that:
//   - Logout succeeds for an authenticated session
//   - The session key is removed from Redis after logout
//   - A subsequent protected call using the same cookie returns Unauthenticated
func TestLogout_SessionDeletedAndCookieInvalidated(t *testing.T) {
	const email = "logout-test@test.local"
	const password = "logout-password"
	seedUser(t, email, password, false)

	// Log in and capture the session id.
	jar := newCookieJar()
	client := newAuthClient(jar)

	loginReq := connect.NewRequest(&authv1.LoginRequest{Email: email, Password: password})
	if _, err := client.Login(context.Background(), loginReq); err != nil {
		t.Fatalf("Login: %v", err)
	}
	sid := jar.cookieValue("sid")
	if sid == "" {
		t.Fatal("no 'sid' cookie after Login")
	}

	// Verify the session exists in Redis before logout.
	ctx := context.Background()
	ttlBefore, err := testRedisClient.TTL(ctx, "session:"+sid).Result()
	if err != nil {
		t.Fatalf("Redis TTL before logout: %v", err)
	}
	if ttlBefore <= 0 {
		t.Fatal("session key not found in Redis before logout")
	}

	// Logout — the jar carries the sid cookie automatically.
	logoutReq := connect.NewRequest(&authv1.LogoutRequest{})
	if _, err := client.Logout(context.Background(), logoutReq); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// The session key must be gone from Redis.
	exists, err := testRedisClient.Exists(ctx, "session:"+sid).Result()
	if err != nil {
		t.Fatalf("Redis EXISTS after logout: %v", err)
	}
	if exists != 0 {
		t.Error("session key still exists in Redis after Logout")
	}

	// A subsequent protected call with the same cookie must be rejected.
	// Logout is the only protected endpoint; reusing the deleted session id must
	// be rejected by the interceptor before the handler runs.
	if err := callLogout(t, baseURL, sid); err == nil {
		t.Fatal("expected Unauthenticated on reuse of logged-out session, got nil error")
	} else {
		assertConnectCode(t, err, connect.CodeUnauthenticated)
	}
}
