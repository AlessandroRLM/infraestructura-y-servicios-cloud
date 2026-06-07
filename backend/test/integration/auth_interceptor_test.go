package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
)

// sessionPayload mirrors the JSON stored by session.redisStore so we can seed
// sessions directly into Redis without going through the auth service.
type sessionPayload struct {
	UserID   uuid.UUID `json:"user_id"`
	IssuedAt time.Time `json:"issued_at"`
}

// seedSessionInRedis writes a session entry directly into Redis with the given TTL.
// Returns the session ID (key suffix) for use in Cookie headers.
func seedSessionInRedis(t *testing.T, userID uuid.UUID, ttl time.Duration) string {
	t.Helper()
	sidRaw := uuid.New()
	sidStr := sidRaw.String()

	payload := sessionPayload{UserID: userID, IssuedAt: time.Now().UTC()}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("seedSessionInRedis: marshal: %v", err)
	}

	ctx := context.Background()
	key := "session:" + sidStr
	if err := testRedisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		t.Fatalf("seedSessionInRedis: Redis SET: %v", err)
	}
	t.Cleanup(func() {
		testRedisClient.Del(context.Background(), key) //nolint:errcheck
	})
	return sidStr
}

// callLogout calls Logout on targetURL with the given session id (or no cookie if sid == "").
// Logout is NOT in the public allowlist so every call goes through the session interceptor.
func callLogout(t *testing.T, targetURL, sid string) error {
	t.Helper()
	req := connect.NewRequest(&authv1.LogoutRequest{})
	if sid != "" {
		req.Header().Set("Cookie", "sid="+sid)
	}
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, targetURL)
	_, err := client.Logout(context.Background(), req)
	return err
}

// TestInterceptor_ValidSessionPassesAndRenewsTTL verifies that a request
// carrying a valid session cookie reaches the handler, and that the session TTL
// in Redis is reset to SESSION_TTL after the interceptor runs (sliding expiration).
//
// Approach: seed a session with a short TTL, make a Logout call (the interceptor
// calls GETEX before the handler, atomically renewing the TTL), then check the TTL
// via testRedisClient immediately after GETEX runs. Because the Logout handler also
// deletes the key, we read the TTL BEFORE the handler by using the interceptor
// directly via the store's Touch method in a side test.
// To avoid coupling to the handler deletion, we verify TTL renewal through the
// store.Touch call which is the same operation the interceptor performs.
func TestInterceptor_ValidSessionPassesAndRenewsTTL(t *testing.T) {
	userID := uuid.New()
	// Seed a session with a short TTL so we can verify the interceptor renews it.
	shortTTL := 30 * time.Second
	sid := seedSessionInRedis(t, userID, shortTTL)

	ctx := context.Background()
	ttlBefore, err := testRedisClient.TTL(ctx, "session:"+sid).Result()
	if err != nil {
		t.Fatalf("TTL before Touch: %v", err)
	}
	if ttlBefore > shortTTL+2*time.Second {
		t.Fatalf("pre-touch TTL unexpectedly high: %v", ttlBefore)
	}

	// Simulate what the interceptor does: Touch (GETEX) with SESSION_TTL.
	// This is the exact same operation NewSessionInterceptor calls on every protected request.
	store := session.NewRedisStore(testRedisClient)
	sess, err := store.Touch(ctx, sid, sharedCfg.SessionTTL)
	if err != nil {
		t.Fatalf("store.Touch: %v", err)
	}
	if sess.UserID != userID {
		t.Errorf("Touch returned wrong UserID: got %v, want %v", sess.UserID, userID)
	}

	// TTL must now be close to the full SESSION_TTL.
	ttlAfter, err := testRedisClient.TTL(ctx, "session:"+sid).Result()
	if err != nil {
		t.Fatalf("TTL after Touch: %v", err)
	}
	wantTTL := sharedCfg.SessionTTL
	tolerance := 5 * time.Second
	if ttlAfter < wantTTL-tolerance {
		t.Errorf("TTL after Touch = %v, want ≥ %v (GETEX did not renew TTL)", ttlAfter, wantTTL-tolerance)
	}

	// Additionally verify the full interceptor path: call Logout on the shared server.
	// The interceptor runs Touch internally and then the handler deletes the key.
	// The call succeeding proves the interceptor validated the session and forwarded the request.
	if err := callLogout(t, baseURL, sid); err != nil {
		t.Errorf("Logout with valid session failed (interceptor should have passed): %v", err)
	}
}

// TestInterceptor_ValidSessionContextCarriesUserID verifies that the
// authenticated user's identity is present in the handler context and that the
// NoopRoleLoader returning an empty slice does not cause the interceptor to error.
//
// We verify this indirectly: a Logout call with a valid session succeeds only when
// the interceptor resolves the session, injects the user_id into context, loads
// roles (noop, returns nil), and calls the handler. Any error in those steps would
// surface as a non-nil return.
func TestInterceptor_ValidSessionContextCarriesUserID(t *testing.T) {
	const email = "interceptor-ctx@test.local"
	const password = "ctx-password"
	seedUser(t, email, password, false)

	// Login to get a real session backed by a real user_id in the database.
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

	// Logout with a valid session: the interceptor resolves session → injects user_id
	// → calls loader.Load (returns empty, nil) → calls handler. Success here means
	// all interceptor steps ran without error, including context propagation.
	logoutReq := connect.NewRequest(&authv1.LogoutRequest{})
	logoutReq.Header().Set("Cookie", "sid="+sid)
	if _, err := client.Logout(context.Background(), logoutReq); err != nil {
		t.Fatalf("Logout with valid session: interceptor should have passed, got: %v", err)
	}
}

// TestInterceptor_MissingCookie verifies that Logout without a session
// cookie is rejected by the interceptor with Unauthenticated before the handler runs.
func TestInterceptor_MissingCookie(t *testing.T) {
	err := callLogout(t, baseURL, "" /* no sid */)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestInterceptor_InvalidCookie verifies that a session id that does not
// exist in Redis is rejected with Unauthenticated.
func TestInterceptor_InvalidCookie(t *testing.T) {
	err := callLogout(t, baseURL, "completely-fake-session-id-that-was-never-issued")
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestInterceptor_ExpiredSession verifies that a session whose Redis TTL
// has elapsed is rejected with Unauthenticated.
func TestInterceptor_ExpiredSession(t *testing.T) {
	userID := uuid.New()
	// Use a 1-second TTL and wait for it to lapse.
	sid := seedSessionInRedis(t, userID, 1*time.Second)
	time.Sleep(1500 * time.Millisecond)

	err := callLogout(t, baseURL, sid)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestInterceptor_PublicProceduresBypassSessionCheck verifies that Login,
// RequestPasswordReset, and ConfirmPasswordReset are reachable without a session
// cookie. The interceptor must not block them with its session-missing guard.
func TestInterceptor_PublicProceduresBypassSessionCheck(t *testing.T) {
	client := newAuthClient(nil) // no cookie jar — no session at all

	t.Run("Login bypasses session check", func(t *testing.T) {
		req := connect.NewRequest(&authv1.LoginRequest{Email: "nobody@example.com", Password: "pw"})
		_, err := client.Login(context.Background(), req)
		// The interceptor must NOT block this with its session-missing sentinel.
		// Any error that comes back is from the handler (e.g. Unauthenticated for bad creds),
		// but must not carry the interceptor's "session cookie required" message.
		if err != nil {
			ce, ok := err.(*connect.Error)
			if ok && ce.Code() == connect.CodeUnauthenticated && ce.Message() == "session cookie required" {
				t.Errorf("Login was blocked by the session interceptor — it should be on the public allowlist")
			}
		}
	})

	t.Run("RequestPasswordReset bypasses session check", func(t *testing.T) {
		req := connect.NewRequest(&authv1.RequestPasswordResetRequest{Email: "nobody@example.com"})
		_, err := client.RequestPasswordReset(context.Background(), req)
		// Should succeed silently for unknown emails — no Unauthenticated from interceptor.
		if err != nil {
			ce, ok := err.(*connect.Error)
			if ok && ce.Code() == connect.CodeUnauthenticated {
				t.Errorf("RequestPasswordReset was blocked by session interceptor: %v", err)
			}
		}
	})

	t.Run("ConfirmPasswordReset bypasses session check", func(t *testing.T) {
		req := connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
			Token:       "nonexistent-token",
			NewPassword: "newpw",
		})
		_, err := client.ConfirmPasswordReset(context.Background(), req)
		// Will fail with InvalidArgument (unknown token), but not Unauthenticated from interceptor.
		if err != nil {
			ce, ok := err.(*connect.Error)
			if ok && ce.Code() == connect.CodeUnauthenticated {
				t.Errorf("ConfirmPasswordReset was blocked by session interceptor: %v", err)
			}
		}
	})
}
