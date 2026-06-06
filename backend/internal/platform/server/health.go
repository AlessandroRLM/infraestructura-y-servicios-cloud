package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
)

const readinessTimeout = 2 * time.Second

// livenessHandler returns 200 OK unconditionally — the process is alive.
func livenessHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// readinessHandler pings both the database and cache with a short timeout.
// Returns 200 when both are healthy; 503 with a body identifying failing deps otherwise.
func readinessHandler(database, cache db.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		var failures []string

		if err := database.Ping(ctx); err != nil {
			failures = append(failures, fmt.Sprintf("postgres: %v", err))
		}
		if err := cache.Ping(ctx); err != nil {
			failures = append(failures, fmt.Sprintf("redis: %v", err))
		}

		if len(failures) > 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			for _, f := range failures {
				_, _ = fmt.Fprintf(w, "%s\n", f)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
