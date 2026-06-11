package enrollment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/enrollment/enrollmentdb"
)

// fakeQuerier is a test double for enrollmentdb.Querier.
// Each method records whether it was called via the called sentinel.
type fakeQuerier struct {
	lockQuotaCalled bool
	lockQuotaErr    error
	lockCapacity    int32

	countActiveCalled bool
	countActiveErr    error
	countActiveResult int64

	getByKeyCalled bool
	getByKeyErr    error
	getByKeyResult enrollmentdb.Enrollment

	insertCalled bool
	insertErr    error
	insertResult enrollmentdb.Enrollment

	reviveCalled bool
	reviveErr    error
	reviveResult enrollmentdb.Enrollment

	markPaidCalled bool
	markPaidErr    error
	markPaidResult enrollmentdb.Enrollment

	cancelCalled bool
	cancelErr    error
	cancelRows   int64

	getEnrollmentCalled bool
	getEnrollmentErr    error
	getEnrollmentResult enrollmentdb.Enrollment

	listCalled bool
	listResult []enrollmentdb.Enrollment
	listErr    error

	listOwnResult []enrollmentdb.Enrollment
	listOwnErr    error
}

func (f *fakeQuerier) LockProgramQuotaForYear(_ context.Context, _ enrollmentdb.LockProgramQuotaForYearParams) (int32, error) {
	f.lockQuotaCalled = true
	return f.lockCapacity, f.lockQuotaErr
}

func (f *fakeQuerier) CountActiveEnrollments(_ context.Context, _ enrollmentdb.CountActiveEnrollmentsParams) (int64, error) {
	f.countActiveCalled = true
	return f.countActiveResult, f.countActiveErr
}

func (f *fakeQuerier) GetEnrollmentByKeyForUpdate(_ context.Context, _ enrollmentdb.GetEnrollmentByKeyForUpdateParams) (enrollmentdb.Enrollment, error) {
	f.getByKeyCalled = true
	return f.getByKeyResult, f.getByKeyErr
}

func (f *fakeQuerier) InsertEnrollment(_ context.Context, _ enrollmentdb.InsertEnrollmentParams) (enrollmentdb.Enrollment, error) {
	f.insertCalled = true
	return f.insertResult, f.insertErr
}

func (f *fakeQuerier) ReviveEnrollment(_ context.Context, _ enrollmentdb.ReviveEnrollmentParams) (enrollmentdb.Enrollment, error) {
	f.reviveCalled = true
	return f.reviveResult, f.reviveErr
}

func (f *fakeQuerier) MarkEnrollmentPaid(_ context.Context, _ enrollmentdb.MarkEnrollmentPaidParams) (enrollmentdb.Enrollment, error) {
	f.markPaidCalled = true
	return f.markPaidResult, f.markPaidErr
}

func (f *fakeQuerier) CancelEnrollment(_ context.Context, _ enrollmentdb.CancelEnrollmentParams) (int64, error) {
	f.cancelCalled = true
	return f.cancelRows, f.cancelErr
}

func (f *fakeQuerier) GetEnrollment(_ context.Context, _ pgtype.UUID) (enrollmentdb.Enrollment, error) {
	f.getEnrollmentCalled = true
	return f.getEnrollmentResult, f.getEnrollmentErr
}

func (f *fakeQuerier) ListEnrollments(_ context.Context, _ enrollmentdb.ListEnrollmentsParams) ([]enrollmentdb.Enrollment, error) {
	f.listCalled = true
	return f.listResult, f.listErr
}

func (f *fakeQuerier) ListOwnEnrollments(_ context.Context, _ pgtype.UUID) ([]enrollmentdb.Enrollment, error) {
	return f.listOwnResult, f.listOwnErr
}

// pgUUID converts a uuid.UUID to pgtype.UUID for test fixtures.
func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// ---- CreateEnrollmentTx unit tests ----

// TestCreateEnrollmentTx_QuotaNotFound verifies that a missing program_quotas row
// returns ErrQuotaNotFound before any seat counting or insert attempt.
func TestCreateEnrollmentTx_QuotaNotFound(t *testing.T) {
	// This test cannot run without a real pool (the tx boundary lives in the repo).
	// It is exercised by the integration test suite. Skipping here to keep unit tests fast.
	t.Skip("CreateEnrollmentTx requires a pgxpool; covered by integration tests")
}

// ---- MarkEnrollmentPaid unit tests ----

// TestMarkEnrollmentPaid_NotFound verifies that when the querier returns ErrNoRows
// (because the row does not exist or is already not-pending), the repo performs a
// GetEnrollment pre-fetch and returns ErrNotFound when the row itself is absent.
func TestMarkEnrollmentPaid_NotFound(t *testing.T) {
	id := uuid.New()
	actor := uuid.New()

	q := &fakeQuerier{
		// MarkEnrollmentPaid returns no rows because the row doesn't exist.
		markPaidErr: pgx.ErrNoRows,
		// Pre-fetch also finds nothing.
		getEnrollmentErr: pgx.ErrNoRows,
	}
	repo := &postgresRepository{q: q}
	_, err := repo.MarkEnrollmentPaid(context.Background(), id, &actor)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("MarkEnrollmentPaid absent row: want ErrNotFound, got %v", err)
	}
	if !q.markPaidCalled {
		t.Error("MarkEnrollmentPaid: querier.MarkEnrollmentPaid was not called")
	}
}

// TestMarkEnrollmentPaid_WrongState verifies that when the row exists but the status
// is not 'pending', MarkEnrollmentPaid returns ErrInvalidTransition.
func TestMarkEnrollmentPaid_WrongState(t *testing.T) {
	id := uuid.New()
	actor := uuid.New()

	q := &fakeQuerier{
		markPaidErr: pgx.ErrNoRows,
		// Pre-fetch returns the row with status 'cancelled'.
		getEnrollmentResult: enrollmentdb.Enrollment{
			ID:     pgUUID(id),
			Status: "cancelled",
		},
		getEnrollmentErr: nil,
	}
	repo := &postgresRepository{q: q}
	_, err := repo.MarkEnrollmentPaid(context.Background(), id, &actor)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("MarkEnrollmentPaid wrong state: want ErrInvalidTransition, got %v", err)
	}
}

// TestMarkEnrollmentPaid_Success verifies that a successful paid transition returns the row.
func TestMarkEnrollmentPaid_Success(t *testing.T) {
	id := uuid.New()
	actor := uuid.New()
	want := enrollmentdb.Enrollment{ID: pgUUID(id), Status: "paid"}

	q := &fakeQuerier{
		markPaidResult: want,
	}
	repo := &postgresRepository{q: q}
	got, err := repo.MarkEnrollmentPaid(context.Background(), id, &actor)
	if err != nil {
		t.Fatalf("MarkEnrollmentPaid success: unexpected error %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("MarkEnrollmentPaid success: got id %v, want %v", got.ID, want.ID)
	}
}

// ---- CancelEnrollment unit tests ----

// TestCancelEnrollment_NotFound verifies that when 0 rows are affected and the row
// does not exist, CancelEnrollment returns ErrNotFound.
func TestCancelEnrollment_NotFound(t *testing.T) {
	id := uuid.New()
	actor := uuid.New()

	q := &fakeQuerier{
		cancelRows: 0,
		// Pre-fetch: row absent.
		getEnrollmentErr: pgx.ErrNoRows,
	}
	repo := &postgresRepository{q: q}
	err := repo.CancelEnrollment(context.Background(), id, &actor)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("CancelEnrollment absent row: want ErrNotFound, got %v", err)
	}
}

// TestCancelEnrollment_AlreadyCancelled verifies that when 0 rows are affected and the
// row exists with status 'cancelled', CancelEnrollment returns ErrInvalidTransition.
func TestCancelEnrollment_AlreadyCancelled(t *testing.T) {
	id := uuid.New()
	actor := uuid.New()

	q := &fakeQuerier{
		cancelRows: 0,
		getEnrollmentResult: enrollmentdb.Enrollment{
			ID:     pgUUID(id),
			Status: "cancelled",
		},
	}
	repo := &postgresRepository{q: q}
	err := repo.CancelEnrollment(context.Background(), id, &actor)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("CancelEnrollment already-cancelled: want ErrInvalidTransition, got %v", err)
	}
}

// TestCancelEnrollment_Success verifies that a successful cancellation returns nil.
func TestCancelEnrollment_Success(t *testing.T) {
	id := uuid.New()
	actor := uuid.New()

	q := &fakeQuerier{cancelRows: 1}
	repo := &postgresRepository{q: q}
	if err := repo.CancelEnrollment(context.Background(), id, &actor); err != nil {
		t.Errorf("CancelEnrollment success: unexpected error %v", err)
	}
}

// ---- GetEnrollment unit tests ----

// TestGetEnrollment_NotFound verifies that a missing row returns ErrNotFound.
func TestGetEnrollment_NotFound(t *testing.T) {
	q := &fakeQuerier{getEnrollmentErr: pgx.ErrNoRows}
	repo := &postgresRepository{q: q}
	_, err := repo.GetEnrollment(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetEnrollment not found: want ErrNotFound, got %v", err)
	}
}

// TestGetEnrollment_Success verifies that a found row is returned without error.
func TestGetEnrollment_Success(t *testing.T) {
	id := uuid.New()
	want := enrollmentdb.Enrollment{ID: pgUUID(id), Status: "pending"}
	q := &fakeQuerier{getEnrollmentResult: want}
	repo := &postgresRepository{q: q}
	got, err := repo.GetEnrollment(context.Background(), id)
	if err != nil {
		t.Fatalf("GetEnrollment success: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("GetEnrollment success: got %v, want %v", got.ID, want.ID)
	}
}

// ---- ListEnrollments unit tests ----

// TestListEnrollments_DelegatesWithFilter verifies that ListEnrollments calls the querier.
func TestListEnrollments_DelegatesWithFilter(t *testing.T) {
	want := []enrollmentdb.Enrollment{{Status: "pending"}}
	q := &fakeQuerier{listResult: want}
	repo := &postgresRepository{q: q}
	got, err := repo.ListEnrollments(context.Background(), ListEnrollmentsFilter{})
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}
	if !q.listCalled {
		t.Error("ListEnrollments: querier.ListEnrollments was not called")
	}
	if len(got) != len(want) {
		t.Errorf("ListEnrollments: got %d rows, want %d", len(got), len(want))
	}
}

// ---- ListOwnEnrollments unit tests ----

// TestListOwnEnrollments_Delegates verifies delegation to the querier.
func TestListOwnEnrollments_Delegates(t *testing.T) {
	studentID := uuid.New()
	want := []enrollmentdb.Enrollment{{StudentID: pgUUID(studentID)}}
	q := &fakeQuerier{listOwnResult: want}
	repo := &postgresRepository{q: q}
	got, err := repo.ListOwnEnrollments(context.Background(), studentID)
	if err != nil {
		t.Fatalf("ListOwnEnrollments: %v", err)
	}
	if len(got) != len(want) {
		t.Errorf("ListOwnEnrollments: got %d rows, want %d", len(got), len(want))
	}
}
