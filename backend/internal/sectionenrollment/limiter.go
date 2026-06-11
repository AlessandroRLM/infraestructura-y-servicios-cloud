package sectionenrollment

import (
	"context"
	"log/slog"
	"math"

	"connectrpc.com/connect"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1/section_enrollmentv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/metrics"
)

// concurrencyLimiter is a bounded in-flight admission control mechanism for the
// inscription hot path. It is backed by a buffered channel acting as a counting
// semaphore: each acquisition sends a token into the channel; release reads it back.
//
// The cap is derived from the configured DB pool size as floor(poolSize * 0.6).
// This ensures that inscription transactions can never consume the entire pool, leaving
// at least 40% of connections available for login, catalog reads, and own-view RPCs.
//
// Admission control pattern: tryAcquire performs a non-blocking channel send.
// A full channel (cap reached) returns ok=false immediately — no goroutine is queued.
// Saturated requests are rejected with CodeResourceExhausted without opening a DB connection.
//
// The cap factor (0.6) and pool size are compile-time/startup tunables. Adjust
// DB_MAX_CONNS (env) and/or the factor in NewConcurrencyLimiter to tune backpressure.
type concurrencyLimiter struct {
	tokens chan struct{}
	m      *metrics.SectionEnrollment
}

// newConcurrencyLimiter constructs a concurrencyLimiter with the given capacity and a
// fresh metrics registry. cap=0 means every acquisition fails (fully closed path).
func newConcurrencyLimiter(cap int) *concurrencyLimiter {
	return newConcurrencyLimiterWithMetrics(cap, metrics.New().SectionEnrollmentMetrics())
}

// newConcurrencyLimiterWithMetrics constructs a concurrencyLimiter with the given capacity
// and metrics struct. Used in tests that verify counter increments.
func newConcurrencyLimiterWithMetrics(cap int, m *metrics.SectionEnrollment) *concurrencyLimiter {
	if cap < 0 {
		cap = 0
	}
	return &concurrencyLimiter{tokens: make(chan struct{}, cap), m: m}
}

// tryAcquire attempts a non-blocking acquisition of one slot.
// On success it returns a release func and ok=true.
// When the cap is exhausted it returns nil, false immediately — no waiting.
func (l *concurrencyLimiter) tryAcquire() (release func(), ok bool) {
	select {
	case l.tokens <- struct{}{}:
		return func() { <-l.tokens }, true
	default:
		return nil, false
	}
}

// NewConcurrencyLimiter constructs a concurrencyLimiter whose cap is
// floor(poolSize * 0.6). The formula ensures inscription transactions cannot
// starve other endpoints of DB connections under stampede load.
//
// poolSize is the value of cfg.DBMaxConns (DB_MAX_CONNS env var, default 10).
// With poolSize=10 the cap is 6; with the test harness poolSize=5 the cap is 3.
// m provides the domain counters; it must not be nil.
func NewConcurrencyLimiter(poolSize int, m *metrics.SectionEnrollment) *concurrencyLimiter {
	cap := int(math.Floor(float64(poolSize) * 0.6))
	return newConcurrencyLimiterWithMetrics(cap, m)
}

// enrollProcedures is the set of inscription procedures subject to admission control.
// Only these two procedures are hot-path; list/get/withdraw are excluded.
var enrollProcedures = map[string]struct{}{
	section_enrollmentv1connect.SectionEnrollmentServiceEnrollOwnSectionProcedure: {},
	section_enrollmentv1connect.SectionEnrollmentServiceEnrollSectionProcedure:    {},
}

// NewConcurrencyLimitInterceptor returns a Connect unary interceptor that enforces
// the admission control cap for the two inscription procedures. Non-enroll procedures
// are passed through without acquiring a slot, so a single handler registration covers
// the whole service with selective admission control.
//
// The interceptor must be chained BEFORE the auth interceptor so that saturated
// enroll requests are rejected before any session/DB work occurs.
func NewConcurrencyLimitInterceptor(lim *concurrencyLimiter) connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, controlled := enrollProcedures[req.Spec().Procedure]; !controlled {
				// Not an enroll procedure — bypass admission control.
				return next(ctx, req)
			}
			release, ok := lim.tryAcquire()
			if !ok {
				slog.WarnContext(ctx, "section enrollment admission rejected: limiter saturated",
					"in_flight", len(lim.tokens),
					"cap", cap(lim.tokens),
					"procedure", req.Spec().Procedure,
				)
				lim.m.AdmissionSaturated.Inc()
				return nil, connect.NewError(connect.CodeResourceExhausted, ErrAdmissionSaturated)
			}
			defer release()
			return next(ctx, req)
		})
	})
}
