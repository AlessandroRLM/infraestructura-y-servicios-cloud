package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
)

// TestSectionEnrollment_SoftDelete_RowInvisibleAfterDelete verifies that a hard-deleted
// section_enrollment row is invisible through the API (GetSectionEnrollment returns NotFound).
func TestSectionEnrollment_SoftDelete_RowInvisibleAfterDelete(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-softdel-invisible-admin@se.test", "admin")
	studentUserID := seedUserWithRole(t, "se-softdel-invisible-stu@se.test", "student")
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

	// Hard-delete via raw SQL to simulate orphan cleanup — deleted rows must be invisible.
	_, err2 := pgxPool.Exec(context.Background(),
		`UPDATE section_enrollments SET deleted_at = NOW() WHERE id = $1`, se.GetId())
	if err2 != nil {
		t.Fatalf("soft-delete via raw SQL: %v", err2)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, se.GetId())
	})

	getReq := connect.NewRequest(&section_enrollmentv1.GetSectionEnrollmentRequest{Id: se.GetId()})
	getReq.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetSectionEnrollment(ctx, getReq)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestSectionEnrollment_Withdraw_SetsUpdatedAt verifies that WithdrawSection updates
// the updated_at timestamp and changes status to withdrawn without setting deleted_at.
func TestSectionEnrollment_Withdraw_SetsUpdatedAt(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-updated-at-admin@se.test", "admin")
	studentUserID := seedUserWithRole(t, "se-updated-at-stu@se.test", "student")
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

	beforeWithdraw := time.Now().UTC()

	wReq := connect.NewRequest(&section_enrollmentv1.WithdrawSectionRequest{Id: se.GetId()})
	wReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.WithdrawSection(ctx, wReq); err != nil {
		t.Fatalf("WithdrawSection: %v", err)
	}

	// Verify the status was set to withdrawn in the DB.
	var statusInDB string
	if err := pgxPool.QueryRow(context.Background(),
		`SELECT status FROM section_enrollments WHERE id = $1`, se.GetId(),
	).Scan(&statusInDB); err != nil {
		t.Fatalf("query status after withdraw: %v", err)
	}
	if statusInDB != "withdrawn" {
		t.Errorf("status in DB after withdraw = %q, want withdrawn", statusInDB)
	}

	// Verify deleted_at IS NULL (withdrawal is NOT a hard delete).
	var deletedAt *time.Time
	if err := pgxPool.QueryRow(context.Background(),
		`SELECT deleted_at FROM section_enrollments WHERE id = $1`, se.GetId(),
	).Scan(&deletedAt); err != nil {
		t.Fatalf("query deleted_at: %v", err)
	}
	if deletedAt != nil {
		t.Error("deleted_at must remain NULL after WithdrawSection — withdraw is not a delete")
	}

	// Verify updated_at was bumped after the withdraw.
	var updatedAt time.Time
	if err := pgxPool.QueryRow(context.Background(),
		`SELECT updated_at FROM section_enrollments WHERE id = $1`, se.GetId(),
	).Scan(&updatedAt); err != nil {
		t.Fatalf("query updated_at: %v", err)
	}
	if !updatedAt.After(beforeWithdraw) {
		t.Errorf("updated_at = %v is not after beforeWithdraw = %v", updatedAt, beforeWithdraw)
	}
}

// TestSectionEnrollment_Withdrawn_NotCountedAsActiveSeat verifies that a withdrawn
// inscription is NOT counted in the active seat tally.
func TestSectionEnrollment_Withdrawn_NotCountedAsActiveSeat(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedUserWithSession(t, "se-wdrn-seat-admin@se.test", "admin")
	studentUserID := seedUserWithRole(t, "se-wdrn-seat-stu@se.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 5)
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

	if activeSeatCount(t, sectionID) != 1 {
		t.Fatal("expected 1 active seat before withdrawal")
	}

	wReq := connect.NewRequest(&section_enrollmentv1.WithdrawSectionRequest{Id: se.GetId()})
	wReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.WithdrawSection(ctx, wReq); err != nil {
		t.Fatalf("WithdrawSection: %v", err)
	}

	if activeSeatCount(t, sectionID) != 0 {
		t.Error("withdrawn inscription must not count as an active seat")
	}
}
