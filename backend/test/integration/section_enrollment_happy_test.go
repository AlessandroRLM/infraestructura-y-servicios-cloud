package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
)

// TestSectionEnrollment_StudentSelfEnroll_OpenWindow verifies the happy path for
// student self-enrollment: paid matrícula, course in program, open window → OK.
func TestSectionEnrollment_StudentSelfEnroll_OpenWindow(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-happy-student@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, enrollmentCleanup := seedPaidEnrollment(t, studentUserID.String(), programID, 2099)
	defer enrollmentCleanup()
	_ = enrollmentID

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	se, err := seEnrollOwn(ctx, client, studentSID, sectionID, programID)
	if err != nil {
		t.Fatalf("EnrollOwnSection: %v", err)
	}
	cleanupSectionEnrollment(t, se.GetId())

	if se.GetStatus() != "in_progress" {
		t.Errorf("status = %q, want in_progress", se.GetStatus())
	}
	if se.GetSectionId() != sectionID {
		t.Errorf("section_id = %q, want %q", se.GetSectionId(), sectionID)
	}
	if activeSeatCount(t, sectionID) != 1 {
		t.Error("active seat count should be 1 after enrollment")
	}
}

// TestSectionEnrollment_AdminEnroll_NotWindowGated verifies that admin EnrollSection
// succeeds even when the enrollment window is closed.
func TestSectionEnrollment_AdminEnroll_NotWindowGated(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-admin-window@se.test", "admin")

	studentUserID := seedUserWithRole(t, "se-admin-window-student@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	// Closed window — self-enroll would fail, admin enroll must succeed.
	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, enrollmentCleanup := seedPaidEnrollment(t, studentUserID.String(), programID, 2099)
	defer enrollmentCleanup()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	se, err := seEnrollAdmin(ctx, client, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("EnrollSection (admin, closed window): %v", err)
	}
	cleanupSectionEnrollment(t, se.GetId())

	if se.GetStatus() != "in_progress" {
		t.Errorf("status = %q, want in_progress", se.GetStatus())
	}
	if activeSeatCount(t, sectionID) != 1 {
		t.Error("active seat count should be 1 after admin enroll")
	}
}

// TestSectionEnrollment_AdminWithdraw_FreesASeat verifies that withdrawing an inscription
// decrements the active seat count and allows a subsequent enrollment to succeed.
func TestSectionEnrollment_AdminWithdraw_FreesASeat(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-withdraw-admin@se.test", "admin")

	studentA := seedUserWithRole(t, "se-withdraw-studentA@se.test", "student")
	studentB := seedUserWithRole(t, "se-withdraw-studentB@se.test", "student")
	seedStudentProfile(t, studentA, 2099)
	seedStudentProfile(t, studentB, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	// Capacity=1 so studentB cannot enroll while studentA is active.
	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 1)
	defer sectionCleanup()

	enrollA, cleanA := seedPaidEnrollment(t, studentA.String(), programID, 2099)
	defer cleanA()
	enrollB, cleanB := seedPaidEnrollment(t, studentB.String(), programID, 2099)
	defer cleanB()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	// Enroll studentA — fills the only seat.
	seA, err := seEnrollAdmin(ctx, client, adminSID, enrollA, sectionID)
	if err != nil {
		t.Fatalf("enroll studentA: %v", err)
	}
	cleanupSectionEnrollment(t, seA.GetId())

	// Verify full.
	if activeSeatCount(t, sectionID) != 1 {
		t.Fatal("expected 1 active seat after enrolling studentA")
	}

	// Withdraw studentA.
	wReq := connect.NewRequest(&section_enrollmentv1.WithdrawSectionRequest{Id: seA.GetId()})
	wReq.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.WithdrawSection(ctx, wReq)
	if err != nil {
		t.Fatalf("WithdrawSection: %v", err)
	}

	// Seat must be freed.
	if activeSeatCount(t, sectionID) != 0 {
		t.Error("active seat count should be 0 after withdrawal")
	}

	// Enroll studentB — must succeed.
	seB, err := seEnrollAdmin(ctx, client, adminSID, enrollB, sectionID)
	if err != nil {
		t.Fatalf("enroll studentB after withdrawal: %v", err)
	}
	cleanupSectionEnrollment(t, seB.GetId())

	if activeSeatCount(t, sectionID) != 1 {
		t.Error("active seat count should be 1 after studentB enrollment")
	}
}

// TestSectionEnrollment_AdminRevival verifies that admin EnrollSection on a withdrawn
// inscription revives it in-place (no new row; UNIQUE constraint stays intact).
func TestSectionEnrollment_AdminRevival(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-revival-admin@se.test", "admin")

	studentUserID := seedUserWithRole(t, "se-revival-student@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programID, 2099)
	defer cleanEnrollment()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	// Enroll then withdraw.
	se, err := seEnrollAdmin(ctx, client, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("initial enroll: %v", err)
	}
	cleanupSectionEnrollment(t, se.GetId())

	wReq := connect.NewRequest(&section_enrollmentv1.WithdrawSectionRequest{Id: se.GetId()})
	wReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.WithdrawSection(ctx, wReq); err != nil {
		t.Fatalf("withdraw: %v", err)
	}

	// Revive via EnrollSection.
	revived, err := seEnrollAdmin(ctx, client, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("revival: %v", err)
	}

	if revived.GetId() != se.GetId() {
		t.Errorf("revival should reuse existing row; got new id %q, want %q", revived.GetId(), se.GetId())
	}
	if revived.GetStatus() != "in_progress" {
		t.Errorf("revived status = %q, want in_progress", revived.GetStatus())
	}
	if activeSeatCount(t, sectionID) != 1 {
		t.Error("active seat count should be 1 after revival")
	}
}

// TestSectionEnrollment_ListSectionEnrollments_Filter verifies that ListSectionEnrollments
// returns only inscriptions matching the section_id filter.
func TestSectionEnrollment_ListSectionEnrollments_Filter(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-list-admin@se.test", "admin")

	studentA := seedUserWithRole(t, "se-list-stuA@se.test", "student")
	studentB := seedUserWithRole(t, "se-list-stuB@se.test", "student")
	seedStudentProfile(t, studentA, 2099)
	seedStudentProfile(t, studentB, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollA, cleanA := seedPaidEnrollment(t, studentA.String(), programID, 2099)
	defer cleanA()
	enrollB, cleanB := seedPaidEnrollment(t, studentB.String(), programID, 2099)
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

	listReq := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		SectionId: sectionID,
	})
	listReq.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSectionEnrollments(ctx, listReq)
	if err != nil {
		t.Fatalf("ListSectionEnrollments: %v", err)
	}

	if len(resp.Msg.GetSectionEnrollments()) != 2 {
		t.Errorf("expected 2 section enrollments, got %d", len(resp.Msg.GetSectionEnrollments()))
	}
	for _, se := range resp.Msg.GetSectionEnrollments() {
		if se.GetSectionId() != sectionID {
			t.Errorf("expected section_id=%q, got %q", sectionID, se.GetSectionId())
		}
	}
}
