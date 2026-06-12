package integration_test

import (
	"context"
	"net/http"
	"sort"
	"testing"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
)

// seedUserWithRoleAndSession inserts a user, assigns a role, and seeds a Redis session.
// Returns the user's UUID string and session cookie value.
func seedUserWithRoleAndSession(t *testing.T, email, roleName string) (string, string) {
	t.Helper()
	userID := seedUserWithRole(t, email, roleName)
	sid := seedSessionInRedis(t, userID, time.Hour)
	return userID.String(), sid
}

// TestGetSession_AuthenticatedWithRoles verifies scenario 1: authenticated admin user
// receives populated sorted roles and permissions with Cache-Control: no-store.
func TestGetSession_AuthenticatedWithRoles(t *testing.T) {
	userIDStr, sid := seedUserWithRoleAndSession(t, "get-session-admin@test.local", "admin")

	req := connect.NewRequest(&authv1.GetSessionRequest{})
	req.Header().Set("Cookie", "sid="+sid)

	var cacheControl string
	rt := &captureTransport{
		inner: http.DefaultTransport,
		onResponse: func(r *http.Response) {
			cacheControl = r.Header.Get("Cache-Control")
		},
	}
	client := newAuthClientWithTransport(rt)
	resp, err := client.GetSession(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSession: unexpected error: %v", err)
	}

	msg := resp.Msg
	if msg.GetUserId() != userIDStr {
		t.Errorf("user_id = %q, want %q", msg.GetUserId(), userIDStr)
	}
	if msg.GetEmail() != "get-session-admin@test.local" {
		t.Errorf("email = %q, want get-session-admin@test.local", msg.GetEmail())
	}
	if len(msg.GetRoles()) == 0 {
		t.Error("roles must be non-empty for admin user")
	}
	if !sort.StringsAreSorted(msg.GetRoles()) {
		t.Errorf("roles not sorted: %v", msg.GetRoles())
	}
	if len(msg.GetPermissions()) == 0 {
		t.Error("permissions must be non-empty for admin user")
	}
	if !sort.StringsAreSorted(msg.GetPermissions()) {
		t.Errorf("permissions not sorted: %v", msg.GetPermissions())
	}
	if cacheControl != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cacheControl)
	}
}

// TestGetSession_RoleLessUser verifies scenario 2: authenticated user with no roles
// receives 200 OK with empty (non-null) role and permission arrays.
func TestGetSession_RoleLessUser(t *testing.T) {
	userID := seedUserNoRole(t, "get-session-norole@test.local")
	sid := seedSessionInRedis(t, userID, time.Hour)

	req := connect.NewRequest(&authv1.GetSessionRequest{})
	req.Header().Set("Cookie", "sid="+sid)

	resp, err := newAuthClient(nil).GetSession(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSession: unexpected error: %v", err)
	}

	msg := resp.Msg
	if msg.GetUserId() == "" {
		t.Error("user_id must not be empty")
	}
	if msg.GetEmail() != "get-session-norole@test.local" {
		t.Errorf("email = %q, want get-session-norole@test.local", msg.GetEmail())
	}
	if len(msg.GetRoles()) != 0 {
		t.Errorf("roles = %v, want empty", msg.GetRoles())
	}
	if len(msg.GetPermissions()) != 0 {
		t.Errorf("permissions = %v, want empty", msg.GetPermissions())
	}
}

// TestGetSession_NoCookie verifies scenario 3: no session cookie returns CodeUnauthenticated.
func TestGetSession_NoCookie(t *testing.T) {
	req := connect.NewRequest(&authv1.GetSessionRequest{})
	// no Cookie header

	_, err := newAuthClient(nil).GetSession(context.Background(), req)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestGetSession_InvalidCookie verifies scenario 4: a session cookie not found in Redis
// returns CodeUnauthenticated.
func TestGetSession_InvalidCookie(t *testing.T) {
	req := connect.NewRequest(&authv1.GetSessionRequest{})
	req.Header().Set("Cookie", "sid=not-a-real-session-id-xyz")

	_, err := newAuthClient(nil).GetSession(context.Background(), req)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestGetSession_ExemptConfirmation verifies scenario 7: an authenticated user with no
// permissions can still call GetSession (exempt path, not policies).
func TestGetSession_ExemptConfirmation(t *testing.T) {
	userID := seedUserNoRole(t, "get-session-exempt@test.local")
	sid := seedSessionInRedis(t, userID, time.Hour)

	req := connect.NewRequest(&authv1.GetSessionRequest{})
	req.Header().Set("Cookie", "sid="+sid)

	resp, err := newAuthClient(nil).GetSession(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSession with zero-permission user: expected 200 OK, got: %v", err)
	}
	if resp.Msg.GetUserId() == "" {
		t.Error("user_id must be populated for zero-permission user")
	}
}
