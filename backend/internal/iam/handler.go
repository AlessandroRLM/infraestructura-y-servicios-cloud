package iam

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	iamv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/iam/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/iam/v1/iamv1connect"
)

// iamService is the handler's consumer-side interface so unit tests can inject fakes.
type iamService interface {
	ListUsers(ctx context.Context, pageSize int32, pageToken string, query string) (ListUsersResult, error)
	GetUser(ctx context.Context, userID uuid.UUID) (UserSummary, error)
}

// Compile-time proof that *Service satisfies iamService.
var _ iamService = (*Service)(nil)

// Handler implements iamv1connect.IamServiceHandler.
type Handler struct {
	svc iamService
}

// NewHandler constructs a Connect handler wrapping the IamService.
func NewHandler(svc iamService) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the IamService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := iamv1connect.NewIamServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// ListUsers validates the request and delegates to the service.
func (h *Handler) ListUsers(
	ctx context.Context,
	req *connect.Request[iamv1.ListUsersRequest],
) (*connect.Response[iamv1.ListUsersResponse], error) {
	result, err := h.svc.ListUsers(ctx, req.Msg.GetPageSize(), req.Msg.GetPageToken(), req.Msg.GetQuery())
	if err != nil {
		return nil, MapError(ctx, err)
	}

	protoUsers := make([]*iamv1.UserSummary, 0, len(result.Users))
	for _, u := range result.Users {
		protoUsers = append(protoUsers, userSummaryToProto(u))
	}

	return connect.NewResponse(&iamv1.ListUsersResponse{
		Users:         protoUsers,
		NextPageToken: result.NextPageToken,
	}), nil
}

// GetUser parses the user_id UUID and delegates to the service.
func (h *Handler) GetUser(
	ctx context.Context,
	req *connect.Request[iamv1.GetUserRequest],
) (*connect.Response[iamv1.GetUserResponse], error) {
	userID, err := parseIAMUUID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}

	summary, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		return nil, MapError(ctx, err)
	}

	return connect.NewResponse(&iamv1.GetUserResponse{
		User: userSummaryToProto(summary),
	}), nil
}

// parseIAMUUID parses a UUID string, returning CodeInvalidArgument on failure.
func parseIAMUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("iam: %s is not a valid UUID: %q", field, value),
		)
	}
	return id, nil
}

// userSummaryToProto converts a domain UserSummary to the proto wire message.
func userSummaryToProto(u UserSummary) *iamv1.UserSummary {
	status := iamv1.UserStatus_USER_STATUS_ACTIVE
	if u.Disabled {
		status = iamv1.UserStatus_USER_STATUS_DISABLED
	}
	return &iamv1.UserSummary{
		Id:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Roles:       u.Roles,
		Status:      status,
	}
}
