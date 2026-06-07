package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
)

// TestReset_ValidEmailStoresTokenAndReturnsDevToken verifies that
// RequestPasswordReset for a known active user:
//   - returns success with a non-empty dev_token (non-production environment)
//   - writes a reset key in Redis with TTL ≤ RESET_TOKEN_TTL
func TestReset_ValidEmailStoresTokenAndReturnsDevToken(t *testing.T) {
	const email = "reset-valid@test.local"
	const password = "any-password"
	seedUser(t, email, password, false)

	client := newAuthClient(nil)
	req := connect.NewRequest(&authv1.RequestPasswordResetRequest{Email: email})
	resp, err := client.RequestPasswordReset(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	devToken := resp.Msg.GetDevToken()
	if devToken == "" {
		t.Fatal("expected dev_token in non-production environment, got empty string")
	}

	// Verify the reset key exists in Redis with the expected TTL.
	ctx := context.Background()
	ttl, err := testRedisClient.TTL(ctx, "reset:"+devToken).Result()
	if err != nil {
		t.Fatalf("Redis TTL(reset:%s): %v", devToken, err)
	}
	if ttl <= 0 {
		t.Fatalf("reset key not found in Redis (TTL=%v)", ttl)
	}
	if ttl > sharedCfg.ResetTokenTTL+2*time.Second {
		t.Errorf("reset key TTL = %v, want ≤ %v", ttl, sharedCfg.ResetTokenTTL)
	}
}

// TestReset_UnknownEmailSucceedsWithNoRedisKey verifies that
// RequestPasswordReset for an email that does not exist returns success without
// writing any key to Redis (no user enumeration).
func TestReset_UnknownEmailSucceedsWithNoRedisKey(t *testing.T) {
	client := newAuthClient(nil)
	req := connect.NewRequest(&authv1.RequestPasswordResetRequest{
		Email: "does-not-exist-reset@test.local",
	})
	resp, err := client.RequestPasswordReset(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestPasswordReset for unknown email: expected success, got: %v", err)
	}

	// In the silent-success path the dev_token must be empty (no token was generated).
	devToken := resp.Msg.GetDevToken()
	if devToken != "" {
		// If a dev_token was returned, assert its key does not exist in Redis.
		// (This branch guards against a bug where a token is returned but no user matched.)
		ctx := context.Background()
		exists, err := testRedisClient.Exists(ctx, "reset:"+devToken).Result()
		if err != nil {
			t.Fatalf("Redis EXISTS check: %v", err)
		}
		if exists != 0 {
			t.Error("reset key exists in Redis for unknown email (expected no key)")
		}
	}
}

// TestReset_ValidTokenUpdatesPasswordAndConsumesToken verifies that
// ConfirmPasswordReset with a valid token:
//   - updates the user's password hash
//   - removes the reset token from Redis
//   - allows the user to log in with the new password
func TestReset_ValidTokenUpdatesPasswordAndConsumesToken(t *testing.T) {
	const email = "reset-confirm@test.local"
	const oldPassword = "old-password"
	const newPassword = "new-password-123"
	seedUser(t, email, oldPassword, false)

	client := newAuthClient(nil)

	// Request reset to obtain the dev_token.
	reqReset := connect.NewRequest(&authv1.RequestPasswordResetRequest{Email: email})
	resetResp, err := client.RequestPasswordReset(context.Background(), reqReset)
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}
	token := resetResp.Msg.GetDevToken()
	if token == "" {
		t.Fatal("no dev_token in non-production response")
	}

	// Confirm the reset.
	reqConfirm := connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
		Token:       token,
		NewPassword: newPassword,
	})
	if _, err := client.ConfirmPasswordReset(context.Background(), reqConfirm); err != nil {
		t.Fatalf("ConfirmPasswordReset: %v", err)
	}

	// Token must be gone from Redis.
	ctx := context.Background()
	exists, err := testRedisClient.Exists(ctx, "reset:"+token).Result()
	if err != nil {
		t.Fatalf("Redis EXISTS after confirm: %v", err)
	}
	if exists != 0 {
		t.Error("reset token still exists in Redis after ConfirmPasswordReset (token not consumed)")
	}

	// The new password must work for login.
	loginReq := connect.NewRequest(&authv1.LoginRequest{Email: email, Password: newPassword})
	if _, err := client.Login(context.Background(), loginReq); err != nil {
		t.Fatalf("Login with new password after reset: %v", err)
	}

	// The old password must no longer work.
	oldLoginReq := connect.NewRequest(&authv1.LoginRequest{Email: email, Password: oldPassword})
	if _, oldErr := client.Login(context.Background(), oldLoginReq); oldErr == nil {
		t.Error("Login with old password succeeded after reset — password was not changed")
	}
}

// TestReset_TokenCannotBeReused verifies that a consumed reset token
// cannot be used a second time.
func TestReset_TokenCannotBeReused(t *testing.T) {
	const email = "reset-reuse@test.local"
	const password = "any-password"
	seedUser(t, email, password, false)

	client := newAuthClient(nil)

	// Obtain a reset token.
	reqReset := connect.NewRequest(&authv1.RequestPasswordResetRequest{Email: email})
	resetResp, err := client.RequestPasswordReset(context.Background(), reqReset)
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}
	token := resetResp.Msg.GetDevToken()
	if token == "" {
		t.Fatal("no dev_token")
	}

	// First confirm — should succeed.
	first := connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
		Token:       token,
		NewPassword: "first-new-password",
	})
	if _, err := client.ConfirmPasswordReset(context.Background(), first); err != nil {
		t.Fatalf("first ConfirmPasswordReset: %v", err)
	}

	// Second confirm with the same token — must fail.
	second := connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
		Token:       token,
		NewPassword: "second-new-password",
	})
	_, err = client.ConfirmPasswordReset(context.Background(), second)
	if err == nil {
		t.Fatal("expected error on reuse of consumed reset token, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeInvalidArgument && ce.Code() != connect.CodeNotFound {
		t.Errorf("reused token error code = %v, want InvalidArgument or NotFound", ce.Code())
	}
}

// TestReset_ExpiredTokenIsRejected verifies that a reset token whose
// Redis TTL has elapsed is rejected and the database is not modified.
func TestReset_ExpiredTokenIsRejected(t *testing.T) {
	const email = "reset-expired@test.local"
	const password = "any-password"
	seedUser(t, email, password, false)

	// Seed a reset token directly into Redis with a 1-second TTL so it expires quickly.
	// The stored value is a user id, so it must be a valid UUID: if expiry ever raced
	// and the value were read, an invalid UUID would surface as Internal and mask the
	// real (expired-token) behaviour under test.
	expiredToken := "test-expired-token-" + email
	ctx := context.Background()
	if err := testRedisClient.Set(ctx, "reset:"+expiredToken, "00000000-0000-7000-8000-000000000000", 1*time.Second).Err(); err != nil {
		t.Fatalf("seed expired token: %v", err)
	}

	// Wait for it to expire.
	time.Sleep(1500 * time.Millisecond)

	client := newAuthClient(nil)
	req := connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
		Token:       expiredToken,
		NewPassword: "should-not-be-set",
	})
	_, err := client.ConfirmPasswordReset(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeInvalidArgument && ce.Code() != connect.CodeNotFound {
		t.Errorf("expired token error code = %v, want InvalidArgument or NotFound", ce.Code())
	}

	// Verify the user's password was not changed (can still log in with original password).
	loginReq := connect.NewRequest(&authv1.LoginRequest{Email: email, Password: password})
	if _, loginErr := newAuthClient(nil).Login(context.Background(), loginReq); loginErr != nil {
		t.Errorf("login with original password should still work after expired token attempt: %v", loginErr)
	}
}

// TestReset_FakeTokenIsRejected verifies that a token that was never
// issued (forged or malformed) is rejected with an appropriate error.
func TestReset_FakeTokenIsRejected(t *testing.T) {
	client := newAuthClient(nil)
	req := connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
		Token:       "this-token-was-never-issued-ever",
		NewPassword: "should-not-be-set",
	})
	_, err := client.ConfirmPasswordReset(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for fake token, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeInvalidArgument && ce.Code() != connect.CodeNotFound {
		t.Errorf("fake token error code = %v, want InvalidArgument or NotFound", ce.Code())
	}
}
