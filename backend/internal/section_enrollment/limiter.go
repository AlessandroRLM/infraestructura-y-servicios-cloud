package section_enrollment

import (
	"context"
	"math"

	"connectrpc.com/connect"
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
}

// newConcurrencyLimiter constructs a concurrencyLimiter with the given capacity.
// cap=0 means every acquisition fails (fully closed path).
func newConcurrencyLimiter(cap int) *concurrencyLimiter {
	if cap < 0 {
		cap = 0
	}
	return &concurrencyLimiter{tokens: make(chan struct{}, cap)}
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
func NewConcurrencyLimiter(poolSize int) *concurrencyLimiter {
	cap := int(math.Floor(float64(poolSize) * 0.6))
	return newConcurrencyLimiter(cap)
}

// NewConcurrencyLimitInterceptor returns a Connect unary interceptor that enforces
// the admission control cap for inscription procedures. It must be chained BEFORE
// the auth interceptor so that saturated requests are rejected before any DB call.
//
// Only the two enroll procedures should use this interceptor; list/get procedures
// are read-only and must not be admission-controlled.
func NewConcurrencyLimitInterceptor(lim *concurrencyLimiter) connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			release, ok := lim.tryAcquire()
			if !ok {
				return nil, connect.NewError(connect.CodeResourceExhausted, ErrAdmissionSaturated)
			}
			defer release()
			return next(ctx, req)
		})
	})
}
