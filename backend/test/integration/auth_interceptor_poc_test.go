package integration_test

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
)

const timeout3s = 3 * time.Second

// TestInterceptorPOC_CookieReadOnRequest proves that a Cookie header set by the
// client is visible inside the handler via req.Header().Get("Cookie").
func TestInterceptorPOC_CookieReadOnRequest(t *testing.T) {
	var observedCookie string

	// Inline handler that captures whatever Cookie header arrives.
	probeRead := &requestCookieCapture{captured: &observedCookie}

	mux := http.NewServeMux()
	path, handler := authv1connect.NewAuthServiceHandler(probeRead)
	mux.Handle(path, handler)

	srv := launchInlineServer(t, mux)
	defer func() { _ = srv.Shutdown(context.Background()) }()

	client := authv1connect.NewAuthServiceClient(http.DefaultClient, "http://"+srv.Addr)
	req := connect.NewRequest(&authv1.LoginRequest{Email: "x@example.com", Password: "pw"})
	req.Header().Set("Cookie", "sid=my-secret-value")

	_, _ = client.Login(context.Background(), req) // error irrelevant

	if observedCookie == "" {
		t.Error("Cookie header was not forwarded to the handler: got empty string")
	}
	if observedCookie != "sid=my-secret-value" {
		t.Errorf("Cookie header = %q, want %q", observedCookie, "sid=my-secret-value")
	}
}

// TestInterceptorPOC_SetCookieOnResponse proves that a Set-Cookie header written
// via resp.Header().Set() inside a handler is delivered to the HTTP client.
func TestInterceptorPOC_SetCookieOnResponse(t *testing.T) {
	jar := &simpleCookieJar{}
	httpClient := &http.Client{Jar: jar}

	mux := http.NewServeMux()
	path, handler := authv1connect.NewAuthServiceHandler(&setCookieHandler{})
	mux.Handle(path, handler)

	srv := launchInlineServer(t, mux)
	defer func() { _ = srv.Shutdown(context.Background()) }()

	client := authv1connect.NewAuthServiceClient(httpClient, "http://"+srv.Addr)
	_, _ = client.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{}))

	if !jar.hasCookieNamed("sid") {
		t.Error("Set-Cookie 'sid' was not observed on the response; cookie wiring is broken")
	}
}

// --- inline handler stubs ---

type requestCookieCapture struct {
	authv1connect.UnimplementedAuthServiceHandler
	captured *string
}

func (h *requestCookieCapture) Login(
	_ context.Context,
	req *connect.Request[authv1.LoginRequest],
) (*connect.Response[authv1.LoginResponse], error) {
	*h.captured = req.Header().Get("Cookie")
	return connect.NewResponse(&authv1.LoginResponse{}), nil
}

type setCookieHandler struct {
	authv1connect.UnimplementedAuthServiceHandler
}

func (h *setCookieHandler) Login(
	_ context.Context,
	_ *connect.Request[authv1.LoginRequest],
) (*connect.Response[authv1.LoginResponse], error) {
	resp := connect.NewResponse(&authv1.LoginResponse{})
	resp.Header().Set("Set-Cookie", "sid=interceptor-poc; HttpOnly; Path=/; SameSite=Lax")
	return resp, nil
}

// --- minimal cookie jar ---

type simpleCookieJar struct {
	cookies []*http.Cookie
}

func (j *simpleCookieJar) SetCookies(_ *url.URL, cookies []*http.Cookie) {
	j.cookies = append(j.cookies, cookies...)
}

func (j *simpleCookieJar) Cookies(_ *url.URL) []*http.Cookie {
	return j.cookies
}

func (j *simpleCookieJar) hasCookieNamed(name string) bool {
	for _, c := range j.cookies {
		if c.Name == name {
			return true
		}
	}
	return false
}

// --- inline test server helpers ---

// launchInlineServer starts a plain HTTP server on a random port and returns it.
// The caller must call srv.Shutdown.
func launchInlineServer(t *testing.T, mux *http.ServeMux) *http.Server {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("launchInlineServer: listen: %v", err)
	}
	srv := &http.Server{ //nolint:gosec // G112: test-only inline server; Slowloris not a concern in unit tests.
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	srv.Addr = ln.Addr().String()
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			t.Logf("inline server: %v", err)
		}
	}()
	waitForServer(srv.Addr, timeout3s)
	return srv
}
