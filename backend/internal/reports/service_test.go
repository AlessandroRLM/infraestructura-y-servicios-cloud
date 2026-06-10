package reports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	reportsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/reports/reportsdb"
)

// --- fakeRepository ---

type fakeRepository struct {
	sectionExistsCalled         bool
	actaAdminCalled             bool
	actaTeacherCalled           bool
	periodExistsCalled          bool
	occupancyForPeriodCalled    bool
	programExistsCalled         bool
	programSummaryCalled        bool
	studentExistsCalled         bool
	fichaForStudentCalled       bool

	sectionExistsResult      bool
	sectionExistsErr         error
	actaAdminResult          []reportsdb.ActaForSectionAdminRow
	actaAdminErr             error
	actaTeacherResult        []reportsdb.ActaForSectionByTeacherRow
	actaTeacherErr           error
	periodExistsResult       bool
	periodExistsErr          error
	occupancyResult          []reportsdb.OccupancyForPeriodRow
	occupancyErr             error
	programExistsResult      bool
	programExistsErr         error
	programSummaryResult     []reportsdb.ProgramSummaryRow
	programSummaryErr        error
	studentExistsResult      bool
	studentExistsErr         error
	fichaResult              []reportsdb.FichaForStudentRow
	fichaErr                 error
}

var _ Repository = (*fakeRepository)(nil)

func (f *fakeRepository) SectionExists(_ context.Context, _ uuid.UUID) (bool, error) {
	f.sectionExistsCalled = true
	return f.sectionExistsResult, f.sectionExistsErr
}

func (f *fakeRepository) ActaForSectionAdmin(_ context.Context, _ uuid.UUID) ([]reportsdb.ActaForSectionAdminRow, error) {
	f.actaAdminCalled = true
	return f.actaAdminResult, f.actaAdminErr
}

func (f *fakeRepository) ActaForSectionByTeacher(_ context.Context, _, _ uuid.UUID) ([]reportsdb.ActaForSectionByTeacherRow, error) {
	f.actaTeacherCalled = true
	return f.actaTeacherResult, f.actaTeacherErr
}

func (f *fakeRepository) PeriodExists(_ context.Context, _ uuid.UUID) (bool, error) {
	f.periodExistsCalled = true
	return f.periodExistsResult, f.periodExistsErr
}

func (f *fakeRepository) OccupancyForPeriod(_ context.Context, _ uuid.UUID) ([]reportsdb.OccupancyForPeriodRow, error) {
	f.occupancyForPeriodCalled = true
	return f.occupancyResult, f.occupancyErr
}

func (f *fakeRepository) ProgramExists(_ context.Context, _ uuid.UUID) (bool, error) {
	f.programExistsCalled = true
	return f.programExistsResult, f.programExistsErr
}

func (f *fakeRepository) ProgramSummary(_ context.Context, _ uuid.UUID, _ int32) ([]reportsdb.ProgramSummaryRow, error) {
	f.programSummaryCalled = true
	return f.programSummaryResult, f.programSummaryErr
}

func (f *fakeRepository) StudentExists(_ context.Context, _ uuid.UUID) (bool, error) {
	f.studentExistsCalled = true
	return f.studentExistsResult, f.studentExistsErr
}

func (f *fakeRepository) FichaForStudent(_ context.Context, _ uuid.UUID) ([]reportsdb.FichaForStudentRow, error) {
	f.fichaForStudentCalled = true
	return f.fichaResult, f.fichaErr
}

// --- helpers ---

// adminCtx returns a context carrying PermCatalogManage (admin marker).
func adminCtx() context.Context {
	perms := authz.NewPermissionSet([]authz.Permission{authz.PermCatalogManage, authz.PermReportsRead})
	return authz.WithPermissions(context.Background(), perms)
}

// teacherCtx returns a context carrying only PermReportsRead (teacher — no admin marker)
// with a fake teacher user ID.
var testTeacherID = uuid.MustParse("01932a81-f801-7a4c-90b4-aaaaaaaaaaaa")

func teacherCtx() context.Context {
	perms := authz.NewPermissionSet([]authz.Permission{authz.PermReportsRead})
	ctx := authz.WithPermissions(context.Background(), perms)
	return auth.WithUserID(ctx, testTeacherID)
}

// newSvc builds a Service with fake dependencies for unit tests.
func newSvc(repo Repository, cache Cache) *Service {
	return NewService(repo, cache, 5*time.Minute)
}

// pgNum converts a string decimal to pgtype.Numeric (panics on bad input — test helper only).
func pgNum(s string) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		panic("pgNum: " + err.Error())
	}
	return n
}

func pgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// --- GetSectionGradeReport tests ---

func TestGetSectionGradeReport_AdminPathHitCache(t *testing.T) {
	sectionID := uuid.New()
	// Build a response proto, marshal to cache bytes.
	resp := &reportsv1.GetSectionGradeReportResponse{
		SectionId:   sectionID.String(),
		GeneratedAt: "2026-06-10T00:00:00Z",
	}
	data, err := protoMarshal(resp)
	if err != nil {
		t.Fatalf("protoMarshal: %v", err)
	}
	cache := &fakeRedis{getVal: data}
	repo := &fakeRepository{}

	svc := newSvc(repo, cache)
	got, err := svc.GetSectionGradeReport(adminCtx(), sectionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SectionId != sectionID.String() {
		t.Errorf("expected SectionId=%s, got %s", sectionID.String(), got.SectionId)
	}
	if repo.actaAdminCalled {
		t.Error("expected repo NOT to be called on cache hit, but it was")
	}
}

func TestGetSectionGradeReport_AdminPathCacheMiss_CallsRepo(t *testing.T) {
	sectionID := uuid.New()
	studentID := uuid.New()
	evalID := uuid.New()

	repo := &fakeRepository{
		sectionExistsResult: true,
		actaAdminResult: []reportsdb.ActaForSectionAdminRow{
			{
				SeID:             pgUUID(sectionID),
				StudentID:        pgUUID(studentID),
				GivenNames:       "Ana",
				LastNamePaternal: "García",
				LastNameMaternal: pgtype.Text{String: "López", Valid: true},
				EvaluationID:     pgUUID(evalID),
				Position:         pgtype.Int4{Int32: 1, Valid: true},
				GradeValue:       pgNum("5.5"),
				FinalGrade:       pgNum("5.5"),
				EnrollmentStatus: "active",
			},
		},
	}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	got, err := svc.GetSectionGradeReport(adminCtx(), sectionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.actaAdminCalled {
		t.Fatal("expected repo.ActaForSectionAdmin to be called, was not")
	}
	if len(got.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got.Rows))
	}
	if !cache.setCalled {
		t.Error("expected cache.Set to be called after repo hit")
	}
}

func TestGetSectionGradeReport_AdminPathSectionNotFound(t *testing.T) {
	repo := &fakeRepository{sectionExistsErr: ErrNotFound}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	_, err := svc.GetSectionGradeReport(adminCtx(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSectionGradeReport_TeacherPath_CallsTeacherRepo(t *testing.T) {
	sectionID := uuid.New()
	repo := &fakeRepository{
		sectionExistsResult:   true,
		actaTeacherResult:     []reportsdb.ActaForSectionByTeacherRow{},
	}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	_, err := svc.GetSectionGradeReport(teacherCtx(), sectionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.actaTeacherCalled {
		t.Fatal("expected ActaForSectionByTeacher to be called for teacher path")
	}
	if repo.actaAdminCalled {
		t.Fatal("expected ActaForSectionAdmin NOT to be called for teacher path")
	}
}

func TestGetSectionGradeReport_Truncation(t *testing.T) {
	sectionID := uuid.New()
	// Build 501 rows (cap 500 → truncated).
	rows := make([]reportsdb.ActaForSectionAdminRow, 501)
	for i := range rows {
		rows[i] = reportsdb.ActaForSectionAdminRow{
			StudentID:        pgUUID(uuid.New()),
			GivenNames:       "Student",
			LastNamePaternal: "Test",
			EvaluationID:     pgUUID(uuid.New()),
		}
	}
	repo := &fakeRepository{sectionExistsResult: true, actaAdminResult: rows}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	got, err := svc.GetSectionGradeReport(adminCtx(), sectionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Truncated {
		t.Error("expected Truncated=true for 501 rows")
	}
	if len(got.Rows) > 500 {
		t.Errorf("expected at most 500 rows in response, got %d", len(got.Rows))
	}
}

// --- GetSectionOccupancyReport tests (admin-only) ---

func TestGetSectionOccupancyReport_TeacherGetsPermissionDenied(t *testing.T) {
	repo := &fakeRepository{}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	_, err := svc.GetSectionOccupancyReport(teacherCtx(), uuid.New())
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
	if cache.getCalled {
		t.Error("cache must NOT be consulted before authz check")
	}
}

func TestGetSectionOccupancyReport_AdminPath_CacheMiss_CallsRepo(t *testing.T) {
	periodID := uuid.New()
	repo := &fakeRepository{
		periodExistsResult: true,
		occupancyResult: []reportsdb.OccupancyForPeriodRow{
			{SectionID: pgUUID(uuid.New()), Capacity: 30, CourseName: pgtype.Text{String: "Math", Valid: true}, ActiveSeatCount: 20},
		},
	}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	got, err := svc.GetSectionOccupancyReport(adminCtx(), periodID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.occupancyForPeriodCalled {
		t.Fatal("expected OccupancyForPeriod to be called")
	}
	if len(got.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got.Rows))
	}
	if !cache.setCalled {
		t.Error("expected cache.Set after repo hit")
	}
}

func TestGetSectionOccupancyReport_AdminPath_PeriodNotFound(t *testing.T) {
	repo := &fakeRepository{periodExistsErr: ErrNotFound}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	_, err := svc.GetSectionOccupancyReport(adminCtx(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- GetProgramSummaryReport tests (admin-only) ---

func TestGetProgramSummaryReport_TeacherGetsPermissionDenied(t *testing.T) {
	repo := &fakeRepository{}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	_, err := svc.GetProgramSummaryReport(teacherCtx(), uuid.New(), 2025)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestGetProgramSummaryReport_AdminPath_CacheMiss_CallsRepo(t *testing.T) {
	programID := uuid.New()
	repo := &fakeRepository{
		programExistsResult: true,
		programSummaryResult: []reportsdb.ProgramSummaryRow{
			{QuotaID: pgUUID(uuid.New()), QuotaCapacity: 100, EnrolledCount: 50},
		},
	}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	got, err := svc.GetProgramSummaryReport(adminCtx(), programID, 2025)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.programSummaryCalled {
		t.Fatal("expected ProgramSummary to be called")
	}
	if len(got.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got.Rows))
	}
}

func TestGetProgramSummaryReport_ProgramNotFound(t *testing.T) {
	repo := &fakeRepository{programExistsErr: ErrNotFound}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	_, err := svc.GetProgramSummaryReport(adminCtx(), uuid.New(), 2025)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- GetStudentRecordReport tests (admin-only) ---

func TestGetStudentRecordReport_TeacherGetsPermissionDenied(t *testing.T) {
	repo := &fakeRepository{}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	_, err := svc.GetStudentRecordReport(teacherCtx(), uuid.New())
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestGetStudentRecordReport_AdminPath_CacheMiss_CallsRepo(t *testing.T) {
	studentID := uuid.New()
	repo := &fakeRepository{
		studentExistsResult: true,
		fichaResult: []reportsdb.FichaForStudentRow{
			{
				AcademicPeriodID:   pgUUID(uuid.New()),
				AcademicPeriodName: "2025-1",
				SectionID:          pgUUID(uuid.New()),
				CourseName:         "Cálculo",
				EnrollmentStatus:   "active",
				FinalGrade:         pgNum("5.5"),
				EvaluationID:       pgUUID(uuid.New()),
				Position:           pgtype.Int4{Int32: 1, Valid: true},
				GradeValue:         pgNum("5.5"),
			},
		},
	}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	got, err := svc.GetStudentRecordReport(adminCtx(), studentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.fichaForStudentCalled {
		t.Fatal("expected FichaForStudent to be called")
	}
	if len(got.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got.Rows))
	}
}

func TestGetStudentRecordReport_StudentNotFound(t *testing.T) {
	repo := &fakeRepository{studentExistsErr: ErrNotFound}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	_, err := svc.GetStudentRecordReport(adminCtx(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- Cache fail-open tests ---

func TestCacheGetError_TreatedAsMiss(t *testing.T) {
	sectionID := uuid.New()
	repo := &fakeRepository{sectionExistsResult: true, actaAdminResult: []reportsdb.ActaForSectionAdminRow{}}
	cache := &fakeRedis{getErr: errors.New("redis: timeout")}
	svc := newSvc(repo, cache)

	// Should not return error; Redis GET failure is treated as miss.
	_, err := svc.GetSectionGradeReport(adminCtx(), sectionID)
	if err != nil {
		t.Fatalf("expected no error on redis get failure (fail-open), got: %v", err)
	}
	if !repo.actaAdminCalled {
		t.Fatal("expected repo to be called after cache miss due to redis error")
	}
}

func TestCacheSetError_IsSwallowed(t *testing.T) {
	sectionID := uuid.New()
	repo := &fakeRepository{sectionExistsResult: true, actaAdminResult: []reportsdb.ActaForSectionAdminRow{}}
	cache := &fakeRedis{setErr: errors.New("redis: full")}
	svc := newSvc(repo, cache)

	// Cache SET error should be swallowed — response still returned.
	_, err := svc.GetSectionGradeReport(adminCtx(), sectionID)
	if err != nil {
		t.Fatalf("expected no error when redis set fails, got: %v", err)
	}
}

// --- Row grouping for acta ---

func TestGetSectionGradeReport_GroupsPartialGrades(t *testing.T) {
	sectionID := uuid.New()
	studentID := uuid.New()
	eval1 := uuid.New()
	eval2 := uuid.New()

	repo := &fakeRepository{
		sectionExistsResult: true,
		actaAdminResult: []reportsdb.ActaForSectionAdminRow{
			{StudentID: pgUUID(studentID), GivenNames: "Ana", LastNamePaternal: "G", EvaluationID: pgUUID(eval1), Position: pgtype.Int4{Int32: 1, Valid: true}, GradeValue: pgNum("5.0"), FinalGrade: pgNum("5.5"), EnrollmentStatus: "active"},
			{StudentID: pgUUID(studentID), GivenNames: "Ana", LastNamePaternal: "G", EvaluationID: pgUUID(eval2), Position: pgtype.Int4{Int32: 2, Valid: true}, GradeValue: pgNum("6.0"), FinalGrade: pgNum("5.5"), EnrollmentStatus: "active"},
		},
	}
	cache := &fakeRedis{}
	svc := newSvc(repo, cache)

	got, err := svc.GetSectionGradeReport(adminCtx(), sectionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Two rows with same studentID → grouped into 1 StudentGradeRow with 2 PartialGrades.
	if len(got.Rows) != 1 {
		t.Fatalf("expected 1 grouped row, got %d", len(got.Rows))
	}
	if len(got.Rows[0].PartialGrades) != 2 {
		t.Fatalf("expected 2 partial grades, got %d", len(got.Rows[0].PartialGrades))
	}
	_ = sectionID // used in repo setup only
}
