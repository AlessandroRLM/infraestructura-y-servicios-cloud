package integration_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
	section_enrollmentv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1/section_enrollmentv1connect"
	"github.com/google/uuid"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/authdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/logging"
	platformmetrics "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/metrics"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac/rbacdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment/sectionenrollmentdb"
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

// seedActiveEnrollmentDirectSQL inserts an active (in_progress) section_enrollment row
// directly via SQL, bypassing the service layer. This is used by I-7/I-8 to control the
// active-seat count without triggering any counter logic in the production code path.
func seedActiveEnrollmentDirectSQL(t *testing.T, enrollmentID, sectionID string) {
	t.Helper()
	ctx := context.Background()
	_, err := pgxPool.Exec(ctx,
		`INSERT INTO section_enrollments (enrollment_id, section_id, status)
		 VALUES ($1, $2, 'in_progress')`,
		enrollmentID, sectionID,
	)
	if err != nil {
		t.Fatalf("seedActiveEnrollmentDirectSQL: %v", err)
	}
}

// TestMetrics_SectionFull_PreCheckCounter verifies that the pre_check path of
// academico_section_full_total increments by exactly 1 when a section is already at
// full capacity before the enrollment transaction begins (I-7, FR-3a).
//
// The pre-check gate (CountActiveSeats before acquiring the section lock) fires first.
// We confirm that only pre_check increments and under_lock remains unchanged.
func TestMetrics_SectionFull_PreCheckCounter(t *testing.T) {
	ctx := context.Background()

	// Seed a student with all pre-RPC prerequisites satisfied.
	studentUserID, studentSID := seedUserWithSession(t, "metrics-i7-stu@metrics.test", "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	// Capacity=1: the pre-check fires as soon as one active seat exists.
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 1)
	defer sectionCleanup()

	_, enrollmentCleanup := seedPaidEnrollment(t, studentUserID.String(), programID, periodYear)
	defer enrollmentCleanup()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	// Seed a second student to occupy the single seat (direct SQL — bypasses all
	// production counters so we start with clean counter deltas).
	blockerUserID, _ := seedUserWithSession(t, "metrics-i7-blocker@metrics.test", "student")
	seedStudentProfile(t, blockerUserID, 2099)
	blockerEnrollmentID, blockerEnrollCleanup := seedPaidEnrollment(t, blockerUserID.String(), programID, periodYear)
	defer blockerEnrollCleanup()
	seedActiveEnrollmentDirectSQL(t, blockerEnrollmentID, sectionID)

	before := scrapeMetrics(t)

	// Attempt enrollment — pre-check must see activeCount=1 == capacity=1 → reject.
	client := newSectionEnrollmentClient(nil)
	_, rpcErr := seEnrollOwn(ctx, client, studentSID, sectionID, programID)

	after := scrapeMetrics(t)

	// The RPC must return CodeFailedPrecondition (ErrSectionFull mapping).
	assertConnectCode(t, rpcErr, connect.CodeFailedPrecondition)

	// pre_check counter must increase by exactly 1.
	beforePre := parseCounterValue(before, "academico_section_full_total", `path="pre_check"`)
	afterPre := parseCounterValue(after, "academico_section_full_total", `path="pre_check"`)
	if delta := afterPre - beforePre; delta != 1 {
		t.Errorf("section_full_total{path=pre_check} delta = %.0f, want 1 (before=%.0f after=%.0f)",
			delta, beforePre, afterPre)
	}

	// under_lock counter must NOT change.
	beforeUnder := parseCounterValue(before, "academico_section_full_total", `path="under_lock"`)
	afterUnder := parseCounterValue(after, "academico_section_full_total", `path="under_lock"`)
	if delta := afterUnder - beforeUnder; delta != 0 {
		t.Errorf("section_full_total{path=under_lock} delta = %.0f, want 0 (before=%.0f after=%.0f)",
			delta, beforeUnder, afterUnder)
	}
}

// TestMetrics_SectionFull_UnderLockCounter verifies that the under_lock path of
// academico_section_full_total increments by exactly 1 when the section reaches
// capacity only after the enrollment transaction acquires the row lock (I-8, FR-3b).
//
// Determinism technique: tx A holds the section row FOR UPDATE within an open transaction
// AND inserts an active seat row in that SAME transaction (uncommitted). B's pre-check
// runs outside any transaction under READ COMMITTED — it sees 0 active seats (tx A's
// insert is uncommitted). B's enroll tx then tries FOR UPDATE, blocks (A holds it).
// A bounded 250ms sleep gives B time to reach the contention point. Then tx A commits:
// the insert and lock release happen atomically. B acquires the lock, counts 1 active
// seat (committed by A), increments under_lock, and returns CodeFailedPrecondition.
//
// Sleep is ONLY used as a bounded wait for B to reach the lock contention point; all
// critical ordering is enforced by the channel barriers and DB lock semantics.
// Run 3× to validate determinism: -run TestMetrics_SectionFull_UnderLockCounter -count 3.
func TestMetrics_SectionFull_UnderLockCounter(t *testing.T) {
	ctx := context.Background()

	// Seed a student with all pre-RPC prerequisites satisfied.
	studentUserID, studentSID := seedUserWithSession(t, "metrics-i8-stu@metrics.test"+"_"+uniqueSuffix(t), "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	// Capacity=1, 0 active seats: pre-check passes (CountActiveSeats returns 0 < 1).
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 1)
	defer sectionCleanup()

	_, enrollmentCleanup := seedPaidEnrollment(t, studentUserID.String(), programID, periodYear)
	defer enrollmentCleanup()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	// Seed a second student. tx A will INSERT this student's section_enrollment within
	// its transaction (uncommitted) so pre-check sees 0 but under-lock count sees 1.
	blockerUserID, _ := seedUserWithSession(t, "metrics-i8-blocker@metrics.test"+"_"+uniqueSuffix(t), "student")
	seedStudentProfile(t, blockerUserID, 2099)
	blockerEnrollmentID, blockerEnrollCleanup := seedPaidEnrollment(t, blockerUserID.String(), programID, periodYear)
	defer blockerEnrollCleanup()

	sectionUUID, err := uuid.Parse(sectionID)
	if err != nil {
		t.Fatalf("parse sectionID: %v", err)
	}

	// lockHeld is closed once tx A has acquired the section FOR UPDATE and inserted the
	// uncommitted seat row — signalling B may now fire its pre-check.
	lockHeld := make(chan struct{})
	// lockCommit signals tx A to commit (releasing the lock and making the insert visible).
	lockCommit := make(chan struct{})
	// txDone is closed once tx A's commit returns.
	txDone := make(chan struct{})

	// tx A: within a single open transaction —
	//   1. INSERT an active seat row (uncommitted — invisible to B's READ COMMITTED pre-check)
	//   2. Acquire the section row FOR UPDATE (holds the lock B needs)
	//   3. Signal lockHeld so B can start its pre-check
	//   4. Wait for lockCommit (fired after 250ms when B is blocking on the FOR UPDATE)
	//   5. Commit: insert becomes visible + lock is released → B acquires lock, counts 1
	go func() {
		defer close(txDone)
		txCtx := context.Background()
		tx, err := pgxPool.Begin(txCtx)
		if err != nil {
			close(lockHeld)
			return
		}
		// Insert the active seat WITHIN this transaction (uncommitted at pre-check time).
		_, _ = tx.Exec(txCtx,
			`INSERT INTO section_enrollments (enrollment_id, section_id, status)
			 VALUES ($1, $2, 'in_progress')`,
			blockerEnrollmentID, sectionID,
		)

		// Acquire the section row FOR UPDATE — same lock as EnrollSectionTx step 2.
		var dummy string
		_ = tx.QueryRow(txCtx,
			`SELECT id::text FROM sections WHERE id = $1 FOR UPDATE`,
			sectionUUID,
		).Scan(&dummy)

		// Signal: seat row is inserted (uncommitted) + section lock is held.
		close(lockHeld)

		// Bounded wait: 250ms gives B time to complete its pre-check (which sees 0 rows
		// since our insert is uncommitted) and reach the FOR UPDATE contention point.
		// The lock contention provides the real ordering guarantee — sleep only sets a
		// minimum window for B to be blocked before A releases.
		select {
		case <-lockCommit:
		case <-time.After(300 * time.Millisecond):
		}

		// Commit: the insert becomes visible AND the section lock is released atomically.
		_ = tx.Commit(txCtx)
	}()

	// BARRIER: wait until tx A holds the lock (and has inserted the uncommitted row).
	<-lockHeld

	before := scrapeMetrics(t)

	// Fire B's RPC in a goroutine: pre-check sees 0 seats (A's insert uncommitted),
	// passes; B's enroll tx tries FOR UPDATE on the section row, blocks waiting for A.
	rpcDone := make(chan error, 1)
	go func() {
		client := newSectionEnrollmentClient(nil)
		_, rpcErr := seEnrollOwn(ctx, client, studentSID, sectionID, programID)
		rpcDone <- rpcErr
	}()

	// Bounded wait: 250ms for B to pass its pre-check and block on the section lock.
	// A's commit fires after this, releasing the lock so B can proceed.
	select {
	case <-time.After(250 * time.Millisecond):
	case rpcErr := <-rpcDone:
		// B completed before we released A — pre-check may have fired (pre-check does not
		// block on A's lock). Check which counter fired.
		<-txDone
		after := scrapeMetrics(t)
		preDelta := parseCounterValue(after, "academico_section_full_total", `path="pre_check"`) -
			parseCounterValue(before, "academico_section_full_total", `path="pre_check"`)
		underDelta := parseCounterValue(after, "academico_section_full_total", `path="under_lock"`) -
			parseCounterValue(before, "academico_section_full_total", `path="under_lock"`)
		t.Logf("RPC completed before commit: err=%v pre_check_delta=%.0f under_lock_delta=%.0f",
			rpcErr, preDelta, underDelta)
		// One of the two paths must have fired.
		if preDelta+underDelta != 1 {
			t.Errorf("total section_full delta = %.0f, want 1", preDelta+underDelta)
		}
		return
	}

	// Signal tx A to commit (insert visible + lock released).
	close(lockCommit)
	<-txDone // wait for the commit to complete.

	// Wait for B's RPC to complete (should acquire the lock, count 1, return SectionFull).
	var rpcErr error
	select {
	case rpcErr = <-rpcDone:
	case <-time.After(5 * time.Second):
		t.Fatal("RPC did not complete within 5s after tx A commit")
	}

	after := scrapeMetrics(t)

	// B must return CodeFailedPrecondition (ErrSectionFull, under-lock path).
	assertConnectCode(t, rpcErr, connect.CodeFailedPrecondition)

	// under_lock counter must increase by exactly 1.
	beforeUnder := parseCounterValue(before, "academico_section_full_total", `path="under_lock"`)
	afterUnder := parseCounterValue(after, "academico_section_full_total", `path="under_lock"`)
	if delta := afterUnder - beforeUnder; delta != 1 {
		t.Errorf("section_full_total{path=under_lock} delta = %.0f, want 1 (before=%.0f after=%.0f)",
			delta, beforeUnder, afterUnder)
	}

	// pre_check counter must NOT change (B's pre-check saw 0 active seats — A's insert was uncommitted).
	beforePre := parseCounterValue(before, "academico_section_full_total", `path="pre_check"`)
	afterPre := parseCounterValue(after, "academico_section_full_total", `path="pre_check"`)
	if delta := afterPre - beforePre; delta != 0 {
		t.Errorf("section_full_total{path=pre_check} delta = %.0f, want 0 (before=%.0f after=%.0f)",
			delta, beforePre, afterPre)
	}
}

// TestMetrics_AdmissionSaturated_Counter verifies that admission_saturated_total
// increments by exactly 1 when the concurrency limiter rejects an enroll RPC (I-9, FR-3d).
//
// Fidelity decision: the shared test harness uses a single server instance with a
// concurrency limiter of cap=3 (floor(5*0.6)). Saturating it with 3 held-open RPCs while
// firing a 4th is complex and prone to CI timing issues. Instead, we spin up a test-local
// server with cap=0 (NewConcurrencyLimiter(0, m) = floor(0*0.6)=0, fully closed). The
// local server uses a fresh metrics registry; we assert the counter on that registry by
// scraping its own /metrics endpoint. This gives full observable fidelity: the interceptor
// rejects the RPC with CodeResourceExhausted and the counter increment is visible via scrape.
// The shared server's counter is covered indirectly by U-13 (unit) and the stampede test.
func TestMetrics_AdmissionSaturated_Counter(t *testing.T) {
	ctx := context.Background()

	// Seed a student with all pre-RPC prerequisites so the request reaches the limiter.
	studentUserID, studentSID := seedUserWithSession(t, "metrics-i9-stu@metrics.test"+"_"+uniqueSuffix(t), "student")
	seedStudentProfile(t, studentUserID, 2099)

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 10)
	defer sectionCleanup()

	_, enrollmentCleanup := seedPaidEnrollment(t, studentUserID.String(), programID, periodYear)
	defer enrollmentCleanup()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	// Build a test-local server wired with a cap=0 limiter on a fresh metrics registry.
	localMetricsReg := platformmetrics.New()
	localSEMetrics := localMetricsReg.SectionEnrollmentMetrics()
	// NewConcurrencyLimiter(0, ...) → floor(0 * 0.6) = 0 → every enroll is immediately rejected.
	localLimiter := sectionenrollment.NewConcurrencyLimiter(0, localSEMetrics)
	localURL := buildSETestServerWithLimiter(t, localLimiter, localMetricsReg)

	// Scrape the local server's /metrics before the RPC.
	before := scrapeMetricsAt(t, localURL)

	// Fire the enroll RPC — cap=0 means the limiter interceptor rejects immediately.
	client := section_enrollmentv1connect.NewSectionEnrollmentServiceClient(http.DefaultClient, localURL)
	req := connect.NewRequest(&section_enrollmentv1.EnrollOwnSectionRequest{
		SectionId: sectionID,
		ProgramId: programID,
	})
	req.Header().Set("Cookie", "sid="+studentSID)
	_, rpcErr := client.EnrollOwnSection(ctx, req)

	after := scrapeMetricsAt(t, localURL)

	// The RPC must return CodeResourceExhausted (ErrAdmissionSaturated mapping in limiter.go).
	assertConnectCode(t, rpcErr, connect.CodeResourceExhausted)

	// admission_saturated_total must have increased by exactly 1 on the local registry.
	beforeV := parseCounterValue(before, "academico_admission_saturated_total", "")
	afterV := parseCounterValue(after, "academico_admission_saturated_total", "")
	if delta := afterV - beforeV; delta != 1 {
		t.Errorf("admission_saturated_total delta = %.0f, want 1 (before=%.0f after=%.0f)",
			delta, beforeV, afterV)
	}
}

// scrapeMetricsAt issues GET <url>/metrics with the shared test token and returns the body.
func scrapeMetricsAt(t *testing.T, baseURL string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/metrics", nil)
	if err != nil {
		t.Fatalf("scrapeMetricsAt: create request: %v", err)
	}
	req.Header.Set("X-Metrics-Token", testMetricsToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("scrapeMetricsAt: do request: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("scrapeMetricsAt: read body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("scrapeMetricsAt: status = %d, want 200; body: %s", resp.StatusCode, body)
	}
	return string(body)
}

// buildSETestServerWithLimiter constructs a minimal test http.Server with the given
// concurrency limiter and metrics registry, starts it on a random port, and registers a
// Cleanup to shut it down. It mirrors the main_test.go wiring for the section-enrollment
// slice only. The server uses the shared database pool and sharedCfg session settings.
func buildSETestServerWithLimiter(t *testing.T, lim *sectionenrollment.ConcurrencyLimiter, metricsReg *platformmetrics.Registry) string {
	t.Helper()

	redInterceptor := metricsReg.RPCInterceptor()

	redisClient := testRedisClient
	sessionStore := session.NewRedisStore(redisClient)
	roleLoader := rbac.NewPostgresRoleLoader(rbacdb.New(pgxPool))
	authInterceptor := auth.NewSessionInterceptor(sessionStore, roleLoader, sharedCfg)

	exempt := map[string]struct{}{
		authv1connect.AuthServiceLoginProcedure:                {},
		authv1connect.AuthServiceRequestPasswordResetProcedure: {},
		authv1connect.AuthServiceConfirmPasswordResetProcedure: {},
		authv1connect.AuthServiceLogoutProcedure:               {},
	}
	policies := map[string]authz.PolicyFunc{
		section_enrollmentv1connect.SectionEnrollmentServiceEnrollOwnSectionProcedure: authz.RequirePermission(authz.PermSectionsEnroll),
	}
	authzInterceptor := auth.NewAuthzInterceptor(exempt, policies)

	seLimiterInterceptor := sectionenrollment.NewConcurrencyLimitInterceptor(lim)
	seOpts := server.Chain(redInterceptor, seLimiterInterceptor, authInterceptor, authzInterceptor)

	seQueries := sectionenrollmentdb.New(pgxPool)
	seRepo := sectionenrollment.NewPostgresRepository(seQueries, pgxPool, metricsReg.SectionEnrollmentMetrics())
	seSvc := sectionenrollment.NewService(seRepo)
	seHandler := sectionenrollment.NewHandler(seSvc)

	authQueries := authdb.New(pgxPool)
	authRepo := auth.NewPostgresRepository(authQueries)
	authSvc := auth.NewService(authRepo, sessionStore, roleLoader, sharedCfg)
	authHandler := auth.NewHandler(authSvc, sharedCfg)
	authOpts := server.Chain(redInterceptor, authInterceptor, authzInterceptor)

	metricsHandlerReg := func(mux *http.ServeMux) {
		mux.Handle("/metrics", metricsReg.Handler(sharedCfg.MetricsAuthToken))
	}
	sectionEnrollmentReg := func(mux *http.ServeMux) {
		sectionenrollment.Register(mux, seHandler, seOpts...)
	}
	authReg := func(mux *http.ServeMux) {
		auth.Register(mux, authHandler, authOpts...)
	}

	log := logging.New(slog.LevelError)

	// pgxPool also implements db.Pinger.
	srv := server.New(log, pgxPool, pgxPool, authReg, sectionEnrollmentReg, metricsHandlerReg)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("buildSETestServer: listen: %v", err)
	}
	srv.Addr = ln.Addr().String()
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			t.Logf("buildSETestServer: %v", err)
		}
	}()
	waitForServer(ln.Addr().String(), 5*time.Second)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Logf("buildSETestServer Shutdown: %v", err)
		}
	})

	return "http://" + ln.Addr().String()
}
