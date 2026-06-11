package metrics

import (
	"crypto/subtle"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an http.Handler that serves the Prometheus text exposition format.
// Requests must carry the configured token in the X-Metrics-Token header; missing or
// incorrect tokens result in HTTP 401 Unauthorized (no body, no WWW-Authenticate header).
//
// The comparison between the provided and configured token uses crypto/subtle.ConstantTimeCompare
// to prevent timing attacks.
func (r *Registry) Handler(token string) http.Handler {
	metricsHandler := promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		provided := req.Header.Get("X-Metrics-Token")
		// ConstantTimeCompare requires equal-length slices; if lengths differ it returns 0.
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		metricsHandler.ServeHTTP(w, req)
	})
}
