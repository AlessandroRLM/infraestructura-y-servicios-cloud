package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1"
)

// TestEnrollment_MarkPaid_OnCancelled_Rejected verifies S-05.
func TestEnrollment_MarkPaid_OnCancelled_Rejected(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-sm-paid-cancelled@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2100)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-sm-student1@enroll.test", "student")
	seedStudentProfile(t, studentID, 2100)

	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2100)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupEnrollment(t, e.GetId())

	// Cancel the enrollment first.
	cancelReq := connect.NewRequest(&enrollmentv1.CancelEnrollmentRequest{Id: e.GetId()})
	cancelReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.CancelEnrollment(ctx, cancelReq); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Now try to mark it paid — must fail with CodeFailedPrecondition.
	paidReq := connect.NewRequest(&enrollmentv1.MarkEnrollmentPaidRequest{Id: e.GetId()})
	paidReq.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.MarkEnrollmentPaid(ctx, paidReq)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestEnrollment_MarkPaid_OnPaid_Rejected verifies S-06.
func TestEnrollment_MarkPaid_OnPaid_Rejected(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-sm-paid-paid@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2101)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-sm-student2@enroll.test", "student")
	seedStudentProfile(t, studentID, 2101)

	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2101)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupEnrollment(t, e.GetId())

	// Mark paid once — OK.
	paidReq := connect.NewRequest(&enrollmentv1.MarkEnrollmentPaidRequest{Id: e.GetId()})
	paidReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.MarkEnrollmentPaid(ctx, paidReq); err != nil {
		t.Fatalf("first mark paid: %v", err)
	}

	// Mark paid again — must fail with CodeFailedPrecondition.
	paidReq2 := connect.NewRequest(&enrollmentv1.MarkEnrollmentPaidRequest{Id: e.GetId()})
	paidReq2.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.MarkEnrollmentPaid(ctx, paidReq2)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestEnrollment_Cancel_AlreadyCancelled_Rejected verifies S-09.
func TestEnrollment_Cancel_AlreadyCancelled_Rejected(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-sm-cancel-twice@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2102)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-sm-student3@enroll.test", "student")
	seedStudentProfile(t, studentID, 2102)

	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2102)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupEnrollment(t, e.GetId())

	// Cancel once — OK.
	cancelReq := connect.NewRequest(&enrollmentv1.CancelEnrollmentRequest{Id: e.GetId()})
	cancelReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.CancelEnrollment(ctx, cancelReq); err != nil {
		t.Fatalf("first cancel: %v", err)
	}

	// Cancel again — must fail with CodeFailedPrecondition.
	cancelReq2 := connect.NewRequest(&enrollmentv1.CancelEnrollmentRequest{Id: e.GetId()})
	cancelReq2.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.CancelEnrollment(ctx, cancelReq2)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestEnrollment_MarkPaid_AbsentID_NotFound verifies S-24.
func TestEnrollment_MarkPaid_AbsentID_NotFound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-sm-absent-paid@enroll.test", "admin")
	client := newEnrollmentClient(nil)

	paidReq := connect.NewRequest(&enrollmentv1.MarkEnrollmentPaidRequest{
		Id: "00000000-dead-beef-0000-000000000001",
	})
	paidReq.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.MarkEnrollmentPaid(ctx, paidReq)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestEnrollment_Get_SoftDeleted_NotFound verifies S-16.
func TestEnrollment_Get_SoftDeleted_NotFound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-sm-softdel@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2103)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-sm-student4@enroll.test", "student")
	seedStudentProfile(t, studentID, 2103)

	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2103)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Soft-delete directly via SQL (no RPC for this — simulates a maintenance operation).
	_, err = pgxPool.Exec(ctx, `UPDATE enrollments SET deleted_at = now() WHERE id = $1`, e.GetId())
	if err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, e.GetId())
	})

	getReq := connect.NewRequest(&enrollmentv1.GetEnrollmentRequest{Id: e.GetId()})
	getReq.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetEnrollment(ctx, getReq)
	assertConnectCode(t, err, connect.CodeNotFound)
}
