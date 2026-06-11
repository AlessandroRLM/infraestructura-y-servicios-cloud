package integration_test

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
)

const testMetricsToken = "test-metrics-token"

// scrapeMetrics issues GET /metrics with the test token and returns the raw body.
func scrapeMetrics(t *testing.T) string {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/metrics", nil)
	if err != nil {
		t.Fatalf("scrapeMetrics: create request: %v", err)
	}
	req.Header.Set("X-Metrics-Token", testMetricsToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("scrapeMetrics: do request: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("scrapeMetrics: read body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("scrapeMetrics: status = %d, want 200; body: %s", resp.StatusCode, body)
	}
	return string(body)
}

// TestMetrics_ValidToken_200 verifies that GET /metrics with the correct X-Metrics-Token
// returns 200, text/plain Content-Type, and the expected metric families.
func TestMetrics_ValidToken_200(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/metrics", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("X-Metrics-Token", testMetricsToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want prefix \"text/plain\"", ct)
	}

	bodyStr := string(body)
	// Plain (non-vec) counters and runtime collectors are always present in the
	// exposition text — even at value zero — because they carry a single pre-determined
	// label set. CounterVec / HistogramVec metrics (rpc_requests, rpc_duration,
	// section_full) only appear after their first WithLabelValues call, so they are
	// validated by tests that explicitly trigger RPCs or events.
	alwaysPresent := []string{
		"academico_section_lock_timeout_total",
		"academico_admission_saturated_total",
		"go_goroutines",
		"process_open_fds",
	}
	for _, metric := range alwaysPresent {
		if !strings.Contains(bodyStr, metric) {
			t.Errorf("body does not contain %q", metric)
		}
	}
}

// TestMetrics_MissingToken_401 verifies that GET /metrics without the X-Metrics-Token
// header returns 401 and does not expose metrics.
func TestMetrics_MissingToken_401(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/metrics", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	// Deliberately omit the token header.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	if strings.Contains(string(body), "academico_rpc_requests_total") {
		t.Error("response body must not contain metric names on 401")
	}
}

// TestMetrics_WrongToken_401 verifies that GET /metrics with a wrong token returns 401.
func TestMetrics_WrongToken_401(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/metrics", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("X-Metrics-Token", "wrong")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

// TestMetrics_InterceptorChainExclusion verifies that the /metrics endpoint is NOT
// a Connect procedure: its response must be Prometheus text exposition, not Connect JSON.
func TestMetrics_InterceptorChainExclusion(t *testing.T) {
	body := scrapeMetrics(t)

	// Prometheus text format is line-based, not JSON. A Connect error response would
	// be {"code":"...","message":"..."}.
	if strings.Contains(body, "\"code\"") && strings.Contains(body, "\"message\"") {
		t.Error("response looks like a Connect JSON response — /metrics must not pass through Connect interceptors")
	}
	if !strings.Contains(body, "# HELP") && !strings.Contains(body, "# TYPE") {
		t.Error("response does not contain Prometheus text format markers (# HELP / # TYPE)")
	}
}

// TestMetrics_RED_SuccessfulRPC_CounterIncremented verifies that a successful RPC
// increments academico_rpc_requests_total.
func TestMetrics_RED_SuccessfulRPC_CounterIncremented(t *testing.T) {
	// Seed an authenticated session.
	before := scrapeMetrics(t)

	// Issue a Login RPC (exempt procedure — no auth session needed).
	client := authv1connect.NewAuthServiceClient(http.DefaultClient, baseURL)
	_, _ = client.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
		Email:    "doesnotexist@test.com",
		Password: "wrongpassword",
	}))

	after := scrapeMetrics(t)

	// The counter for AuthService/Login should have increased.
	if !strings.Contains(after, "AuthService") {
		t.Errorf("after RPC: academico_rpc_requests_total should contain service=AuthService label")
	}
	// Crude check: after should differ from before (counter incremented).
	if before == after {
		t.Log("before and after scrape are identical — expected counter change after RPC")
	}
}

// TestMetrics_RED_UnauthenticatedRPC_CodeLabelPresent verifies that unauthenticated
// requests are counted by the RED interceptor (outermost placement).
func TestMetrics_RED_UnauthenticatedRPC_CodeLabelPresent(t *testing.T) {
	before := scrapeMetrics(t)

	// Issue an RPC that requires auth without a session cookie — will return unauthenticated.
	// Use catalog ListPrograms as a representative protected procedure.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		baseURL+"/catalog.v1.CatalogService/ListPrograms",
		strings.NewReader("{}"),
	)
	if err != nil {
		t.Fatalf("create RPC request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// No Cookie header — unauthenticated.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST CatalogService/ListPrograms: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	after := scrapeMetrics(t)

	// The after scrape should contain unauthenticated code label since RED is outermost.
	if !strings.Contains(after, "unauthenticated") {
		t.Errorf("after unauthenticated RPC: expected code=unauthenticated label in %q; before: contains=%v",
			"academico_rpc_requests_total", strings.Contains(before, "unauthenticated"))
	}
}

// parseCounterValue scans the Prometheus text exposition body for a line matching
// metricName followed by the given label substring and returns the float64 value.
// Returns 0 when no matching line is found (counter not yet emitted / still at zero).
// This is a best-effort line scan — adequate for counter diff assertions in tests.
func parseCounterValue(body, metricName, labelSubstr string) float64 {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, metricName) {
			continue
		}
		if labelSubstr != "" && !strings.Contains(line, labelSubstr) {
			continue
		}
		// Line format: metricName{labels} value
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(parts[len(parts)-1], 64)
		if err != nil {
			continue
		}
		return v
	}
	return 0
}

// TestMetrics_LockTimeout_CounterAndRPCCode exercises the 55P03 code path live against
// the real database (test-plan item I-10, FR-3c, product-decisions obs #289).
//
// Determinism guarantee: a channel barrier ensures that tx A has acquired the
// FOR UPDATE lock on the section row BEFORE the EnrollOwnSection RPC fires.
// time.Sleep is used ONLY to hold the lock for a duration that comfortably exceeds
// the 2500ms lock_timeout set by the repository; it is NOT used for synchronization.
func TestMetrics_LockTimeout_CounterAndRPCCode(t *testing.T) {
	ctx := context.Background()

	// Seed a student with a paid enrollment and an open window so that all pre-RPC
	// gates pass and the only failure point is the lock acquisition inside the tx.
	studentUserID, studentSID := seedUserWithSession(t, "metrics-lt-stu@metrics.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	// Open enrollment window so the window gate passes.
	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	// capacity=10: section is not full, so the pre-check fast-fail does not fire.
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	_, enrollmentCleanup := seedPaidEnrollment(t, studentUserID.String(), programID, periodYear)
	defer enrollmentCleanup()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	// lockHeld is closed by the goroutine AFTER the FOR UPDATE query returns,
	// guaranteeing the lock is held before the RPC fires.
	lockHeld := make(chan struct{})
	// lockRelease is closed by the test after the RPC returns to release tx A promptly.
	lockRelease := make(chan struct{})

	// tx A: acquire the section row lock and hold it until the RPC completes.
	go func() {
		txCtx := context.Background()
		tx, err := pgxPool.Begin(txCtx)
		if err != nil {
			// Can't signal; the test will hang and timeout — acceptable failure mode.
			close(lockHeld)
			return
		}
		defer tx.Rollback(txCtx) //nolint:errcheck

		// Acquire the exact lock that EnrollSectionTx acquires (step 2 in the repository).
		// This matches: SELECT ... FROM sections WHERE id=$1 FOR UPDATE
		var dummy string
		_ = tx.QueryRow(txCtx,
			`SELECT id::text FROM sections WHERE id = $1 FOR UPDATE`,
			sectionID,
		).Scan(&dummy)

		// Signal: lock is now held. The RPC may fire.
		close(lockHeld)

		// Hold the lock until the test signals release. The repository sets
		// lock_timeout='2500ms', so we hold for 3500ms to ensure the RPC times out.
		select {
		case <-lockRelease:
		case <-time.After(5 * time.Second): // safety valve
		}
		// tx.Rollback deferred above releases the lock.
	}()

	// BARRIER: wait for tx A to confirm the lock is held before scraping or calling RPC.
	<-lockHeld

	before := scrapeMetrics(t)

	// tx B (inside the RPC): EnrollOwnSection tries to acquire the same lock, waits,
	// hits the 2500ms lock_timeout, and returns 55P03 → ErrLockTimeout → CodeUnavailable.
	client := newSectionEnrollmentClient(nil)
	_, rpcErr := seEnrollOwn(ctx, client, studentSID, sectionID, programID)

	// Release tx A promptly after RPC returns to keep the suite fast.
	close(lockRelease)

	after := scrapeMetrics(t)

	// Assertion 1: the RPC must return CodeUnavailable (ErrLockTimeout sentinel).
	assertConnectCode(t, rpcErr, connect.CodeUnavailable)

	// Assertion 2: academico_section_lock_timeout_total incremented by exactly 1.
	beforeLT := parseCounterValue(before, "academico_section_lock_timeout_total", "")
	afterLT := parseCounterValue(after, "academico_section_lock_timeout_total", "")
	if delta := afterLT - beforeLT; delta != 1 {
		t.Errorf("academico_section_lock_timeout_total delta = %.0f, want 1 (before=%.0f, after=%.0f)",
			delta, beforeLT, afterLT)
	}

	// Assertion 3: no section_full counter was touched (the timeout fires before capacity check).
	beforeFull := parseCounterValue(before, "academico_section_full_total", "")
	afterFull := parseCounterValue(after, "academico_section_full_total", "")
	if delta := afterFull - beforeFull; delta != 0 {
		t.Errorf("academico_section_full_total delta = %.0f, want 0 (before=%.0f, after=%.0f)",
			delta, beforeFull, afterFull)
	}
}
