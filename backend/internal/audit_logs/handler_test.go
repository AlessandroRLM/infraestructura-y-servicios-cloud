package audit_logs

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
)

// fakeAuditService stubs the handler's service dependency.
type fakeAuditService struct {
	called bool
	resp   *auditlogsv1.ListAuditLogsResponse
	err    error
}

func (f *fakeAuditService) ListAuditLogs(
	_ context.Context,
	_ *auditlogsv1.ListAuditLogsRequest,
) (*auditlogsv1.ListAuditLogsResponse, error) {
	f.called = true
	if f.resp != nil {
		return f.resp, f.err
	}
	return &auditlogsv1.ListAuditLogsResponse{}, f.err
}

// TestHandler_ListAuditLogs_MissingEntity_ReturnsCodeInvalidArgument verifies that
// an empty entity returns CodeInvalidArgument without calling the service.
func TestHandler_ListAuditLogs_MissingEntity_ReturnsCodeInvalidArgument(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditService{}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "",
		EntityId: uuid.New().String(),
	})
	_, err := h.ListAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInvalidArgument)
	if svc.called {
		t.Error("service must NOT be called when entity is empty")
	}
}

// TestHandler_ListAuditLogs_MalformedEntityID verifies that a non-UUID entity_id
// returns CodeInvalidArgument without calling the service.
func TestHandler_ListAuditLogs_MalformedEntityID(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditService{}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: "not-a-uuid",
	})
	_, err := h.ListAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInvalidArgument)
	if svc.called {
		t.Error("service must NOT be called when entity_id is malformed")
	}
}

// TestHandler_ListAuditLogs_MalformedPageToken verifies that an invalid page_token
// (non-empty, non-UUID) returns CodeInvalidArgument without calling the service.
func TestHandler_ListAuditLogs_MalformedPageToken(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditService{}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:    "grades",
		EntityId:  uuid.New().String(),
		PageToken: "not-a-uuid",
	})
	_, err := h.ListAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInvalidArgument)
	if svc.called {
		t.Error("service must NOT be called when page_token is malformed")
	}
}

// TestHandler_ListAuditLogs_MalformedActorID verifies that a non-empty invalid actor_id
// returns CodeInvalidArgument without calling the service.
func TestHandler_ListAuditLogs_MalformedActorID(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditService{}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
		ActorId:  "bad-uuid",
	})
	_, err := h.ListAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInvalidArgument)
	if svc.called {
		t.Error("service must NOT be called when actor_id is malformed")
	}
}

// TestHandler_ListAuditLogs_BadCreatedAtRange verifies that a malformed created_from
// is passed to the service (timestamp parsing is service-level), and the service
// returns ErrInvalidInput which the handler maps to CodeInvalidArgument.
func TestHandler_ListAuditLogs_BadCreatedAtRange(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditService{
		err: fmt.Errorf("%w: created_from is not a valid RFC3339 timestamp", ErrInvalidInput),
	}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:      "grades",
		EntityId:    uuid.New().String(),
		CreatedFrom: "not-a-date",
	})
	_, err := h.ListAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInvalidArgument)
	if !svc.called {
		t.Error("service MUST be called for timestamp parsing (service-level validation)")
	}
}

// TestHandler_ListAuditLogs_ValidRequest_DelegatesToService verifies that a well-formed
// request with entity+entity_id calls the service and proxies the response correctly.
func TestHandler_ListAuditLogs_ValidRequest_DelegatesToService(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditService{
		resp: &auditlogsv1.ListAuditLogsResponse{
			Logs: []*auditlogsv1.AuditLog{{Id: uuid.New().String()}},
		},
	}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
	})
	resp, err := h.ListAuditLogs(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svc.called {
		t.Error("service must be called for valid request")
	}
	if len(resp.Msg.Logs) != 1 {
		t.Errorf("expected 1 log row proxied from service, got %d", len(resp.Msg.Logs))
	}
}

// TestHandler_ListAuditLogs_ServiceErrNotFound_MapsToCodeNotFound verifies that
// ErrNotFound from the service maps to CodeNotFound.
func TestHandler_ListAuditLogs_ServiceErrNotFound_MapsToCodeNotFound(t *testing.T) {
	t.Parallel()

	svc := &fakeAuditService{err: ErrNotFound}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
	})
	_, err := h.ListAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeNotFound)
}

// TestHandler_ListAuditLogs_ServiceUnknownErr_MapsToCodeInternal verifies that an unknown
// service error maps to CodeInternal with a generic message (no leak).
func TestHandler_ListAuditLogs_ServiceUnknownErr_MapsToCodeInternal(t *testing.T) {
	t.Parallel()

	secretMsg := "secret internal detail"
	svc := &fakeAuditService{err: errors.New(secretMsg)}
	h := NewHandler(svc)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
	})
	_, err := h.ListAuditLogs(context.Background(), req)
	assertConnectErr(t, err, connect.CodeInternal)

	if connectErr, ok := errors.AsType[*connect.Error](err); ok {
		if connectErr.Message() == secretMsg {
			t.Error("handler must not leak the original error message for internal errors")
		}
	}
}

// assertConnectErr is a helper that asserts err is a *connect.Error with the expected code.
func assertConnectErr(t *testing.T, err error, wantCode connect.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a connect error with code %v, got nil", wantCode)
	}
	connectErr, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if connectErr.Code() != wantCode {
		t.Errorf("connect error code = %v, want %v", connectErr.Code(), wantCode)
	}
}
