package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
)

// validRecentAuditLogsRequest returns a minimal valid ListRecentAuditLogsRequest.
func validRecentAuditLogsRequest() *auditlogsv1.ListRecentAuditLogsRequest {
	return &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	}
}

// TestRecentAuditLogs_Authz_AdminWithAuditRead_Returns200 verifies that an admin session
// (which holds audit.read via the admin role) can call ListRecentAuditLogs and receives
// HTTP 200. An empty log list is acceptable.
func TestRecentAuditLogs_Authz_AdminWithAuditRead_Returns200(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-authz-admin@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(validRecentAuditLogsRequest())
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListRecentAuditLogs(ctx, req)
	if err != nil {
		t.Fatalf("expected success for admin with audit.read, got: %v", err)
	}
	if resp.Msg == nil {
		t.Error("expected non-nil response body")
	}
}

// TestRecentAuditLogs_Authz_Teacher_ReturnsPermissionDenied verifies that a teacher session
// (which does NOT hold audit.read) receives CodePermissionDenied.
func TestRecentAuditLogs_Authz_Teacher_ReturnsPermissionDenied(t *testing.T) {
	ctx := context.Background()
	_, teacherSID := seedTeacherProfile(t, "recent-authz-teacher@audit.test")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(validRecentAuditLogsRequest())
	req.Header().Set("Cookie", "sid="+teacherSID)

	_, err := client.ListRecentAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestRecentAuditLogs_Authz_Student_ReturnsPermissionDenied verifies that a student session
// (which does NOT hold audit.read) receives CodePermissionDenied.
func TestRecentAuditLogs_Authz_Student_ReturnsPermissionDenied(t *testing.T) {
	ctx := context.Background()
	_, studentSID := seedUserWithSession(t, "recent-authz-student@audit.test", "student")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(validRecentAuditLogsRequest())
	req.Header().Set("Cookie", "sid="+studentSID)

	_, err := client.ListRecentAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestRecentAuditLogs_Authz_NoSession_ReturnsUnauthenticated verifies that a request
// with no session cookie is rejected with CodeUnauthenticated.
func TestRecentAuditLogs_Authz_NoSession_ReturnsUnauthenticated(t *testing.T) {
	ctx := context.Background()
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(validRecentAuditLogsRequest())

	_, err := client.ListRecentAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}
