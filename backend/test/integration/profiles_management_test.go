package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1/profilesv1connect"
)

// newProfilesClient returns a Connect ProfileService client targeting the shared test server.
// Pass a non-nil CookieJar to carry session cookies.
func newProfilesClient(jar http.CookieJar) profilesv1connect.ProfileServiceClient {
	return profilesv1connect.NewProfileServiceClient(&http.Client{Jar: jar}, baseURL)
}

// seedUserWithSession seeds a user with the given role, creates a Redis session,
// and returns (userID, sessionID) ready for use in Cookie headers.
func seedUserWithSession(t *testing.T, email, roleName string) (uuid.UUID, string) {
	t.Helper()
	userID := seedUserWithRole(t, email, roleName)
	sid := seedSessionInRedis(t, userID, time.Hour)
	return userID, sid
}

// profilesRequest wraps a connect.Request with the session cookie.
func withSession[T any](req *connect.Request[T], sid string) *connect.Request[T] {
	req.Header().Set("Cookie", "sid="+sid)
	return req
}

// TestProfilesManagement_AdminUpsertAndGetUserProfile verifies that an admin
// can upsert and then retrieve a user profile.
func TestProfilesManagement_AdminUpsertAndGetUserProfile(t *testing.T) {
	ctx := context.Background()
	targetID := seedUserWithRole(t, "target-userprofile@profiles.test", "student")
	_, adminSID := seedUserWithSession(t, "admin-upsert-up@profiles.test", "admin")

	client := newProfilesClient(nil)

	upsertReq := &profilesv1.UpsertUserProfileRequest{
		UserId:           targetID.String(),
		GivenNames:       "Juan",
		LastNamePaternal: "Pérez",
		NationalIdType:   "RUT",
		NationalId:       "12345678-" + targetID.String()[:4],
	}
	resp, err := client.UpsertUserProfile(ctx, withSession(connect.NewRequest(upsertReq), adminSID))
	if err != nil {
		t.Fatalf("UpsertUserProfile: %v", err)
	}
	if resp.Msg.GetGivenNames() != "Juan" {
		t.Errorf("GivenNames = %q, want %q", resp.Msg.GetGivenNames(), "Juan")
	}

	getResp, err := client.GetUserProfile(ctx, withSession(connect.NewRequest(&profilesv1.GetUserProfileRequest{
		UserId: targetID.String(),
	}), adminSID))
	if err != nil {
		t.Fatalf("GetUserProfile: %v", err)
	}
	if getResp.Msg.GetGivenNames() != "Juan" {
		t.Errorf("GetUserProfile GivenNames = %q, want %q", getResp.Msg.GetGivenNames(), "Juan")
	}
}

// TestProfilesManagement_AdminReUpsertNosDuplicate verifies that upserting an existing
// user profile updates the row without creating a duplicate.
func TestProfilesManagement_AdminReUpsertNoDuplicate(t *testing.T) {
	ctx := context.Background()
	targetID := seedUserWithRole(t, "target-reupsert@profiles.test", "student")
	_, adminSID := seedUserWithSession(t, "admin-reupsert@profiles.test", "admin")

	client := newProfilesClient(nil)
	nationalID := "UNIQUE-REUPSERT-" + targetID.String()[:8]

	upsert := func(name string) {
		req := &profilesv1.UpsertUserProfileRequest{
			UserId:           targetID.String(),
			GivenNames:       name,
			LastNamePaternal: "Lopez",
			NationalIdType:   "RUT",
			NationalId:       nationalID,
		}
		_, err := client.UpsertUserProfile(ctx, withSession(connect.NewRequest(req), adminSID))
		if err != nil {
			t.Fatalf("UpsertUserProfile(%s): %v", name, err)
		}
	}

	upsert("First")
	upsert("Updated")

	getResp, err := client.GetUserProfile(ctx, withSession(connect.NewRequest(&profilesv1.GetUserProfileRequest{
		UserId: targetID.String(),
	}), adminSID))
	if err != nil {
		t.Fatalf("GetUserProfile after re-upsert: %v", err)
	}
	if getResp.Msg.GetGivenNames() != "Updated" {
		t.Errorf("GivenNames after re-upsert = %q, want %q", getResp.Msg.GetGivenNames(), "Updated")
	}

	var rowCount int
	err = pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM user_profiles WHERE user_id = $1`, targetID).Scan(&rowCount)
	if err != nil {
		t.Fatalf("count user_profiles: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("user_profiles rows for target = %d, want 1 (no duplicate)", rowCount)
	}
}

// TestProfilesManagement_AdminUpsertStudentProfile verifies student profile creation.
func TestProfilesManagement_AdminUpsertStudentProfile(t *testing.T) {
	ctx := context.Background()
	targetID := seedUserWithRole(t, "target-student@profiles.test", "student")
	_, adminSID := seedUserWithSession(t, "admin-student@profiles.test", "admin")

	client := newProfilesClient(nil)

	req := &profilesv1.UpsertStudentProfileRequest{
		UserId:        targetID.String(),
		AdmissionYear: 2023,
	}
	resp, err := client.UpsertStudentProfile(ctx, withSession(connect.NewRequest(req), adminSID))
	if err != nil {
		t.Fatalf("UpsertStudentProfile: %v", err)
	}
	if resp.Msg.GetAdmissionYear() != 2023 {
		t.Errorf("AdmissionYear = %d, want 2023", resp.Msg.GetAdmissionYear())
	}
}

// TestProfilesManagement_AdminUpsertTeacherProfile verifies teacher profile creation.
func TestProfilesManagement_AdminUpsertTeacherProfile(t *testing.T) {
	ctx := context.Background()
	targetID := seedUserWithRole(t, "target-teacher@profiles.test", "teacher")
	_, adminSID := seedUserWithSession(t, "admin-teacher@profiles.test", "admin")

	client := newProfilesClient(nil)

	dept := "Computer Science"
	title := "Professor"
	req := &profilesv1.UpsertTeacherProfileRequest{
		UserId:     targetID.String(),
		Department: &dept,
		Title:      &title,
	}
	resp, err := client.UpsertTeacherProfile(ctx, withSession(connect.NewRequest(req), adminSID))
	if err != nil {
		t.Fatalf("UpsertTeacherProfile: %v", err)
	}
	if resp.Msg.GetDepartment() != dept {
		t.Errorf("Department = %q, want %q", resp.Msg.GetDepartment(), dept)
	}
}

// TestProfilesManagement_AddAndListTeacherQualifications verifies the qualification lifecycle.
func TestProfilesManagement_AddAndListTeacherQualifications(t *testing.T) {
	ctx := context.Background()
	teacherID := seedUserWithRole(t, "target-qualifications@profiles.test", "teacher")
	_, adminSID := seedUserWithSession(t, "admin-quals@profiles.test", "admin")

	client := newProfilesClient(nil)

	// First create the teacher profile so the FK is satisfied.
	_, err := client.UpsertTeacherProfile(ctx, withSession(connect.NewRequest(&profilesv1.UpsertTeacherProfileRequest{
		UserId: teacherID.String(),
	}), adminSID))
	if err != nil {
		t.Fatalf("UpsertTeacherProfile (prerequisite): %v", err)
	}

	qualifications := []struct {
		degree string
		year   int32
	}{
		{"MSc Computer Science", 2015},
		{"PhD Machine Learning", 2020},
	}

	for _, q := range qualifications {
		req := &profilesv1.AddTeacherQualificationRequest{
			TeacherId: teacherID.String(),
			Degree:    q.degree,
			Year:      q.year,
		}
		_, err := client.AddTeacherQualification(ctx, withSession(connect.NewRequest(req), adminSID))
		if err != nil {
			t.Fatalf("AddTeacherQualification(%q): %v", q.degree, err)
		}
	}

	listResp, err := client.ListTeacherQualifications(ctx, withSession(connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
	}), adminSID))
	if err != nil {
		t.Fatalf("ListTeacherQualifications: %v", err)
	}
	if len(listResp.Msg.GetQualifications()) != 2 {
		t.Errorf("qualifications count = %d, want 2", len(listResp.Msg.GetQualifications()))
	}
}

// TestProfilesManagement_NonAdminIsDenied verifies that a student cannot call management procedures.
func TestProfilesManagement_NonAdminIsDenied(t *testing.T) {
	ctx := context.Background()
	targetID := seedUserWithRole(t, "target-deny@profiles.test", "student")
	_, studentSID := seedUserWithSession(t, "student-deny@profiles.test", "student")

	client := newProfilesClient(nil)

	req := &profilesv1.UpsertUserProfileRequest{
		UserId:           targetID.String(),
		GivenNames:       "Denied",
		LastNamePaternal: "User",
		NationalIdType:   "RUT",
		NationalId:       "DENIED-" + targetID.String()[:8],
	}
	_, err := client.UpsertUserProfile(ctx, withSession(connect.NewRequest(req), studentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestProfilesManagement_NoSessionIsUnauthenticated verifies that unauthenticated callers
// receive CodeUnauthenticated on management procedures.
func TestProfilesManagement_NoSessionIsUnauthenticated(t *testing.T) {
	ctx := context.Background()
	client := newProfilesClient(nil)

	req := &profilesv1.UpsertUserProfileRequest{
		UserId:           uuid.New().String(),
		GivenNames:       "Ghost",
		LastNamePaternal: "User",
		NationalIdType:   "RUT",
		NationalId:       "NOSESSION-1",
	}
	_, err := client.UpsertUserProfile(ctx, connect.NewRequest(req))
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}
