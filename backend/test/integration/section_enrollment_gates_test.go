package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
)

// TestSectionEnrollment_WindowClosed_RejectsStudentSelfEnroll verifies that a student
// cannot self-enroll when the academic period's enrollment window is in the past.
func TestSectionEnrollment_WindowClosed_RejectsStudentSelfEnroll(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-window-closed-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	// Closed window (ends in the past).
	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programID, 2099)
	defer cleanEnrollment()
	_ = enrollmentID

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	_, err := seEnrollOwn(ctx, client, studentSID, sectionID, programID)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	if activeSeatCount(t, sectionID) != 0 {
		t.Error("no seat should be consumed when window is closed")
	}
}

// TestSectionEnrollment_FutureWindow_RejectsStudentSelfEnroll verifies that a student
// cannot self-enroll when the enrollment window starts in the future (not yet open).
func TestSectionEnrollment_FutureWindow_RejectsStudentSelfEnroll(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-future-window-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	// Future window: starts=now+1h / ends=now+2h.
	futurePeriodID, futureCleanup := seedAcademicPeriodFutureWindow(t)
	defer futureCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, futurePeriodID, 10)
	defer sectionCleanup()

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programID, 2099)
	defer cleanEnrollment()
	_ = enrollmentID

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	_, err := seEnrollOwn(ctx, client, studentSID, sectionID, programID)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	if activeSeatCount(t, sectionID) != 0 {
		t.Error("no seat should be consumed when window has not opened yet")
	}
}

// TestSectionEnrollment_NullWindow_FailClosed verifies that a NULL enrollment window
// is treated as closed (fail-closed), rejecting the student self-enroll.
func TestSectionEnrollment_NullWindow_FailClosed(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-null-window-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	// NULL window columns.
	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, false, true)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programID, 2099)
	defer cleanEnrollment()
	_ = enrollmentID

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	_, err := seEnrollOwn(ctx, client, studentSID, sectionID, programID)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestSectionEnrollment_UnpaidEnrollment_RejectsEnroll verifies that a pending (unpaid)
// matrícula causes EnrollOwnSection to return FailedPrecondition.
func TestSectionEnrollment_UnpaidEnrollment_RejectsEnroll(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-unpaid-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	// Pending enrollment (not paid).
	enrollmentID, cleanEnrollment := seedPendingEnrollment(t, studentUserID.String(), programID, 2099)
	defer cleanEnrollment()
	_ = enrollmentID

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	_, err := seEnrollOwn(ctx, client, studentSID, sectionID, programID)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	if activeSeatCount(t, sectionID) != 0 {
		t.Error("no seat should be consumed when enrollment is not paid")
	}
}

// TestSectionEnrollment_WrongProgramID_RejectsEnroll verifies that supplying a program_id
// for which the student has no enrollment returns NotFound (not FailedPrecondition).
func TestSectionEnrollment_WrongProgramID_RejectsEnroll(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-wrong-program-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programA, courseID, cleanupA := seedProgramWithCourse(t)
	defer cleanupA()

	// programB exists but the student has no enrollment in it.
	programB, _, cleanupB := seedProgramWithCourse(t)
	defer cleanupB()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	// Student has a paid enrollment in programA only.
	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programA, 2099)
	defer cleanEnrollment()
	_ = enrollmentID

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	// Pass programB — no enrollment exists there → NotFound.
	_, err := seEnrollOwn(ctx, client, studentSID, sectionID, programB)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestSectionEnrollment_CourseNotInProgram_RejectsEnroll verifies that a section whose
// course is NOT in the student's program causes FailedPrecondition.
func TestSectionEnrollment_CourseNotInProgram_RejectsEnroll(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-course-not-in-prog-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	// Program A where the student is enrolled.
	programA, _, cleanupA := seedProgramWithCourse(t)
	defer cleanupA()

	// Program B with a different course — student is NOT enrolled in B.
	_, courseB, cleanupB := seedProgramWithCourse(t)
	defer cleanupB()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	// Section uses courseB (not in programA).
	sectionID, sectionCleanup := seedSection(t, courseB, periodID, 10)
	defer sectionCleanup()

	// Enrollment in programA (paid).
	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programA, 2099)
	defer cleanEnrollment()
	_ = enrollmentID

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	// Pass programA — student has a paid enrollment there, but courseB is not in programA.
	_, err := seEnrollOwn(ctx, client, studentSID, sectionID, programA)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	if activeSeatCount(t, sectionID) != 0 {
		t.Error("no seat should be consumed when course is not in program")
	}
}

// TestSectionEnrollment_TwoPrograms_Unambiguous verifies that a student enrolled in two
// programs sharing a course can enroll in a section by specifying the program_id.
func TestSectionEnrollment_TwoPrograms_Unambiguous(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-two-prog-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	// Both programs contain the same course.
	programA, courseID, cleanupA := seedProgramWithCourse(t)
	defer cleanupA()

	programB, _, cleanupB := seedProgramWithCourse(t)
	defer cleanupB()

	// Link the same course to programB as well.
	if _, err := pgxPool.Exec(context.Background(),
		`INSERT INTO program_courses (program_id, course_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		programB, courseID,
	); err != nil {
		t.Fatalf("link course to programB: %v", err)
	}
	defer pgxPool.Exec(context.Background(), //nolint:errcheck
		`DELETE FROM program_courses WHERE program_id = $1 AND course_id = $2`, programB, courseID)

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	// Student has paid enrollments in BOTH programs.
	enrollA, cleanA := seedPaidEnrollment(t, studentUserID.String(), programA, 2099)
	defer cleanA()
	_ = enrollA

	enrollB, cleanB := seedPaidEnrollment(t, studentUserID.String(), programB, 2099)
	defer cleanB()
	_ = enrollB

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	// Enroll via programA — must succeed unambiguously.
	se, err := seEnrollOwn(ctx, client, studentSID, sectionID, programA)
	if err != nil {
		t.Fatalf("EnrollOwnSection via programA: %v", err)
	}
	cleanupSectionEnrollment(t, se.GetId())

	if se.GetStatus() != "in_progress" {
		t.Errorf("status = %q, want in_progress", se.GetStatus())
	}
	if activeSeatCount(t, sectionID) != 1 {
		t.Error("expected 1 active seat after enrollment via programA")
	}
}

// TestSectionEnrollment_AdminRevival_RejectsWhenFull verifies that admin cannot revive
// a withdrawn inscription when the section is already at full capacity.
func TestSectionEnrollment_AdminRevival_RejectsWhenFull(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-revival-full-admin@se.test", "admin")

	studentA := seedUserWithRole(t, "se-revival-full-stuA@se.test", "student")
	studentB := seedUserWithRole(t, "se-revival-full-stuB@se.test", "student")
	seedStudentProfile(t, studentA, 2099)
	seedStudentProfile(t, studentB, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	// Capacity=1 so that when A is withdrawn and B occupies the seat, revival fails.
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 1)
	defer sectionCleanup()

	enrollA, cleanA := seedPaidEnrollment(t, studentA.String(), programID, 2099)
	defer cleanA()
	enrollB, cleanB := seedPaidEnrollment(t, studentB.String(), programID, 2099)
	defer cleanB()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	// Enroll A, then withdraw A.
	seA, err := seEnrollAdmin(ctx, client, adminSID, enrollA, sectionID)
	if err != nil {
		t.Fatalf("enroll A: %v", err)
	}
	cleanupSectionEnrollment(t, seA.GetId())

	withdrawReq := connect.NewRequest(&section_enrollmentv1.WithdrawSectionRequest{Id: seA.GetId()})
	withdrawReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.WithdrawSection(ctx, withdrawReq); err != nil {
		t.Fatalf("withdraw A: %v", err)
	}

	// Enroll B — takes the freed seat.
	seB, err := seEnrollAdmin(ctx, client, adminSID, enrollB, sectionID)
	if err != nil {
		t.Fatalf("enroll B: %v", err)
	}
	cleanupSectionEnrollment(t, seB.GetId())

	// Attempt to revive A — should fail (section full).
	_, err = seEnrollAdmin(ctx, client, adminSID, enrollA, sectionID)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestSectionEnrollment_StudentCannotSelfRevive verifies that a student calling
// EnrollOwnSection when a withdrawn inscription already exists returns FailedPrecondition.
func TestSectionEnrollment_StudentCannotSelfRevive(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-no-self-revive-admin@se.test", "admin")
	studentUserID, studentSID := seedUserWithSession(t, "se-no-self-revive-stu@se.test", "student")
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

	// Admin enroll then withdraw.
	se, err := seEnrollAdmin(ctx, client, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("admin enroll: %v", err)
	}
	cleanupSectionEnrollment(t, se.GetId())

	withdrawReq := connect.NewRequest(&section_enrollmentv1.WithdrawSectionRequest{Id: se.GetId()})
	withdrawReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.WithdrawSection(ctx, withdrawReq); err != nil {
		t.Fatalf("withdraw: %v", err)
	}

	// Student tries to re-enroll — must fail.
	_, err = seEnrollOwn(ctx, client, studentSID, sectionID, programID)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestSectionEnrollment_IdempotentRetry_AlreadyExists verifies that a second identical
// EnrollOwnSection call returns CodeAlreadyExists without consuming a second seat.
func TestSectionEnrollment_IdempotentRetry_AlreadyExists(t *testing.T) {
	ctx := context.Background()

	studentUserID, studentSID := seedUserWithSession(t, "se-idempotent-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentUserID.String(), programID, 2099)
	defer cleanEnrollment()
	_ = enrollmentID

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)

	se, err := seEnrollOwn(ctx, client, studentSID, sectionID, programID)
	if err != nil {
		t.Fatalf("first enroll: %v", err)
	}
	cleanupSectionEnrollment(t, se.GetId())

	_, err = seEnrollOwn(ctx, client, studentSID, sectionID, programID)
	assertConnectCode(t, err, connect.CodeAlreadyExists)

	if activeSeatCount(t, sectionID) != 1 {
		t.Error("duplicate enroll must not consume a second seat")
	}
}

// TestSectionEnrollment_PaidGate_AdminPath verifies that the paid gate applies to
// admin EnrollSection as well (enrollment must be paid, not pending).
func TestSectionEnrollment_PaidGate_AdminPath(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-paid-gate-admin@se.test", "admin")

	studentUserID := seedUserWithRole(t, "se-paid-gate-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	// Pending enrollment — not paid.
	pendingEnrollmentID, cleanEnrollment := seedPendingEnrollment(t, studentUserID.String(), programID, 2099)
	defer cleanEnrollment()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	client := newSectionEnrollmentClient(nil)
	_, err := seEnrollAdmin(ctx, client, adminSID, pendingEnrollmentID, sectionID)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}
