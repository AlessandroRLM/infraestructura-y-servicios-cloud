package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
)

// seedUserProfile inserts a user_profiles row directly for the given userID.
// Uses a unique national_id derived from the userID to avoid UNIQUE constraint collisions.
func seedUserProfile(t *testing.T, userID interface{ String() string }, givenNames string) {
	t.Helper()
	_, err := pgxPool.Exec(context.Background(), `
		INSERT INTO user_profiles (
			user_id, given_names, last_name_paternal,
			national_id_type, national_id
		) VALUES ($1, $2, 'Test', 'RUT', $3)
		ON CONFLICT (user_id) DO UPDATE SET given_names = EXCLUDED.given_names
	`, userID.String(), givenNames, "SEED-"+userID.String()[:12])
	if err != nil {
		t.Fatalf("seedUserProfile: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM user_profiles WHERE user_id = $1`, userID.String())
	})
}

// TestProfilesSelfRead_AuthenticatedUserGetsOwnProfile verifies that an authenticated user
// with profile.view_own gets their own profile row from GetOwnProfile.
func TestProfilesSelfRead_AuthenticatedUserGetsOwnProfile(t *testing.T) {
	ctx := context.Background()
	userID, sid := seedUserWithSession(t, "self-read-own@profiles.test", "student")
	seedUserProfile(t, userID, "SelfReadUser")

	client := newProfilesClient(nil)

	resp, err := client.GetOwnProfile(ctx, withSession(connect.NewRequest(&profilesv1.GetOwnProfileRequest{}), sid))
	if err != nil {
		t.Fatalf("GetOwnProfile: %v", err)
	}

	if resp.Msg.GetGivenNames() != "SelfReadUser" {
		t.Errorf("GivenNames = %q, want %q", resp.Msg.GetGivenNames(), "SelfReadUser")
	}

	// The returned user_id must match the caller's own user_id.
	if resp.Msg.GetUserId() != userID.String() {
		t.Errorf("returned user_id = %q, want caller's user_id %q", resp.Msg.GetUserId(), userID.String())
	}
}

// TestProfilesSelfRead_IsolationBetweenUsers verifies that two users cannot access each other's
// profiles via GetOwnProfile (no user_id argument means always own profile).
func TestProfilesSelfRead_IsolationBetweenUsers(t *testing.T) {
	ctx := context.Background()

	userAID, userASID := seedUserWithSession(t, "self-read-a@profiles.test", "student")
	userBID, _ := seedUserWithSession(t, "self-read-b@profiles.test", "student")

	seedUserProfile(t, userAID, "UserAProfile")
	seedUserProfile(t, userBID, "UserBProfile")

	client := newProfilesClient(nil)

	// User A calls GetOwnProfile — must get A's profile, not B's.
	resp, err := client.GetOwnProfile(ctx, withSession(connect.NewRequest(&profilesv1.GetOwnProfileRequest{}), userASID))
	if err != nil {
		t.Fatalf("GetOwnProfile (user A): %v", err)
	}

	if resp.Msg.GetUserId() != userAID.String() {
		t.Errorf("user A's GetOwnProfile returned user_id %q, want %q", resp.Msg.GetUserId(), userAID.String())
	}
	if resp.Msg.GetGivenNames() != "UserAProfile" {
		t.Errorf("user A's GetOwnProfile GivenNames = %q, want %q", resp.Msg.GetGivenNames(), "UserAProfile")
	}
	// Ensure B's name is not returned.
	if resp.Msg.GetGivenNames() == "UserBProfile" {
		t.Error("user A's GetOwnProfile returned user B's profile")
	}
}

// TestProfilesSelfRead_NotFoundWhenNoProfile verifies that CodeNotFound is returned when
// the authenticated user has no user_profiles row.
func TestProfilesSelfRead_NotFoundWhenNoProfile(t *testing.T) {
	ctx := context.Background()
	_, sid := seedUserWithSession(t, "self-read-noprofile@profiles.test", "student")

	client := newProfilesClient(nil)

	_, err := client.GetOwnProfile(ctx, withSession(connect.NewRequest(&profilesv1.GetOwnProfileRequest{}), sid))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestProfilesSelfRead_NoSessionIsUnauthenticated verifies CodeUnauthenticated without a session.
func TestProfilesSelfRead_NoSessionIsUnauthenticated(t *testing.T) {
	ctx := context.Background()
	client := newProfilesClient(nil)

	_, err := client.GetOwnProfile(ctx, connect.NewRequest(&profilesv1.GetOwnProfileRequest{}))
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}
