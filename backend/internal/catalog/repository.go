package catalog

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog/catalogdb"
)

// Repository provides data access for all core catalog entities.
type Repository interface {
	// Programs
	CreateProgram(ctx context.Context, p CreateProgramParams, actor *uuid.UUID) (catalogdb.Program, error)
	UpdateProgram(ctx context.Context, id uuid.UUID, p UpdateProgramParams, actor *uuid.UUID) (catalogdb.Program, error)
	GetProgram(ctx context.Context, id uuid.UUID) (catalogdb.Program, error)
	ListPrograms(ctx context.Context) ([]catalogdb.Program, error)
	SoftDeleteProgram(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error
	CountProgramCourses(ctx context.Context, programID uuid.UUID) (int64, error)
	CountLiveProgramQuotas(ctx context.Context, programID uuid.UUID) (int64, error)

	// Courses
	CreateCourse(ctx context.Context, p CreateCourseParams, actor *uuid.UUID) (catalogdb.Course, error)
	UpdateCourse(ctx context.Context, id uuid.UUID, p UpdateCourseParams, actor *uuid.UUID) (catalogdb.Course, error)
	GetCourse(ctx context.Context, id uuid.UUID) (catalogdb.Course, error)
	ListCourses(ctx context.Context) ([]catalogdb.Course, error)
	SoftDeleteCourse(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error
	CountCourseProgramAssociations(ctx context.Context, courseID uuid.UUID) (int64, error)

	// Program-course M:N
	AddCourseToProgram(ctx context.Context, programID, courseID uuid.UUID) (catalogdb.ProgramCourse, error)
	RemoveCourseFromProgram(ctx context.Context, programID, courseID uuid.UUID) error
	ListProgramCourses(ctx context.Context, programID uuid.UUID) ([]catalogdb.ProgramCourse, error)

	// Academic periods
	CreateAcademicPeriod(ctx context.Context, p CreateAcademicPeriodParams) (catalogdb.AcademicPeriod, error)
	UpdateAcademicPeriod(ctx context.Context, id uuid.UUID, p UpdateAcademicPeriodParams) (catalogdb.AcademicPeriod, error)
	GetAcademicPeriod(ctx context.Context, id uuid.UUID) (catalogdb.AcademicPeriod, error)
	ListAcademicPeriods(ctx context.Context) ([]catalogdb.AcademicPeriod, error)
	SoftDeleteAcademicPeriod(ctx context.Context, id uuid.UUID) error

	// Program quotas
	CreateProgramQuota(ctx context.Context, p CreateProgramQuotaParams, actor *uuid.UUID) (catalogdb.ProgramQuota, error)
	UpdateProgramQuota(ctx context.Context, id uuid.UUID, p UpdateProgramQuotaParams, actor *uuid.UUID) (catalogdb.ProgramQuota, error)
	GetProgramQuota(ctx context.Context, id uuid.UUID) (catalogdb.ProgramQuota, error)
	ListProgramQuotas(ctx context.Context, programID uuid.UUID) ([]catalogdb.ProgramQuota, error)
	SoftDeleteProgramQuota(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error

	// Sections
	CreateSection(ctx context.Context, p CreateSectionParams, actor *uuid.UUID) (catalogdb.Section, error)
	UpdateSection(ctx context.Context, id uuid.UUID, p UpdateSectionParams, actor *uuid.UUID) (catalogdb.Section, error)
	GetSection(ctx context.Context, id uuid.UUID) (catalogdb.Section, error)
	ListSections(ctx context.Context, courseID *uuid.UUID, academicPeriodID *uuid.UUID) ([]catalogdb.Section, error)
	SoftDeleteSection(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error
	CountLiveSectionsByCourse(ctx context.Context, courseID uuid.UUID) (int64, error)
	CountLiveSectionsByAcademicPeriod(ctx context.Context, academicPeriodID uuid.UUID) (int64, error)
	CountSectionTeachers(ctx context.Context, sectionID uuid.UUID) (int64, error)

	// Section-teacher M:N
	AssignTeacherToSection(ctx context.Context, sectionID, teacherID uuid.UUID) (catalogdb.SectionTeacher, error)
	RemoveTeacherFromSection(ctx context.Context, sectionID, teacherID uuid.UUID) error
	ListSectionTeachers(ctx context.Context, sectionID uuid.UUID) ([]catalogdb.SectionTeacher, error)
}

// Parameter types for repository operations.

// CreateProgramParams holds data for inserting a new program.
type CreateProgramParams struct {
	Code string
	Name string
}

// UpdateProgramParams holds data for updating an existing program.
type UpdateProgramParams struct {
	Code string
	Name string
}

// CreateCourseParams holds data for inserting a new course.
type CreateCourseParams struct {
	Code    string
	Name    string
	Credits int32
}

// UpdateCourseParams holds data for updating an existing course.
type UpdateCourseParams struct {
	Code    string
	Name    string
	Credits int32
}

// CreateAcademicPeriodParams holds data for inserting an academic period.
type CreateAcademicPeriodParams struct {
	Year      int32
	Term      int32
	StartDate pgtype.Date
	EndDate   pgtype.Date
}

// UpdateAcademicPeriodParams holds data for updating an academic period.
type UpdateAcademicPeriodParams struct {
	Year      int32
	Term      int32
	StartDate pgtype.Date
	EndDate   pgtype.Date
}

// CreateProgramQuotaParams holds data for inserting a program quota.
type CreateProgramQuotaParams struct {
	ProgramID uuid.UUID
	Year      int32
	Capacity  int32
}

// UpdateProgramQuotaParams holds data for updating a program quota.
type UpdateProgramQuotaParams struct {
	Year     int32
	Capacity int32
}

// CreateSectionParams holds data for inserting a new section.
type CreateSectionParams struct {
	CourseID         uuid.UUID
	AcademicPeriodID uuid.UUID
	SeatCapacity     int32
}

// UpdateSectionParams holds data for updating an existing section.
type UpdateSectionParams struct {
	SeatCapacity int32
}

// postgresRepository is the production implementation backed by a sqlc Querier.
type postgresRepository struct {
	q catalogdb.Querier
}

// Compile-time proof that *postgresRepository satisfies the Repository interface.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by the given sqlc Querier.
func NewPostgresRepository(q catalogdb.Querier) Repository {
	return &postgresRepository{q: q}
}

// --- Programs ---

func (r *postgresRepository) CreateProgram(ctx context.Context, p CreateProgramParams, actor *uuid.UUID) (catalogdb.Program, error) {
	row, err := r.q.InsertProgram(ctx, catalogdb.InsertProgramParams{
		Code:      p.Code,
		Name:      p.Name,
		CreatedBy: optionalUUID(actor),
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return catalogdb.Program{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) UpdateProgram(ctx context.Context, id uuid.UUID, p UpdateProgramParams, actor *uuid.UUID) (catalogdb.Program, error) {
	row, err := r.q.UpdateProgram(ctx, catalogdb.UpdateProgramParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Code:      p.Code,
		Name:      p.Name,
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return catalogdb.Program{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) GetProgram(ctx context.Context, id uuid.UUID) (catalogdb.Program, error) {
	row, err := r.q.GetProgram(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return catalogdb.Program{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) ListPrograms(ctx context.Context) ([]catalogdb.Program, error) {
	rows, err := r.q.ListPrograms(ctx)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteProgram(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error {
	n, err := r.q.SoftDeleteProgram(ctx, catalogdb.SoftDeleteProgramParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return TranslatePgError(err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) CountProgramCourses(ctx context.Context, programID uuid.UUID) (int64, error) {
	n, err := r.q.CountProgramCourses(ctx, pgtype.UUID{Bytes: programID, Valid: true})
	if err != nil {
		return 0, TranslatePgError(err)
	}
	return n, nil
}

func (r *postgresRepository) CountLiveProgramQuotas(ctx context.Context, programID uuid.UUID) (int64, error) {
	n, err := r.q.CountLiveProgramQuotas(ctx, pgtype.UUID{Bytes: programID, Valid: true})
	if err != nil {
		return 0, TranslatePgError(err)
	}
	return n, nil
}

// --- Courses ---

func (r *postgresRepository) CreateCourse(ctx context.Context, p CreateCourseParams, actor *uuid.UUID) (catalogdb.Course, error) {
	row, err := r.q.InsertCourse(ctx, catalogdb.InsertCourseParams{
		Code:      p.Code,
		Name:      p.Name,
		Credits:   p.Credits,
		CreatedBy: optionalUUID(actor),
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return catalogdb.Course{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) UpdateCourse(ctx context.Context, id uuid.UUID, p UpdateCourseParams, actor *uuid.UUID) (catalogdb.Course, error) {
	row, err := r.q.UpdateCourse(ctx, catalogdb.UpdateCourseParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Code:      p.Code,
		Name:      p.Name,
		Credits:   p.Credits,
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return catalogdb.Course{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) GetCourse(ctx context.Context, id uuid.UUID) (catalogdb.Course, error) {
	row, err := r.q.GetCourse(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return catalogdb.Course{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) ListCourses(ctx context.Context) ([]catalogdb.Course, error) {
	rows, err := r.q.ListCourses(ctx)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteCourse(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error {
	n, err := r.q.SoftDeleteCourse(ctx, catalogdb.SoftDeleteCourseParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return TranslatePgError(err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) CountCourseProgramAssociations(ctx context.Context, courseID uuid.UUID) (int64, error) {
	n, err := r.q.CountCourseProgramAssociations(ctx, pgtype.UUID{Bytes: courseID, Valid: true})
	if err != nil {
		return 0, TranslatePgError(err)
	}
	return n, nil
}

// --- Program courses (M:N) ---

func (r *postgresRepository) AddCourseToProgram(ctx context.Context, programID, courseID uuid.UUID) (catalogdb.ProgramCourse, error) {
	row, err := r.q.InsertProgramCourse(ctx, catalogdb.InsertProgramCourseParams{
		ProgramID: pgtype.UUID{Bytes: programID, Valid: true},
		CourseID:  pgtype.UUID{Bytes: courseID, Valid: true},
	})
	if err != nil {
		return catalogdb.ProgramCourse{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) RemoveCourseFromProgram(ctx context.Context, programID, courseID uuid.UUID) error {
	n, err := r.q.DeleteProgramCourse(ctx, catalogdb.DeleteProgramCourseParams{
		ProgramID: pgtype.UUID{Bytes: programID, Valid: true},
		CourseID:  pgtype.UUID{Bytes: courseID, Valid: true},
	})
	if err != nil {
		return TranslatePgError(err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) ListProgramCourses(ctx context.Context, programID uuid.UUID) ([]catalogdb.ProgramCourse, error) {
	rows, err := r.q.ListProgramCourses(ctx, pgtype.UUID{Bytes: programID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// --- Academic periods ---

func (r *postgresRepository) CreateAcademicPeriod(ctx context.Context, p CreateAcademicPeriodParams) (catalogdb.AcademicPeriod, error) {
	row, err := r.q.InsertAcademicPeriod(ctx, catalogdb.InsertAcademicPeriodParams{
		Year:      p.Year,
		Term:      p.Term,
		StartDate: p.StartDate,
		EndDate:   p.EndDate,
	})
	if err != nil {
		return catalogdb.AcademicPeriod{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) UpdateAcademicPeriod(ctx context.Context, id uuid.UUID, p UpdateAcademicPeriodParams) (catalogdb.AcademicPeriod, error) {
	row, err := r.q.UpdateAcademicPeriod(ctx, catalogdb.UpdateAcademicPeriodParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Year:      p.Year,
		Term:      p.Term,
		StartDate: p.StartDate,
		EndDate:   p.EndDate,
	})
	if err != nil {
		return catalogdb.AcademicPeriod{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) GetAcademicPeriod(ctx context.Context, id uuid.UUID) (catalogdb.AcademicPeriod, error) {
	row, err := r.q.GetAcademicPeriod(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return catalogdb.AcademicPeriod{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) ListAcademicPeriods(ctx context.Context) ([]catalogdb.AcademicPeriod, error) {
	rows, err := r.q.ListAcademicPeriods(ctx)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteAcademicPeriod(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteAcademicPeriod(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return TranslatePgError(err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

// --- Program quotas ---

func (r *postgresRepository) CreateProgramQuota(ctx context.Context, p CreateProgramQuotaParams, actor *uuid.UUID) (catalogdb.ProgramQuota, error) {
	row, err := r.q.UpsertProgramQuota(ctx, catalogdb.UpsertProgramQuotaParams{
		ProgramID: pgtype.UUID{Bytes: p.ProgramID, Valid: true},
		Year:      p.Year,
		Capacity:  p.Capacity,
		CreatedBy: optionalUUID(actor),
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return catalogdb.ProgramQuota{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) UpdateProgramQuota(ctx context.Context, id uuid.UUID, p UpdateProgramQuotaParams, actor *uuid.UUID) (catalogdb.ProgramQuota, error) {
	row, err := r.q.UpdateProgramQuota(ctx, catalogdb.UpdateProgramQuotaParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Year:      p.Year,
		Capacity:  p.Capacity,
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return catalogdb.ProgramQuota{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) GetProgramQuota(ctx context.Context, id uuid.UUID) (catalogdb.ProgramQuota, error) {
	row, err := r.q.GetProgramQuota(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return catalogdb.ProgramQuota{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) ListProgramQuotas(ctx context.Context, programID uuid.UUID) ([]catalogdb.ProgramQuota, error) {
	rows, err := r.q.ListProgramQuotas(ctx, pgtype.UUID{Bytes: programID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteProgramQuota(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error {
	n, err := r.q.SoftDeleteProgramQuota(ctx, catalogdb.SoftDeleteProgramQuotaParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return TranslatePgError(err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

// --- Sections ---

func (r *postgresRepository) CreateSection(ctx context.Context, p CreateSectionParams, actor *uuid.UUID) (catalogdb.Section, error) {
	row, err := r.q.InsertSection(ctx, catalogdb.InsertSectionParams{
		CourseID:         pgtype.UUID{Bytes: p.CourseID, Valid: true},
		AcademicPeriodID: pgtype.UUID{Bytes: p.AcademicPeriodID, Valid: true},
		Capacity:         p.SeatCapacity,
		CreatedBy:        optionalUUID(actor),
		UpdatedBy:        optionalUUID(actor),
	})
	if err != nil {
		return catalogdb.Section{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) UpdateSection(ctx context.Context, id uuid.UUID, p UpdateSectionParams, actor *uuid.UUID) (catalogdb.Section, error) {
	row, err := r.q.UpdateSection(ctx, catalogdb.UpdateSectionParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Capacity:  p.SeatCapacity,
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return catalogdb.Section{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) GetSection(ctx context.Context, id uuid.UUID) (catalogdb.Section, error) {
	row, err := r.q.GetSection(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return catalogdb.Section{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) ListSections(ctx context.Context, courseID *uuid.UUID, academicPeriodID *uuid.UUID) ([]catalogdb.Section, error) {
	rows, err := r.q.ListSections(ctx, catalogdb.ListSectionsParams{
		CourseID:         optionalUUID(courseID),
		AcademicPeriodID: optionalUUID(academicPeriodID),
	})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteSection(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error {
	n, err := r.q.SoftDeleteSection(ctx, catalogdb.SoftDeleteSectionParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return TranslatePgError(err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) CountLiveSectionsByCourse(ctx context.Context, courseID uuid.UUID) (int64, error) {
	n, err := r.q.CountLiveSectionsByCourse(ctx, pgtype.UUID{Bytes: courseID, Valid: true})
	if err != nil {
		return 0, TranslatePgError(err)
	}
	return n, nil
}

func (r *postgresRepository) CountLiveSectionsByAcademicPeriod(ctx context.Context, academicPeriodID uuid.UUID) (int64, error) {
	n, err := r.q.CountLiveSectionsByAcademicPeriod(ctx, pgtype.UUID{Bytes: academicPeriodID, Valid: true})
	if err != nil {
		return 0, TranslatePgError(err)
	}
	return n, nil
}

func (r *postgresRepository) CountSectionTeachers(ctx context.Context, sectionID uuid.UUID) (int64, error) {
	n, err := r.q.CountSectionTeachers(ctx, pgtype.UUID{Bytes: sectionID, Valid: true})
	if err != nil {
		return 0, TranslatePgError(err)
	}
	return n, nil
}

// --- Section-teacher M:N ---

func (r *postgresRepository) AssignTeacherToSection(ctx context.Context, sectionID, teacherID uuid.UUID) (catalogdb.SectionTeacher, error) {
	row, err := r.q.InsertSectionTeacher(ctx, catalogdb.InsertSectionTeacherParams{
		SectionID: pgtype.UUID{Bytes: sectionID, Valid: true},
		TeacherID: pgtype.UUID{Bytes: teacherID, Valid: true},
	})
	if err != nil {
		return catalogdb.SectionTeacher{}, TranslatePgError(err)
	}
	return row, nil
}

func (r *postgresRepository) RemoveTeacherFromSection(ctx context.Context, sectionID, teacherID uuid.UUID) error {
	n, err := r.q.DeleteSectionTeacher(ctx, catalogdb.DeleteSectionTeacherParams{
		SectionID: pgtype.UUID{Bytes: sectionID, Valid: true},
		TeacherID: pgtype.UUID{Bytes: teacherID, Valid: true},
	})
	if err != nil {
		return TranslatePgError(err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) ListSectionTeachers(ctx context.Context, sectionID uuid.UUID) ([]catalogdb.SectionTeacher, error) {
	rows, err := r.q.ListSectionTeachers(ctx, pgtype.UUID{Bytes: sectionID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// optionalUUID converts a *uuid.UUID to pgtype.UUID.
func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}
