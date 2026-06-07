package auth

import (
	"net/http"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

const cookieName = "sid"

// BuildCookie constructs the session cookie for the given session ID.
// HttpOnly and SameSite=Lax are always set. Secure follows cfg.CookieSecure,
// which is configurable so local development over plaintext h2c can disable it
// while production over HTTPS enables it. MaxAge is the session TTL in seconds.
//
//nolint:gosec // G124: the Secure attribute is intentionally configurable (off for local h2c, on for HTTPS); SameSite=Lax is correct for a first-party session cookie.
func BuildCookie(sid string, cfg config.Config) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.CookieSecure,
		MaxAge:   int(cfg.SessionTTL.Seconds()),
	}
}

// ClearCookie returns a cookie that instructs the browser to delete the session cookie.
// Secure mirrors cfg.CookieSecure so browsers honour the deletion directive under HTTPS;
// a deletion cookie must carry the same attributes as the cookie it replaces.
//
//nolint:gosec // G124: the Secure attribute is intentionally configurable to match the original session cookie; SameSite=Lax is correct for a first-party session cookie.
func ClearCookie(cfg config.Config) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.CookieSecure,
		MaxAge:   -1,
	}
}

// parseSID extracts the "sid" session cookie value from request headers,
// returning "" when the cookie is absent or unparseable. It is the single
// shared parser used by both the handler and the session interceptor.
func parseSID(headers http.Header) string {
	cookieHeader := headers.Get("Cookie")
	if cookieHeader == "" {
		return ""
	}
	r := &http.Request{Header: http.Header{"Cookie": {cookieHeader}}}
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
