package section_enrollment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/section_enrollment/section_enrollmentdb"
)

// Repository provides data access for the section_enrollment domain.
type Repository interface {
	// EnrollSectionTx runs the full seat-accounting transaction with a FOR UPDATE
	// lock on the section row. isAdmin=true skips the enrollment window gate and
	// enables revival of withdrawn inscriptions.
	EnrollSectionTx(ctx context.Context, p EnrollSectionParams, isAdmin bool) (section_enrollmentdb.SectionEnrollment, error)

	// WithdrawSection transitions an in_progress inscription to withdrawn.
	// Returns ErrNotFound when the inscription is absent or not in_progress.
	WithdrawSection(ctx context.Context, id uuid.UUID) (section_enrollmentdb.SectionEnrollment, error)

	// GetSectionEnrollment returns a live inscription by id. Soft-deleted rows return ErrNotFound.
	GetSectionEnrollment(ctx context.Context, id uuid.UUID) (section_enrollmentdb.SectionEnrollment, error)

	// ListSectionEnrollments returns live inscriptions matching the optional filter.
	ListSectionEnrollments(ctx context.Context, f ListSectionEnrollmentsFilter) ([]section_enrollmentdb.SectionEnrollment, error)

	// ListOwnSectionEnrollments returns all live inscriptions for the given student.
	ListOwnSectionEnrollments(ctx context.Context, studentID uuid.UUID) ([]section_enrollmentdb.SectionEnrollment, error)

	// GetOwnSectionEnrollment returns a live inscription by id without the ownership check
	// (ownership is enforced by the service). It is distinct from GetSectionEnrollment to
	// allow the service to apply scoping after the fetch.
	GetOwnSectionEnrollment(ctx context.Context, id uuid.UUID) (section_enrollmentdb.SectionEnrollment, error)
}

// EnrollSectionParams holds the validated inputs for EnrollSectionTx.
type EnrollSectionParams struct {
	// SectionID is the target section.
	SectionID uuid.UUID
	// EnrollmentID is the explicit enrollment_id (admin path). Ignored on the student
	// self-service path — the repository resolves the paid enrollment for the student.
	EnrollmentID uuid.UUID
	// StudentID is used only on the self-service path (isAdmin=false) to resolve the
	// paid enrollment for the section's program. Not used when isAdmin=true.
	StudentID uuid.UUID
}

// ListSectionEnrollmentsFilter holds optional filter parameters.
type ListSectionEnrollmentsFilter struct {
	SectionID    *uuid.UUID
	EnrollmentID *uuid.UUID
	Status       *string
}

// postgresRepository is the production implementation backed by a sqlc Querier and
// a connection pool used exclusively by EnrollSectionTx.
type postgresRepository struct {
	q    section_enrollmentdb.Querier
	pool *pgxpool.Pool
}

// Compile-time proof that *postgresRepository satisfies the Repository interface.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by the given sqlc Querier.
// pool is required by EnrollSectionTx, which opens a transaction.
func NewPostgresRepository(q section_enrollmentdb.Querier, pool *pgxpool.Pool) Repository {
	return &postgresRepository{q: q, pool: pool}
}

// EnrollSectionTx executes the seat-accounting transaction under READ COMMITTED isolation.
//
// Lock order (consistent everywhere — prevents deadlocks):
//  1. SELECT sections ... FOR UPDATE      (lock #1: section row)
//  2. SELECT section_enrollments ... FOR UPDATE  (lock #2: key row, revival detection)
//
// Transaction steps:
//  1. SET LOCAL lock_timeout='2500ms' — aborts if lock #1 waits too long.
//  2. Lock section row FOR UPDATE; fetch capacity, course_id, window columns.
//  3. Window gate (self-enroll only): now() ∈ [starts, ends]; null window → fail-closed.
//  4. Paid gate: enrollment must have status='paid'.
//  5. Course-in-program gate: section.course_id ∈ program_courses[enrollment.program_id].
//  6. Lock key row (enrollment_id, section_id) FOR UPDATE; detect existing/withdrawn.
//  7. COUNT active seats under the lock; n >= capacity → ErrSectionFull.
//  8. INSERT (fresh) or UPDATE (revival, admin only).
func (r *postgresRepository) EnrollSectionTx(ctx context.Context, p EnrollSectionParams, isAdmin bool) (section_enrollmentdb.SectionEnrollment, error) {
	// Non-locking fast-fail pre-check: avoids queuing on the lock for obviously-full sections.
	// The under-lock count below is authoritative; this is a stale-tolerant optimisation only.
	preCount, err := r.q.CountActiveSeats(ctx, pgtype.UUID{Bytes: p.SectionID, Valid: true})
	if err != nil {
		slog.ErrorContext(ctx, "section_enrollment: pre-check count failed", "err", err)
		return section_enrollmentdb.SectionEnrollment{}, fmt.Errorf("%w", ErrInvalidInput)
	}
	// We need the capacity for the pre-check comparison. Fetch it without locking.
	// The section capacity is read again under the lock for authoritative enforcement.
	// If the section is absent at pre-check we fall through to the tx (which will NotFound).

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := section_enrollmentdb.New(tx)

	// 1. Set per-transaction lock timeout. This bounds how long the hot path can block
	//    waiting for the section row lock under a registration stampede.
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '2500ms'"); err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}

	// 2. Lock section row FOR UPDATE. Fetches capacity, course_id, and enrollment window.
	sectionRow, err := q.GetSectionForUpdateWithWindow(ctx, pgtype.UUID{Bytes: p.SectionID, Valid: true})
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}

	// Now that we have the capacity we can apply the pre-check result.
	// If the pre-check already showed full, abort without lock overhead (optimisation only).
	if preCount >= int64(sectionRow.Capacity) {
		return section_enrollmentdb.SectionEnrollment{}, ErrSectionFull
	}

	// 3. Window gate (self-enrollment only). Admin path is never window-gated.
	if !isAdmin {
		if err := checkEnrollmentWindow(sectionRow); err != nil {
			return section_enrollmentdb.SectionEnrollment{}, err
		}
	}

	// 4. Paid gate: resolve the linked enrollment and verify status='paid'.
	// Admin path: enrollment_id is supplied directly; student path resolves via student+program.
	var enrollmentID pgtype.UUID
	var programID pgtype.UUID

	if isAdmin {
		eRow, err := q.ResolvePaidEnrollmentByID(ctx, pgtype.UUID{Bytes: p.EnrollmentID, Valid: true})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return section_enrollmentdb.SectionEnrollment{}, ErrNotPaid
			}
			return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
		}
		enrollmentID = eRow.ID
		programID = eRow.ProgramID
	} else {
		// Self-service path: the request carries no enrollment_id.
		// Resolve the paid enrollment for this student by joining through program_courses
		// using the section's course_id. A student has at most one paid enrollment per program,
		// and the section's course appears in exactly the programs they are enrolled in.
		eRow, err := q.ResolvePaidEnrollmentForStudentAndCourse(ctx, section_enrollmentdb.ResolvePaidEnrollmentForStudentAndCourseParams{
			StudentID: pgtype.UUID{Bytes: p.StudentID, Valid: true},
			CourseID:  sectionRow.CourseID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return section_enrollmentdb.SectionEnrollment{}, ErrNotPaid
			}
			return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
		}
		enrollmentID = eRow.ID
		programID = eRow.ProgramID
	}

	// 5. Course-in-program gate.
	inProgram, err := q.CourseInProgram(ctx, section_enrollmentdb.CourseInProgramParams{
		ProgramID: programID,
		CourseID:  sectionRow.CourseID,
	})
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	if !inProgram {
		return section_enrollmentdb.SectionEnrollment{}, ErrCourseNotInProgram
	}

	// 6. Lock key row FOR UPDATE (revival detection). Includes withdrawn rows.
	existingRow, keyErr := q.GetSectionEnrollmentByKeyForUpdate(ctx, section_enrollmentdb.GetSectionEnrollmentByKeyForUpdateParams{
		EnrollmentID: enrollmentID,
		SectionID:    pgtype.UUID{Bytes: p.SectionID, Valid: true},
	})

	var isRevival bool
	switch {
	case keyErr == nil:
		if existingRow.Status == "in_progress" && !existingRow.DeletedAt.Valid {
			// Live in_progress row exists.
			return section_enrollmentdb.SectionEnrollment{}, ErrAlreadyExists
		}
		if existingRow.Status == "withdrawn" {
			if !isAdmin {
				// Students cannot self-revive a withdrawn inscription.
				return section_enrollmentdb.SectionEnrollment{}, ErrWithdrawnNotRevivable
			}
			isRevival = true
		}
	case errors.Is(keyErr, pgx.ErrNoRows):
		// No existing row — fresh insert path.
	default:
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(keyErr)
	}

	// 7. COUNT active seats under the lock. This is the authoritative check.
	activeCount, err := q.CountActiveSeats(ctx, pgtype.UUID{Bytes: p.SectionID, Valid: true})
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	if activeCount >= int64(sectionRow.Capacity) {
		return section_enrollmentdb.SectionEnrollment{}, ErrSectionFull
	}

	// 8. Insert (fresh) or revive (admin only).
	var result section_enrollmentdb.SectionEnrollment
	if isRevival {
		result, err = q.ReviveSectionEnrollment(ctx, existingRow.ID)
	} else {
		result, err = q.InsertSectionEnrollment(ctx, section_enrollmentdb.InsertSectionEnrollmentParams{
			EnrollmentID: enrollmentID,
			SectionID:    pgtype.UUID{Bytes: p.SectionID, Valid: true},
		})
	}
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	return result, nil
}

// WithdrawSection transitions an in_progress inscription to withdrawn.
// Returns ErrNotFound when the inscription is absent, already withdrawn, or soft-deleted.
func (r *postgresRepository) WithdrawSection(ctx context.Context, id uuid.UUID) (section_enrollmentdb.SectionEnrollment, error) {
	row, err := r.q.WithdrawSectionEnrollment(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	return row, nil
}

// GetSectionEnrollment returns a live inscription by id.
func (r *postgresRepository) GetSectionEnrollment(ctx context.Context, id uuid.UUID) (section_enrollmentdb.SectionEnrollment, error) {
	row, err := r.q.GetSectionEnrollmentByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return section_enrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	return row, nil
}

// GetOwnSectionEnrollment returns a live inscription by id (same as GetSectionEnrollment;
// ownership is enforced by the service after the fetch).
func (r *postgresRepository) GetOwnSectionEnrollment(ctx context.Context, id uuid.UUID) (section_enrollmentdb.SectionEnrollment, error) {
	return r.GetSectionEnrollment(ctx, id)
}

// ListSectionEnrollments returns live inscriptions filtered by optional criteria.
func (r *postgresRepository) ListSectionEnrollments(ctx context.Context, f ListSectionEnrollmentsFilter) ([]section_enrollmentdb.SectionEnrollment, error) {
	params := section_enrollmentdb.ListSectionEnrollmentsParams{}
	if f.SectionID != nil {
		params.SectionID = pgtype.UUID{Bytes: *f.SectionID, Valid: true}
	}
	if f.EnrollmentID != nil {
		params.EnrollmentID = pgtype.UUID{Bytes: *f.EnrollmentID, Valid: true}
	}
	if f.Status != nil {
		params.Status = pgtype.Text{String: *f.Status, Valid: true}
	}
	rows, err := r.q.ListSectionEnrollments(ctx, params)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// ListOwnSectionEnrollments returns all live inscriptions for the given student.
func (r *postgresRepository) ListOwnSectionEnrollments(ctx context.Context, studentID uuid.UUID) ([]section_enrollmentdb.SectionEnrollment, error) {
	rows, err := r.q.ListOwnSectionEnrollments(ctx, pgtype.UUID{Bytes: studentID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// checkEnrollmentWindow verifies that the current time falls within the section's
// academic period enrollment window. Null/unset window columns → fail-closed.
func checkEnrollmentWindow(row section_enrollmentdb.GetSectionForUpdateWithWindowRow) error {
	if !row.EnrollmentStartsAt.Valid || !row.EnrollmentEndsAt.Valid {
		return fmt.Errorf("%w: window not configured", ErrWindowClosed)
	}
	now := time.Now().UTC()
	starts := row.EnrollmentStartsAt.Time.UTC()
	ends := row.EnrollmentEndsAt.Time.UTC()
	if now.Before(starts) || now.After(ends) {
		return fmt.Errorf("%w: current time is outside [%s, %s]", ErrWindowClosed, starts.Format(time.RFC3339), ends.Format(time.RFC3339))
	}
	return nil
}

// newFakeRepository constructs a repository backed by the given Querier without a
// pool (the pool is only used for EnrollSectionTx, which opens a real transaction).
// Used exclusively in unit tests that exercise non-transactional repository methods.
func newFakeRepository(q section_enrollmentdb.Querier) Repository {
	return &postgresRepository{q: q, pool: nil}
}
