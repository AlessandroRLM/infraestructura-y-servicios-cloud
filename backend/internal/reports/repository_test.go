package reports

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/reports/reportsdb"
)

// fakeQuerier implements reportsdb.Querier with explicit called sentinels per method.
type fakeQuerier struct {
	// per-method called flags
	actaSectionExistsCalled     bool
	isTeacherForSectionCalled   bool
	actaForSectionAdminCalled   bool
	actaForSectionTeacherCalled bool
	occupancyPeriodExistsCalled bool
	occupancyForPeriodCalled    bool
	programExistsCalled         bool
	programSummaryCalled        bool
	studentExistsCalled         bool
	fichaForStudentCalled       bool

	// configurable return values
	actaSectionExistsResult     bool
	actaSectionExistsErr        error
	isTeacherForSectionResult   bool
	isTeacherForSectionErr      error
	actaForSectionAdminResult   []reportsdb.ActaForSectionAdminRow
	actaForSectionAdminErr      error
	actaForSectionTeacherResult []reportsdb.ActaForSectionByTeacherRow
	actaForSectionTeacherErr    error
	occupancyPeriodExistsResult bool
	occupancyPeriodExistsErr    error
	occupancyForPeriodResult    []reportsdb.OccupancyForPeriodRow
	occupancyForPeriodErr       error
	programExistsResult         bool
	programExistsErr            error
	programSummaryResult        []reportsdb.ProgramSummaryRow
	programSummaryErr           error
	studentExistsResult         bool
	studentExistsErr            error
	fichaForStudentResult       []reportsdb.FichaForStudentRow
	fichaForStudentErr          error
}

var _ reportsdb.Querier = (*fakeQuerier)(nil)

func (f *fakeQuerier) ActaSectionExists(_ context.Context, _ pgtype.UUID) (bool, error) {
	f.actaSectionExistsCalled = true
	return f.actaSectionExistsResult, f.actaSectionExistsErr
}

func (f *fakeQuerier) IsTeacherForSection(_ context.Context, _ reportsdb.IsTeacherForSectionParams) (bool, error) {
	f.isTeacherForSectionCalled = true
	return f.isTeacherForSectionResult, f.isTeacherForSectionErr
}

func (f *fakeQuerier) ActaForSectionAdmin(_ context.Context, _ pgtype.UUID) ([]reportsdb.ActaForSectionAdminRow, error) {
	f.actaForSectionAdminCalled = true
	return f.actaForSectionAdminResult, f.actaForSectionAdminErr
}

func (f *fakeQuerier) ActaForSectionByTeacher(_ context.Context, _ reportsdb.ActaForSectionByTeacherParams) ([]reportsdb.ActaForSectionByTeacherRow, error) {
	f.actaForSectionTeacherCalled = true
	return f.actaForSectionTeacherResult, f.actaForSectionTeacherErr
}

func (f *fakeQuerier) OccupancyPeriodExists(_ context.Context, _ pgtype.UUID) (bool, error) {
	f.occupancyPeriodExistsCalled = true
	return f.occupancyPeriodExistsResult, f.occupancyPeriodExistsErr
}

func (f *fakeQuerier) OccupancyForPeriod(_ context.Context, _ pgtype.UUID) ([]reportsdb.OccupancyForPeriodRow, error) {
	f.occupancyForPeriodCalled = true
	return f.occupancyForPeriodResult, f.occupancyForPeriodErr
}

func (f *fakeQuerier) ProgramExists(_ context.Context, _ pgtype.UUID) (bool, error) {
	f.programExistsCalled = true
	return f.programExistsResult, f.programExistsErr
}

func (f *fakeQuerier) ProgramSummary(_ context.Context, _ reportsdb.ProgramSummaryParams) ([]reportsdb.ProgramSummaryRow, error) {
	f.programSummaryCalled = true
	return f.programSummaryResult, f.programSummaryErr
}

func (f *fakeQuerier) StudentExists(_ context.Context, _ pgtype.UUID) (bool, error) {
	f.studentExistsCalled = true
	return f.studentExistsResult, f.studentExistsErr
}

func (f *fakeQuerier) FichaForStudent(_ context.Context, _ pgtype.UUID) ([]reportsdb.FichaForStudentRow, error) {
	f.fichaForStudentCalled = true
	return f.fichaForStudentResult, f.fichaForStudentErr
}

// --- Repository delegation tests ---

func TestPostgresRepository_SectionExists_DelegatesAndReturnsBool(t *testing.T) {
	fq := &fakeQuerier{actaSectionExistsResult: true}
	repo := NewPostgresRepository(fq)
	id := uuid.MustParse("01932a81-f801-7a4c-90b4-111111111111")

	exists, err := repo.SectionExists(context.Background(), id)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.actaSectionExistsCalled {
		t.Fatal("expected ActaSectionExists to be called, was not")
	}
	if !exists {
		t.Fatal("expected exists=true, got false")
	}
}

func TestPostgresRepository_SectionExists_TranslatesErrNoRows(t *testing.T) {
	fq := &fakeQuerier{actaSectionExistsErr: pgx.ErrNoRows}
	repo := NewPostgresRepository(fq)
	id := uuid.New()

	_, err := repo.SectionExists(context.Background(), id)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostgresRepository_IsTeacherForSection_Delegates(t *testing.T) {
	fq := &fakeQuerier{isTeacherForSectionResult: true}
	repo := NewPostgresRepository(fq)

	isMember, err := repo.IsTeacherForSection(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.isTeacherForSectionCalled {
		t.Fatal("expected IsTeacherForSection to be called, was not")
	}
	if !isMember {
		t.Fatal("expected isMember=true")
	}
}

func TestPostgresRepository_ActaForSectionAdmin_Delegates(t *testing.T) {
	rows := []reportsdb.ActaForSectionAdminRow{{GivenNames: "Ana"}}
	fq := &fakeQuerier{actaForSectionAdminResult: rows}
	repo := NewPostgresRepository(fq)
	id := uuid.New()

	got, err := repo.ActaForSectionAdmin(context.Background(), id)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.actaForSectionAdminCalled {
		t.Fatal("expected ActaForSectionAdmin to be called, was not")
	}
	if len(got) != 1 || got[0].GivenNames != "Ana" {
		t.Fatalf("unexpected rows: %v", got)
	}
}

func TestPostgresRepository_ActaForSectionByTeacher_Delegates(t *testing.T) {
	rows := []reportsdb.ActaForSectionByTeacherRow{{GivenNames: "Pedro"}}
	fq := &fakeQuerier{actaForSectionTeacherResult: rows}
	repo := NewPostgresRepository(fq)

	got, err := repo.ActaForSectionByTeacher(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.actaForSectionTeacherCalled {
		t.Fatal("expected ActaForSectionByTeacher to be called, was not")
	}
	if len(got) != 1 || got[0].GivenNames != "Pedro" {
		t.Fatalf("unexpected rows: %v", got)
	}
}

func TestPostgresRepository_PeriodExists_Delegates(t *testing.T) {
	fq := &fakeQuerier{occupancyPeriodExistsResult: true}
	repo := NewPostgresRepository(fq)

	exists, err := repo.PeriodExists(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.occupancyPeriodExistsCalled {
		t.Fatal("expected OccupancyPeriodExists to be called, was not")
	}
	if !exists {
		t.Fatal("expected exists=true")
	}
}

func TestPostgresRepository_OccupancyForPeriod_Delegates(t *testing.T) {
	rows := []reportsdb.OccupancyForPeriodRow{{Capacity: 30}}
	fq := &fakeQuerier{occupancyForPeriodResult: rows}
	repo := NewPostgresRepository(fq)

	got, err := repo.OccupancyForPeriod(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.occupancyForPeriodCalled {
		t.Fatal("expected OccupancyForPeriod to be called, was not")
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
}

func TestPostgresRepository_ProgramExists_Delegates(t *testing.T) {
	fq := &fakeQuerier{programExistsResult: true}
	repo := NewPostgresRepository(fq)

	exists, err := repo.ProgramExists(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.programExistsCalled {
		t.Fatal("expected ProgramExists to be called, was not")
	}
	if !exists {
		t.Fatal("expected exists=true")
	}
}

func TestPostgresRepository_ProgramSummary_Delegates(t *testing.T) {
	rows := []reportsdb.ProgramSummaryRow{{QuotaCapacity: 100, EnrolledCount: 50}}
	fq := &fakeQuerier{programSummaryResult: rows}
	repo := NewPostgresRepository(fq)

	got, err := repo.ProgramSummary(context.Background(), uuid.New(), 2025)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.programSummaryCalled {
		t.Fatal("expected ProgramSummary to be called, was not")
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
}

func TestPostgresRepository_StudentExists_Delegates(t *testing.T) {
	fq := &fakeQuerier{studentExistsResult: true}
	repo := NewPostgresRepository(fq)

	exists, err := repo.StudentExists(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.studentExistsCalled {
		t.Fatal("expected StudentExists to be called, was not")
	}
	if !exists {
		t.Fatal("expected exists=true")
	}
}

func TestPostgresRepository_FichaForStudent_Delegates(t *testing.T) {
	rows := []reportsdb.FichaForStudentRow{{CourseName: "Matemáticas"}}
	fq := &fakeQuerier{fichaForStudentResult: rows}
	repo := NewPostgresRepository(fq)

	got, err := repo.FichaForStudent(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fq.fichaForStudentCalled {
		t.Fatal("expected FichaForStudent to be called, was not")
	}
	if len(got) != 1 || got[0].CourseName != "Matemáticas" {
		t.Fatalf("unexpected rows: %v", got)
	}
}
