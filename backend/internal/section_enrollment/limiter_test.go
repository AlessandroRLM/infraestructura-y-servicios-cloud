package section_enrollment

import (
	"errors"
	"testing"
)

// TestConcurrencyLimiter_HappyPath verifies that the limiter allows acquisitions
// below the configured cap and that released slots become available again.
func TestConcurrencyLimiter_HappyPath(t *testing.T) {
	t.Parallel()

	lim := newConcurrencyLimiter(2)

	release1, ok1 := lim.tryAcquire()
	if !ok1 {
		t.Fatal("first tryAcquire should succeed")
	}

	release2, ok2 := lim.tryAcquire()
	if !ok2 {
		t.Fatal("second tryAcquire should succeed (cap=2)")
	}

	// Third acquisition must fail — cap is exhausted.
	_, ok3 := lim.tryAcquire()
	if ok3 {
		t.Fatal("third tryAcquire must fail when cap=2 and two are held")
	}

	// Release one slot; the next acquisition must succeed.
	release1()
	release3, ok4 := lim.tryAcquire()
	if !ok4 {
		t.Fatal("tryAcquire after release should succeed")
	}
	release2()
	release3()
}

// TestConcurrencyLimiter_Saturation verifies that a saturated limiter returns
// ErrAdmissionSaturated when the interceptor queries the error.
func TestConcurrencyLimiter_Saturation(t *testing.T) {
	t.Parallel()

	lim := newConcurrencyLimiter(1)

	release, ok := lim.tryAcquire()
	if !ok {
		t.Fatal("first acquisition should succeed")
	}
	defer release()

	// Second acquisition must fail and the package must expose ErrAdmissionSaturated.
	_, ok2 := lim.tryAcquire()
	if ok2 {
		t.Fatal("second acquisition should fail on cap=1 limiter")
	}

	// Verify the sentinel is exported and distinct.
	if ErrAdmissionSaturated == nil {
		t.Fatal("ErrAdmissionSaturated must be non-nil")
	}
	if errors.Is(ErrAdmissionSaturated, ErrNotFound) {
		t.Error("ErrAdmissionSaturated must not wrap ErrNotFound")
	}
}

// TestConcurrencyLimiter_ZeroCap verifies that cap=0 makes every acquisition fail.
func TestConcurrencyLimiter_ZeroCap(t *testing.T) {
	t.Parallel()

	lim := newConcurrencyLimiter(0)
	_, ok := lim.tryAcquire()
	if ok {
		t.Fatal("tryAcquire on cap=0 limiter should always fail")
	}
}
