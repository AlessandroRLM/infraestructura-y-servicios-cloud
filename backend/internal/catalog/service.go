package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog/catalogdb"
)

// Service orchestrates catalog business logic: validation, audit-column population,
// dependent-blocking soft-delete enforcement, and repository delegation.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the given Repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// --- Service-layer parameter types ---
// These differ from repository params where the service needs string inputs
// (e.g. dates as ISO strings, UUIDs as strings) before conversion.

// CreateAcademicPeriodServiceParams carries the raw inputs from the handler.
type CreateAcademicPeriodServiceParams struct {
	Year      int32
	Term      int32
	StartDate string
	EndDate   string
}

// UpdateAcademicPeriodServiceParams carries the raw inputs from the handler.
type UpdateAcademicPeriodServiceParams struct {
	Year      int32
	Term      int32
	StartDate string
	EndDate   string
}

// CreateProgramQuotaServiceParams carries the raw inputs from the handler.
type CreateProgramQuotaServiceParams struct {
	ProgramID string
	Year      int32
	Capacity  int32
}

// UpdateProgramQuotaServiceParams carries the raw inputs from the handler.
type UpdateProgramQuotaServiceParams struct {
	Year     int32
	Capacity int32
}

// --- Programs ---

// CreateProgram validates input, populates audit columns, and delegates to the repository.
func (s *Service) CreateProgram(ctx context.Context, p CreateProgramParams) (catalogdb.Program, error) {
	if p.Code == "" {
		return catalogdb.Program{}, fmt.Errorf("%w: code is required", ErrInvalidInput)
	}
	if p.Name == "" {
		return catalogdb.Program{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	return s.repo.CreateProgram(ctx, p, actorFromContext(ctx))
}

// UpdateProgram validates input and delegates to the repository.
func (s *Service) UpdateProgram(ctx context.Context, id uuid.UUID, p UpdateProgramParams) (catalogdb.Program, error) {
	if p.Code == "" {
		return catalogdb.Program{}, fmt.Errorf("%w: code is required", ErrInvalidInput)
	}
	if p.Name == "" {
		return catalogdb.Program{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	return s.repo.UpdateProgram(ctx, id, p, actorFromContext(ctx))
}

// GetProgram retrieves a live program by id.
func (s *Service) GetProgram(ctx context.Context, id uuid.UUID) (catalogdb.Program, error) {
	return s.repo.GetProgram(ctx, id)
}

// ListPrograms returns all live programs.
func (s *Service) ListPrograms(ctx context.Context) ([]catalogdb.Program, error) {
	return s.repo.ListPrograms(ctx)
}

// DeleteProgram soft-deletes a program after verifying no live dependents exist.
// Blocked by: live program_courses rows OR live program_quotas rows.
func (s *Service) DeleteProgram(ctx context.Context, id uuid.UUID) error {
	n, err := s.repo.CountProgramCourses(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("%w: program has %d course association(s)", ErrHasDependents, n)
	}

	q, err := s.repo.CountLiveProgramQuotas(ctx, id)
	if err != nil {
		return err
	}
	if q > 0 {
		return fmt.Errorf("%w: program has %d live quota(s)", ErrHasDependents, q)
	}

	return s.repo.SoftDeleteProgram(ctx, id, actorFromContext(ctx))
}

// --- Courses ---

// CreateCourse validates input, populates audit columns, and delegates to the repository.
func (s *Service) CreateCourse(ctx context.Context, p CreateCourseParams) (catalogdb.Course, error) {
	if p.Code == "" {
		return catalogdb.Course{}, fmt.Errorf("%w: code is required", ErrInvalidInput)
	}
	if p.Name == "" {
		return catalogdb.Course{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if p.Credits <= 0 {
		return catalogdb.Course{}, fmt.Errorf("%w: credits must be greater than 0", ErrInvalidInput)
	}
	return s.repo.CreateCourse(ctx, p, actorFromContext(ctx))
}

// UpdateCourse validates input and delegates to the repository.
func (s *Service) UpdateCourse(ctx context.Context, id uuid.UUID, p UpdateCourseParams) (catalogdb.Course, error) {
	if p.Code == "" {
		return catalogdb.Course{}, fmt.Errorf("%w: code is required", ErrInvalidInput)
	}
	if p.Name == "" {
		return catalogdb.Course{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if p.Credits <= 0 {
		return catalogdb.Course{}, fmt.Errorf("%w: credits must be greater than 0", ErrInvalidInput)
	}
	return s.repo.UpdateCourse(ctx, id, p, actorFromContext(ctx))
}

// GetCourse retrieves a live course by id.
func (s *Service) GetCourse(ctx context.Context, id uuid.UUID) (catalogdb.Course, error) {
	return s.repo.GetCourse(ctx, id)
}

// ListCourses returns all live courses.
func (s *Service) ListCourses(ctx context.Context) ([]catalogdb.Course, error) {
	return s.repo.ListCourses(ctx)
}

// DeleteCourse soft-deletes a course after verifying no live program associations or live sections exist.
func (s *Service) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	n, err := s.repo.CountCourseProgramAssociations(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("%w: course has %d program association(s)", ErrHasDependents, n)
	}

	sec, err := s.repo.CountLiveSectionsByCourse(ctx, id)
	if err != nil {
		return err
	}
	if sec > 0 {
		return fmt.Errorf("%w: course has %d live section(s)", ErrHasDependents, sec)
	}

	return s.repo.SoftDeleteCourse(ctx, id, actorFromContext(ctx))
}

// --- Program-course M:N ---

// AddCourseToProgram inserts a program_courses row. Duplicate returns ErrAlreadyExists.
func (s *Service) AddCourseToProgram(ctx context.Context, programID, courseID uuid.UUID) (catalogdb.ProgramCourse, error) {
	return s.repo.AddCourseToProgram(ctx, programID, courseID)
}

// RemoveCourseFromProgram hard-deletes a program_courses row.
func (s *Service) RemoveCourseFromProgram(ctx context.Context, programID, courseID uuid.UUID) error {
	return s.repo.RemoveCourseFromProgram(ctx, programID, courseID)
}

// ListProgramCourses returns all course associations for the given program.
func (s *Service) ListProgramCourses(ctx context.Context, programID uuid.UUID) ([]catalogdb.ProgramCourse, error) {
	return s.repo.ListProgramCourses(ctx, programID)
}

// --- Academic periods ---

// CreateAcademicPeriod validates input and delegates to the repository.
// No created_by/updated_by: academic_periods does not carry those columns per §10.1.
func (s *Service) CreateAcademicPeriod(ctx context.Context, p CreateAcademicPeriodServiceParams) (catalogdb.AcademicPeriod, error) {
	if err := validateAcademicPeriod(p.Year, p.Term, p.StartDate, p.EndDate); err != nil {
		return catalogdb.AcademicPeriod{}, err
	}

	start, _ := ParseDate(p.StartDate)
	end, _ := ParseDate(p.EndDate)

	return s.repo.CreateAcademicPeriod(ctx, CreateAcademicPeriodParams{
		Year:      p.Year,
		Term:      p.Term,
		StartDate: start,
		EndDate:   end,
	})
}

// UpdateAcademicPeriod validates input and delegates to the repository.
func (s *Service) UpdateAcademicPeriod(ctx context.Context, id uuid.UUID, p UpdateAcademicPeriodServiceParams) (catalogdb.AcademicPeriod, error) {
	if err := validateAcademicPeriod(p.Year, p.Term, p.StartDate, p.EndDate); err != nil {
		return catalogdb.AcademicPeriod{}, err
	}

	start, _ := ParseDate(p.StartDate)
	end, _ := ParseDate(p.EndDate)

	return s.repo.UpdateAcademicPeriod(ctx, id, UpdateAcademicPeriodParams{
		Year:      p.Year,
		Term:      p.Term,
		StartDate: start,
		EndDate:   end,
	})
}

// GetAcademicPeriod retrieves a live academic period by id.
func (s *Service) GetAcademicPeriod(ctx context.Context, id uuid.UUID) (catalogdb.AcademicPeriod, error) {
	return s.repo.GetAcademicPeriod(ctx, id)
}

// ListAcademicPeriods returns all live academic periods.
func (s *Service) ListAcademicPeriods(ctx context.Context) ([]catalogdb.AcademicPeriod, error) {
	return s.repo.ListAcademicPeriods(ctx)
}

// DeleteAcademicPeriod soft-deletes an academic period after verifying no live sections reference it.
func (s *Service) DeleteAcademicPeriod(ctx context.Context, id uuid.UUID) error {
	sec, err := s.repo.CountLiveSectionsByAcademicPeriod(ctx, id)
	if err != nil {
		return err
	}
	if sec > 0 {
		return fmt.Errorf("%w: academic period has %d live section(s)", ErrHasDependents, sec)
	}

	return s.repo.SoftDeleteAcademicPeriod(ctx, id)
}

// --- Program quotas ---

// CreateProgramQuota validates input, populates audit columns, and delegates to the repository.
func (s *Service) CreateProgramQuota(ctx context.Context, p CreateProgramQuotaServiceParams) (catalogdb.ProgramQuota, error) {
	if p.Year <= 0 {
		return catalogdb.ProgramQuota{}, fmt.Errorf("%w: year must be greater than 0", ErrInvalidInput)
	}
	if p.Capacity <= 0 {
		return catalogdb.ProgramQuota{}, fmt.Errorf("%w: capacity must be greater than 0", ErrInvalidInput)
	}

	programID, err := uuid.Parse(p.ProgramID)
	if err != nil {
		return catalogdb.ProgramQuota{}, fmt.Errorf("%w: invalid program_id", ErrInvalidInput)
	}

	return s.repo.CreateProgramQuota(ctx, CreateProgramQuotaParams{
		ProgramID: programID,
		Year:      p.Year,
		Capacity:  p.Capacity,
	}, actorFromContext(ctx))
}

// UpdateProgramQuota validates input and delegates to the repository.
func (s *Service) UpdateProgramQuota(ctx context.Context, id uuid.UUID, p UpdateProgramQuotaServiceParams) (catalogdb.ProgramQuota, error) {
	if p.Year <= 0 {
		return catalogdb.ProgramQuota{}, fmt.Errorf("%w: year must be greater than 0", ErrInvalidInput)
	}
	if p.Capacity <= 0 {
		return catalogdb.ProgramQuota{}, fmt.Errorf("%w: capacity must be greater than 0", ErrInvalidInput)
	}

	return s.repo.UpdateProgramQuota(ctx, id, UpdateProgramQuotaParams(p), actorFromContext(ctx))
}

// GetProgramQuota retrieves a live program quota by id.
func (s *Service) GetProgramQuota(ctx context.Context, id uuid.UUID) (catalogdb.ProgramQuota, error) {
	return s.repo.GetProgramQuota(ctx, id)
}

// ListProgramQuotas returns all live quotas for the given program.
func (s *Service) ListProgramQuotas(ctx context.Context, programID uuid.UUID) ([]catalogdb.ProgramQuota, error) {
	return s.repo.ListProgramQuotas(ctx, programID)
}

// DeleteProgramQuota soft-deletes a program quota.
// program_quotas has no downstream dependents; the soft-delete always succeeds when the row is live.
func (s *Service) DeleteProgramQuota(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDeleteProgramQuota(ctx, id, actorFromContext(ctx))
}

// --- Sections ---

// CreateSectionServiceParams carries the raw inputs from the handler.
type CreateSectionServiceParams struct {
	CourseID         string
	AcademicPeriodID string
	SeatCapacity     int32
}

// UpdateSectionServiceParams carries the raw inputs from the handler.
type UpdateSectionServiceParams struct {
	SeatCapacity int32
}

// CreateSection validates input, populates audit columns, and delegates to the repository.
func (s *Service) CreateSection(ctx context.Context, p CreateSectionServiceParams) (catalogdb.Section, error) {
	if p.SeatCapacity <= 0 {
		return catalogdb.Section{}, fmt.Errorf("%w: seat_capacity must be greater than 0", ErrInvalidInput)
	}

	courseID, err := uuid.Parse(p.CourseID)
	if err != nil {
		return catalogdb.Section{}, fmt.Errorf("%w: invalid course_id", ErrInvalidInput)
	}

	periodID, err := uuid.Parse(p.AcademicPeriodID)
	if err != nil {
		return catalogdb.Section{}, fmt.Errorf("%w: invalid academic_period_id", ErrInvalidInput)
	}

	return s.repo.CreateSection(ctx, CreateSectionParams{
		CourseID:         courseID,
		AcademicPeriodID: periodID,
		SeatCapacity:     p.SeatCapacity,
	}, actorFromContext(ctx))
}

// UpdateSection validates input and delegates to the repository.
func (s *Service) UpdateSection(ctx context.Context, id uuid.UUID, p UpdateSectionServiceParams) (catalogdb.Section, error) {
	if p.SeatCapacity <= 0 {
		return catalogdb.Section{}, fmt.Errorf("%w: seat_capacity must be greater than 0", ErrInvalidInput)
	}
	return s.repo.UpdateSection(ctx, id, UpdateSectionParams(p), actorFromContext(ctx))
}

// GetSection retrieves a live section by id.
func (s *Service) GetSection(ctx context.Context, id uuid.UUID) (catalogdb.Section, error) {
	return s.repo.GetSection(ctx, id)
}

// ListSections returns live sections, optionally filtered by course or academic period.
// Nil filters return all live sections.
func (s *Service) ListSections(ctx context.Context, courseID *uuid.UUID, academicPeriodID *uuid.UUID) ([]catalogdb.Section, error) {
	return s.repo.ListSections(ctx, courseID, academicPeriodID)
}

// DeleteSection soft-deletes a section after verifying no section_teachers rows exist.
// section_teachers is append-only; any existing row blocks deletion.
func (s *Service) DeleteSection(ctx context.Context, id uuid.UUID) error {
	n, err := s.repo.CountSectionTeachers(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("%w: section has %d teacher assignment(s)", ErrHasDependents, n)
	}

	return s.repo.SoftDeleteSection(ctx, id, actorFromContext(ctx))
}

// --- Section-teacher M:N ---

// AssignTeacherToSection inserts a section_teachers row. Duplicate returns ErrAlreadyExists; bad FK returns ErrInvalidInput.
func (s *Service) AssignTeacherToSection(ctx context.Context, sectionID, teacherID uuid.UUID) (catalogdb.SectionTeacher, error) {
	return s.repo.AssignTeacherToSection(ctx, sectionID, teacherID)
}

// RemoveTeacherFromSection hard-deletes a section_teachers row. Returns ErrNotFound when absent.
func (s *Service) RemoveTeacherFromSection(ctx context.Context, sectionID, teacherID uuid.UUID) error {
	return s.repo.RemoveTeacherFromSection(ctx, sectionID, teacherID)
}

// ListSectionTeachers returns all teacher assignments for the given section.
func (s *Service) ListSectionTeachers(ctx context.Context, sectionID uuid.UUID) ([]catalogdb.SectionTeacher, error) {
	return s.repo.ListSectionTeachers(ctx, sectionID)
}

// --- Helpers ---

// actorFromContext extracts the authenticated user_id from context and returns a pointer.
// Returns nil when no actor is present (system or background operations).
func actorFromContext(ctx context.Context) *uuid.UUID {
	id, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil
	}
	return &id
}

// validateAcademicPeriod enforces all academic period business rules.
func validateAcademicPeriod(year, term int32, startDate, endDate string) error {
	if year <= 0 {
		return fmt.Errorf("%w: year must be greater than 0", ErrInvalidInput)
	}
	if term != 1 && term != 2 {
		return fmt.Errorf("%w: term must be 1 or 2", ErrInvalidInput)
	}

	start, err := ParseDate(startDate)
	if err != nil {
		return fmt.Errorf("%w: invalid start_date: %v", ErrInvalidInput, err)
	}
	end, err := ParseDate(endDate)
	if err != nil {
		return fmt.Errorf("%w: invalid end_date: %v", ErrInvalidInput, err)
	}
	if !start.Time.Before(end.Time) {
		return fmt.Errorf("%w: start_date must be before end_date", ErrInvalidInput)
	}

	return nil
}

// ParseDate parses an ISO 8601 date string (YYYY-MM-DD) into a pgtype.Date.
func ParseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("invalid date %q: %w", s, err)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

// FormatDate converts a pgtype.Date to an ISO 8601 string.
// Returns an empty string when the date is null.
func FormatDate(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}
