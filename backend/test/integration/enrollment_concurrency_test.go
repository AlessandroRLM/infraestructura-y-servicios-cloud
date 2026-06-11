package integration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1/enrollmentv1connect"
)

// TestEnrollment_OversellRace verifies the quota invariant under concurrent load (S-03).
//
// Setup: capacity=N, fill N-1 active seats. Release two goroutines via a sync barrier,
// each calling CreateEnrollment for a distinct student targeting the last seat.
// Invariant: exactly one succeeds (CodeOK), the other fails (CodeFailedPrecondition),
// and the final active-seat count equals N — never N+1.
func TestEnrollment_OversellRace(t *testing.T) {
	ctx := context.Background()
	const capacity = 3

	_, adminSID1 := seedUserWithSession(t, "enroll-race-admin1@enroll.test", "admin")
	_, adminSID2 := seedUserWithSession(t, "enroll-race-admin2@enroll.test", "admin")

	programID, cleanup := seedProgramWithQuota(t, capacity, 2120)
	defer cleanup()

	// Seed capacity+1 students. The first capacity-1 pre-fill seats; the last two race.
	students := make([]string, capacity+1)
	for i := range students {
		uid := seedUserWithRole(t, "enroll-race-s"+string(rune('a'+i))+"@enroll.test", "student")
		seedStudentProfile(t, uid, 2120)
		students[i] = uid.String()
	}

	client1 := newEnrollmentClient(nil)
	client2 := newEnrollmentClient(nil)

	// Fill N-1 seats sequentially.
	for i := 0; i < capacity-1; i++ {
		e, err := enrollmentCreate(ctx, client1, adminSID1, students[i], programID, 2120)
		if err != nil {
			t.Fatalf("pre-fill seat %d: %v", i, err)
		}
		cleanupEnrollment(t, e.GetId())
	}

	// Two goroutines race for the last seat, released together via a buffered channel.
	type result struct {
		id  string
		err error
	}
	results := make(chan result, 2)

	// barrier is a channel closed at once to release both goroutines simultaneously.
	barrier := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(2)

	race := func(adminSID, studentID string, client enrollmentv1connect.EnrollmentServiceClient) {
		ready.Done()
		<-barrier // block until barrier is closed

		// Use a bounded context so a deadlock in CI does not hang indefinitely.
		raceCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		req := connect.NewRequest(&enrollmentv1.CreateEnrollmentRequest{
			StudentId: studentID,
			ProgramId: programID,
			Year:      2120,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		resp, err := client.CreateEnrollment(raceCtx, req)
		if err == nil {
			results <- result{id: resp.Msg.GetId(), err: nil}
		} else {
			results <- result{err: err}
		}
	}

	go race(adminSID1, students[capacity-1], client1)
	go race(adminSID2, students[capacity], client2)

	ready.Wait()   // wait until both goroutines are set up and blocked at the barrier
	close(barrier) // release both simultaneously

	// Collect results.
	r1 := <-results
	r2 := <-results

	// Count outcomes.
	var successCount, failCount int
	for _, r := range []result{r1, r2} {
		if r.err == nil {
			successCount++
			cleanupEnrollment(t, r.id)
		} else {
			ce, ok := r.err.(*connect.Error)
			if !ok || ce.Code() != connect.CodeFailedPrecondition {
				t.Errorf("loser should return CodeFailedPrecondition, got %v", r.err)
			}
			failCount++
		}
	}

	if successCount != 1 {
		t.Errorf("expected exactly 1 success, got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("expected exactly 1 failure, got %d", failCount)
	}

	// Assert: final active-seat count equals capacity — oversell must never occur.
	var activeCount int
	err := pgxPool.QueryRow(ctx,
		`SELECT count(*) FROM enrollments WHERE program_id = $1 AND year = $2 AND status <> 'cancelled' AND deleted_at IS NULL`,
		programID, 2120,
	).Scan(&activeCount)
	if err != nil {
		t.Fatalf("count active seats: %v", err)
	}
	if activeCount != capacity {
		t.Errorf("active seat count = %d, want exactly %d (oversell detected!)", activeCount, capacity)
	}
}
