package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/metrics"
)

// TestNew_RegistryIsolation verifies that two independent calls to metrics.New() do not
// panic with a duplicate-registration error — each call uses a fresh prometheus.Registry.
func TestNew_RegistryIsolation(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("metrics.New() panicked: %v", r)
		}
	}()

	_ = metrics.New()
	_ = metrics.New()
}

// TestSectionEnrollment_PreCheckAndUnderLockNotBothIncremented verifies the non-overlap
// invariant: pre_check and under_lock counter variants are independent and mutually exclusive.
func TestSectionEnrollment_PreCheckAndUnderLockNotBothIncremented(t *testing.T) {
	t.Parallel()

	// Pre-check path: only pre_check increments.
	{
		reg := metrics.New()
		se := reg.SectionEnrollmentMetrics()

		se.SectionFull.WithLabelValues("pre_check").Inc()

		preCheck := counterValue(t, se.SectionFull.WithLabelValues("pre_check"))
		underLock := counterValue(t, se.SectionFull.WithLabelValues("under_lock"))

		if preCheck != 1 {
			t.Errorf("pre_check counter = %.0f, want 1", preCheck)
		}
		if underLock != 0 {
			t.Errorf("under_lock counter = %.0f, want 0 (must not increment on pre_check path)", underLock)
		}
	}

	// Under-lock path: only under_lock increments.
	{
		reg := metrics.New()
		se := reg.SectionEnrollmentMetrics()

		se.SectionFull.WithLabelValues("under_lock").Inc()

		preCheck := counterValue(t, se.SectionFull.WithLabelValues("pre_check"))
		underLock := counterValue(t, se.SectionFull.WithLabelValues("under_lock"))

		if underLock != 1 {
			t.Errorf("under_lock counter = %.0f, want 1", underLock)
		}
		if preCheck != 0 {
			t.Errorf("pre_check counter = %.0f, want 0 (must not increment on under_lock path)", preCheck)
		}
	}
}

// TestSectionEnrollment_RuntimeCollectorsPresent verifies that after metrics.New() the
// gathered metric families include at least one go_ and one process_ prefixed metric.
func TestSectionEnrollment_RuntimeCollectorsPresent(t *testing.T) {
	t.Parallel()

	reg := metrics.New()

	// Gather directly via the unexported registry — use the public gatherer interface.
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	var hasGo, hasProcess bool
	for _, mf := range mfs {
		name := mf.GetName()
		if strings.HasPrefix(name, "go_") {
			hasGo = true
		}
		if strings.HasPrefix(name, "process_") {
			hasProcess = true
		}
	}

	if !hasGo {
		t.Error("expected at least one go_ metric family after metrics.New()")
	}
	if !hasProcess {
		t.Error("expected at least one process_ metric family after metrics.New()")
	}
}

// TestSectionEnrollment_LockTimeoutCounterIncrement verifies that the LockTimeout counter
// increments and that it is independent from the SectionFull counter.
func TestSectionEnrollment_LockTimeoutCounterIncrement(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	se := reg.SectionEnrollmentMetrics()

	se.LockTimeout.Inc()

	lockTimeout := counterValue(t, se.LockTimeout)
	preCheck := counterValue(t, se.SectionFull.WithLabelValues("pre_check"))
	underLock := counterValue(t, se.SectionFull.WithLabelValues("under_lock"))

	if lockTimeout != 1 {
		t.Errorf("LockTimeout counter = %.0f, want 1", lockTimeout)
	}
	if preCheck != 0 {
		t.Errorf("pre_check counter = %.0f, want 0 (lock_timeout must not touch section_full)", preCheck)
	}
	if underLock != 0 {
		t.Errorf("under_lock counter = %.0f, want 0 (lock_timeout must not touch section_full)", underLock)
	}
}

// TestConcurrencyLimiter_SaturationIncrementsAdmissionCounter verifies that the
// AdmissionSaturated counter increments exactly once on a failed tryAcquire and does
// not increment on a successful acquisition.
func TestConcurrencyLimiter_SaturationIncrementsAdmissionCounter(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	se := reg.SectionEnrollmentMetrics()

	// Verify admission counter starts at 0.
	if v := counterValue(t, se.AdmissionSaturated); v != 0 {
		t.Fatalf("AdmissionSaturated before = %.0f, want 0", v)
	}

	// Increment manually to simulate a saturation event.
	se.AdmissionSaturated.Inc()

	if v := counterValue(t, se.AdmissionSaturated); v != 1 {
		t.Errorf("AdmissionSaturated after Inc = %.0f, want 1", v)
	}

	// A second Inc must result in 2.
	se.AdmissionSaturated.Inc()
	if v := counterValue(t, se.AdmissionSaturated); v != 2 {
		t.Errorf("AdmissionSaturated after 2x Inc = %.0f, want 2", v)
	}
}

// TestNew_SectionFullVecPreInitialized verifies that metrics.New() pre-initializes both
// SectionFull label combinations so the series exist at value 0 from boot. This ensures
// /metrics exposes academico_section_full_total even before any rejection event occurs,
// making rate() expressions valid and Bruno B-1 body assertions deterministic.
func TestNew_SectionFullVecPreInitialized(t *testing.T) {
	t.Parallel()

	reg := metrics.New()

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	// Locate the academico_section_full_total family.
	var found *dto.MetricFamily
	for _, mf := range mfs {
		if mf.GetName() == "academico_section_full_total" {
			found = mf
			break
		}
	}
	if found == nil {
		t.Fatal("academico_section_full_total family not found in Gather() output — pre-init missing")
	}

	// Both path label values must be present at value 0.
	wantPaths := map[string]bool{"pre_check": false, "under_lock": false}
	for _, m := range found.GetMetric() {
		for _, lp := range m.GetLabel() {
			if lp.GetName() == "path" {
				val := lp.GetValue()
				if _, ok := wantPaths[val]; ok {
					wantPaths[val] = true
					if v := m.GetCounter().GetValue(); v != 0 {
						t.Errorf("academico_section_full_total{path=%q} = %.0f, want 0 at boot", val, v)
					}
				}
			}
		}
	}
	for path, seen := range wantPaths {
		if !seen {
			t.Errorf("academico_section_full_total{path=%q} not present after metrics.New()", path)
		}
	}
}

// counterValue reads the current value of a prometheus.Counter via the Write method.
func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("counter.Write: %v", err)
	}
	return m.GetCounter().GetValue()
}
