package auditlogs

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1/auditlogsv1connect"
)

// auditLogsService is the handler's consumer-side interface so unit tests can inject fakes.
type auditLogsService interface {
	ListAuditLogs(ctx context.Context, req *auditlogsv1.ListAuditLogsRequest) (*auditlogsv1.ListAuditLogsResponse, error)
	ListRecentAuditLogs(ctx context.Context, req *auditlogsv1.ListRecentAuditLogsRequest) (*auditlogsv1.ListRecentAuditLogsResponse, error)
}

// Compile-time proof that *Service satisfies auditLogsService.
var _ auditLogsService = (*Service)(nil)

// Handler implements auditlogsv1connect.AuditLogsServiceHandler.
type Handler struct {
	svc auditLogsService
}

// NewHandler constructs a Connect handler wrapping the AuditLogsService.
func NewHandler(svc auditLogsService) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the AuditLogsService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := auditlogsv1connect.NewAuditLogsServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// ListAuditLogs validates required UUID fields and delegates to the service.
func (h *Handler) ListAuditLogs(
	ctx context.Context,
	req *connect.Request[auditlogsv1.ListAuditLogsRequest],
) (*connect.Response[auditlogsv1.ListAuditLogsResponse], error) {
	// entity must be non-empty — guard before UUID parsing to give a clear error.
	if req.Msg.GetEntity() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("audit_logs: entity is required"))
	}

	// entity_id is required and must be a valid UUID.
	if _, err := parseAuditUUID("entity_id", req.Msg.GetEntityId()); err != nil {
		return nil, err
	}

	// page_token is optional; if non-empty it must be a valid UUID.
	if pt := req.Msg.GetPageToken(); pt != "" {
		if _, err := parseAuditUUID("page_token", pt); err != nil {
			return nil, err
		}
	}

	// actor_id is optional; if non-empty it must be a valid UUID.
	if a := req.Msg.GetActorId(); a != "" {
		if _, err := parseAuditUUID("actor_id", a); err != nil {
			return nil, err
		}
	}

	// Delegate to service — timestamp parsing and remaining validation happen there.
	resp, err := h.svc.ListAuditLogs(ctx, req.Msg)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(resp), nil
}

// ListRecentAuditLogs validates optional UUID fields and delegates to the service.
// No entity/entity_id required — this is the global feed RPC.
func (h *Handler) ListRecentAuditLogs(
	ctx context.Context,
	req *connect.Request[auditlogsv1.ListRecentAuditLogsRequest],
) (*connect.Response[auditlogsv1.ListRecentAuditLogsResponse], error) {
	// actor_id is optional; if non-empty it must be a valid UUID.
	if a := req.Msg.GetActorId(); a != "" {
		if _, err := parseAuditUUID("actor_id", a); err != nil {
			return nil, err
		}
	}

	// page_token is optional; if non-empty it must parse as a valid cursor (validated by service).
	// Handler does a lightweight check: the service owns cursor format; we only reject completely
	// empty-but-whitespace tokens.
	if pt := req.Msg.GetPageToken(); pt != "" {
		// Validate early by delegating cursor parsing — returns ErrInvalidInput on failure.
		_, _, parseErr := decodeRecentCursor(pt)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("audit_logs: page_token is not a valid cursor"))
		}
	}

	// Delegate to service — timestamp and actor_id parsing happen there.
	resp, err := h.svc.ListRecentAuditLogs(ctx, req.Msg)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(resp), nil
}

// parseAuditUUID parses a string UUID, returning a Connect CodeInvalidArgument error on failure.
// The field parameter names the field for error context (e.g. "entity_id").
func parseAuditUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("audit_logs: %s is not a valid UUID: %q", field, value),
		)
	}
	return id, nil
}
