package catalog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog/catalogdb"
)

// fakeRepository is a test double for catalog.Repository.
type fakeRepository struct {
	// Programs
	createProgramRow    catalogdb.Program
	createProgramErr    error
	createProgramActor  *uuid.UUID
	updateProgramRow    catalogdb.Program
	updateProgramErr    error
	getProgramRow       catalogdb.Program
	getProgramErr       error
	listProgramsRows    []catalogdb.Program
	softDeleteProgramE  error
	countProgramCourses int64
	countLiveQuotas     int64

	// Courses
	createCourseRow            catalogdb.Course
	createCourseErr            error
	createCourseActor          *uuid.UUID
	updateCourseRow            catalogdb.Course
	updateCourseErr            error
	getCourseRow               catalogdb.Course
	getCourseErr               error
	listCoursesRows            []catalogdb.Course
	softDeleteCourseE          error
	countCourseAssociations    int64
	countCourseAssociationsErr error

	// Program courses
	addCourseToProgramRow  catalogdb.ProgramCourse
	addCourseToProgramErr  error
	removeCourseErr        error
	listProgramCoursesRows []catalogdb.ProgramCourse

	// Academic periods
	createAcademicPeriodRow   catalogdb.AcademicPeriod
	createAcademicPeriodErr   error
	updateAcademicPeriodRow   catalogdb.AcademicPeriod
	getAcademicPeriodRow      catalogdb.AcademicPeriod
	getAcademicPeriodErr      error
	listAcademicPeriodsRows   []catalogdb.AcademicPeriod
	softDeleteAcademicPeriodE error

	// Program quotas
	createProgramQuotaRow   catalogdb.ProgramQuota
	createProgramQuotaErr   error
	createProgramQuotaActor *uuid.UUID
	updateProgramQuotaRow   catalogdb.ProgramQuota
	updateProgramQuotaErr   error
	getProgramQuotaRow      catalogdb.ProgramQuota
	getProgramQuotaErr      error
	listProgramQuotasRows   []catalogdb.ProgramQuota
	softDeleteProgramQuotaE error

	// Sections
	createSectionRow          catalogdb.Section
	createSectionErr          error
	createSectionActor        *uuid.UUID
	updateSectionRow          catalogdb.Section
	updateSectionErr          error
	getSectionRow             catalogdb.Section
	getSectionErr             error
	listSectionsRows          []catalogdb.Section
	softDeleteSectionE        error
	countLiveSectionsByCourse int64
	countLiveSectionsByPeriod int64
	countSectionTeachersN     int64

	// Section teachers
	assignTeacherRow        catalogdb.SectionTeacher
	assignTeacherErr        error
	removeTeacherErr        error
	listSectionTeachersRows []catalogdb.SectionTeacher
}

// Compile-time check: fakeRepository must satisfy catalog.Repository.
var _ catalog.Repository = (*fakeRepository)(nil)

func (f *fakeRepository) CreateProgram(_ context.Context, _ catalog.CreateProgramParams, actor *uuid.UUID) (catalogdb.Program, error) {
	f.createProgramActor = actor
	return f.createProgramRow, f.createProgramErr
}
func (f *fakeRepository) UpdateProgram(_ context.Context, _ uuid.UUID, _ catalog.UpdateProgramParams, _ *uuid.UUID) (catalogdb.Program, error) {
	return f.updateProgramRow, f.updateProgramErr
}
func (f *fakeRepository) GetProgram(_ context.Context, _ uuid.UUID) (catalogdb.Program, error) {
	return f.getProgramRow, f.getProgramErr
}
func (f *fakeRepository) ListPrograms(_ context.Context) ([]catalogdb.Program, error) {
	return f.listProgramsRows, nil
}
func (f *fakeRepository) DeleteProgramTx(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	if f.countProgramCourses > 0 {
		return catalog.ErrHasDependents
	}
	if f.countLiveQuotas > 0 {
		return catalog.ErrHasDependents
	}
	return f.softDeleteProgramE
}
func (f *fakeRepository) CountProgramCourses(_ context.Context, _ uuid.UUID) (int64, error) {
	return f.countProgramCourses, nil
}
func (f *fakeRepository) CountLiveProgramQuotas(_ context.Context, _ uuid.UUID) (int64, error) {
	return f.countLiveQuotas, nil
}
func (f *fakeRepository) CreateCourse(_ context.Context, _ catalog.CreateCourseParams, actor *uuid.UUID) (catalogdb.Course, error) {
	f.createCourseActor = actor
	return f.createCourseRow, f.createCourseErr
}
func (f *fakeRepository) UpdateCourse(_ context.Context, _ uuid.UUID, _ catalog.UpdateCourseParams, _ *uuid.UUID) (catalogdb.Course, error) {
	return f.updateCourseRow, f.updateCourseErr
}
func (f *fakeRepository) GetCourse(_ context.Context, _ uuid.UUID) (catalogdb.Course, error) {
	return f.getCourseRow, f.getCourseErr
}
func (f *fakeRepository) ListCourses(_ context.Context) ([]catalogdb.Course, error) {
	return f.listCoursesRows, nil
}
func (f *fakeRepository) DeleteCourseTx(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	if f.countCourseAssociationsErr != nil {
		return f.countCourseAssociationsErr
	}
	if f.countCourseAssociations > 0 || f.countLiveSectionsByCourse > 0 {
		return catalog.ErrHasDependents
	}
	return f.softDeleteCourseE
}
func (f *fakeRepository) CountCourseProgramAssociations(_ context.Context, _ uuid.UUID) (int64, error) {
	return f.countCourseAssociations, f.countCourseAssociationsErr
}
func (f *fakeRepository) AddCourseToProgram(_ context.Context, _, _ uuid.UUID) (catalogdb.ProgramCourse, error) {
	return f.addCourseToProgramRow, f.addCourseToProgramErr
}
func (f *fakeRepository) RemoveCourseFromProgram(_ context.Context, _, _ uuid.UUID) error {
	return f.removeCourseErr
}
func (f *fakeRepository) ListProgramCourses(_ context.Context, _ uuid.UUID) ([]catalogdb.ProgramCourse, error) {
	return f.listProgramCoursesRows, nil
}
func (f *fakeRepository) CreateAcademicPeriod(_ context.Context, _ catalog.CreateAcademicPeriodParams) (catalogdb.AcademicPeriod, error) {
	return f.createAcademicPeriodRow, f.createAcademicPeriodErr
}
func (f *fakeRepository) UpdateAcademicPeriod(_ context.Context, _ uuid.UUID, _ catalog.UpdateAcademicPeriodParams) (catalogdb.AcademicPeriod, error) {
	return f.updateAcademicPeriodRow, nil
}
func (f *fakeRepository) GetAcademicPeriod(_ context.Context, _ uuid.UUID) (catalogdb.AcademicPeriod, error) {
	return f.getAcademicPeriodRow, f.getAcademicPeriodErr
}
func (f *fakeRepository) ListAcademicPeriods(_ context.Context) ([]catalogdb.AcademicPeriod, error) {
	return f.listAcademicPeriodsRows, nil
}
func (f *fakeRepository) DeleteAcademicPeriodTx(_ context.Context, _ uuid.UUID) error {
	if f.countLiveSectionsByPeriod > 0 {
		return catalog.ErrHasDependents
	}
	return f.softDeleteAcademicPeriodE
}
func (f *fakeRepository) CreateProgramQuota(_ context.Context, _ catalog.CreateProgramQuotaParams, actor *uuid.UUID) (catalogdb.ProgramQuota, error) {
	f.createProgramQuotaActor = actor
	return f.createProgramQuotaRow, f.createProgramQuotaErr
}
func (f *fakeRepository) UpdateProgramQuota(_ context.Context, _ uuid.UUID, _ catalog.UpdateProgramQuotaParams, _ *uuid.UUID) (catalogdb.ProgramQuota, error) {
	return f.updateProgramQuotaRow, f.updateProgramQuotaErr
}
func (f *fakeRepository) GetProgramQuota(_ context.Context, _ uuid.UUID) (catalogdb.ProgramQuota, error) {
	return f.getProgramQuotaRow, f.getProgramQuotaErr
}
func (f *fakeRepository) ListProgramQuotas(_ context.Context, _ uuid.UUID) ([]catalogdb.ProgramQuota, error) {
	return f.listProgramQuotasRows, nil
}
func (f *fakeRepository) SoftDeleteProgramQuota(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	return f.softDeleteProgramQuotaE
}
func (f *fakeRepository) CreateSection(_ context.Context, _ catalog.CreateSectionParams, actor *uuid.UUID) (catalogdb.Section, error) {
	f.createSectionActor = actor
	return f.createSectionRow, f.createSectionErr
}
func (f *fakeRepository) UpdateSection(_ context.Context, _ uuid.UUID, _ catalog.UpdateSectionParams, _ *uuid.UUID) (catalogdb.Section, error) {
	return f.updateSectionRow, f.updateSectionErr
}
func (f *fakeRepository) GetSection(_ context.Context, _ uuid.UUID) (catalogdb.Section, error) {
	return f.getSectionRow, f.getSectionErr
}
func (f *fakeRepository) ListSections(_ context.Context, _ *uuid.UUID, _ *uuid.UUID) ([]catalogdb.Section, error) {
	return f.listSectionsRows, nil
}
func (f *fakeRepository) DeleteSectionTx(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	if f.countSectionTeachersN > 0 {
		return catalog.ErrHasDependents
	}
	return f.softDeleteSectionE
}
func (f *fakeRepository) CountLiveSectionsByCourse(_ context.Context, _ uuid.UUID) (int64, error) {
	return f.countLiveSectionsByCourse, nil
}
func (f *fakeRepository) CountLiveSectionsByAcademicPeriod(_ context.Context, _ uuid.UUID) (int64, error) {
	return f.countLiveSectionsByPeriod, nil
}
func (f *fakeRepository) CountSectionTeachers(_ context.Context, _ uuid.UUID) (int64, error) {
	return f.countSectionTeachersN, nil
}
func (f *fakeRepository) AssignTeacherToSection(_ context.Context, _, _ uuid.UUID) (catalogdb.SectionTeacher, error) {
	return f.assignTeacherRow, f.assignTeacherErr
}
func (f *fakeRepository) RemoveTeacherFromSection(_ context.Context, _, _ uuid.UUID) error {
	return f.removeTeacherErr
}
func (f *fakeRepository) ListSectionTeachers(_ context.Context, _ uuid.UUID) ([]catalogdb.SectionTeacher, error) {
	return f.listSectionTeachersRows, nil
}

// --- Validation tests ---

func TestService_CreateProgram_Validation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params catalog.CreateProgramParams
	}{
		{"empty code", catalog.CreateProgramParams{Code: "", Name: "Engineering"}},
		{"empty name", catalog.CreateProgramParams{Code: "ENG", Name: ""}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := catalog.NewService(repo)
			_, err := svc.CreateProgram(context.Background(), tc.params)
			if !errors.Is(err, catalog.ErrInvalidInput) {
				t.Errorf("CreateProgram(%s): got %v, want ErrInvalidInput", tc.name, err)
			}
		})
	}
}

func TestService_CreateCourse_Validation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params catalog.CreateCourseParams
	}{
		{"empty code", catalog.CreateCourseParams{Code: "", Name: "Math", Credits: 3}},
		{"empty name", catalog.CreateCourseParams{Code: "MAT", Name: "", Credits: 3}},
		{"zero credits", catalog.CreateCourseParams{Code: "MAT", Name: "Math", Credits: 0}},
		{"negative credits", catalog.CreateCourseParams{Code: "MAT", Name: "Math", Credits: -1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := catalog.NewService(repo)
			_, err := svc.CreateCourse(context.Background(), tc.params)
			if !errors.Is(err, catalog.ErrInvalidInput) {
				t.Errorf("CreateCourse(%s): got %v, want ErrInvalidInput", tc.name, err)
			}
		})
	}
}

func TestService_CreateAcademicPeriod_Validation(t *testing.T) {
	t.Parallel()

	validStart := "2025-03-01"
	validEnd := "2025-07-31"

	cases := []struct {
		name   string
		params catalog.CreateAcademicPeriodServiceParams
	}{
		{"zero year", catalog.CreateAcademicPeriodServiceParams{Year: 0, Term: 1, StartDate: validStart, EndDate: validEnd}},
		{"negative year", catalog.CreateAcademicPeriodServiceParams{Year: -1, Term: 1, StartDate: validStart, EndDate: validEnd}},
		{"term zero", catalog.CreateAcademicPeriodServiceParams{Year: 2025, Term: 0, StartDate: validStart, EndDate: validEnd}},
		{"term three", catalog.CreateAcademicPeriodServiceParams{Year: 2025, Term: 3, StartDate: validStart, EndDate: validEnd}},
		{"term negative", catalog.CreateAcademicPeriodServiceParams{Year: 2025, Term: -1, StartDate: validStart, EndDate: validEnd}},
		{"equal dates", catalog.CreateAcademicPeriodServiceParams{Year: 2025, Term: 1, StartDate: "2025-03-01", EndDate: "2025-03-01"}},
		{"start after end", catalog.CreateAcademicPeriodServiceParams{Year: 2025, Term: 1, StartDate: "2025-08-01", EndDate: "2025-03-01"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := catalog.NewService(repo)
			_, err := svc.CreateAcademicPeriod(context.Background(), tc.params)
			if !errors.Is(err, catalog.ErrInvalidInput) {
				t.Errorf("CreateAcademicPeriod(%s): got %v, want ErrInvalidInput", tc.name, err)
			}
		})
	}
}

func TestService_CreateAcademicPeriod_ValidTerms(t *testing.T) {
	t.Parallel()

	for _, term := range []int32{1, 2} {
		repo := &fakeRepository{}
		svc := catalog.NewService(repo)
		p := catalog.CreateAcademicPeriodServiceParams{
			Year:      2025,
			Term:      term,
			StartDate: "2025-03-01",
			EndDate:   "2025-07-31",
		}
		_, err := svc.CreateAcademicPeriod(context.Background(), p)
		if err != nil {
			t.Errorf("CreateAcademicPeriod(term=%d): unexpected error: %v", term, err)
		}
	}
}

func TestService_CreateProgramQuota_Validation(t *testing.T) {
	t.Parallel()

	pid := uuid.New()
	cases := []struct {
		name   string
		params catalog.CreateProgramQuotaServiceParams
	}{
		{"zero capacity", catalog.CreateProgramQuotaServiceParams{ProgramID: pid.String(), Year: 2025, Capacity: 0}},
		{"negative capacity", catalog.CreateProgramQuotaServiceParams{ProgramID: pid.String(), Year: 2025, Capacity: -1}},
		{"zero year", catalog.CreateProgramQuotaServiceParams{ProgramID: pid.String(), Year: 0, Capacity: 40}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := catalog.NewService(repo)
			_, err := svc.CreateProgramQuota(context.Background(), tc.params)
			if !errors.Is(err, catalog.ErrInvalidInput) {
				t.Errorf("CreateProgramQuota(%s): got %v, want ErrInvalidInput", tc.name, err)
			}
		})
	}
}

// --- Audit tests ---

func TestService_CreateProgram_AuditFromContext(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)

	repo := &fakeRepository{
		createProgramRow: catalogdb.Program{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			CreatedBy: pgtype.UUID{Bytes: actorID, Valid: true},
		},
	}
	svc := catalog.NewService(repo)

	_, err := svc.CreateProgram(ctx, catalog.CreateProgramParams{Code: "ENG", Name: "Engineering"})
	if err != nil {
		t.Fatalf("CreateProgram: unexpected error: %v", err)
	}

	if repo.createProgramActor == nil || *repo.createProgramActor != actorID {
		t.Errorf("createProgram actor = %v, want %v", repo.createProgramActor, actorID)
	}
}

func TestService_CreateAcademicPeriod_NoAudit(t *testing.T) {
	t.Parallel()

	// Academic periods must NOT receive created_by/updated_by from context.
	// Since the repository method signature for CreateAcademicPeriod has no actor param,
	// compilation itself proves the invariant — the service cannot pass actor to the repo.
	// This test verifies the service call succeeds without panic for completeness.
	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)

	repo := &fakeRepository{}
	svc := catalog.NewService(repo)

	_, err := svc.CreateAcademicPeriod(ctx, catalog.CreateAcademicPeriodServiceParams{
		Year:      2025,
		Term:      1,
		StartDate: "2025-03-01",
		EndDate:   "2025-07-31",
	})
	if err != nil {
		t.Fatalf("CreateAcademicPeriod: unexpected error: %v", err)
	}
}

func TestService_CreateProgramQuota_AuditFromContext(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)

	repo := &fakeRepository{
		createProgramQuotaRow: catalogdb.ProgramQuota{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			CreatedBy: pgtype.UUID{Bytes: actorID, Valid: true},
		},
	}
	svc := catalog.NewService(repo)

	pid := uuid.New()
	_, err := svc.CreateProgramQuota(ctx, catalog.CreateProgramQuotaServiceParams{
		ProgramID: pid.String(),
		Year:      2025,
		Capacity:  40,
	})
	if err != nil {
		t.Fatalf("CreateProgramQuota: unexpected error: %v", err)
	}

	if repo.createProgramQuotaActor == nil || *repo.createProgramQuotaActor != actorID {
		t.Errorf("createProgramQuota actor = %v, want %v", repo.createProgramQuotaActor, actorID)
	}
}

// --- Dependent-blocking soft-delete tests ---

func TestService_DeleteProgram_BlockedByProgramCourses(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		countProgramCourses: 1,
		getProgramRow:       catalogdb.Program{ID: pgtype.UUID{Bytes: uuid.New(), Valid: true}},
	}
	svc := catalog.NewService(repo)

	err := svc.DeleteProgram(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrHasDependents) {
		t.Errorf("DeleteProgram (with program_courses): got %v, want ErrHasDependents", err)
	}
}

func TestService_DeleteProgram_BlockedByLiveQuotas(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		countProgramCourses: 0,
		countLiveQuotas:     1,
		getProgramRow:       catalogdb.Program{ID: pgtype.UUID{Bytes: uuid.New(), Valid: true}},
	}
	svc := catalog.NewService(repo)

	err := svc.DeleteProgram(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrHasDependents) {
		t.Errorf("DeleteProgram (with live quotas): got %v, want ErrHasDependents", err)
	}
}

func TestService_DeleteProgram_AllowedWhenNoLiveDependents(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		countProgramCourses: 0,
		countLiveQuotas:     0,
		getProgramRow:       catalogdb.Program{ID: pgtype.UUID{Bytes: uuid.New(), Valid: true}},
	}
	svc := catalog.NewService(repo)

	err := svc.DeleteProgram(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("DeleteProgram (no dependents): unexpected error: %v", err)
	}
}

func TestService_DeleteProgramQuota_NoDependentCheck(t *testing.T) {
	t.Parallel()

	// program_quotas have no downstream dependents — soft-delete always proceeds.
	repo := &fakeRepository{
		getProgramQuotaRow: catalogdb.ProgramQuota{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			DeletedAt: pgtype.Timestamptz{}, // live (null)
		},
	}
	svc := catalog.NewService(repo)

	err := svc.DeleteProgramQuota(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("DeleteProgramQuota: unexpected error: %v", err)
	}
}

func TestService_DeleteProgram_ActorPassedToRepo(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)

	var capturedActor *uuid.UUID
	repo := &fakeRepositoryWithDeleteActor{
		fakeRepository: fakeRepository{
			countProgramCourses: 0,
			countLiveQuotas:     0,
		},
		captureActor: &capturedActor,
	}
	svc := catalog.NewService(repo)

	if err := svc.DeleteProgram(ctx, uuid.New()); err != nil {
		t.Fatalf("DeleteProgram: unexpected error: %v", err)
	}
	if capturedActor == nil || *capturedActor != actorID {
		t.Errorf("DeleteProgramTx actor = %v, want %v", capturedActor, actorID)
	}
}

func TestService_DeleteCourse_ActorPassedToRepo(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)

	var capturedActor *uuid.UUID
	repo := &fakeRepositoryWithDeleteActor{
		fakeRepository: fakeRepository{
			countCourseAssociations:   0,
			countLiveSectionsByCourse: 0,
		},
		captureActor:    &capturedActor,
		captureIsCourse: true,
	}
	svc := catalog.NewService(repo)

	if err := svc.DeleteCourse(ctx, uuid.New()); err != nil {
		t.Fatalf("DeleteCourse: unexpected error: %v", err)
	}
	if capturedActor == nil || *capturedActor != actorID {
		t.Errorf("DeleteCourseTx actor = %v, want %v", capturedActor, actorID)
	}
}

// fakeRepositoryWithDeleteActor extends fakeRepository to capture the actor
// passed to DeleteProgramTx / DeleteCourseTx / DeleteSectionTx.
type fakeRepositoryWithDeleteActor struct {
	fakeRepository
	captureActor     **uuid.UUID
	captureIsCourse  bool
	captureIsSection bool
}

func (f *fakeRepositoryWithDeleteActor) DeleteProgramTx(_ context.Context, _ uuid.UUID, actor *uuid.UUID) error {
	if !f.captureIsCourse {
		*f.captureActor = actor
	}
	return f.softDeleteProgramE
}

func (f *fakeRepositoryWithDeleteActor) DeleteCourseTx(_ context.Context, _ uuid.UUID, actor *uuid.UUID) error {
	if f.captureIsCourse {
		*f.captureActor = actor
	}
	return f.softDeleteCourseE
}

func (f *fakeRepositoryWithDeleteActor) DeleteSectionTx(_ context.Context, _ uuid.UUID, actor *uuid.UUID) error {
	if f.captureIsSection {
		*f.captureActor = actor
	}
	return f.softDeleteSectionE
}

func TestService_DeleteSection_ActorPassedToRepo(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)

	var capturedActor *uuid.UUID
	repo := &fakeRepositoryWithDeleteActor{
		fakeRepository: fakeRepository{
			countSectionTeachersN: 0,
		},
		captureActor:     &capturedActor,
		captureIsSection: true,
	}
	svc := catalog.NewService(repo)

	if err := svc.DeleteSection(ctx, uuid.New()); err != nil {
		t.Fatalf("DeleteSection: unexpected error: %v", err)
	}
	if capturedActor == nil || *capturedActor != actorID {
		t.Errorf("DeleteSectionTx actor = %v, want %v", capturedActor, actorID)
	}
}

func TestService_DeleteCourse_BlockedByProgramAssociations(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		countCourseAssociations: 2,
	}
	svc := catalog.NewService(repo)

	err := svc.DeleteCourse(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrHasDependents) {
		t.Errorf("DeleteCourse (with program associations): got %v, want ErrHasDependents", err)
	}
}

func TestService_DeleteCourse_AllowedWhenNoAssociations(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		countCourseAssociations: 0,
		softDeleteCourseE:       nil,
	}
	svc := catalog.NewService(repo)

	err := svc.DeleteCourse(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("DeleteCourse (no associations): unexpected error: %v", err)
	}
}

// --- Soft-deleted entity is not found ---

func TestService_GetProgram_NotFound(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{getProgramErr: catalog.ErrNotFound}
	svc := catalog.NewService(repo)

	_, err := svc.GetProgram(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("GetProgram (not found): got %v, want ErrNotFound", err)
	}
}

// --- AddCourseToProgram propagates ErrAlreadyExists ---

func TestService_AddCourseToProgram_AlreadyExists(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{addCourseToProgramErr: catalog.ErrAlreadyExists}
	svc := catalog.NewService(repo)

	_, err := svc.AddCourseToProgram(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, catalog.ErrAlreadyExists) {
		t.Errorf("AddCourseToProgram (duplicate): got %v, want ErrAlreadyExists", err)
	}
}

// --- Section validation tests ---

func TestService_CreateSection_Validation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params catalog.CreateSectionServiceParams
	}{
		{"zero capacity", catalog.CreateSectionServiceParams{
			CourseID:         uuid.New().String(),
			AcademicPeriodID: uuid.New().String(),
			SeatCapacity:     0,
		}},
		{"negative capacity", catalog.CreateSectionServiceParams{
			CourseID:         uuid.New().String(),
			AcademicPeriodID: uuid.New().String(),
			SeatCapacity:     -5,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := catalog.NewService(repo)
			_, err := svc.CreateSection(context.Background(), tc.params)
			if !errors.Is(err, catalog.ErrInvalidInput) {
				t.Errorf("CreateSection(%s): got %v, want ErrInvalidInput", tc.name, err)
			}
		})
	}
}

func TestService_CreateSection_AuditFromContext(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)

	repo := &fakeRepository{
		createSectionRow: catalogdb.Section{
			ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
			CreatedBy: pgtype.UUID{Bytes: actorID, Valid: true},
		},
	}
	svc := catalog.NewService(repo)

	_, err := svc.CreateSection(ctx, catalog.CreateSectionServiceParams{
		CourseID:         uuid.New().String(),
		AcademicPeriodID: uuid.New().String(),
		SeatCapacity:     20,
	})
	if err != nil {
		t.Fatalf("CreateSection: unexpected error: %v", err)
	}
	if repo.createSectionActor == nil || *repo.createSectionActor != actorID {
		t.Errorf("createSection actor = %v, want %v", repo.createSectionActor, actorID)
	}
}

func TestService_DeleteSection_BlockedBySectionTeachers(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{countSectionTeachersN: 1}
	svc := catalog.NewService(repo)

	err := svc.DeleteSection(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrHasDependents) {
		t.Errorf("DeleteSection (with teachers): got %v, want ErrHasDependents", err)
	}
}

func TestService_DeleteSection_AllowedWhenNoTeachers(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{countSectionTeachersN: 0}
	svc := catalog.NewService(repo)

	err := svc.DeleteSection(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("DeleteSection (no teachers): unexpected error: %v", err)
	}
}

func TestService_DeleteCourse_BlockedByLiveSections(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		countCourseAssociations:   0,
		countLiveSectionsByCourse: 3,
	}
	svc := catalog.NewService(repo)

	err := svc.DeleteCourse(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrHasDependents) {
		t.Errorf("DeleteCourse (with live sections): got %v, want ErrHasDependents", err)
	}
}

func TestService_DeleteAcademicPeriod_BlockedByLiveSections(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{countLiveSectionsByPeriod: 2}
	svc := catalog.NewService(repo)

	err := svc.DeleteAcademicPeriod(context.Background(), uuid.New())
	if !errors.Is(err, catalog.ErrHasDependents) {
		t.Errorf("DeleteAcademicPeriod (with live sections): got %v, want ErrHasDependents", err)
	}
}

func TestService_DeleteAcademicPeriod_AllowedWhenNoSections(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{countLiveSectionsByPeriod: 0}
	svc := catalog.NewService(repo)

	err := svc.DeleteAcademicPeriod(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("DeleteAcademicPeriod (no sections): unexpected error: %v", err)
	}
}

func TestService_AssignTeacherToSection_AlreadyExists(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{assignTeacherErr: catalog.ErrAlreadyExists}
	svc := catalog.NewService(repo)

	_, err := svc.AssignTeacherToSection(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, catalog.ErrAlreadyExists) {
		t.Errorf("AssignTeacherToSection (duplicate): got %v, want ErrAlreadyExists", err)
	}
}

// --- AcademicPeriod date helper round-trip ---

func TestParseDate_ValidAndInvalid(t *testing.T) {
	t.Parallel()

	_, err := catalog.ParseDate("2025-03-01")
	if err != nil {
		t.Errorf("ParseDate(valid): unexpected error: %v", err)
	}

	_, err = catalog.ParseDate("not-a-date")
	if err == nil {
		t.Error("ParseDate(invalid): expected error, got nil")
	}
}

// --- FormatDate round-trip ---

func TestFormatDate(t *testing.T) {
	t.Parallel()

	d := pgtype.Date{
		Time:  time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
	got := catalog.FormatDate(d)
	if got != "2025-03-01" {
		t.Errorf("FormatDate: got %q, want %q", got, "2025-03-01")
	}

	null := pgtype.Date{}
	got = catalog.FormatDate(null)
	if got != "" {
		t.Errorf("FormatDate(null): got %q, want empty string", got)
	}
}
