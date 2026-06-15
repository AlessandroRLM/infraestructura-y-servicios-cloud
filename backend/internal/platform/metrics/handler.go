package metrics

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler serves the Prometheus text exposition format, gated by the configured token.
// The token may arrive in the X-Metrics-Token header or as "Authorization: Bearer <token>"
// (what Prometheus and Managed Prometheus send); a missing or wrong token yields 401.
// Comparison is constant-time.
func (r *Registry) Handler(token string) http.Handler {
	metricsHandler := promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		provided := req.Header.Get("X-Metrics-Token")
		if provided == "" {
			if auth := req.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				provided = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		metricsHandler.ServeHTTP(w, req)
	})
}
