package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1"
)

// TestEnrollment_GetOwn_Self verifies S-18: a student can fetch their own enrollment.
func TestEnrollment_GetOwn_Self(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-own-admin-self@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2130)
	defer cleanup()

	studentID, studentSID := seedUserWithSession(t, "enroll-own-student-self@enroll.test", "student")
	seedStudentProfile(t, studentID, 2130)

	// Admin creates the enrollment.
	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2130)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupEnrollment(t, e.GetId())

	// Student fetches their own enrollment.
	getReq := connect.NewRequest(&enrollmentv1.GetOwnEnrollmentRequest{Id: e.GetId()})
	getReq.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.GetOwnEnrollment(ctx, getReq)
	if err != nil {
		t.Fatalf("GetOwnEnrollment self: %v", err)
	}
	if resp.Msg.GetId() != e.GetId() {
		t.Errorf("id = %q, want %q", resp.Msg.GetId(), e.GetId())
	}
}

// TestEnrollment_GetOwn_OtherStudent_NotFound verifies S-19: fetching another student's
// enrollment returns CodeNotFound — existence is never disclosed.
func TestEnrollment_GetOwn_OtherStudent_NotFound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-own-admin-mismatch@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2131)
	defer cleanup()

	ownerID := seedUserWithRole(t, "enroll-own-owner@enroll.test", "student")
	_, callerSID := seedUserWithSession(t, "enroll-own-caller@enroll.test", "student")
	seedStudentProfile(t, ownerID, 2131)

	// Admin creates enrollment for the owner student.
	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, ownerID.String(), programID, 2131)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupEnrollment(t, e.GetId())

	// A different student tries to read the enrollment — must get CodeNotFound.
	getReq := connect.NewRequest(&enrollmentv1.GetOwnEnrollmentRequest{Id: e.GetId()})
	getReq.Header().Set("Cookie", "sid="+callerSID)
	_, err = client.GetOwnEnrollment(ctx, getReq)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestEnrollment_GetOwn_AbsentID_NotFound verifies S-19: a UUID that does not exist
// also returns CodeNotFound (both absence and mismatch are indistinguishable).
func TestEnrollment_GetOwn_AbsentID_NotFound(t *testing.T) {
	ctx := context.Background()
	_, studentSID := seedUserWithSession(t, "enroll-own-absent@enroll.test", "student")
	client := newEnrollmentClient(nil)

	getReq := connect.NewRequest(&enrollmentv1.GetOwnEnrollmentRequest{Id: uuid.New().String()})
	getReq.Header().Set("Cookie", "sid="+studentSID)
	_, err := client.GetOwnEnrollment(ctx, getReq)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestEnrollment_ListOwn_Isolation verifies S-20: each student sees only their own enrollments.
func TestEnrollment_ListOwn_Isolation(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-own-admin-iso@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2132)
	defer cleanup()

	s1ID, s1SID := seedUserWithSession(t, "enroll-own-iso-s1@enroll.test", "student")
	s2ID, s2SID := seedUserWithSession(t, "enroll-own-iso-s2@enroll.test", "student")
	seedStudentProfile(t, s1ID, 2132)
	seedStudentProfile(t, s2ID, 2132)

	client := newEnrollmentClient(nil)

	// Admin creates one enrollment per student.
	e1, err := enrollmentCreate(ctx, client, adminSID, s1ID.String(), programID, 2132)
	if err != nil {
		t.Fatalf("create s1: %v", err)
	}
	cleanupEnrollment(t, e1.GetId())

	e2, err := enrollmentCreate(ctx, client, adminSID, s2ID.String(), programID, 2132)
	if err != nil {
		t.Fatalf("create s2: %v", err)
	}
	cleanupEnrollment(t, e2.GetId())

	// s1 lists — must see only their own enrollment.
	listReq := connect.NewRequest(&enrollmentv1.ListOwnEnrollmentsRequest{})
	listReq.Header().Set("Cookie", "sid="+s1SID)
	resp, err := client.ListOwnEnrollments(ctx, listReq)
	if err != nil {
		t.Fatalf("ListOwnEnrollments s1: %v", err)
	}
	for _, row := range resp.Msg.GetEnrollments() {
		if row.GetStudentId() != s1ID.String() {
			t.Errorf("s1 list: got student_id %q, want %q", row.GetStudentId(), s1ID.String())
		}
	}

	// s2 lists — must see only their own enrollment and not s1's.
	listReq2 := connect.NewRequest(&enrollmentv1.ListOwnEnrollmentsRequest{})
	listReq2.Header().Set("Cookie", "sid="+s2SID)
	resp2, err := client.ListOwnEnrollments(ctx, listReq2)
	if err != nil {
		t.Fatalf("ListOwnEnrollments s2: %v", err)
	}
	for _, row := range resp2.Msg.GetEnrollments() {
		if row.GetStudentId() != s2ID.String() {
			t.Errorf("s2 list: got student_id %q, want %q", row.GetStudentId(), s2ID.String())
		}
	}
}
