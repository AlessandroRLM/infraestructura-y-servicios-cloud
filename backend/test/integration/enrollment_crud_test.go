package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1/enrollmentv1connect"
)

// TestEnrollment_Create_Pending verifies that CreateEnrollment returns status=pending
// with the correct student, program, and year, and that audit columns are set to the
// acting admin's user id on insert.
func TestEnrollment_Create_Pending(t *testing.T) {
	ctx := context.Background()
	adminID, adminSID := seedUserWithSession(t, "enroll-crud-admin-create@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 30, 2091)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-crud-student-create@enroll.test", "student")
	seedStudentProfile(t, studentID, 2091)

	client := newEnrollmentClient(nil)
	req := connect.NewRequest(&enrollmentv1.CreateEnrollmentRequest{
		StudentId: studentID.String(),
		ProgramId: programID,
		Year:      2091,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.CreateEnrollment(ctx, req)
	if err != nil {
		t.Fatalf("CreateEnrollment: %v", err)
	}
	cleanupEnrollment(t, resp.Msg.GetId())

	if resp.Msg.GetStatus() != "pending" {
		t.Errorf("status = %q, want pending", resp.Msg.GetStatus())
	}
	if resp.Msg.GetStudentId() != studentID.String() {
		t.Errorf("student_id = %q, want %q", resp.Msg.GetStudentId(), studentID.String())
	}
	if resp.Msg.PaidAt != nil {
		t.Error("paid_at should be nil on create")
	}

	// Assert audit columns: both created_by and updated_by must be the acting admin.
	var createdBy, updatedBy uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`SELECT created_by, updated_by FROM enrollments WHERE id = $1`,
		resp.Msg.GetId(),
	).Scan(&createdBy, &updatedBy); err != nil {
		t.Fatalf("SELECT audit columns: %v", err)
	}
	if createdBy != adminID {
		t.Errorf("created_by = %v, want admin user_id %v", createdBy, adminID)
	}
	if updatedBy != adminID {
		t.Errorf("updated_by = %v, want admin user_id %v", updatedBy, adminID)
	}
}

// TestEnrollment_MarkPaid_SetsStatusAndPaidAt verifies S-04.
func TestEnrollment_MarkPaid_SetsStatusAndPaidAt(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-crud-admin-paid@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 30, 2092)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-crud-student-paid@enroll.test", "student")
	seedStudentProfile(t, studentID, 2092)

	client := newEnrollmentClient(nil)
	createResp, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2092)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupEnrollment(t, createResp.GetId())

	paidReq := connect.NewRequest(&enrollmentv1.MarkEnrollmentPaidRequest{Id: createResp.GetId()})
	paidReq.Header().Set("Cookie", "sid="+adminSID)
	paidResp, err := client.MarkEnrollmentPaid(ctx, paidReq)
	if err != nil {
		t.Fatalf("MarkEnrollmentPaid: %v", err)
	}
	if paidResp.Msg.GetStatus() != "paid" {
		t.Errorf("status = %q, want paid", paidResp.Msg.GetStatus())
	}
	if paidResp.Msg.PaidAt == nil {
		t.Error("paid_at should be non-nil after MarkEnrollmentPaid")
	}
}

// TestEnrollment_Cancel_FreeSeat verifies that cancelling an enrollment frees
// the seat, allowing a subsequent create to succeed when quota was full. (S-07)
func TestEnrollment_Cancel_FreeSeat(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-crud-admin-cancel@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 1, 2093)
	defer cleanup()

	studentA := seedUserWithRole(t, "enroll-crud-studentA@enroll.test", "student")
	studentB := seedUserWithRole(t, "enroll-crud-studentB@enroll.test", "student")
	seedStudentProfile(t, studentA, 2093)
	seedStudentProfile(t, studentB, 2093)

	client := newEnrollmentClient(nil)

	// Fill the only seat.
	enrollA, err := enrollmentCreate(ctx, client, adminSID, studentA.String(), programID, 2093)
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	cleanupEnrollment(t, enrollA.GetId())

	// Seat is full — studentB should fail.
	_, err = enrollmentCreate(ctx, client, adminSID, studentB.String(), programID, 2093)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	// Cancel studentA's enrollment.
	cancelReq := connect.NewRequest(&enrollmentv1.CancelEnrollmentRequest{Id: enrollA.GetId()})
	cancelReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.CancelEnrollment(ctx, cancelReq); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Now studentB can enroll.
	enrollB, err := enrollmentCreate(ctx, client, adminSID, studentB.String(), programID, 2093)
	if err != nil {
		t.Fatalf("create B after cancel: %v", err)
	}
	cleanupEnrollment(t, enrollB.GetId())
}

// TestEnrollment_CancelPaid verifies that a paid enrollment can be cancelled. (S-08)
func TestEnrollment_CancelPaid(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-crud-admin-cpaid@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 5, 2094)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-crud-student-cpaid@enroll.test", "student")
	seedStudentProfile(t, studentID, 2094)

	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2094)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupEnrollment(t, e.GetId())

	// Mark paid first.
	paidReq := connect.NewRequest(&enrollmentv1.MarkEnrollmentPaidRequest{Id: e.GetId()})
	paidReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.MarkEnrollmentPaid(ctx, paidReq); err != nil {
		t.Fatalf("mark paid: %v", err)
	}

	// Now cancel the paid enrollment.
	cancelReq := connect.NewRequest(&enrollmentv1.CancelEnrollmentRequest{Id: e.GetId()})
	cancelReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.CancelEnrollment(ctx, cancelReq); err != nil {
		t.Fatalf("cancel paid: %v", err)
	}
}

// TestEnrollment_GetEnrollment_Admin verifies that an admin can read any live enrollment. (S-15)
func TestEnrollment_GetEnrollment_Admin(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-crud-admin-get@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2095)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-crud-student-get@enroll.test", "student")
	seedStudentProfile(t, studentID, 2095)

	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2095)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupEnrollment(t, e.GetId())

	getReq := connect.NewRequest(&enrollmentv1.GetEnrollmentRequest{Id: e.GetId()})
	getReq.Header().Set("Cookie", "sid="+adminSID)
	got, err := client.GetEnrollment(ctx, getReq)
	if err != nil {
		t.Fatalf("GetEnrollment: %v", err)
	}
	if got.Msg.GetId() != e.GetId() {
		t.Errorf("id = %q, want %q", got.Msg.GetId(), e.GetId())
	}
}

// TestEnrollment_ListEnrollments_StudentIDFilter verifies S-17.
func TestEnrollment_ListEnrollments_StudentIDFilter(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-crud-admin-list@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2096)
	defer cleanup()

	s1 := seedUserWithRole(t, "enroll-crud-s1@enroll.test", "student")
	s2 := seedUserWithRole(t, "enroll-crud-s2@enroll.test", "student")
	seedStudentProfile(t, s1, 2096)
	seedStudentProfile(t, s2, 2096)

	client := newEnrollmentClient(nil)
	e1, err := enrollmentCreate(ctx, client, adminSID, s1.String(), programID, 2096)
	if err != nil {
		t.Fatalf("create s1: %v", err)
	}
	cleanupEnrollment(t, e1.GetId())

	e2, err := enrollmentCreate(ctx, client, adminSID, s2.String(), programID, 2096)
	if err != nil {
		t.Fatalf("create s2: %v", err)
	}
	cleanupEnrollment(t, e2.GetId())

	// Filter by s1's ID — only s1's enrollment should appear.
	listReq := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{StudentId: s1.String()})
	listReq.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListEnrollments(ctx, listReq)
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}
	for _, row := range resp.Msg.GetEnrollments() {
		if row.GetStudentId() != s1.String() {
			t.Errorf("filter by s1: got student_id %q, want %q", row.GetStudentId(), s1.String())
		}
	}
	found := false
	for _, row := range resp.Msg.GetEnrollments() {
		if row.GetId() == e1.GetId() {
			found = true
		}
	}
	if !found {
		t.Error("ListEnrollments: s1's enrollment not in filtered result")
	}
}

// TestEnrollment_EmptyList_ReturnsOK verifies S-25.
func TestEnrollment_EmptyList_ReturnsOK(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-crud-admin-empty@enroll.test", "admin")
	client := newEnrollmentClient(nil)

	// Use a non-existent student_id to ensure the list is empty.
	listReq := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{
		StudentId: uuid.New().String(),
	})
	listReq.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListEnrollments(ctx, listReq)
	if err != nil {
		t.Fatalf("ListEnrollments empty: unexpected error %v", err)
	}
	if len(resp.Msg.GetEnrollments()) != 0 {
		t.Errorf("expected empty list, got %d rows", len(resp.Msg.GetEnrollments()))
	}
}

// --- Helpers ---

// enrollmentCreate is a convenience helper that creates an enrollment and returns the proto.
func enrollmentCreate(ctx context.Context, client enrollmentv1connect.EnrollmentServiceClient, adminSID, studentID, programID string, year int32) (*enrollmentv1.Enrollment, error) {
	req := connect.NewRequest(&enrollmentv1.CreateEnrollmentRequest{
		StudentId: studentID,
		ProgramId: programID,
		Year:      year,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.CreateEnrollment(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// cleanupEnrollment deletes an enrollment row by id string.
func cleanupEnrollment(t *testing.T, id string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, id)
	})
}
