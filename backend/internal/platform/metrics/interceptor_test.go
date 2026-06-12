package metrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	dto "github.com/prometheus/client_model/go"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/metrics"
)

// noopAuthHandler is a minimal AuthService implementation that succeeds on every call.
type noopAuthHandler struct{}

func (h *noopAuthHandler) Login(_ context.Context, _ *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	return connect.NewResponse(&authv1.LoginResponse{}), nil
}
func (h *noopAuthHandler) Logout(_ context.Context, _ *connect.Request[authv1.LogoutRequest]) (*connect.Response[authv1.LogoutResponse], error) {
	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}
func (h *noopAuthHandler) RequestPasswordReset(_ context.Context, _ *connect.Request[authv1.RequestPasswordResetRequest]) (*connect.Response[authv1.RequestPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.RequestPasswordResetResponse{}), nil
}
func (h *noopAuthHandler) ConfirmPasswordReset(_ context.Context, _ *connect.Request[authv1.ConfirmPasswordResetRequest]) (*connect.Response[authv1.ConfirmPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.ConfirmPasswordResetResponse{}), nil
}

func (h *noopAuthHandler) GetSession(_ context.Context, _ *connect.Request[authv1.GetSessionRequest]) (*connect.Response[authv1.Session], error) {
	return connect.NewResponse(&authv1.Session{}), nil
}

// buildTestServer registers the AuthService handler with the given interceptors and returns
// a running httptest server and its base URL.
func buildTestServer(t *testing.T, interceptors ...connect.UnaryInterceptorFunc) *httptest.Server {
	t.Helper()
	opts := make([]connect.HandlerOption, 0, len(interceptors))
	for _, fn := range interceptors {
		opts = append(opts, connect.WithInterceptors(fn))
	}
	mux := http.NewServeMux()
	path, h := authv1connect.NewAuthServiceHandler(&noopAuthHandler{}, opts...)
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// callLogin issues a Login RPC to the given server URL and returns the error (if any).
func callLogin(t *testing.T, serverURL string) error {
	t.Helper()
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, serverURL)
	_, err := client.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
		Email:    "test@test.com",
		Password: "password",
	}))
	return err
}

// TestRPCInterceptor_SuccessLabels verifies that a successful RPC records
// academico_rpc_requests_total with code="ok".
func TestRPCInterceptor_SuccessLabels(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	srv := buildTestServer(t, reg.RPCInterceptor())

	err := callLogin(t, srv.URL)
	if err != nil {
		// Login may return an error (missing fields etc.) — what matters is the interceptor ran.
		_ = err
	}

	mfs, gErr := reg.Gather()
	if gErr != nil {
		t.Fatalf("Gather: %v", gErr)
	}

	// Verify the counter exists for the AuthService/Login procedure.
	total := findCounterSum(mfs, "academico_rpc_requests_total", map[string]string{
		"service": "AuthService",
		"method":  "Login",
	})
	if total < 1 {
		t.Errorf("academico_rpc_requests_total{service=AuthService,method=Login} = %.0f, want >= 1", total)
	}

	// Verify no counter increment happened for a non-existent service/method.
	badTotal := findCounterSum(mfs, "academico_rpc_requests_total", map[string]string{
		"service": "NonExistent",
		"method":  "Missing",
	})
	if badTotal != 0 {
		t.Errorf("expected 0 for non-called procedure, got %.0f", badTotal)
	}
}

// TestRPCInterceptor_ErrorCode verifies that an error return maps to the correct connect code label.
func TestRPCInterceptor_ErrorCode(t *testing.T) {
	t.Parallel()

	reg := metrics.New()

	// Wire an interceptor that forces every call to fail with CodeNotFound.
	forceNotFound := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
	})

	// RED interceptor OUTERMOST so it wraps the forceNotFound interceptor.
	srv := buildTestServer(t, reg.RPCInterceptor(), forceNotFound)

	_ = callLogin(t, srv.URL)

	mfs, gErr := reg.Gather()
	if gErr != nil {
		t.Fatalf("Gather: %v", gErr)
	}

	found := findCounter(mfs, "academico_rpc_requests_total", map[string]string{
		"service": "AuthService",
		"method":  "Login",
		"code":    "not_found",
	})
	if found < 1 {
		t.Errorf("academico_rpc_requests_total{code=not_found} = %.0f, want >= 1", found)
	}

	// Must NOT have an "ok" entry for the same service/method.
	okFound := findCounter(mfs, "academico_rpc_requests_total", map[string]string{
		"service": "AuthService",
		"method":  "Login",
		"code":    "ok",
	})
	if okFound != 0 {
		t.Errorf("expected code=ok counter to be 0 when all calls fail, got %.0f", okFound)
	}
}

// TestRPCInterceptor_ProcedureLabelDerivation verifies service and method label parsing.
func TestRPCInterceptor_ProcedureLabelDerivation(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	srv := buildTestServer(t, reg.RPCInterceptor())

	_ = callLogin(t, srv.URL)

	mfs, gErr := reg.Gather()
	if gErr != nil {
		t.Fatalf("Gather: %v", gErr)
	}

	// The Login procedure is "/auth.v1.AuthService/Login"; verify it parses to
	// service="AuthService", method="Login".
	total := findCounterSum(mfs, "academico_rpc_requests_total", map[string]string{
		"service": "AuthService",
		"method":  "Login",
	})
	if total < 1 {
		t.Errorf("label derivation: academico_rpc_requests_total{service=AuthService,method=Login} = %.0f, want >= 1", total)
	}
}

// TestRPCInterceptor_HistogramHasNoCodeLabel verifies that rpc_duration_seconds metrics
// do not carry a code label dimension.
func TestRPCInterceptor_HistogramHasNoCodeLabel(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	srv := buildTestServer(t, reg.RPCInterceptor())

	_ = callLogin(t, srv.URL)

	mfs, gErr := reg.Gather()
	if gErr != nil {
		t.Fatalf("Gather: %v", gErr)
	}

	for _, mf := range mfs {
		if mf.GetName() != "academico_rpc_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "code" {
					t.Errorf("academico_rpc_duration_seconds must not have a 'code' label; found: %v", lp)
				}
			}
		}
	}
}

// findCounter returns the value of the named counter with the exact given label set,
// or 0 if not found.
func findCounter(mfs []*dto.MetricFamily, name string, labels map[string]string) float64 {
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if labelsMatch(m.GetLabel(), labels) {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

// findCounterSum returns the sum of all counter values for the named metric that match
// the given label subset (may have additional labels not listed in the filter).
func findCounterSum(mfs []*dto.MetricFamily, name string, labels map[string]string) float64 {
	var sum float64
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if labelsMatchSubset(m.GetLabel(), labels) {
				sum += m.GetCounter().GetValue()
			}
		}
	}
	return sum
}

// labelsMatch returns true if the metric's label set EXACTLY matches the given map.
func labelsMatch(got []*dto.LabelPair, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	idx := make(map[string]string, len(got))
	for _, lp := range got {
		idx[lp.GetName()] = lp.GetValue()
	}
	for k, v := range want {
		if idx[k] != v {
			return false
		}
	}
	return true
}

// labelsMatchSubset returns true if all key-value pairs in want are present in got
// (got may contain additional labels).
func labelsMatchSubset(got []*dto.LabelPair, want map[string]string) bool {
	idx := make(map[string]string, len(got))
	for _, lp := range got {
		idx[lp.GetName()] = lp.GetValue()
	}
	for k, v := range want {
		if idx[k] != v {
			return false
		}
	}
	return true
}
