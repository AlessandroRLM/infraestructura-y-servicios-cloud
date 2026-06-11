// Package metrics provides a custom Prometheus registry, domain counters, and runtime
// collectors for the application. All metrics use the academico_ namespace prefix to
// avoid collisions with standard Go/process collector names.
package metrics

import (
	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry wraps a custom *prometheus.Registry. Using a custom registry (rather than
// the global DefaultRegisterer) prevents duplicate-registration panics when multiple
// test instances construct the same metrics objects in the same process.
type Registry struct {
	reg               *prometheus.Registry
	sectionEnrollment *SectionEnrollment
	rpcRequests       *prometheus.CounterVec
	rpcDuration       *prometheus.HistogramVec
}

// SectionEnrollment holds the domain rejection counters for the section enrollment
// hot path. All fields are injected into the postgresRepository and concurrencyLimiter
// via their constructors (AD7).
type SectionEnrollment struct {
	// SectionFull counts capacity-rejection events. The path label differentiates
	// the pre-check gate (path="pre_check") from the authoritative under-lock gate
	// (path="under_lock"). These two variants are mutually exclusive per call.
	SectionFull *prometheus.CounterVec
	// LockTimeout counts 55P03 Postgres lock-timeout errors on the section row lock.
	LockTimeout prometheus.Counter
	// AdmissionSaturated counts requests rejected by the concurrency limiter when
	// the in-flight semaphore is at capacity.
	AdmissionSaturated prometheus.Counter
}

// New constructs a Registry with a fresh *prometheus.Registry, registers the Go runtime
// and OS process collectors, and creates all application-level metrics. Each call returns
// an independent registry — safe for concurrent use in tests.
func New() *Registry {
	reg := prometheus.NewRegistry()

	// Register standard runtime collectors on the custom registry. These keep their
	// canonical go_ / process_ prefixes for compatibility with off-the-shelf dashboards.
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	sectionFull := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "academico",
		Name:      "section_full_total",
		Help:      "Total number of section enrollment rejections due to capacity, by gate path.",
	}, []string{"path"})

	lockTimeout := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "academico",
		Name:      "section_lock_timeout_total",
		Help:      "Total number of Postgres lock-timeout errors (55P03) on the section row lock.",
	})

	admissionSaturated := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "academico",
		Name:      "admission_saturated_total",
		Help:      "Total number of enroll requests rejected by the concurrency limiter.",
	})

	rpcRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "academico",
		Name:      "rpc_requests_total",
		Help:      "Total number of Connect RPC calls, labeled by service, method, and status code.",
	}, []string{"service", "method", "code"})

	rpcDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "academico",
		Name:      "rpc_duration_seconds",
		Help:      "Duration of Connect RPC calls in seconds, labeled by service and method.",
		// Custom buckets tuned for this workload: fine sub-100ms resolution for
		// single-query RPCs, a 10-second ceiling for cached report generation.
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"service", "method"})

	reg.MustRegister(sectionFull, lockTimeout, admissionSaturated, rpcRequests, rpcDuration)

	// Pre-initialize both SectionFull label combinations so that the series appear in
	// the /metrics output from the first scrape — even before any capacity rejection
	// has occurred. This makes rate() expressions valid from boot and ensures Bruno /
	// integration tests that assert body.includes('academico_section_full_total') pass
	// without needing a prior rejection event.
	sectionFull.WithLabelValues("pre_check")
	sectionFull.WithLabelValues("under_lock")

	se := &SectionEnrollment{
		SectionFull:        sectionFull,
		LockTimeout:        lockTimeout,
		AdmissionSaturated: admissionSaturated,
	}

	return &Registry{
		reg:               reg,
		sectionEnrollment: se,
		rpcRequests:       rpcRequests,
		rpcDuration:       rpcDuration,
	}
}

// SectionEnrollmentMetrics returns the domain counters for the section enrollment slice.
// The returned pointer is the same instance for the lifetime of the Registry.
func (r *Registry) SectionEnrollmentMetrics() *SectionEnrollment {
	return r.sectionEnrollment
}

// Gather delegates to the underlying prometheus.Registry.Gather. Exposed primarily for
// testing; the /metrics handler uses promhttp.HandlerFor(r.reg, ...) instead.
func (r *Registry) Gather() ([]*dto.MetricFamily, error) {
	return r.reg.Gather()
}
