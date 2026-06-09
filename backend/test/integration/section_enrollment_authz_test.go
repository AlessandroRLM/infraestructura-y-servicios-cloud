package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
)

// TestSectionEnrollment_StudentCannotWithdrawOwnSection verifies that there is no
// WithdrawOwnSection RPC — the student cannot withdraw themselves; only admin can.
// The student calling WithdrawSection without admin permission must return PermissionDenied.
func TestSectionEnrollment_StudentCannotWithdrawOwnSection(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-withdraw-perm-admin@se.test", "admin")
	studentUserID, studentSID := seedUserWithSession(t, "se-withdraw-perm-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programID, periodYear)
	defer cleanEnrollment()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	se, err := seEnrollAdmin(ctx, client, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("admin enroll: %v", err)
	}
	cleanupSectionEnrollment(t, se.GetId())

	// Student tries to call WithdrawSection — must be denied.
	withdrawReq := connect.NewRequest(&section_enrollmentv1.WithdrawSectionRequest{Id: se.GetId()})
	withdrawReq.Header().Set("Cookie", "sid="+studentSID)
	_, err = client.WithdrawSection(ctx, withdrawReq)
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestSectionEnrollment_OwnershipMismatch_GetOwn verifies that a student cannot
// retrieve another student's section enrollment via GetOwnSectionEnrollment.
func TestSectionEnrollment_OwnershipMismatch_GetOwn(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-ownership-admin@se.test", "admin")
	ownerUserID, _ := seedUserWithSession(t, "se-ownership-owner@se.test", "student")
	_, otherSID := seedUserWithSession(t, "se-ownership-other@se.test", "student")
	seedStudentProfile(t, ownerUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, ownerUserID.String(), programID, periodYear)
	defer cleanEnrollment()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	se, err := seEnrollAdmin(ctx, client, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("admin enroll owner: %v", err)
	}
	cleanupSectionEnrollment(t, se.GetId())

	// Another student tries to get this enrollment via GetOwnSectionEnrollment.
	getReq := connect.NewRequest(&section_enrollmentv1.GetOwnSectionEnrollmentRequest{Id: se.GetId()})
	getReq.Header().Set("Cookie", "sid="+otherSID)
	_, err = client.GetOwnSectionEnrollment(ctx, getReq)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestSectionEnrollment_ListOwnIsolation verifies that ListOwnSectionEnrollments only
// returns enrollments belonging to the authenticated student, not other students'.
func TestSectionEnrollment_ListOwnIsolation(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-list-own-admin@se.test", "admin")
	studentA, studentASID := seedUserWithSession(t, "se-list-own-stuA@se.test", "student")
	studentB, _ := seedUserWithSession(t, "se-list-own-stuB@se.test", "student")
	seedStudentProfile(t, studentA, 2099)
	seedStudentProfile(t, studentB, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollA, cleanA := seedPaidEnrollment(t, studentA.String(), programID, periodYear)
	defer cleanA()
	enrollB, cleanB := seedPaidEnrollment(t, studentB.String(), programID, periodYear)
	defer cleanB()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	seA, err := seEnrollAdmin(ctx, client, adminSID, enrollA, sectionID)
	if err != nil {
		t.Fatalf("enroll A: %v", err)
	}
	cleanupSectionEnrollment(t, seA.GetId())

	seB, err := seEnrollAdmin(ctx, client, adminSID, enrollB, sectionID)
	if err != nil {
		t.Fatalf("enroll B: %v", err)
	}
	cleanupSectionEnrollment(t, seB.GetId())

	// Student A lists own — must see only their own inscription.
	listReq := connect.NewRequest(&section_enrollmentv1.ListOwnSectionEnrollmentsRequest{})
	listReq.Header().Set("Cookie", "sid="+studentASID)
	resp, err := client.ListOwnSectionEnrollments(ctx, listReq)
	if err != nil {
		t.Fatalf("ListOwnSectionEnrollments: %v", err)
	}

	for _, se := range resp.Msg.GetSectionEnrollments() {
		if se.GetId() == seB.GetId() {
			t.Errorf("student A must not see student B's enrollment %q in ListOwn", seB.GetId())
		}
	}

	foundOwn := false
	for _, se := range resp.Msg.GetSectionEnrollments() {
		if se.GetId() == seA.GetId() {
			foundOwn = true
		}
	}
	if !foundOwn {
		t.Error("student A should see their own enrollment in ListOwn")
	}
}

// TestSectionEnrollment_StudentCallingAdminRPC_Denied verifies that a student
// calling EnrollSection (admin-only) is rejected with PermissionDenied.
func TestSectionEnrollment_StudentCallingAdminRPC_Denied(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-admin-rpc-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programID, periodYear)
	defer cleanEnrollment()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	// Student calls admin EnrollSection — not their RPC.
	_, err := seEnrollAdmin(ctx, client, studentSID, enrollmentID, sectionID)
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestSectionEnrollment_Unauthenticated_Rejected verifies that unauthenticated requests
// to any section enrollment RPC return Unauthenticated.
func TestSectionEnrollment_Unauthenticated_Rejected(t *testing.T) {
	ctx := context.Background()

	client := newSectionEnrollmentClient(nil)

	req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{})
	_, err := client.ListSectionEnrollments(ctx, req)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}
