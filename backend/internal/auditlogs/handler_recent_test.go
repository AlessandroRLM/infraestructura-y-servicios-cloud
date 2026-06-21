package auditlogs

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
)

// fakeAuditServiceRecent stubs the handler's service interface for ListRecentAuditLogs.
type fakeAuditServiceRecent struct {
	fakeAuditService

	recentCalled bool
	recentResp   *auditlogsv1.ListRecentAuditLogsResponse
	recentErr    error
}

func (f *fakeAuditServiceRecent) ListAuditLogs(
	ctx context.Context,
	req *auditlogsv1.ListAuditLogsRequest,
) (*auditlogsv1.ListAuditLogsResponse, error) {
	return f.fakeAuditService.ListAuditLogs(ctx, req)
}

func (f *fakeAuditServiceRecent) ListRecentAuditLogs(
	ctx context.Context,
	req *auditlogsv1.ListRecentAuditLogsRequest,
) (*auditlogsv1.ListRecentAuditLogsResponse, error) {
	f.recentCalled = true
	if f.recentResp != nil {
		return f.recentResp, f.recentErr
	}
	return &auditlogsv1.ListRecentAuditLogsResponse{}, f.recentErr
}

// TestHandler_ListRecentAuditLogs_MalformedActorID_ReturnsCodeInvalidArgument verifies
// that a non-UUID actor_id returns CodeInvalidArgument without calling the service.
func TestHandler_ListRecentAuditLogs_MalformedActorID_ReturnsCodeInvalidArgument(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditServiceRecent{}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		ActorId: "not-a-uuid",
	})
	_, err := h.ListRecentAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInvalidArgument)
	if svc.recentCalled {
		t.Error("service must NOT be called when actor_id is malformed")
	}
}

// TestHandler_ListRecentAuditLogs_MalformedPageToken_ReturnsCodeInvalidArgument verifies
// that a non-empty invalid page_token returns CodeInvalidArgument without calling the service.
func TestHandler_ListRecentAuditLogs_MalformedPageToken_ReturnsCodeInvalidArgument(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditServiceRecent{}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		PageToken: "garbage",
	})
	_, err := h.ListRecentAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInvalidArgument)
	if svc.recentCalled {
		t.Error("service must NOT be called when page_token is malformed")
	}
}

// TestHandler_ListRecentAuditLogs_ValidRequest_DelegatesToService verifies that a
// minimal valid request calls the service and proxies the response.
func TestHandler_ListRecentAuditLogs_ValidRequest_DelegatesToService(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditServiceRecent{
		recentResp: &auditlogsv1.ListRecentAuditLogsResponse{
			Logs: []*auditlogsv1.AuditLog{{Id: uuid.New().String()}},
		},
	}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	})
	resp, err := h.ListRecentAuditLogs(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svc.recentCalled {
		t.Error("service must be called for valid request")
	}
	if len(resp.Msg.Logs) != 1 {
		t.Errorf("expected 1 log row proxied from service, got %d", len(resp.Msg.Logs))
	}
}

// TestHandler_ListRecentAuditLogs_ValidActorID_DelegatesToService verifies that a valid
// actor_id is accepted and delegated to the service.
func TestHandler_ListRecentAuditLogs_ValidActorID_DelegatesToService(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditServiceRecent{}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		ActorId:  uuid.New().String(),
		PageSize: 20,
	})
	_, err := h.ListRecentAuditLogs(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svc.recentCalled {
		t.Error("service must be called for valid actor_id")
	}
}

// TestHandler_ListRecentAuditLogs_ServiceErrInvalidInput_MapsToCodeInvalidArgument verifies
// that ErrInvalidInput from the service maps to CodeInvalidArgument.
func TestHandler_ListRecentAuditLogs_ServiceErrInvalidInput_MapsToCodeInvalidArgument(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditServiceRecent{
		recentErr: fmt.Errorf("%w: created_from is not a valid RFC3339 timestamp", ErrInvalidInput),
	}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		CreatedFrom: "not-a-date",
	})
	_, err := h.ListRecentAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInvalidArgument)
}

// TestHandler_ListRecentAuditLogs_ServiceUnknownErr_MapsToCodeInternal verifies that an
// unknown service error maps to CodeInternal with a generic message (no leak).
func TestHandler_ListRecentAuditLogs_ServiceUnknownErr_MapsToCodeInternal(t *testing.T) {
	t.Parallel()

	secretMsg := "secret internal detail"
	svc := &fakeAuditServiceRecent{recentErr: errors.New(secretMsg)}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	})
	_, err := h.ListRecentAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInternal)

	if connectErr, ok := errors.AsType[*connect.Error](err); ok {
		if connectErr.Message() == secretMsg {
			t.Error("handler must not leak the original error message for internal errors")
		}
	}
}
