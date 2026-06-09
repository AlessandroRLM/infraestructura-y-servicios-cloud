package integration_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1/enrollmentv1connect"
)

// newEnrollmentClient returns a Connect EnrollmentService client targeting the shared test server.
func newEnrollmentClient(jar http.CookieJar) enrollmentv1connect.EnrollmentServiceClient {
	return enrollmentv1connect.NewEnrollmentServiceClient(&http.Client{Jar: jar}, baseURL)
}

// TestEnrollment_Unauthenticated_CodeUnauthenticated verifies that a caller without
// a session cookie receives CodeUnauthenticated on any enrollment RPC.
func TestEnrollment_Unauthenticated_CodeUnauthenticated(t *testing.T) {
	client := newEnrollmentClient(nil)
	ctx := context.Background()

	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{})
	_, err := client.ListEnrollments(ctx, req)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestEnrollment_StudentRole_ManageRPC_PermissionDenied verifies that a student
// (who has enrollment.view_own but NOT enrollment.manage) is denied on management RPCs.
func TestEnrollment_StudentRole_ManageRPC_PermissionDenied(t *testing.T) {
	_, studentSID := seedUserWithSession(t, "enroll-authz-student@enroll.test", "student")
	client := newEnrollmentClient(nil)
	ctx := context.Background()

	managementCases := []struct {
		name string
		call func() error
	}{
		{
			name: "CreateEnrollment",
			call: func() error {
				req := connect.NewRequest(&enrollmentv1.CreateEnrollmentRequest{})
				req.Header().Set("Cookie", "sid="+studentSID)
				_, err := client.CreateEnrollment(ctx, req)
				return err
			},
		},
		{
			name: "MarkEnrollmentPaid",
			call: func() error {
				req := connect.NewRequest(&enrollmentv1.MarkEnrollmentPaidRequest{Id: "00000000-0000-0000-0000-000000000001"})
				req.Header().Set("Cookie", "sid="+studentSID)
				_, err := client.MarkEnrollmentPaid(ctx, req)
				return err
			},
		},
		{
			name: "CancelEnrollment",
			call: func() error {
				req := connect.NewRequest(&enrollmentv1.CancelEnrollmentRequest{Id: "00000000-0000-0000-0000-000000000001"})
				req.Header().Set("Cookie", "sid="+studentSID)
				_, err := client.CancelEnrollment(ctx, req)
				return err
			},
		},
		{
			name: "GetEnrollment",
			call: func() error {
				req := connect.NewRequest(&enrollmentv1.GetEnrollmentRequest{Id: "00000000-0000-0000-0000-000000000001"})
				req.Header().Set("Cookie", "sid="+studentSID)
				_, err := client.GetEnrollment(ctx, req)
				return err
			},
		},
		{
			name: "ListEnrollments",
			call: func() error {
				req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{})
				req.Header().Set("Cookie", "sid="+studentSID)
				_, err := client.ListEnrollments(ctx, req)
				return err
			},
		},
	}

	for _, tc := range managementCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assertConnectCode(t, tc.call(), connect.CodePermissionDenied)
		})
	}
}

// TestEnrollment_Admin_BasicCreateAllowed verifies that an admin with enrollment.manage
// can create an enrollment (basic happy-path authz check; full CRUD is in crud_test.go).
func TestEnrollment_Admin_BasicCreateAllowed(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-authz-admin@enroll.test", "admin")

	programID, quotaCleanup := seedProgramWithQuota(t, 30, 2090)
	defer quotaCleanup()

	studentUserID := seedUserWithRole(t, "enroll-authz-target@enroll.test", "student")
	seedStudentProfile(t, studentUserID, 2090)

	client := newEnrollmentClient(nil)
	req := connect.NewRequest(&enrollmentv1.CreateEnrollmentRequest{
		StudentId: studentUserID.String(),
		ProgramId: programID,
		Year:      2090,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.CreateEnrollment(ctx, req)
	if err != nil {
		t.Fatalf("CreateEnrollment as admin: %v", err)
	}
	cleanupEnrollment(t, resp.Msg.GetId())
}
