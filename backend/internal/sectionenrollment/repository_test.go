package sectionenrollment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment/sectionenrollmentdb"
)

// fakeQuerier is a fake implementation of sectionenrollmentdb.Querier for unit testing.
// It uses explicit called bool sentinels and configurable responses.
type fakeQuerier struct {
	// GetSectionCapacity (non-locking pre-check)
	getSectionCapacityCalled bool
	getSectionCapacityRow    sectionenrollmentdb.GetSectionCapacityRow
	getSectionCapacityErr    error

	// GetSectionForUpdateWithWindow
	getSectionCalled bool
	getSectionRow    sectionenrollmentdb.GetSectionForUpdateWithWindowRow
	getSectionErr    error

	// CountActiveSeats
	countSeatsCalled bool
	countSeatsResult int64
	countSeatsErr    error

	// ResolveEnrollmentByID (admin path)
	resolveByIDCalled bool
	resolveByIDRow    sectionenrollmentdb.ResolveEnrollmentByIDRow
	resolveByIDErr    error

	// ResolveEnrollmentByStudentAndProgram (student path)
	resolveByStudentProgramCalled bool
	resolveByStudentProgramRow    sectionenrollmentdb.ResolveEnrollmentByStudentAndProgramRow
	resolveByStudentProgramErr    error

	// CourseInProgram
	courseInProgramCalled bool
	courseInProgramResult bool
	courseInProgramErr    error

	// GetSectionEnrollmentByKeyForUpdate
	getKeyForUpdateCalled bool
	getKeyForUpdateRow    sectionenrollmentdb.SectionEnrollment
	getKeyForUpdateErr    error

	// InsertSectionEnrollment
	insertCalled bool
	insertRow    sectionenrollmentdb.SectionEnrollment
	insertErr    error

	// ReviveSectionEnrollment
	reviveCalled bool
	reviveRow    sectionenrollmentdb.SectionEnrollment
	reviveErr    error

	// WithdrawSectionEnrollment
	withdrawCalled bool
	withdrawRow    sectionenrollmentdb.SectionEnrollment
	withdrawErr    error

	// GetSectionEnrollmentByID
	getByIDCalled bool
	getByIDRow    sectionenrollmentdb.SectionEnrollment
	getByIDErr    error

	// ListSectionEnrollments
	listCalled bool
	listRows   []sectionenrollmentdb.SectionEnrollment
	listErr    error

	// ListOwnSectionEnrollments
	listOwnCalled bool
	listOwnRows   []sectionenrollmentdb.SectionEnrollment
	listOwnErr    error

	// SetSectionEnrollmentOutcome
	setOutcomeCalled     bool
	setOutcomeRow        sectionenrollmentdb.SectionEnrollment
	setOutcomeErr        error
	setOutcomeLastStatus string
	setOutcomeLastGrade  pgtype.Numeric
}

func (f *fakeQuerier) GetSectionCapacity(_ context.Context, _ pgtype.UUID) (sectionenrollmentdb.GetSectionCapacityRow, error) {
	f.getSectionCapacityCalled = true
	return f.getSectionCapacityRow, f.getSectionCapacityErr
}

func (f *fakeQuerier) GetSectionForUpdateWithWindow(_ context.Context, _ pgtype.UUID) (sectionenrollmentdb.GetSectionForUpdateWithWindowRow, error) {
	f.getSectionCalled = true
	return f.getSectionRow, f.getSectionErr
}

func (f *fakeQuerier) CountActiveSeats(_ context.Context, _ pgtype.UUID) (int64, error) {
	f.countSeatsCalled = true
	return f.countSeatsResult, f.countSeatsErr
}

func (f *fakeQuerier) ResolveEnrollmentByID(_ context.Context, _ pgtype.UUID) (sectionenrollmentdb.ResolveEnrollmentByIDRow, error) {
	f.resolveByIDCalled = true
	return f.resolveByIDRow, f.resolveByIDErr
}

func (f *fakeQuerier) ResolveEnrollmentByStudentAndProgram(_ context.Context, _ sectionenrollmentdb.ResolveEnrollmentByStudentAndProgramParams) (sectionenrollmentdb.ResolveEnrollmentByStudentAndProgramRow, error) {
	f.resolveByStudentProgramCalled = true
	return f.resolveByStudentProgramRow, f.resolveByStudentProgramErr
}

func (f *fakeQuerier) CourseInProgram(_ context.Context, _ sectionenrollmentdb.CourseInProgramParams) (bool, error) {
	f.courseInProgramCalled = true
	return f.courseInProgramResult, f.courseInProgramErr
}

func (f *fakeQuerier) GetSectionEnrollmentByKeyForUpdate(_ context.Context, _ sectionenrollmentdb.GetSectionEnrollmentByKeyForUpdateParams) (sectionenrollmentdb.SectionEnrollment, error) {
	f.getKeyForUpdateCalled = true
	return f.getKeyForUpdateRow, f.getKeyForUpdateErr
}

func (f *fakeQuerier) InsertSectionEnrollment(_ context.Context, _ sectionenrollmentdb.InsertSectionEnrollmentParams) (sectionenrollmentdb.SectionEnrollment, error) {
	f.insertCalled = true
	return f.insertRow, f.insertErr
}

func (f *fakeQuerier) ReviveSectionEnrollment(_ context.Context, _ pgtype.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	f.reviveCalled = true
	return f.reviveRow, f.reviveErr
}

func (f *fakeQuerier) WithdrawSectionEnrollment(_ context.Context, _ pgtype.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	f.withdrawCalled = true
	return f.withdrawRow, f.withdrawErr
}

func (f *fakeQuerier) GetSectionEnrollmentByID(_ context.Context, _ pgtype.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	f.getByIDCalled = true
	return f.getByIDRow, f.getByIDErr
}

func (f *fakeQuerier) ListSectionEnrollments(_ context.Context, _ sectionenrollmentdb.ListSectionEnrollmentsParams) ([]sectionenrollmentdb.SectionEnrollment, error) {
	f.listCalled = true
	return f.listRows, f.listErr
}

func (f *fakeQuerier) ListOwnSectionEnrollments(_ context.Context, _ pgtype.UUID) ([]sectionenrollmentdb.SectionEnrollment, error) {
	f.listOwnCalled = true
	return f.listOwnRows, f.listOwnErr
}

func (f *fakeQuerier) SetSectionEnrollmentOutcome(_ context.Context, arg sectionenrollmentdb.SetSectionEnrollmentOutcomeParams) (sectionenrollmentdb.SectionEnrollment, error) {
	f.setOutcomeCalled = true
	f.setOutcomeLastStatus = arg.Status
	f.setOutcomeLastGrade = arg.FinalGrade
	return f.setOutcomeRow, f.setOutcomeErr
}

// makePgUUID creates a valid pgtype.UUID from a uuid.UUID.
func makePgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// makeTimestamptz creates a valid pgtype.Timestamptz from a time.Time.
func makeTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// newInsertedRow creates a fake SectionEnrollment as returned by InsertSectionEnrollment.
func newInsertedRow(seID, enrollmentID, sectionID uuid.UUID) sectionenrollmentdb.SectionEnrollment {
	now := time.Now()
	return sectionenrollmentdb.SectionEnrollment{
		ID:           makePgUUID(seID),
		EnrollmentID: makePgUUID(enrollmentID),
		SectionID:    makePgUUID(sectionID),
		Status:       "in_progress",
		RegisteredAt: makeTimestamptz(now),
		CreatedAt:    makeTimestamptz(now),
		UpdatedAt:    makeTimestamptz(now),
	}
}

// --- Transaction wrapper used by the repository tests ---
// The repository opens a real transaction via pgxpool.Pool. To unit-test without a real DB,
// we test the happy/error paths through the Repository interface using a fake pool-free
// constructor that takes a Querier directly (bypasses BeginTx). In practice these paths are
// covered by integration tests; here we test only the paths that can be exercised without
// a live transaction (GetSectionEnrollment, ListSectionEnrollments, ListOwnSectionEnrollments,
// WithdrawSection, and error propagation through TranslatePgError).

func TestRepository_GetSectionEnrollmentByID_NotFound(t *testing.T) {
	t.Parallel()

	fq := &fakeQuerier{getByIDErr: pgx.ErrNoRows}
	repo := newFakeRepository(fq)

	_, err := repo.GetSectionEnrollment(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetSectionEnrollment(absent) = %v; want ErrNotFound", err)
	}
	if !fq.getByIDCalled {
		t.Error("GetSectionEnrollmentByID was not called")
	}
}

func TestRepository_GetSectionEnrollmentByID_Happy(t *testing.T) {
	t.Parallel()

	seID := uuid.New()
	fq := &fakeQuerier{getByIDRow: newInsertedRow(seID, uuid.New(), uuid.New())}
	repo := newFakeRepository(fq)

	row, err := repo.GetSectionEnrollment(context.Background(), seID)
	if err != nil {
		t.Fatalf("GetSectionEnrollment: unexpected error %v", err)
	}
	if row.ID != makePgUUID(seID) {
		t.Errorf("returned row id = %v, want %v", row.ID, makePgUUID(seID))
	}
}

func TestRepository_ListSectionEnrollments_Empty(t *testing.T) {
	t.Parallel()

	fq := &fakeQuerier{listRows: nil}
	repo := newFakeRepository(fq)

	rows, err := repo.ListSectionEnrollments(context.Background(), ListSectionEnrollmentsFilter{})
	if err != nil {
		t.Fatalf("ListSectionEnrollments: unexpected error %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
	if !fq.listCalled {
		t.Error("ListSectionEnrollments was not called on querier")
	}
}

func TestRepository_WithdrawSection_NotFound(t *testing.T) {
	t.Parallel()

	fq := &fakeQuerier{withdrawErr: pgx.ErrNoRows}
	repo := newFakeRepository(fq)

	_, err := repo.WithdrawSection(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("WithdrawSection(absent) = %v; want ErrNotFound", err)
	}
}

func TestRepository_WithdrawSection_Happy(t *testing.T) {
	t.Parallel()

	seID := uuid.New()
	row := newInsertedRow(seID, uuid.New(), uuid.New())
	row.Status = "withdrawn"
	fq := &fakeQuerier{withdrawRow: row}
	repo := newFakeRepository(fq)

	got, err := repo.WithdrawSection(context.Background(), seID)
	if err != nil {
		t.Fatalf("WithdrawSection: unexpected error %v", err)
	}
	if got.Status != "withdrawn" {
		t.Errorf("status = %q, want withdrawn", got.Status)
	}
}

func TestTranslatePgError_LockTimeout(t *testing.T) {
	t.Parallel()
	pgErr := &pgconn.PgError{Code: "55P03"}
	got := TranslatePgError(pgErr)
	if !errors.Is(got, ErrLockTimeout) {
		t.Errorf("TranslatePgError(55P03) = %v; want ErrLockTimeout", got)
	}
}

// =====================================================================================
// SetSectionEnrollmentOutcome transition guard tests (via setOutcomeWithQuerier seam).
// The SQL WHERE clause enforces: source IN (in_progress, passed, failed) AND target IN
// (passed, failed). Zero rows → ErrInvalidTransition. The tests here verify that the
// repository layer correctly maps the 0-row signal to ErrInvalidTransition and
// propagates real DB errors via TranslatePgError.
// =====================================================================================

func TestSetOutcomeWithQuerier_WithdrawnSource_ReturnsInvalidTransition(t *testing.T) {
	t.Parallel()

	// The SQL returns 0 rows (ErrNoRows) when the source status is withdrawn.
	fq := &fakeQuerier{setOutcomeErr: pgx.ErrNoRows}
	id := uuid.New()

	_, err := setOutcomeWithQuerier(context.Background(), fq, id, "passed", pgtype.Numeric{})
	if !fq.setOutcomeCalled {
		t.Fatal("SetSectionEnrollmentOutcome was not called")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("withdrawn source = %v; want ErrInvalidTransition", err)
	}
}

func TestSetOutcomeWithQuerier_InProgressTarget_ReturnsInvalidTransition(t *testing.T) {
	t.Parallel()

	// The SQL returns 0 rows when the target is in_progress (not in allowed targets).
	fq := &fakeQuerier{setOutcomeErr: pgx.ErrNoRows}
	id := uuid.New()

	_, err := setOutcomeWithQuerier(context.Background(), fq, id, "in_progress", pgtype.Numeric{})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("in_progress target = %v; want ErrInvalidTransition", err)
	}
}

func TestSetOutcomeWithQuerier_PassedToPassed_Allowed(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	expectedRow := newInsertedRow(id, uuid.New(), uuid.New())
	expectedRow.Status = "passed"
	fq := &fakeQuerier{setOutcomeRow: expectedRow}

	got, err := setOutcomeWithQuerier(context.Background(), fq, id, "passed", pgtype.Numeric{})
	if err != nil {
		t.Fatalf("passed→passed: unexpected error %v", err)
	}
	if got.Status != "passed" {
		t.Errorf("status = %q, want passed", got.Status)
	}
}

func TestSetOutcomeWithQuerier_FailedToPassed_Allowed(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	expectedRow := newInsertedRow(id, uuid.New(), uuid.New())
	expectedRow.Status = "passed"
	fq := &fakeQuerier{setOutcomeRow: expectedRow}

	got, err := setOutcomeWithQuerier(context.Background(), fq, id, "passed", pgtype.Numeric{})
	if err != nil {
		t.Fatalf("failed→passed: unexpected error %v", err)
	}
	if got.Status != "passed" {
		t.Errorf("status = %q, want passed", got.Status)
	}
}

func TestSetOutcomeWithQuerier_FinalGradeWrittenWithOutcome(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	expectedRow := newInsertedRow(id, uuid.New(), uuid.New())
	expectedRow.Status = "passed"

	var finalGrade pgtype.Numeric
	_ = finalGrade.Scan("4.0")

	fq := &fakeQuerier{setOutcomeRow: expectedRow}
	_, err := setOutcomeWithQuerier(context.Background(), fq, id, "passed", finalGrade)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.setOutcomeCalled {
		t.Fatal("SetSectionEnrollmentOutcome was not called")
	}
	// Verify the grade was forwarded to the querier.
	if !fq.setOutcomeLastGrade.Valid {
		t.Error("final_grade forwarded to querier must be valid (non-null)")
	}
}

func TestSetOutcomeWithQuerier_DBError_Propagated(t *testing.T) {
	t.Parallel()

	dbErr := &pgconn.PgError{Code: "23503"}
	fq := &fakeQuerier{setOutcomeErr: dbErr}
	id := uuid.New()

	_, err := setOutcomeWithQuerier(context.Background(), fq, id, "passed", pgtype.Numeric{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("FK violation = %v; want ErrInvalidInput", err)
	}
}
