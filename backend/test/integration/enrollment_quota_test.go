package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1"
)

// TestEnrollment_QuotaFull_SingleThreaded verifies S-02: with capacity=1 and one active
// enrollment, a second create for a different student fails with CodeFailedPrecondition.
func TestEnrollment_QuotaFull_SingleThreaded(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-quota-full@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 1, 2110)
	defer cleanup()

	s1 := seedUserWithRole(t, "enroll-quota-s1@enroll.test", "student")
	s2 := seedUserWithRole(t, "enroll-quota-s2@enroll.test", "student")
	seedStudentProfile(t, s1, 2110)
	seedStudentProfile(t, s2, 2110)

	client := newEnrollmentClient(nil)

	// Fill the only seat.
	e1, err := enrollmentCreate(ctx, client, adminSID, s1.String(), programID, 2110)
	if err != nil {
		t.Fatalf("create s1: %v", err)
	}
	cleanupEnrollment(t, e1.GetId())

	// Second student should be denied.
	_, err = enrollmentCreate(ctx, client, adminSID, s2.String(), programID, 2110)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestEnrollment_NoQuotaRow_QuotaNotFound verifies S-13: when no program_quotas row exists
// for the (program_id, year) pair, CreateEnrollment returns CodeFailedPrecondition.
func TestEnrollment_NoQuotaRow_QuotaNotFound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-no-quota@enroll.test", "admin")

	// Create a bare program with no quota row.
	programID := seedBareProgram(t)
	studentID := seedUserWithRole(t, "enroll-no-quota-student@enroll.test", "student")
	seedStudentProfile(t, studentID, 2111)

	client := newEnrollmentClient(nil)
	_, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2111)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestEnrollment_Revival_HappyPath verifies S-10: a cancelled enrollment can be revived
// to pending, consuming a quota seat.
func TestEnrollment_Revival_HappyPath(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-revival@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 5, 2112)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-revival-student@enroll.test", "student")
	seedStudentProfile(t, studentID, 2112)

	client := newEnrollmentClient(nil)

	// Create, then cancel.
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2112)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cancelReq := connect.NewRequest(&enrollmentv1.CancelEnrollmentRequest{Id: e.GetId()})
	cancelReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.CancelEnrollment(ctx, cancelReq); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Revive (create again for same key).
	e2, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2112)
	if err != nil {
		t.Fatalf("revive create: %v", err)
	}
	cleanupEnrollment(t, e2.GetId())

	if e2.GetStatus() != "pending" {
		t.Errorf("revived status = %q, want pending", e2.GetStatus())
	}
	// The revived row reuses the same id (in-place update).
	if e2.GetId() != e.GetId() {
		t.Errorf("revival should reuse the same row: got %q, want %q", e2.GetId(), e.GetId())
	}
}

// TestEnrollment_Revival_QuotaFull verifies S-11: when reviving but quota is full,
// the cancelled row remains cancelled and CodeFailedPrecondition is returned.
func TestEnrollment_Revival_QuotaFull(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-revival-full@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 1, 2113)
	defer cleanup()

	s1 := seedUserWithRole(t, "enroll-revival-s1@enroll.test", "student")
	s2 := seedUserWithRole(t, "enroll-revival-s2@enroll.test", "student")
	seedStudentProfile(t, s1, 2113)
	seedStudentProfile(t, s2, 2113)

	client := newEnrollmentClient(nil)

	// s1 creates, then cancels — free seat for s2.
	e1, err := enrollmentCreate(ctx, client, adminSID, s1.String(), programID, 2113)
	if err != nil {
		t.Fatalf("create s1: %v", err)
	}
	cancelReq := connect.NewRequest(&enrollmentv1.CancelEnrollmentRequest{Id: e1.GetId()})
	cancelReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.CancelEnrollment(ctx, cancelReq); err != nil {
		t.Fatalf("cancel s1: %v", err)
	}

	// s2 takes the seat.
	e2, err := enrollmentCreate(ctx, client, adminSID, s2.String(), programID, 2113)
	if err != nil {
		t.Fatalf("create s2: %v", err)
	}
	cleanupEnrollment(t, e2.GetId())
	cleanupEnrollment(t, e1.GetId())

	// Now try to revive s1 (quota full with s2's active seat).
	_, err = enrollmentCreate(ctx, client, adminSID, s1.String(), programID, 2113)
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestEnrollment_DoubleEnroll_LiveKey_AlreadyExists verifies S-12.
func TestEnrollment_DoubleEnroll_LiveKey_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-double@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2114)
	defer cleanup()
	studentID := seedUserWithRole(t, "enroll-double-student@enroll.test", "student")
	seedStudentProfile(t, studentID, 2114)

	client := newEnrollmentClient(nil)
	e, err := enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2114)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	cleanupEnrollment(t, e.GetId())

	// Second create for the same key — must return CodeAlreadyExists.
	_, err = enrollmentCreate(ctx, client, adminSID, studentID.String(), programID, 2114)
	assertConnectCode(t, err, connect.CodeAlreadyExists)
}

// TestEnrollment_BadStudentFK_InvalidArgument verifies S-14.
func TestEnrollment_BadStudentFK_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-bad-fk@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, 10, 2115)
	defer cleanup()

	client := newEnrollmentClient(nil)
	// Use a random UUID that has no student_profiles row (only a UUID, not seeded).
	_, err := enrollmentCreate(ctx, client, adminSID, uuid.New().String(), programID, 2115)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}
