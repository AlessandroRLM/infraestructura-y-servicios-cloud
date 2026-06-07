package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
)

// TestProfilesDecoupling_StudentProfileDoesNotAssignRole verifies that upserting a student
// profile leaves user_roles unchanged.
func TestProfilesDecoupling_StudentProfileDoesNotAssignRole(t *testing.T) {
	ctx := context.Background()

	// A bare user with no role assignment.
	userID := seedUserNoRole(t, "decoupling-student@profiles.test")
	cleanupStudentProfile(t, userID)
	_, adminSID := seedUserWithSession(t, "decoupling-admin-s@profiles.test", "admin")

	var before int
	if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, userID).Scan(&before); err != nil {
		t.Fatalf("count user_roles before: %v", err)
	}

	client := newProfilesClient(nil)
	req := &profilesv1.UpsertStudentProfileRequest{
		UserId:        userID.String(),
		AdmissionYear: 2024,
	}
	_, err := client.UpsertStudentProfile(ctx, withSession(connect.NewRequest(req), adminSID))
	if err != nil {
		t.Fatalf("UpsertStudentProfile: %v", err)
	}

	var after int
	if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, userID).Scan(&after); err != nil {
		t.Fatalf("count user_roles after: %v", err)
	}

	if after != before {
		t.Errorf("user_roles count changed: before=%d, after=%d (profile upsert must not touch user_roles)", before, after)
	}
	if after != 0 {
		t.Errorf("user_roles count = %d, want 0 (no role auto-assigned)", after)
	}
}

// TestProfilesDecoupling_TeacherProfileDoesNotAssignRole verifies that upserting a teacher
// profile leaves user_roles unchanged.
func TestProfilesDecoupling_TeacherProfileDoesNotAssignRole(t *testing.T) {
	ctx := context.Background()

	userID := seedUserNoRole(t, "decoupling-teacher@profiles.test")
	cleanupTeacherProfile(t, userID)
	_, adminSID := seedUserWithSession(t, "decoupling-admin-t@profiles.test", "admin")

	var before int
	if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, userID).Scan(&before); err != nil {
		t.Fatalf("count user_roles before: %v", err)
	}

	client := newProfilesClient(nil)
	_, err := client.UpsertTeacherProfile(ctx, withSession(connect.NewRequest(&profilesv1.UpsertTeacherProfileRequest{
		UserId: userID.String(),
	}), adminSID))
	if err != nil {
		t.Fatalf("UpsertTeacherProfile: %v", err)
	}

	var after int
	if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, userID).Scan(&after); err != nil {
		t.Fatalf("count user_roles after: %v", err)
	}

	if after != before {
		t.Errorf("user_roles count changed: before=%d, after=%d", before, after)
	}
	if after != 0 {
		t.Errorf("user_roles count = %d, want 0", after)
	}
}

// TestProfilesDecoupling_NoAuthRegressionFromExistingTests verifies that the shared server
// now wired with profiles continues to serve existing auth endpoints without regression.
func TestProfilesDecoupling_NoAuthRegressionFromExistingTests(t *testing.T) {
	// Verify that Login is still reachable and the session interceptor correctly
	// blocks Logout without a session (the simplest auth regression check).
	err := callLogout(t, baseURL, "" /* no session */)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}
