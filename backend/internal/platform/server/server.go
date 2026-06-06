package server

import (
	"context"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
)

const (
	shutdownTimeout    = 15 * time.Second
	readHeaderTimeout  = 5 * time.Second
)

// HandlerReg is a function that registers one or more routes on the given mux.
// Each domain package exposes a compatible Register function.
type HandlerReg func(*http.ServeMux)

// New constructs an *http.Server with h2c transport, liveness/readiness routes,
// and any additional handler registrations provided by callers.
// Unencrypted HTTP/2 (h2c) is enabled via http.Server.Protocols (Go 1.24+).
func New(log *slog.Logger, database, cache db.Pinger, handlers ...HandlerReg) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", livenessHandler)
	mux.HandleFunc("/readyz", readinessHandler(database, cache))

	for _, reg := range handlers {
		reg(mux)
	}

	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{
		Handler:           mux,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
		ReadHeaderTimeout: readHeaderTimeout,
		Protocols:         &protocols,
	}

	return srv
}

// RunWithGracefulShutdown starts srv.ListenAndServe and blocks until SIGINT or SIGTERM
// is received. On signal, it initiates a graceful shutdown allowing in-flight requests
// to complete within shutdownTimeout.
func RunWithGracefulShutdown(ctx context.Context, srv *http.Server) {
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	<-sigCtx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
}
