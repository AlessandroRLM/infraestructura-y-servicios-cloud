package auth

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

// Handler implements authv1connect.AuthServiceHandler.
type Handler struct {
	svc *Service
	cfg config.Config
}

// NewHandler constructs a Connect handler wrapping the AuthService.
func NewHandler(svc *Service, cfg config.Config) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// Register mounts the AuthService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := authv1connect.NewAuthServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// Login handles credential validation and mints a session cookie on success.
func (h *Handler) Login(
	ctx context.Context,
	req *connect.Request[authv1.LoginRequest],
) (*connect.Response[authv1.LoginResponse], error) {
	result, err := h.svc.Login(ctx, req.Msg.GetEmail(), req.Msg.GetPassword())
	if err != nil {
		return nil, mapError(err)
	}

	resp := connect.NewResponse(&authv1.LoginResponse{})
	resp.Header().Set("Set-Cookie", BuildCookie(result.SessionID, h.cfg).String())
	return resp, nil
}

// Logout deletes the session and clears the cookie.
func (h *Handler) Logout(
	ctx context.Context,
	req *connect.Request[authv1.LogoutRequest],
) (*connect.Response[authv1.LogoutResponse], error) {
	sid := parseSID(req.Header())
	if sid == "" {
		// Nothing to invalidate — idempotent success.
		resp := connect.NewResponse(&authv1.LogoutResponse{})
		resp.Header().Set("Set-Cookie", ClearCookie(h.cfg).String())
		return resp, nil
	}

	if err := h.svc.Logout(ctx, sid); err != nil {
		return nil, mapError(err)
	}

	resp := connect.NewResponse(&authv1.LogoutResponse{})
	resp.Header().Set("Set-Cookie", ClearCookie(h.cfg).String())
	return resp, nil
}

// RequestPasswordReset initiates the password-reset flow.
func (h *Handler) RequestPasswordReset(
	ctx context.Context,
	req *connect.Request[authv1.RequestPasswordResetRequest],
) (*connect.Response[authv1.RequestPasswordResetResponse], error) {
	result, err := h.svc.RequestPasswordReset(ctx, req.Msg.GetEmail())
	if err != nil {
		return nil, mapError(err)
	}

	msg := &authv1.RequestPasswordResetResponse{}
	if result.DevToken != "" {
		msg.DevToken = &result.DevToken
	}
	return connect.NewResponse(msg), nil
}

// ConfirmPasswordReset consumes a reset token and updates the user's password.
func (h *Handler) ConfirmPasswordReset(
	ctx context.Context,
	req *connect.Request[authv1.ConfirmPasswordResetRequest],
) (*connect.Response[authv1.ConfirmPasswordResetResponse], error) {
	if err := h.svc.ConfirmPasswordReset(ctx, req.Msg.GetToken(), req.Msg.GetNewPassword()); err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&authv1.ConfirmPasswordResetResponse{}), nil
}

// mapError converts domain errors to connect.Error codes.
func mapError(err error) error {
	if errors.Is(err, ErrInvalidCredentials) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}
	if errors.Is(err, ErrInvalidInput) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, ErrInvalidToken) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
