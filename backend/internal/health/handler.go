// Package health implements the HealthService Connect RPC handler.
package health

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	healthv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/health/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/health/v1/healthv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
)

// Handler implements healthv1connect.HealthServiceHandler.
type Handler struct{}

// Ping returns a PingResponse with status "ok" on every call.
// It never checks infrastructure state — liveness, not readiness.
func (h *Handler) Ping(
	_ context.Context,
	_ *connect.Request[healthv1.PingRequest],
) (*connect.Response[healthv1.PingResponse], error) {
	return connect.NewResponse(&healthv1.PingResponse{Status: "ok"}), nil
}

// Register mounts the HealthService Connect handler on mux.
// It is compatible with server.HandlerReg and applies the shared interceptor chain.
func Register(mux *http.ServeMux) {
	path, handler := healthv1connect.NewHealthServiceHandler(
		&Handler{},
		connect.WithInterceptors(server.Chain()...),
	)
	mux.Handle(path, handler)
}
