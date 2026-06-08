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
	SoftDeleteProgram(ctx context.Context, id uuid.UUID) error
	CountProgramCourses(ctx context.Context, programID uuid.UUID) (int64, error)
	CountLiveProgramQuotas(ctx context.Context, programID uuid.UUID) (int64, error)

	// Courses
	CreateCourse(ctx context.Context, p CreateCourseParams, actor *uuid.UUID) (catalogdb.Course, error)
	UpdateCourse(ctx context.Context, id uuid.UUID, p UpdateCourseParams, actor *uuid.UUID) (catalogdb.Course, error)
	GetCourse(ctx context.Context, id uuid.UUID) (catalogdb.Course, error)
	ListCourses(ctx context.Context) ([]catalogdb.Course, error)
	SoftDeleteCourse(ctx context.Context, id uuid.UUID) error
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

// postgresRepository is the production implementation backed by a sqlc Querier.
type postgresRepository struct {
	q catalogdb.Querier
}

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
		return nil, fmt.Errorf("catalog: ListPrograms: %w", err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteProgram(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteProgram(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return fmt.Errorf("catalog: SoftDeleteProgram: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) CountProgramCourses(ctx context.Context, programID uuid.UUID) (int64, error) {
	n, err := r.q.CountProgramCourses(ctx, pgtype.UUID{Bytes: programID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("catalog: CountProgramCourses: %w", err)
	}
	return n, nil
}

func (r *postgresRepository) CountLiveProgramQuotas(ctx context.Context, programID uuid.UUID) (int64, error) {
	n, err := r.q.CountLiveProgramQuotas(ctx, pgtype.UUID{Bytes: programID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("catalog: CountLiveProgramQuotas: %w", err)
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
		return nil, fmt.Errorf("catalog: ListCourses: %w", err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteCourse(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteCourse(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return fmt.Errorf("catalog: SoftDeleteCourse: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) CountCourseProgramAssociations(ctx context.Context, courseID uuid.UUID) (int64, error) {
	n, err := r.q.CountCourseProgramAssociations(ctx, pgtype.UUID{Bytes: courseID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("catalog: CountCourseProgramAssociations: %w", err)
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
		return fmt.Errorf("catalog: RemoveCourseFromProgram: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

func (r *postgresRepository) ListProgramCourses(ctx context.Context, programID uuid.UUID) ([]catalogdb.ProgramCourse, error) {
	rows, err := r.q.ListProgramCourses(ctx, pgtype.UUID{Bytes: programID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("catalog: ListProgramCourses: %w", err)
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
		return nil, fmt.Errorf("catalog: ListAcademicPeriods: %w", err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteAcademicPeriod(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteAcademicPeriod(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return fmt.Errorf("catalog: SoftDeleteAcademicPeriod: %w", err)
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
		return nil, fmt.Errorf("catalog: ListProgramQuotas: %w", err)
	}
	return rows, nil
}

func (r *postgresRepository) SoftDeleteProgramQuota(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error {
	n, err := r.q.SoftDeleteProgramQuota(ctx, catalogdb.SoftDeleteProgramQuotaParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return fmt.Errorf("catalog: SoftDeleteProgramQuota: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w", ErrNotFound)
	}
	return nil
}

// optionalUUID converts a *uuid.UUID to pgtype.UUID.
func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}
