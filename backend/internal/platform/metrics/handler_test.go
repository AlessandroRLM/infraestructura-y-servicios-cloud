package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/metrics"
)

const testToken = "correcttoken"

// TestHandler_ValidToken_200 verifies that a GET /metrics request with the correct
// X-Metrics-Token header returns 200, a text/plain Content-Type, and a non-empty body.
func TestHandler_ValidToken_200(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	handler := reg.Handler(testToken)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("X-Metrics-Token", testToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want prefix \"text/plain\"", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("response body is empty, want Prometheus text exposition")
	}
}

// TestHandler_MissingToken_401 verifies that a GET /metrics request without the
// X-Metrics-Token header returns 401 and no WWW-Authenticate header.
func TestHandler_MissingToken_401(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	handler := reg.Handler(testToken)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	// Deliberately omit the X-Metrics-Token header.
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if www := rec.Header().Get("WWW-Authenticate"); www != "" {
		t.Errorf("WWW-Authenticate header must be absent; got %q", www)
	}
}

// TestHandler_WrongToken_401 verifies that a GET /metrics request with an incorrect
// X-Metrics-Token value returns 401 and no WWW-Authenticate header.
func TestHandler_WrongToken_401(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	handler := reg.Handler(testToken)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("X-Metrics-Token", "wrong")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if www := rec.Header().Get("WWW-Authenticate"); www != "" {
		t.Errorf("WWW-Authenticate header must be absent; got %q", www)
	}
}

// TestHandler_ConstantTimeCompareSafety verifies that a token that is a prefix of the
// real token is still rejected. This is a behavioral proxy for constant-time comparison:
// a naive byte-by-byte compare that short-circuits on mismatch would accept a prefix
// if the comparison logic had an off-by-one error.
func TestHandler_ConstantTimeCompareSafety(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	handler := reg.Handler(testToken) // "correcttoken"

	// "correct" is a strict prefix of "correcttoken" — must be rejected.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("X-Metrics-Token", "correct")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("prefix token: status = %d, want 401 (constant-time compare must reject length mismatch)", rec.Code)
	}
}
