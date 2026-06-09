package integration_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
)

const (
	stampedeN = 50
	stampedeK = 5
)

// TestSectionEnrollment_Stampede_ExactlyKSucceed is a HARD GATE test.
//
// N=50 goroutines attempt to self-enroll concurrently into a section with capacity K=5.
// All goroutines start simultaneously behind a sync.WaitGroup barrier (no time.Sleep in
// the hot path — the barrier uses WaitGroup channels). Goroutines retry on
// ResourceExhausted (admission limiter busy) with a short random jitter; they stop
// retrying on FailedPrecondition (section full) or a successful enrollment.
//
// Invariants:
//   - Exactly K goroutines succeed.
//   - Zero oversell: active seat count is exactly K after the stampede.
//   - No deadlocks: all goroutines complete within 30 seconds.
//   - The remaining N-K goroutines receive FailedPrecondition (section full) as their
//     terminal rejection (ResourceExhausted may appear transiently but is retried).
func TestSectionEnrollment_Stampede_ExactlyKSucceed(t *testing.T) {
	if testing.Short() {
		t.Skip("stampede test skipped in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, adminSID := seedUserWithSession(t, "se-stampede-admin@se.test", "admin")

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	defer programCleanup()

	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, true, false)
	defer periodCleanup()

	sectionID, sectionCleanup := seedSection(t, courseID, periodID, stampedeK)
	defer sectionCleanup()

	cleanupAllSectionEnrollmentsForSection(t, sectionID)

	// Seed N distinct students, each with a paid enrollment in the program for the period's year.
	type studentRecord struct {
		sessionID    string
		enrollmentID string
	}
	students := make([]studentRecord, stampedeN)
	for i := range stampedeN {
		email := "se-stampede-stu-" + uniqueSuffix(t) + "@se.test"
		userID, sid := seedUserWithSession(t, email, "student")
		seedStudentProfile(t, userID, 2099)
		enrollmentID, cleanup := seedPaidEnrollment(t, userID.String(), programID, periodYear)
		defer cleanup()
		students[i] = studentRecord{sessionID: sid, enrollmentID: enrollmentID}
	}

	client := newSectionEnrollmentClient(nil)

	// Barrier: all goroutines signal ready, then all wait for the start gun.
	var readyWG sync.WaitGroup
	readyWG.Add(stampedeN)
	startCh := make(chan struct{})

	var successCount atomic.Int32
	var errSectionFull atomic.Int32
	var errOther atomic.Int32

	var doneWG sync.WaitGroup
	doneWG.Add(stampedeN)

	for i := range stampedeN {
		go func(rec studentRecord) {
			defer doneWG.Done()
			readyWG.Done() // Signal this goroutine is positioned at the barrier.
			<-startCh      // Wait for the start gun (no time.Sleep).

			// Retry loop: back off only on ResourceExhausted (limiter busy).
			for {
				se, err := seEnrollOwn(ctx, client, rec.sessionID, sectionID, programID)
				if err == nil {
					successCount.Add(1)
					cleanupSectionEnrollment(t, se.GetId())
					return
				}

				var connectErr *connect.Error
				if !errors.As(err, &connectErr) {
					errOther.Add(1)
					return
				}

				switch connectErr.Code() {
				case connect.CodeFailedPrecondition:
					// Section is full — terminal rejection.
					errSectionFull.Add(1)
					return
				case connect.CodeResourceExhausted:
					// Admission limiter is saturated — back off and retry.
					if ctx.Err() != nil {
						errOther.Add(1)
						return
					}
					// Jitter: 5–25ms — just enough to desynchronise retries.
					jitter := time.Duration(5+rand.IntN(20)) * time.Millisecond //nolint:gosec
					select {
					case <-time.After(jitter):
					case <-ctx.Done():
						errOther.Add(1)
						return
					}
				default:
					errOther.Add(1)
					return
				}
			}
		}(students[i])
	}

	readyWG.Wait() // All goroutines are positioned at the barrier.
	close(startCh) // Fire the start gun — all goroutines unblock simultaneously.

	// Hard deadline: all goroutines must complete within the ctx deadline (30s).
	completedCh := make(chan struct{})
	go func() {
		doneWG.Wait()
		close(completedCh)
	}()

	select {
	case <-completedCh:
		// All goroutines completed — proceed to assertions.
	case <-time.After(30 * time.Second):
		t.Fatal("stampede test deadlocked: goroutines did not complete within 30s")
	}

	// --- Invariant checks ---

	// Exactly K must succeed.
	succeeded := int(successCount.Load())
	if succeeded != stampedeK {
		t.Errorf("exactly %d goroutines should succeed; got %d (full=%d, other=%d)",
			stampedeK, succeeded, errSectionFull.Load(), errOther.Load(),
		)
	}

	// Zero oversell: DB must reflect exactly K active seats.
	seats := activeSeatCount(t, sectionID)
	if seats != stampedeK {
		t.Errorf("zero-oversell violated: DB shows %d active seats, want %d", seats, stampedeK)
	}

	// No unexpected error codes.
	if errOther.Load() > 0 {
		t.Errorf("unexpected error codes in %d goroutines", errOther.Load())
	}

	// All N goroutines must be accounted for.
	total := int(successCount.Load()) + int(errSectionFull.Load()) + int(errOther.Load())
	if total != stampedeN {
		t.Errorf("goroutine tally mismatch: %d accounted for, want %d", total, stampedeN)
	}

	// Log breakdown for visibility.
	t.Logf("stampede result: success=%d full=%d other=%d / N=%d K=%d",
		successCount.Load(), errSectionFull.Load(), errOther.Load(),
		stampedeN, stampedeK,
	)

	// Admin must not be able to enroll one more — section is full.
	extraStudent := seedUserWithRole(t, "se-stampede-extra@se.test", "student")
	seedStudentProfile(t, extraStudent, 2099)
	extraEnrollmentID, cleanExtra := seedPaidEnrollment(t, extraStudent.String(), programID, periodYear)
	defer cleanExtra()
	_, err := seEnrollAdmin(context.Background(), client, adminSID, extraEnrollmentID, sectionID)
	if err == nil {
		t.Error("admin enroll after full stampede should fail (section full); got nil error")
	}
}
