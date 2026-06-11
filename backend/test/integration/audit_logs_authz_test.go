package integration_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1/auditlogsv1connect"
)

// newAuditLogsClient returns a Connect AuditLogsService client targeting the shared test server.
func newAuditLogsClient(jar http.CookieJar) auditlogsv1connect.AuditLogsServiceClient {
	return auditlogsv1connect.NewAuditLogsServiceClient(&http.Client{Jar: jar}, baseURL)
}

// validAuditLogsRequest returns a minimal valid ListAuditLogsRequest for use in authz tests.
// Uses a random entity_id so tests are isolated even without actual audit rows.
func validAuditLogsRequest() *auditlogsv1.ListAuditLogsRequest {
	return &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
	}
}

// TestAuditLogs_Authz_AdminWithAuditRead_Returns200 verifies that an admin session
// (which holds audit.read via the admin role) can call ListAuditLogs and receives
// HTTP 200. An empty log list is acceptable (no audit rows required for this test).
func TestAuditLogs_Authz_AdminWithAuditRead_Returns200(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-authz-admin@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(validAuditLogsRequest())
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListAuditLogs(ctx, req)
	if err != nil {
		t.Fatalf("expected success for admin with audit.read, got: %v", err)
	}
	// Empty list is valid — no audit rows exist for the random entity_id.
	if resp.Msg == nil {
		t.Error("expected non-nil response body")
	}
}

// TestAuditLogs_Authz_Teacher_ReturnsPermissionDenied verifies that a teacher session
// (which does NOT hold audit.read) receives CodePermissionDenied from the interceptor.
func TestAuditLogs_Authz_Teacher_ReturnsPermissionDenied(t *testing.T) {
	ctx := context.Background()
	_, teacherSID := seedTeacherProfile(t, "audit-authz-teacher@audit.test")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(validAuditLogsRequest())
	req.Header().Set("Cookie", "sid="+teacherSID)

	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestAuditLogs_Authz_Student_ReturnsPermissionDenied verifies that a student session
// (which does NOT hold audit.read) receives CodePermissionDenied from the interceptor.
func TestAuditLogs_Authz_Student_ReturnsPermissionDenied(t *testing.T) {
	ctx := context.Background()
	_, studentSID := seedUserWithSession(t, "audit-authz-student@audit.test", "student")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(validAuditLogsRequest())
	req.Header().Set("Cookie", "sid="+studentSID)

	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestAuditLogs_Authz_NoSession_ReturnsUnauthenticated verifies that a request
// with no session cookie is rejected with CodeUnauthenticated by the auth interceptor.
func TestAuditLogs_Authz_NoSession_ReturnsUnauthenticated(t *testing.T) {
	ctx := context.Background()
	client := newAuditLogsClient(nil)

	// No Cookie header — unauthenticated request.
	req := connect.NewRequest(validAuditLogsRequest())

	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}
