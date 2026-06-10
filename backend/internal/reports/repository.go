package reports

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/reports/reportsdb"
)

// Repository is the consumer-side data-access seam for the reports slice.
// All methods are read-only (pure SELECTs). No pool or transaction is exposed.
type Repository interface {
	// SectionExists returns true if the section with the given ID exists and is not soft-deleted.
	SectionExists(ctx context.Context, id uuid.UUID) (bool, error)
	// IsTeacherForSection returns true when teacherID appears in section_teachers for sectionID.
	// This check MUST be performed before any cache lookup to prevent cross-teacher cache leakage.
	IsTeacherForSection(ctx context.Context, sectionID, teacherID uuid.UUID) (bool, error)
	// ActaForSectionAdmin returns all grade rows for a section, visible to admin callers.
	// Returns up to LIMIT cap+1 rows; caller detects truncation.
	ActaForSectionAdmin(ctx context.Context, sectionID uuid.UUID) ([]reportsdb.ActaForSectionAdminRow, error)
	// ActaForSectionByTeacher returns grade rows for a section owned by the given teacher.
	// Returns up to LIMIT cap+1 rows; caller detects truncation.
	ActaForSectionByTeacher(ctx context.Context, sectionID, teacherID uuid.UUID) ([]reportsdb.ActaForSectionByTeacherRow, error)
	// PeriodExists returns true if the academic period with the given ID exists and is not soft-deleted.
	PeriodExists(ctx context.Context, id uuid.UUID) (bool, error)
	// OccupancyForPeriod returns occupancy rows for all sections in a given academic period.
	// Returns up to LIMIT cap+1 rows; caller detects truncation.
	OccupancyForPeriod(ctx context.Context, periodID uuid.UUID) ([]reportsdb.OccupancyForPeriodRow, error)
	// ProgramExists returns true if the program with the given ID exists and is not soft-deleted.
	ProgramExists(ctx context.Context, id uuid.UUID) (bool, error)
	// ProgramSummary returns quota and enrollment counts for a program in a given year.
	// Returns up to LIMIT cap+1 rows; caller detects truncation.
	ProgramSummary(ctx context.Context, programID uuid.UUID, year int32) ([]reportsdb.ProgramSummaryRow, error)
	// StudentExists returns true if the student with the given ID exists and is not soft-deleted.
	StudentExists(ctx context.Context, id uuid.UUID) (bool, error)
	// FichaForStudent returns the complete academic record for a given student.
	// Returns up to LIMIT cap+1 rows; caller detects truncation.
	FichaForStudent(ctx context.Context, studentID uuid.UUID) ([]reportsdb.FichaForStudentRow, error)
}

// postgresRepository wraps reportsdb.Querier and implements Repository.
type postgresRepository struct {
	q reportsdb.Querier
}

// Compile-time proof that *postgresRepository satisfies Repository.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by a reportsdb.Querier.
// The querier is typically a *reportsdb.Queries obtained from reportsdb.New(pool).
func NewPostgresRepository(q reportsdb.Querier) *postgresRepository {
	return &postgresRepository{q: q}
}

func (r *postgresRepository) SectionExists(ctx context.Context, id uuid.UUID) (bool, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	exists, err := r.q.ActaSectionExists(ctx, pgID)
	if err != nil {
		return false, TranslatePgError(err)
	}
	return exists, nil
}

func (r *postgresRepository) IsTeacherForSection(ctx context.Context, sectionID, teacherID uuid.UUID) (bool, error) {
	params := reportsdb.IsTeacherForSectionParams{
		SectionID: pgtype.UUID{Bytes: sectionID, Valid: true},
		TeacherID: pgtype.UUID{Bytes: teacherID, Valid: true},
	}
	isMember, err := r.q.IsTeacherForSection(ctx, params)
	if err != nil {
		return false, TranslatePgError(err)
	}
	return isMember, nil
}

func (r *postgresRepository) ActaForSectionAdmin(ctx context.Context, sectionID uuid.UUID) ([]reportsdb.ActaForSectionAdminRow, error) {
	pgID := pgtype.UUID{Bytes: sectionID, Valid: true}
	rows, err := r.q.ActaForSectionAdmin(ctx, pgID)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) ActaForSectionByTeacher(ctx context.Context, sectionID, teacherID uuid.UUID) ([]reportsdb.ActaForSectionByTeacherRow, error) {
	params := reportsdb.ActaForSectionByTeacherParams{
		ID:        pgtype.UUID{Bytes: sectionID, Valid: true},
		TeacherID: pgtype.UUID{Bytes: teacherID, Valid: true},
	}
	rows, err := r.q.ActaForSectionByTeacher(ctx, params)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) PeriodExists(ctx context.Context, id uuid.UUID) (bool, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	exists, err := r.q.OccupancyPeriodExists(ctx, pgID)
	if err != nil {
		return false, TranslatePgError(err)
	}
	return exists, nil
}

func (r *postgresRepository) OccupancyForPeriod(ctx context.Context, periodID uuid.UUID) ([]reportsdb.OccupancyForPeriodRow, error) {
	pgID := pgtype.UUID{Bytes: periodID, Valid: true}
	rows, err := r.q.OccupancyForPeriod(ctx, pgID)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) ProgramExists(ctx context.Context, id uuid.UUID) (bool, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	exists, err := r.q.ProgramExists(ctx, pgID)
	if err != nil {
		return false, TranslatePgError(err)
	}
	return exists, nil
}

func (r *postgresRepository) ProgramSummary(ctx context.Context, programID uuid.UUID, year int32) ([]reportsdb.ProgramSummaryRow, error) {
	params := reportsdb.ProgramSummaryParams{
		ProgramID: pgtype.UUID{Bytes: programID, Valid: true},
		Year:      year,
	}
	rows, err := r.q.ProgramSummary(ctx, params)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

func (r *postgresRepository) StudentExists(ctx context.Context, id uuid.UUID) (bool, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	exists, err := r.q.StudentExists(ctx, pgID)
	if err != nil {
		return false, TranslatePgError(err)
	}
	return exists, nil
}

func (r *postgresRepository) FichaForStudent(ctx context.Context, studentID uuid.UUID) ([]reportsdb.FichaForStudentRow, error) {
	pgID := pgtype.UUID{Bytes: studentID, Valid: true}
	rows, err := r.q.FichaForStudent(ctx, pgID)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}
