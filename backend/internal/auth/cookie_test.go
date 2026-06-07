package auth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

func TestClearCookie(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config.Config
		wantSecure bool
	}{
		{
			name:       "Secure=false mirrors config",
			cfg:        config.Config{CookieSecure: false},
			wantSecure: false,
		},
		{
			name:       "Secure=true mirrors config",
			cfg:        config.Config{CookieSecure: true},
			wantSecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := auth.ClearCookie(tt.cfg)
			if c.Name != "sid" {
				t.Errorf("Name = %q, want %q", c.Name, "sid")
			}
			if c.Value != "" {
				t.Errorf("Value = %q, want empty", c.Value)
			}
			if c.MaxAge != -1 {
				t.Errorf("MaxAge = %d, want -1", c.MaxAge)
			}
			if c.HttpOnly != true {
				t.Errorf("HttpOnly = %v, want true", c.HttpOnly)
			}
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("SameSite = %v, want SameSiteLaxMode", c.SameSite)
			}
			if c.Path != "/" {
				t.Errorf("Path = %q, want /", c.Path)
			}
			if c.Secure != tt.wantSecure {
				t.Errorf("Secure = %v, want %v", c.Secure, tt.wantSecure)
			}
		})
	}
}

func TestBuildCookie(t *testing.T) {
	baseConfig := config.Config{
		SessionTTL:   30 * time.Minute,
		CookieSecure: false,
	}

	tests := []struct {
		name         string
		sid          string
		cfg          config.Config
		wantName     string
		wantHTTPOnly bool
		wantSameSite http.SameSite
		wantPath     string
		wantSecure   bool
		wantMaxAge   int
	}{
		{
			name:         "COOKIE_SECURE=false",
			sid:          "test-session-id",
			cfg:          baseConfig,
			wantName:     "sid",
			wantHTTPOnly: true,
			wantSameSite: http.SameSiteLaxMode,
			wantPath:     "/",
			wantSecure:   false,
			wantMaxAge:   int(30 * time.Minute / time.Second),
		},
		{
			name: "COOKIE_SECURE=true",
			sid:  "test-session-id",
			cfg: config.Config{
				SessionTTL:   1 * time.Hour,
				CookieSecure: true,
			},
			wantName:     "sid",
			wantHTTPOnly: true,
			wantSameSite: http.SameSiteLaxMode,
			wantPath:     "/",
			wantSecure:   true,
			wantMaxAge:   3600,
		},
		{
			name: "MaxAge equals SESSION_TTL seconds",
			sid:  "abc123",
			cfg: config.Config{
				SessionTTL:   45 * time.Minute,
				CookieSecure: false,
			},
			wantName:     "sid",
			wantHTTPOnly: true,
			wantSameSite: http.SameSiteLaxMode,
			wantPath:     "/",
			wantSecure:   false,
			wantMaxAge:   2700,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := auth.BuildCookie(tt.sid, tt.cfg)
			if c.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", c.Name, tt.wantName)
			}
			if c.Value != tt.sid {
				t.Errorf("Value = %q, want %q", c.Value, tt.sid)
			}
			if c.HttpOnly != tt.wantHTTPOnly {
				t.Errorf("HttpOnly = %v, want %v", c.HttpOnly, tt.wantHTTPOnly)
			}
			if c.SameSite != tt.wantSameSite {
				t.Errorf("SameSite = %v, want %v", c.SameSite, tt.wantSameSite)
			}
			if c.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", c.Path, tt.wantPath)
			}
			if c.Secure != tt.wantSecure {
				t.Errorf("Secure = %v, want %v", c.Secure, tt.wantSecure)
			}
			if c.MaxAge != tt.wantMaxAge {
				t.Errorf("MaxAge = %d, want %d", c.MaxAge, tt.wantMaxAge)
			}
		})
	}
}
