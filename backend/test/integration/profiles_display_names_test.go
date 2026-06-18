package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
)

// --- S-19: ListDisplayNamesByIDs happy path (teacher with profile.view_names) ---

// TestListDisplayNamesByIDs_HappyPath verifies that a teacher with profile.view_names
// can resolve display names for a set of known user IDs.
func TestListDisplayNamesByIDs_HappyPath(t *testing.T) {
	ctx := context.Background()

	// Create a teacher (has profile.view_names via migration 000017).
	_, teacherSID := seedUserWithSession(t, "ldbi-happy-teacher@profiles.test", "teacher")

	// Seed three users with user_profiles.
	userA, _ := seedUserWithSession(t, "ldbi-user-a@profiles.test", "student")
	userB, _ := seedUserWithSession(t, "ldbi-user-b@profiles.test", "student")
	userC, _ := seedUserWithSession(t, "ldbi-user-c@profiles.test", "student")
	seedUserProfile(t, userA, "AlphaFirst")
	seedUserProfile(t, userB, "BetaFirst")
	seedUserProfile(t, userC, "GammaFirst")

	client := newProfilesClient(nil)
	resp, err := client.ListDisplayNamesByIDs(ctx, withSession(connect.NewRequest(
		&profilesv1.ListDisplayNamesByIDsRequest{
			UserIds: []string{userA.String(), userB.String(), userC.String()},
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListDisplayNamesByIDs: %v", err)
	}

	names := resp.Msg.GetNames()
	if len(names) != 3 {
		t.Fatalf("got %d names, want 3", len(names))
	}

	// Verify each returned entry has required fields and matches one of the seeded IDs.
	gotIDs := make(map[string]string)
	for _, n := range names {
		if n.GetUserId() == "" {
			t.Errorf("entry has empty user_id")
		}
		if n.GetGivenNames() == "" {
			t.Errorf("entry %s has empty given_names", n.GetUserId())
		}
		if n.GetLastNamePaternal() == "" {
			t.Errorf("entry %s has empty last_name_paternal", n.GetUserId())
		}
		gotIDs[n.GetUserId()] = n.GetGivenNames()
	}

	for _, uid := range []uuid.UUID{userA, userB, userC} {
		if _, ok := gotIDs[uid.String()]; !ok {
			t.Errorf("user_id %s missing from response", uid.String())
		}
	}
}

// --- S-20: Unknown user IDs are omitted (not an error) ---

// TestListDisplayNamesByIDs_UnknownIDOmitted verifies that unknown user IDs are
// silently omitted from the response rather than causing an error or empty-string placeholder.
func TestListDisplayNamesByIDs_UnknownIDOmitted(t *testing.T) {
	ctx := context.Background()

	_, teacherSID := seedUserWithSession(t, "ldbi-omit-teacher@profiles.test", "teacher")
	userA, _ := seedUserWithSession(t, "ldbi-omit-usera@profiles.test", "student")
	seedUserProfile(t, userA, "OmitTestUser")

	unknownID := uuid.New() // no user_profiles row for this

	client := newProfilesClient(nil)
	resp, err := client.ListDisplayNamesByIDs(ctx, withSession(connect.NewRequest(
		&profilesv1.ListDisplayNamesByIDsRequest{
			UserIds: []string{userA.String(), unknownID.String()},
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListDisplayNamesByIDs: %v", err)
	}

	names := resp.Msg.GetNames()
	if len(names) != 1 {
		t.Fatalf("got %d names, want 1 (unknown id should be omitted)", len(names))
	}
	if names[0].GetUserId() != userA.String() {
		t.Errorf("got user_id %q, want %q", names[0].GetUserId(), userA.String())
	}
}

// --- S-21: Empty user_ids returns empty response OK ---

// TestListDisplayNamesByIDs_EmptyInput verifies that an empty user_ids slice
// returns an empty response with status OK (not an error).
func TestListDisplayNamesByIDs_EmptyInput(t *testing.T) {
	ctx := context.Background()

	_, teacherSID := seedUserWithSession(t, "ldbi-empty-teacher@profiles.test", "teacher")

	client := newProfilesClient(nil)
	resp, err := client.ListDisplayNamesByIDs(ctx, withSession(connect.NewRequest(
		&profilesv1.ListDisplayNamesByIDsRequest{
			UserIds: []string{},
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListDisplayNamesByIDs (empty): %v", err)
	}

	if len(resp.Msg.GetNames()) != 0 {
		t.Errorf("got %d names, want 0 for empty input", len(resp.Msg.GetNames()))
	}
}

// --- S-22: Student without profile.view_names gets PermissionDenied ---

// TestListDisplayNamesByIDs_StudentPermissionDenied verifies that a student
// (no profile.view_names permission) receives PermissionDenied.
func TestListDisplayNamesByIDs_StudentPermissionDenied(t *testing.T) {
	ctx := context.Background()

	_, studentSID := seedUserWithSession(t, "ldbi-student-denied@profiles.test", "student")

	client := newProfilesClient(nil)
	_, err := client.ListDisplayNamesByIDs(ctx, withSession(connect.NewRequest(
		&profilesv1.ListDisplayNamesByIDsRequest{
			UserIds: []string{uuid.New().String()},
		},
	), studentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// --- S-23: Admin with profile.view_names can call ListDisplayNamesByIDs ---

// TestListDisplayNamesByIDs_AdminHappyPath verifies that an admin user (has
// profile.view_names via seed grant) receives the same results as a teacher.
func TestListDisplayNamesByIDs_AdminHappyPath(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "ldbi-admin@profiles.test", "admin")
	userA, _ := seedUserWithSession(t, "ldbi-admin-usera@profiles.test", "student")
	userB, _ := seedUserWithSession(t, "ldbi-admin-userb@profiles.test", "student")
	seedUserProfile(t, userA, "AdminTestA")
	seedUserProfile(t, userB, "AdminTestB")

	client := newProfilesClient(nil)
	resp, err := client.ListDisplayNamesByIDs(ctx, withSession(connect.NewRequest(
		&profilesv1.ListDisplayNamesByIDsRequest{
			UserIds: []string{userA.String(), userB.String()},
		},
	), adminSID))
	if err != nil {
		t.Fatalf("ListDisplayNamesByIDs (admin): %v", err)
	}

	if len(resp.Msg.GetNames()) != 2 {
		t.Fatalf("admin: got %d names, want 2", len(resp.Msg.GetNames()))
	}
}

// --- S-25: Malformed UUID in batch returns CodeInvalidArgument (whole call fails) ---

// TestListDisplayNamesByIDs_MalformedUUID verifies that a batch containing a syntactically
// invalid UUID (e.g. "not-a-uuid") returns CodeInvalidArgument for the entire call.
// This is intentionally asymmetric with the omit-unknown-valid-id behavior (S-20):
// a malformed UUID is a caller error; a valid UUID with no matching profile is silently omitted.
func TestListDisplayNamesByIDs_MalformedUUID(t *testing.T) {
	ctx := context.Background()

	_, teacherSID := seedUserWithSession(t, "ldbi-malformed-teacher@profiles.test", "teacher")
	userA, _ := seedUserWithSession(t, "ldbi-malformed-usera@profiles.test", "student")
	seedUserProfile(t, userA, "MalformedTestUser")

	client := newProfilesClient(nil)
	_, err := client.ListDisplayNamesByIDs(ctx, withSession(connect.NewRequest(
		&profilesv1.ListDisplayNamesByIDsRequest{
			// Mix of a valid UUID and a malformed one — the malformed entry fails the whole call.
			UserIds: []string{userA.String(), "not-a-uuid"},
		},
	), teacherSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// --- S-24: Trust boundary — unscoped lookup works even for "outsider" teacher ---

// TestListDisplayNamesByIDs_UnscopedTrustBoundary verifies that a teacher can
// resolve the display name of any user by ID, even a student they do not teach.
// Profiles does NOT check section_teachers — this is the accepted trust model (ADR-4).
func TestListDisplayNamesByIDs_UnscopedTrustBoundary(t *testing.T) {
	ctx := context.Background()

	// Create a teacher with no section assignments.
	_, teacherSID := seedUserWithSession(t, "ldbi-unscoped-teacher@profiles.test", "teacher")

	// Create a student the teacher has NO teaching relationship with.
	studentZ, _ := seedUserWithSession(t, "ldbi-student-z@profiles.test", "student")
	seedUserProfile(t, studentZ, "StudentZFirst")

	client := newProfilesClient(nil)
	resp, err := client.ListDisplayNamesByIDs(ctx, withSession(connect.NewRequest(
		&profilesv1.ListDisplayNamesByIDsRequest{
			UserIds: []string{studentZ.String()},
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListDisplayNamesByIDs (unscoped): %v", err)
	}

	names := resp.Msg.GetNames()
	if len(names) != 1 {
		t.Fatalf("got %d names, want 1 (trust boundary: id-possession is the scope)", len(names))
	}
	if names[0].GetUserId() != studentZ.String() {
		t.Errorf("got user_id %q, want %q", names[0].GetUserId(), studentZ.String())
	}
}
