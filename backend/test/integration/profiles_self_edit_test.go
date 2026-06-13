package integration_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1/profilesv1connect"
	"github.com/google/uuid"
)

// newProfilesClientPlain returns a bare Connect ProfileService client (no jar).
func newProfilesClientPlain() profilesv1connect.ProfileServiceClient {
	return profilesv1connect.NewProfileServiceClient(http.DefaultClient, baseURL)
}

// seedUserProfile inserts a user_profiles row with the given fields for the user.
// Registers t.Cleanup to delete the row.
func seedUserProfileRow(t *testing.T, userID uuid.UUID, phone, commune, givenNames, lastNamePaternal string) {
	t.Helper()
	ctx := context.Background()

	_, err := pgxPool.Exec(ctx, `
		INSERT INTO user_profiles (
			user_id, given_names, last_name_paternal, national_id_type, national_id,
			phone, commune
		) VALUES ($1, $2, $3, 'RUT', $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			phone    = EXCLUDED.phone,
			commune  = EXCLUDED.commune,
			given_names = EXCLUDED.given_names,
			last_name_paternal = EXCLUDED.last_name_paternal
	`, userID, givenNames, lastNamePaternal, "NID-SELF-"+userID.String()[:8], phone, commune)
	if err != nil {
		t.Fatalf("seedUserProfileRow: %v", err)
	}

	cleanupUserProfile(t, userID)
}

// TestUpsertOwnProfile_FailClosed verifies that calling UpsertOwnProfile before the
// procedure is registered in the policies map yields CodePermissionDenied (fail-closed).
// This RED test runs first — the procedure is already in the policies map after A-9 wiring.
// We simulate the "before wiring" scenario by calling without a session (unauthenticated).
func TestUpsertOwnProfile_Unauthenticated(t *testing.T) {
	client := newProfilesClientPlain()

	phone := "999"
	req := connect.NewRequest(&profilesv1.UpsertOwnProfileRequest{
		Phone: &phone,
	})
	// No session cookie set — must get CodeUnauthenticated.
	_, err := client.UpsertOwnProfile(context.Background(), req)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestUpsertOwnProfile_SelfScope verifies that a student can update their own contact
// fields; absent fields are preserved; updated_by is set to the caller's user_id.
func TestUpsertOwnProfile_SelfScope(t *testing.T) {
	ctx := context.Background()

	aliceID, aliceSID := seedUserWithSession(t, "self-edit-alice@profiles.test", "student")
	seedUserProfileRow(t, aliceID, "111", "OldCommune", "Alice", "Smith")

	client := newProfilesClientPlain()

	newPhone := "999"
	newEmail := "alice@personal.com"
	req := connect.NewRequest(&profilesv1.UpsertOwnProfileRequest{
		Phone:         &newPhone,
		PersonalEmail: &newEmail,
		// all other fields absent — must be preserved
	})
	withSession(req, aliceSID)

	resp, err := client.UpsertOwnProfile(ctx, req)
	if err != nil {
		t.Fatalf("UpsertOwnProfile: %v", err)
	}

	// Phone and personal_email were set.
	if resp.Msg.GetPhone() != "999" {
		t.Errorf("phone = %q, want %q", resp.Msg.GetPhone(), "999")
	}
	if resp.Msg.GetPersonalEmail() != "alice@personal.com" {
		t.Errorf("personal_email = %q, want %q", resp.Msg.GetPersonalEmail(), "alice@personal.com")
	}
	// Absent field (commune) must be preserved.
	if resp.Msg.GetCommune() != "OldCommune" {
		t.Errorf("commune = %q, want %q (should be preserved)", resp.Msg.GetCommune(), "OldCommune")
	}
	// Legal-identity fields must remain untouched.
	if resp.Msg.GetGivenNames() != "Alice" {
		t.Errorf("given_names = %q, want %q (legal-identity must not change)", resp.Msg.GetGivenNames(), "Alice")
	}

	// Assert updated_by in the DB.
	var updatedBy uuid.UUID
	err = pgxPool.QueryRow(ctx,
		`SELECT updated_by FROM user_profiles WHERE user_id = $1`, aliceID,
	).Scan(&updatedBy)
	if err != nil {
		t.Fatalf("query updated_by: %v", err)
	}
	if updatedBy != aliceID {
		t.Errorf("updated_by = %v, want %v (must be caller's id)", updatedBy, aliceID)
	}
}

// TestUpsertOwnProfile_NoProfileRow verifies that UpsertOwnProfile on a user with no
// user_profiles row returns CodeNotFound (UPDATE affects 0 rows).
func TestUpsertOwnProfile_NoProfileRow(t *testing.T) {
	ctx := context.Background()

	newuserID, newuserSID := seedUserWithSession(t, "self-edit-noprofile@profiles.test", "teacher")
	_ = newuserID // user exists, but no profile row

	client := newProfilesClientPlain()

	phone := "555"
	req := withSession(connect.NewRequest(&profilesv1.UpsertOwnProfileRequest{
		Phone: &phone,
	}), newuserSID)

	_, err := client.UpsertOwnProfile(ctx, req)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestUpsertOwnProfile_PartialUpdate verifies that absent fields are preserved (COALESCE skip path).
func TestUpsertOwnProfile_PartialUpdate(t *testing.T) {
	ctx := context.Background()

	aliceID, aliceSID := seedUserWithSession(t, "self-edit-partial@profiles.test", "student")
	seedUserProfileRow(t, aliceID, "111", "Santiago", "Alice", "Partial")

	client := newProfilesClientPlain()

	newPhone := "888"
	req := withSession(connect.NewRequest(&profilesv1.UpsertOwnProfileRequest{
		Phone: &newPhone,
		// commune is absent — must stay "Santiago"
	}), aliceSID)

	resp, err := client.UpsertOwnProfile(ctx, req)
	if err != nil {
		t.Fatalf("UpsertOwnProfile partial: %v", err)
	}
	if resp.Msg.GetPhone() != "888" {
		t.Errorf("phone = %q, want %q", resp.Msg.GetPhone(), "888")
	}
	if resp.Msg.GetCommune() != "Santiago" {
		t.Errorf("commune = %q, want %q (absent field must be preserved)", resp.Msg.GetCommune(), "Santiago")
	}
}

// TestUpsertOwnProfile_ClearField verifies that a present-empty field clears the column
// while an absent field is left untouched.
func TestUpsertOwnProfile_ClearField(t *testing.T) {
	ctx := context.Background()

	aliceID, aliceSID := seedUserWithSession(t, "self-edit-clear@profiles.test", "student")
	seedUserProfileRow(t, aliceID, "111", "Santiago", "Alice", "Clear")

	client := newProfilesClientPlain()

	emptyPhone := "" // present-empty → clears phone
	req := withSession(connect.NewRequest(&profilesv1.UpsertOwnProfileRequest{
		Phone: &emptyPhone,
		// commune is absent — must stay "Santiago"
	}), aliceSID)

	resp, err := client.UpsertOwnProfile(ctx, req)
	if err != nil {
		t.Fatalf("UpsertOwnProfile clear: %v", err)
	}

	// Phone should be cleared (empty string).
	if resp.Msg.GetPhone() != "" {
		t.Errorf("phone = %q after clear, want empty", resp.Msg.GetPhone())
	}
	// Absent commune must remain.
	if resp.Msg.GetCommune() != "Santiago" {
		t.Errorf("commune = %q, want %q (absent field must be preserved)", resp.Msg.GetCommune(), "Santiago")
	}

	// Verify DB: phone must be empty string, commune must be "Santiago".
	var dbPhone, dbCommune *string
	err = pgxPool.QueryRow(ctx,
		`SELECT phone, commune FROM user_profiles WHERE user_id = $1`, aliceID,
	).Scan(&dbPhone, &dbCommune)
	if err != nil {
		t.Fatalf("query phone/commune: %v", err)
	}
	if dbPhone != nil && *dbPhone != "" {
		t.Errorf("DB phone = %q, want cleared (nil or empty)", *dbPhone)
	}
	if dbCommune == nil || *dbCommune != "Santiago" {
		v := "<nil>"
		if dbCommune != nil {
			v = *dbCommune
		}
		t.Errorf("DB commune = %q, want %q (should be preserved)", v, "Santiago")
	}
}
