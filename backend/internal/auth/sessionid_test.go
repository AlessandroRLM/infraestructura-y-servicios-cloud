package auth_test

import (
	"strings"
	"testing"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
)

func TestNewSessionID(t *testing.T) {
	t.Run("length is at least 43 chars", func(t *testing.T) {
		sid, err := auth.NewSessionID()
		if err != nil {
			t.Fatalf("NewSessionID() returned error: %v", err)
		}
		if len(sid) < 43 {
			t.Errorf("NewSessionID() len = %d, want >= 43", len(sid))
		}
	})

	t.Run("URL-safe base64 — no padding", func(t *testing.T) {
		sid, err := auth.NewSessionID()
		if err != nil {
			t.Fatalf("NewSessionID() returned error: %v", err)
		}
		if strings.ContainsAny(sid, "+/=") {
			t.Errorf("NewSessionID() contains non-URL-safe chars: %q", sid)
		}
	})

	t.Run("two calls produce different values", func(t *testing.T) {
		a, err := auth.NewSessionID()
		if err != nil {
			t.Fatalf("first NewSessionID() error: %v", err)
		}
		b, err := auth.NewSessionID()
		if err != nil {
			t.Fatalf("second NewSessionID() error: %v", err)
		}
		if a == b {
			t.Error("NewSessionID() returned identical values on consecutive calls")
		}
	})

	t.Run("no padding characters", func(t *testing.T) {
		for range 10 {
			sid, err := auth.NewSessionID()
			if err != nil {
				t.Fatalf("NewSessionID() error: %v", err)
			}
			if strings.Contains(sid, "=") {
				t.Errorf("NewSessionID() contains padding '=': %q", sid)
			}
		}
	})
}
