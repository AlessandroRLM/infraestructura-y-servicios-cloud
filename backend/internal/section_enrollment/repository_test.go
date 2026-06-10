package section_enrollment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/section_enrollment/section_enrollmentdb"
)

// fakeQuerier is a fake implementation of section_enrollmentdb.Querier for unit testing.
// It uses explicit called bool sentinels and configurable responses.
type fakeQuerier struct {
	// GetSectionCapacity (non-locking pre-check)
	getSectionCapacityCalled bool
	getSectionCapacityRow    section_enrollmentdb.GetSectionCapacityRow
	getSectionCapacityErr    error

	// GetSectionForUpdateWithWindow
	getSectionCalled bool
	getSectionRow    section_enrollmentdb.GetSectionForUpdateWithWindowRow
	getSectionErr    error

	// CountActiveSeats
	countSeatsCalled bool
	countSeatsResult int64
	countSeatsErr    error

	// ResolveEnrollmentByID (admin path)
	resolveByIDCalled bool
	resolveByIDRow    section_enrollmentdb.ResolveEnrollmentByIDRow
	resolveByIDErr    error

	// ResolveEnrollmentByStudentAndProgram (student path)
	resolveByStudentProgramCalled bool
	resolveByStudentProgramRow    section_enrollmentdb.ResolveEnrollmentByStudentAndProgramRow
	resolveByStudentProgramErr    error

	// CourseInProgram
	courseInProgramCalled bool
	courseInProgramResult bool
	courseInProgramErr    error

	// GetSectionEnrollmentByKeyForUpdate
	getKeyForUpdateCalled bool
	getKeyForUpdateRow    section_enrollmentdb.SectionEnrollment
	getKeyForUpdateErr    error

	// InsertSectionEnrollment
	insertCalled bool
	insertRow    section_enrollmentdb.SectionEnrollment
	insertErr    error

	// ReviveSectionEnrollment
	reviveCalled bool
	reviveRow    section_enrollmentdb.SectionEnrollment
	reviveErr    error

	// WithdrawSectionEnrollment
	withdrawCalled bool
	withdrawRow    section_enrollmentdb.SectionEnrollment
	withdrawErr    error

	// GetSectionEnrollmentByID
	getByIDCalled bool
	getByIDRow    section_enrollmentdb.SectionEnrollment
	getByIDErr    error

	// ListSectionEnrollments
	listCalled bool
	listRows   []section_enrollmentdb.SectionEnrollment
	listErr    error

	// ListOwnSectionEnrollments
	listOwnCalled bool
	listOwnRows   []section_enrollmentdb.SectionEnrollment
	listOwnErr    error
}

func (f *fakeQuerier) GetSectionCapacity(_ context.Context, _ pgtype.UUID) (section_enrollmentdb.GetSectionCapacityRow, error) {
	f.getSectionCapacityCalled = true
	return f.getSectionCapacityRow, f.getSectionCapacityErr
}

func (f *fakeQuerier) GetSectionForUpdateWithWindow(_ context.Context, _ pgtype.UUID) (section_enrollmentdb.GetSectionForUpdateWithWindowRow, error) {
	f.getSectionCalled = true
	return f.getSectionRow, f.getSectionErr
}

func (f *fakeQuerier) CountActiveSeats(_ context.Context, _ pgtype.UUID) (int64, error) {
	f.countSeatsCalled = true
	return f.countSeatsResult, f.countSeatsErr
}

func (f *fakeQuerier) ResolveEnrollmentByID(_ context.Context, _ pgtype.UUID) (section_enrollmentdb.ResolveEnrollmentByIDRow, error) {
	f.resolveByIDCalled = true
	return f.resolveByIDRow, f.resolveByIDErr
}

func (f *fakeQuerier) ResolveEnrollmentByStudentAndProgram(_ context.Context, _ section_enrollmentdb.ResolveEnrollmentByStudentAndProgramParams) (section_enrollmentdb.ResolveEnrollmentByStudentAndProgramRow, error) {
	f.resolveByStudentProgramCalled = true
	return f.resolveByStudentProgramRow, f.resolveByStudentProgramErr
}

func (f *fakeQuerier) CourseInProgram(_ context.Context, _ section_enrollmentdb.CourseInProgramParams) (bool, error) {
	f.courseInProgramCalled = true
	return f.courseInProgramResult, f.courseInProgramErr
}

func (f *fakeQuerier) GetSectionEnrollmentByKeyForUpdate(_ context.Context, _ section_enrollmentdb.GetSectionEnrollmentByKeyForUpdateParams) (section_enrollmentdb.SectionEnrollment, error) {
	f.getKeyForUpdateCalled = true
	return f.getKeyForUpdateRow, f.getKeyForUpdateErr
}

func (f *fakeQuerier) InsertSectionEnrollment(_ context.Context, _ section_enrollmentdb.InsertSectionEnrollmentParams) (section_enrollmentdb.SectionEnrollment, error) {
	f.insertCalled = true
	return f.insertRow, f.insertErr
}

func (f *fakeQuerier) ReviveSectionEnrollment(_ context.Context, _ pgtype.UUID) (section_enrollmentdb.SectionEnrollment, error) {
	f.reviveCalled = true
	return f.reviveRow, f.reviveErr
}

func (f *fakeQuerier) WithdrawSectionEnrollment(_ context.Context, _ pgtype.UUID) (section_enrollmentdb.SectionEnrollment, error) {
	f.withdrawCalled = true
	return f.withdrawRow, f.withdrawErr
}

func (f *fakeQuerier) GetSectionEnrollmentByID(_ context.Context, _ pgtype.UUID) (section_enrollmentdb.SectionEnrollment, error) {
	f.getByIDCalled = true
	return f.getByIDRow, f.getByIDErr
}

func (f *fakeQuerier) ListSectionEnrollments(_ context.Context, _ section_enrollmentdb.ListSectionEnrollmentsParams) ([]section_enrollmentdb.SectionEnrollment, error) {
	f.listCalled = true
	return f.listRows, f.listErr
}

func (f *fakeQuerier) ListOwnSectionEnrollments(_ context.Context, _ pgtype.UUID) ([]section_enrollmentdb.SectionEnrollment, error) {
	f.listOwnCalled = true
	return f.listOwnRows, f.listOwnErr
}

func (f *fakeQuerier) SetSectionEnrollmentOutcome(_ context.Context, _ section_enrollmentdb.SetSectionEnrollmentOutcomeParams) (section_enrollmentdb.SectionEnrollment, error) {
	return section_enrollmentdb.SectionEnrollment{}, nil
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
func newInsertedRow(seID, enrollmentID, sectionID uuid.UUID) section_enrollmentdb.SectionEnrollment {
	now := time.Now()
	return section_enrollmentdb.SectionEnrollment{
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
