package catalog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog/catalogdb"
)

// fakeQuerier is a stub implementing catalogdb.Querier for unit tests.
type fakeQuerier struct {
	insertProgramErr           error
	insertProgramRow           catalogdb.Program
	updateProgramErr           error
	updateProgramRow           catalogdb.Program
	getProgramErr              error
	getProgramRow              catalogdb.Program
	listProgramsRows           []catalogdb.Program
	listProgramsErr            error
	softDeleteProgRows         int64
	softDeleteProgErr          error
	countProgCoursesN          int64
	countProgCoursesE          error
	countLiveQuotasN           int64
	countLiveQuotasE           error

	insertCourseErr            error
	insertCourseRow            catalogdb.Course
	updateCourseErr            error
	updateCourseRow            catalogdb.Course
	getCourseErr               error
	getCourseRow               catalogdb.Course
	listCoursesRows            []catalogdb.Course
	listCoursesErr             error
	softDeleteCourseRows       int64
	softDeleteCourseE          error
	countCourseAssociationsN   int64
	countCourseAssociationsE   error

	insertProgramCourseErr      error
	insertProgramCourseRow      catalogdb.ProgramCourse
	deleteProgramCourseRows     int64
	deleteProgramCourseErr      error
	listProgramCoursesRows      []catalogdb.ProgramCourse
	listProgramCoursesErr       error

	insertAcademicPeriodErr       error
	insertAcademicPeriodRow       catalogdb.AcademicPeriod
	updateAcademicPeriodErr       error
	updateAcademicPeriodRow       catalogdb.AcademicPeriod
	getAcademicPeriodErr          error
	getAcademicPeriodRow          catalogdb.AcademicPeriod
	listAcademicPeriodsRows       []catalogdb.AcademicPeriod
	listAcademicPeriodsErr        error
	softDeleteAcademicPeriodRows  int64
	softDeleteAcademicPeriodE     error

	upsertProgramQuotaErr        error
	upsertProgramQuotaRow        catalogdb.ProgramQuota
	updateProgramQuotaErr        error
	updateProgramQuotaRow        catalogdb.ProgramQuota
	getProgramQuotaErr           error
	getProgramQuotaRow           catalogdb.ProgramQuota
	listProgramQuotasRows        []catalogdb.ProgramQuota
	listProgramQuotasErr         error
	softDeleteProgramQuotaRows   int64
	softDeleteProgramQuotaE      error

	insertSectionErr                      error
	insertSectionRow                      catalogdb.Section
	updateSectionErr                      error
	updateSectionRow                      catalogdb.Section
	getSectionErr                         error
	getSectionRow                         catalogdb.Section
	listSectionsRows                      []catalogdb.Section
	listSectionsErr                       error
	softDeleteSectionRows                 int64
	softDeleteSectionE                    error
	countLiveSectionsByCourseN            int64
	countLiveSectionsByCourseE            error
	countLiveSectionsByAcademicPeriodN    int64
	countLiveSectionsByAcademicPeriodE    error
	countSectionTeachersN                 int64
	countSectionTeachersE                 error

	insertSectionTeacherErr              error
	insertSectionTeacherRow              catalogdb.SectionTeacher
	deleteSectionTeacherRows             int64
	deleteSectionTeacherErr              error
	listSectionTeachersRows              []catalogdb.SectionTeacher
	listSectionTeachersErr               error
}

// Compile-time check: fakeQuerier must implement catalogdb.Querier.
var _ catalogdb.Querier = (*fakeQuerier)(nil)

func (f *fakeQuerier) InsertProgram(_ context.Context, _ catalogdb.InsertProgramParams) (catalogdb.Program, error) {
	return f.insertProgramRow, f.insertProgramErr
}
func (f *fakeQuerier) UpdateProgram(_ context.Context, _ catalogdb.UpdateProgramParams) (catalogdb.Program, error) {
	return f.updateProgramRow, f.updateProgramErr
}
func (f *fakeQuerier) GetProgram(_ context.Context, _ pgtype.UUID) (catalogdb.Program, error) {
	return f.getProgramRow, f.getProgramErr
}
func (f *fakeQuerier) ListPrograms(_ context.Context) ([]catalogdb.Program, error) {
	return f.listProgramsRows, f.listProgramsErr
}
func (f *fakeQuerier) SoftDeleteProgram(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.softDeleteProgRows, f.softDeleteProgErr
}
func (f *fakeQuerier) CountProgramCourses(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.countProgCoursesN, f.countProgCoursesE
}
func (f *fakeQuerier) CountLiveProgramQuotas(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.countLiveQuotasN, f.countLiveQuotasE
}
func (f *fakeQuerier) InsertCourse(_ context.Context, _ catalogdb.InsertCourseParams) (catalogdb.Course, error) {
	return f.insertCourseRow, f.insertCourseErr
}
func (f *fakeQuerier) UpdateCourse(_ context.Context, _ catalogdb.UpdateCourseParams) (catalogdb.Course, error) {
	return f.updateCourseRow, f.updateCourseErr
}
func (f *fakeQuerier) GetCourse(_ context.Context, _ pgtype.UUID) (catalogdb.Course, error) {
	return f.getCourseRow, f.getCourseErr
}
func (f *fakeQuerier) ListCourses(_ context.Context) ([]catalogdb.Course, error) {
	return f.listCoursesRows, f.listCoursesErr
}
func (f *fakeQuerier) SoftDeleteCourse(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.softDeleteCourseRows, f.softDeleteCourseE
}
func (f *fakeQuerier) CountCourseProgramAssociations(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.countCourseAssociationsN, f.countCourseAssociationsE
}
func (f *fakeQuerier) InsertProgramCourse(_ context.Context, _ catalogdb.InsertProgramCourseParams) (catalogdb.ProgramCourse, error) {
	return f.insertProgramCourseRow, f.insertProgramCourseErr
}
func (f *fakeQuerier) DeleteProgramCourse(_ context.Context, _ catalogdb.DeleteProgramCourseParams) (int64, error) {
	return f.deleteProgramCourseRows, f.deleteProgramCourseErr
}
func (f *fakeQuerier) ListProgramCourses(_ context.Context, _ pgtype.UUID) ([]catalogdb.ProgramCourse, error) {
	return f.listProgramCoursesRows, f.listProgramCoursesErr
}
func (f *fakeQuerier) InsertAcademicPeriod(_ context.Context, _ catalogdb.InsertAcademicPeriodParams) (catalogdb.AcademicPeriod, error) {
	return f.insertAcademicPeriodRow, f.insertAcademicPeriodErr
}
func (f *fakeQuerier) UpdateAcademicPeriod(_ context.Context, _ catalogdb.UpdateAcademicPeriodParams) (catalogdb.AcademicPeriod, error) {
	return f.updateAcademicPeriodRow, f.updateAcademicPeriodErr
}
func (f *fakeQuerier) GetAcademicPeriod(_ context.Context, _ pgtype.UUID) (catalogdb.AcademicPeriod, error) {
	return f.getAcademicPeriodRow, f.getAcademicPeriodErr
}
func (f *fakeQuerier) ListAcademicPeriods(_ context.Context) ([]catalogdb.AcademicPeriod, error) {
	return f.listAcademicPeriodsRows, f.listAcademicPeriodsErr
}
func (f *fakeQuerier) SoftDeleteAcademicPeriod(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.softDeleteAcademicPeriodRows, f.softDeleteAcademicPeriodE
}
func (f *fakeQuerier) UpsertProgramQuota(_ context.Context, _ catalogdb.UpsertProgramQuotaParams) (catalogdb.ProgramQuota, error) {
	return f.upsertProgramQuotaRow, f.upsertProgramQuotaErr
}
func (f *fakeQuerier) UpdateProgramQuota(_ context.Context, _ catalogdb.UpdateProgramQuotaParams) (catalogdb.ProgramQuota, error) {
	return f.updateProgramQuotaRow, f.updateProgramQuotaErr
}
func (f *fakeQuerier) GetProgramQuota(_ context.Context, _ pgtype.UUID) (catalogdb.ProgramQuota, error) {
	return f.getProgramQuotaRow, f.getProgramQuotaErr
}
func (f *fakeQuerier) ListProgramQuotas(_ context.Context, _ pgtype.UUID) ([]catalogdb.ProgramQuota, error) {
	return f.listProgramQuotasRows, f.listProgramQuotasErr
}
func (f *fakeQuerier) SoftDeleteProgramQuota(_ context.Context, _ catalogdb.SoftDeleteProgramQuotaParams) (int64, error) {
	return f.softDeleteProgramQuotaRows, f.softDeleteProgramQuotaE
}
func (f *fakeQuerier) InsertSection(_ context.Context, _ catalogdb.InsertSectionParams) (catalogdb.Section, error) {
	return f.insertSectionRow, f.insertSectionErr
}
func (f *fakeQuerier) UpdateSection(_ context.Context, _ catalogdb.UpdateSectionParams) (catalogdb.Section, error) {
	return f.updateSectionRow, f.updateSectionErr
}
func (f *fakeQuerier) GetSection(_ context.Context, _ pgtype.UUID) (catalogdb.Section, error) {
	return f.getSectionRow, f.getSectionErr
}
func (f *fakeQuerier) ListSections(_ context.Context) ([]catalogdb.Section, error) {
	return f.listSectionsRows, f.listSectionsErr
}
func (f *fakeQuerier) SoftDeleteSection(_ context.Context, _ catalogdb.SoftDeleteSectionParams) (int64, error) {
	return f.softDeleteSectionRows, f.softDeleteSectionE
}
func (f *fakeQuerier) CountLiveSectionsByCourse(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.countLiveSectionsByCourseN, f.countLiveSectionsByCourseE
}
func (f *fakeQuerier) CountLiveSectionsByAcademicPeriod(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.countLiveSectionsByAcademicPeriodN, f.countLiveSectionsByAcademicPeriodE
}
func (f *fakeQuerier) CountSectionTeachers(_ context.Context, _ pgtype.UUID) (int64, error) {
	return f.countSectionTeachersN, f.countSectionTeachersE
}
func (f *fakeQuerier) InsertSectionTeacher(_ context.Context, _ catalogdb.InsertSectionTeacherParams) (catalogdb.SectionTeacher, error) {
	return f.insertSectionTeacherRow, f.insertSectionTeacherErr
}
func (f *fakeQuerier) DeleteSectionTeacher(_ context.Context, _ catalogdb.DeleteSectionTeacherParams) (int64, error) {
	return f.deleteSectionTeacherRows, f.deleteSectionTeacherErr
}
func (f *fakeQuerier) ListSectionTeachers(_ context.Context, _ pgtype.UUID) ([]catalogdb.SectionTeacher, error) {
	return f.listSectionTeachersRows, f.listSectionTeachersErr
}

// --- Repository unit tests ---

func TestRepository_GetProgram_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{getProgramErr: pgx.ErrNoRows}
	repo := catalog.NewPostgresRepository(q)

	_, err := repo.GetProgram(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("GetProgram (no rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_InsertProgramCourse_Duplicate(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{insertProgramCourseErr: &pgconn.PgError{Code: "23505"}}
	repo := catalog.NewPostgresRepository(q)

	_, err := repo.AddCourseToProgram(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, catalog.ErrAlreadyExists) {
		t.Errorf("AddCourseToProgram (duplicate): got %v, want ErrAlreadyExists", err)
	}
}

func TestRepository_InsertCourse_FKViolation(t *testing.T) {
	t.Parallel()

	// This scenario exercises the FK path via a fakeQuerier returning 23503.
	// For courses, no FK violation is expected in practice, but the translation layer must work.
	q := &fakeQuerier{insertCourseErr: &pgconn.PgError{Code: "23503"}}
	repo := catalog.NewPostgresRepository(q)

	p := catalog.CreateCourseParams{Code: "C1", Name: "Course", Credits: 3}
	_, err := repo.CreateCourse(context.Background(), p, nil)
	if !errors.Is(err, catalog.ErrInvalidInput) {
		t.Errorf("CreateCourse (FK violation): got %v, want ErrInvalidInput", err)
	}
}

func TestRepository_CountProgramCourses(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{countProgCoursesN: 2}
	repo := catalog.NewPostgresRepository(q)

	n, err := repo.CountProgramCourses(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("CountProgramCourses: unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("CountProgramCourses: got %d, want 2", n)
	}
}

func TestRepository_CountLiveProgramQuotas(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{countLiveQuotasN: 1}
	repo := catalog.NewPostgresRepository(q)

	n, err := repo.CountLiveProgramQuotas(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("CountLiveProgramQuotas: unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("CountLiveProgramQuotas: got %d, want 1", n)
	}
}

func TestRepository_CountCourseProgramAssociations(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{countCourseAssociationsN: 3}
	repo := catalog.NewPostgresRepository(q)

	n, err := repo.CountCourseProgramAssociations(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("CountCourseProgramAssociations: unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("CountCourseProgramAssociations: got %d, want 3", n)
	}
}

func TestRepository_SoftDeleteProgram_NotFound(t *testing.T) {
	t.Parallel()

	// 0 rows affected means the row does not exist or is already deleted.
	q := &fakeQuerier{softDeleteProgRows: 0}
	repo := catalog.NewPostgresRepository(q)

	err := repo.SoftDeleteProgram(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("SoftDeleteProgram (0 rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_SoftDeleteCourse_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{softDeleteCourseRows: 0}
	repo := catalog.NewPostgresRepository(q)

	err := repo.SoftDeleteCourse(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("SoftDeleteCourse (0 rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_SoftDeleteAcademicPeriod_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{softDeleteAcademicPeriodRows: 0}
	repo := catalog.NewPostgresRepository(q)

	err := repo.SoftDeleteAcademicPeriod(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("SoftDeleteAcademicPeriod (0 rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_SoftDeleteProgramQuota_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{softDeleteProgramQuotaRows: 0}
	repo := catalog.NewPostgresRepository(q)

	err := repo.SoftDeleteProgramQuota(context.Background(), uuid.New(), nil)
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("SoftDeleteProgramQuota (0 rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_RemoveCourseFromProgram_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{deleteProgramCourseRows: 0}
	repo := catalog.NewPostgresRepository(q)

	err := repo.RemoveCourseFromProgram(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("RemoveCourseFromProgram (0 rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_GetSection_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{getSectionErr: pgx.ErrNoRows}
	repo := catalog.NewPostgresRepository(q)

	_, err := repo.GetSection(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("GetSection (no rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_SoftDeleteSection_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{softDeleteSectionRows: 0}
	repo := catalog.NewPostgresRepository(q)

	err := repo.SoftDeleteSection(context.Background(), uuid.New(), nil)
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("SoftDeleteSection (0 rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_InsertSectionTeacher_Duplicate(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{insertSectionTeacherErr: &pgconn.PgError{Code: "23505"}}
	repo := catalog.NewPostgresRepository(q)

	_, err := repo.AssignTeacherToSection(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, catalog.ErrAlreadyExists) {
		t.Errorf("AssignTeacherToSection (duplicate): got %v, want ErrAlreadyExists", err)
	}
}

func TestRepository_InsertSectionTeacher_BadFK(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{insertSectionTeacherErr: &pgconn.PgError{Code: "23503"}}
	repo := catalog.NewPostgresRepository(q)

	_, err := repo.AssignTeacherToSection(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, catalog.ErrInvalidInput) {
		t.Errorf("AssignTeacherToSection (FK violation): got %v, want ErrInvalidInput", err)
	}
}

func TestRepository_RemoveTeacherFromSection_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{deleteSectionTeacherRows: 0}
	repo := catalog.NewPostgresRepository(q)

	err := repo.RemoveTeacherFromSection(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("RemoveTeacherFromSection (0 rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_CountLiveSectionsByCourse(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{countLiveSectionsByCourseN: 2}
	repo := catalog.NewPostgresRepository(q)

	n, err := repo.CountLiveSectionsByCourse(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("CountLiveSectionsByCourse: unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("CountLiveSectionsByCourse: got %d, want 2", n)
	}
}

func TestRepository_CountLiveSectionsByAcademicPeriod(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{countLiveSectionsByAcademicPeriodN: 1}
	repo := catalog.NewPostgresRepository(q)

	n, err := repo.CountLiveSectionsByAcademicPeriod(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("CountLiveSectionsByAcademicPeriod: unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("CountLiveSectionsByAcademicPeriod: got %d, want 1", n)
	}
}

func TestRepository_InsertSection_FKViolation(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{insertSectionErr: &pgconn.PgError{Code: "23503"}}
	repo := catalog.NewPostgresRepository(q)

	_, err := repo.CreateSection(context.Background(), catalog.CreateSectionParams{
		CourseID:         uuid.New(),
		AcademicPeriodID: uuid.New(),
		SeatCapacity:     10,
	}, nil)
	if !errors.Is(err, catalog.ErrInvalidInput) {
		t.Errorf("CreateSection (FK violation): got %v, want ErrInvalidInput", err)
	}
}
