package sectionenrollment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment/sectionenrollmentdb"
)

// Repository provides data access for the section_enrollment domain.
type Repository interface {
	// EnrollSectionTx runs the full seat-accounting transaction with a FOR UPDATE
	// lock on the section row. isAdmin=true skips the enrollment window gate and
	// enables revival of withdrawn inscriptions.
	EnrollSectionTx(ctx context.Context, p EnrollSectionParams, isAdmin bool) (sectionenrollmentdb.SectionEnrollment, error)

	// SetSectionEnrollmentOutcomeTx transitions a section_enrollment to passed or failed
	// and writes the rounded final grade, executing within a caller-owned transaction.
	// Valid source states: in_progress, passed, failed (allows passed<->failed corrections).
	// withdrawn source is rejected → ErrInvalidTransition. target must be passed or failed.
	// finalGrade is the rounded NUMERIC(3,1) value to persist alongside the outcome.
	SetSectionEnrollmentOutcomeTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, outcome string, finalGrade pgtype.Numeric) (sectionenrollmentdb.SectionEnrollment, error)

	// WithdrawSection transitions an in_progress inscription to withdrawn.
	// Returns ErrNotFound when the inscription is absent or not in_progress.
	WithdrawSection(ctx context.Context, id uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error)

	// GetSectionEnrollment returns a live inscription by id. Soft-deleted rows return ErrNotFound.
	GetSectionEnrollment(ctx context.Context, id uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error)

	// ListSectionEnrollments returns live inscriptions matching the optional filter.
	ListSectionEnrollments(ctx context.Context, f ListSectionEnrollmentsFilter) ([]sectionenrollmentdb.SectionEnrollment, error)

	// ListOwnSectionEnrollments returns all live inscriptions for the given student.
	ListOwnSectionEnrollments(ctx context.Context, studentID uuid.UUID) ([]sectionenrollmentdb.SectionEnrollment, error)

	// GetOwnSectionEnrollment returns a live inscription by id without the ownership check
	// (ownership is enforced by the service). It is distinct from GetSectionEnrollment to
	// allow the service to apply scoping after the fetch.
	GetOwnSectionEnrollment(ctx context.Context, id uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error)
}

// EnrollSectionParams holds the validated inputs for EnrollSectionTx.
type EnrollSectionParams struct {
	// SectionID is the target section.
	SectionID uuid.UUID
	// EnrollmentID is the explicit enrollment_id (admin path). Ignored on the student
	// self-service path — the repository resolves the paid enrollment for the student.
	EnrollmentID uuid.UUID
	// StudentID is used only on the self-service path (isAdmin=false) to resolve the
	// paid enrollment by (student_id, program_id). Not used when isAdmin=true.
	StudentID uuid.UUID
	// ProgramID is required on the self-service path to identify which enrollment to link.
	// The student passes this explicitly to disambiguate when enrolled in multiple programs.
	ProgramID uuid.UUID
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
	q    sectionenrollmentdb.Querier
	pool *pgxpool.Pool
}

// Compile-time proof that *postgresRepository satisfies the Repository interface.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by the given sqlc Querier.
// pool is required by EnrollSectionTx, which opens a transaction.
func NewPostgresRepository(q sectionenrollmentdb.Querier, pool *pgxpool.Pool) Repository {
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
//  2. Lock section row FOR UPDATE; fetch capacity, course_id, and window_open (DB clock).
//  3. Window gate (self-enroll only): window_open evaluated by DB clock; null window → fail-closed.
//  4. Paid gate: enrollment must have status='paid'.
//  5. Course-in-program gate: section.course_id ∈ program_courses[enrollment.program_id].
//  6. Lock key row (enrollment_id, section_id) FOR UPDATE; detect existing/withdrawn/terminal.
//  7. COUNT active seats under the lock; n >= capacity → ErrSectionFull.
//  8. INSERT (fresh) or UPDATE (revival, admin only).
func (r *postgresRepository) EnrollSectionTx(ctx context.Context, p EnrollSectionParams, isAdmin bool) (sectionenrollmentdb.SectionEnrollment, error) {
	// Non-locking fast-fail pre-check: avoids queuing on the lock for obviously-full sections.
	// Run BEFORE BeginTx so that a full section is rejected without ever acquiring the row lock.
	// The under-lock count below is authoritative; this is a stale-tolerant optimisation only.
	// ErrNoRows here means the section doesn't exist — skip the pre-check and fall through to
	// the transaction, which will surface NotFound authoritatively.
	preCapRow, err := r.q.GetSectionCapacity(ctx, pgtype.UUID{Bytes: p.SectionID, Valid: true})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.ErrorContext(ctx, "section_enrollment: pre-check capacity read failed", "err", err)
			return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
		}
		// Section absent at pre-check: fall through to the tx, which will return NotFound.
	} else {
		preCount, err := r.q.CountActiveSeats(ctx, pgtype.UUID{Bytes: p.SectionID, Valid: true})
		if err != nil {
			slog.ErrorContext(ctx, "section_enrollment: pre-check count failed", "err", err)
			return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
		}
		if preCount >= int64(preCapRow.Capacity) {
			slog.WarnContext(ctx, "section enrollment rejected: section full (pre-check)",
				"section_id", p.SectionID,
				"capacity", preCapRow.Capacity,
				"active", preCount,
			)
			// TODO(metrics): increment section_full_total{path="pre_check"} when a
			// Prometheus/OTel pipeline is wired into internal/platform.
			return sectionenrollmentdb.SectionEnrollment{}, ErrSectionFull
		}
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := sectionenrollmentdb.New(tx)

	// 1. Set per-transaction lock timeout. This bounds how long the hot path can block
	//    waiting for the section row lock under a registration stampede.
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '2500ms'"); err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}

	// 2. Lock section row FOR UPDATE. Fetches capacity, course_id, period_year, and
	//    window_open (computed by the DB clock — avoids Go clock skew).
	lockStart := time.Now()
	sectionRow, err := q.GetSectionForUpdateWithWindow(ctx, pgtype.UUID{Bytes: p.SectionID, Valid: true})
	if err != nil {
		// 55P03: lock wait exceeded the SET LOCAL lock_timeout budget.
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "55P03" {
			slog.WarnContext(ctx, "section enrollment lock timeout",
				"section_id", p.SectionID,
			)
		}
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	slog.DebugContext(ctx, "section lock acquired",
		"section_id", p.SectionID,
		"wait_ms", time.Since(lockStart).Milliseconds(),
	)

	// 3. Window gate (self-enrollment only). The window_open boolean is evaluated by the
	//    database server clock, making this skew-free. NULL window columns → window_open=false
	//    (fail-closed). Admin path is never window-gated.
	if !isAdmin {
		if !sectionRow.WindowOpen.Valid || !sectionRow.WindowOpen.Bool {
			return sectionenrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: enrollment window is closed or not configured", ErrWindowClosed)
		}
	}

	// 4. Paid gate: resolve the linked enrollment and verify status='paid'.
	// Admin path: enrollment_id is supplied directly.
	// Student path: enrollment is resolved by (student_id, program_id, period_year) so
	//   the student explicitly identifies which program they are enrolling under, removing
	//   ambiguity when a student has paid enrollments in multiple programs.
	var enrollmentID pgtype.UUID
	var programID pgtype.UUID

	if isAdmin {
		eRow, err := q.ResolveEnrollmentByID(ctx, pgtype.UUID{Bytes: p.EnrollmentID, Valid: true})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return sectionenrollmentdb.SectionEnrollment{}, ErrNotFound
			}
			return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
		}
		if eRow.Status != "paid" {
			return sectionenrollmentdb.SectionEnrollment{}, ErrNotPaid
		}
		// Enforce annual-matrícula semantics: the enrollment's year must match the
		// section's academic period year. A matrícula is valid only within its own year.
		if eRow.Year != sectionRow.PeriodYear {
			return sectionenrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: enrollment year %d, period year %d",
				ErrEnrollmentYearMismatch, eRow.Year, sectionRow.PeriodYear)
		}
		enrollmentID = eRow.ID
		programID = eRow.ProgramID
	} else {
		// Self-service path: resolve the enrollment by (student_id, program_id, period_year).
		// The year is taken from the locked section row so the student cannot use a
		// matrícula from a different year to enroll in this section.
		eRow, err := q.ResolveEnrollmentByStudentAndProgram(ctx, sectionenrollmentdb.ResolveEnrollmentByStudentAndProgramParams{
			StudentID: pgtype.UUID{Bytes: p.StudentID, Valid: true},
			ProgramID: pgtype.UUID{Bytes: p.ProgramID, Valid: true},
			Year:      sectionRow.PeriodYear,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return sectionenrollmentdb.SectionEnrollment{}, ErrNotFound
			}
			return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
		}
		if eRow.Status != "paid" {
			return sectionenrollmentdb.SectionEnrollment{}, ErrNotPaid
		}
		enrollmentID = eRow.ID
		programID = eRow.ProgramID
	}

	// 5. Course-in-program gate.
	inProgram, err := q.CourseInProgram(ctx, sectionenrollmentdb.CourseInProgramParams{
		ProgramID: programID,
		CourseID:  sectionRow.CourseID,
	})
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	if !inProgram {
		return sectionenrollmentdb.SectionEnrollment{}, ErrCourseNotInProgram
	}

	// 6. Lock key row FOR UPDATE (revival detection). Only LIVE rows are returned by the query
	//    (deleted_at IS NULL filter) so a soft-deleted row never triggers AlreadyExists or revival.
	existingRow, keyErr := q.GetSectionEnrollmentByKeyForUpdate(ctx, sectionenrollmentdb.GetSectionEnrollmentByKeyForUpdateParams{
		EnrollmentID: enrollmentID,
		SectionID:    pgtype.UUID{Bytes: p.SectionID, Valid: true},
	})

	var isRevival bool
	switch {
	case keyErr == nil:
		switch existingRow.Status {
		case "in_progress":
			// Live active row — duplicate enrollment attempt.
			return sectionenrollmentdb.SectionEnrollment{}, ErrAlreadyExists
		case "withdrawn":
			if !isAdmin {
				// Students cannot self-revive a withdrawn inscription.
				return sectionenrollmentdb.SectionEnrollment{}, ErrWithdrawnNotRevivable
			}
			isRevival = true
		default:
			// passed or failed: a completed course may not be re-enrolled.
			return sectionenrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: inscription is in terminal status %q", ErrInvalidTransition, existingRow.Status)
		}
	case errors.Is(keyErr, pgx.ErrNoRows):
		// No existing live row — fresh insert path.
	default:
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(keyErr)
	}

	// 7. COUNT active seats under the lock. This is the authoritative check.
	activeCount, err := q.CountActiveSeats(ctx, pgtype.UUID{Bytes: p.SectionID, Valid: true})
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	if activeCount >= int64(sectionRow.Capacity) {
		slog.WarnContext(ctx, "section enrollment rejected: section full (under lock)",
			"section_id", p.SectionID,
			"capacity", sectionRow.Capacity,
			"active", activeCount,
		)
		// TODO(metrics): increment section_full_total{path="under_lock"} when a
		// Prometheus/OTel pipeline is wired into internal/platform.
		return sectionenrollmentdb.SectionEnrollment{}, ErrSectionFull
	}

	// 8. Insert (fresh) or revive (admin only).
	var result sectionenrollmentdb.SectionEnrollment
	if isRevival {
		result, err = q.ReviveSectionEnrollment(ctx, existingRow.ID)
	} else {
		result, err = q.InsertSectionEnrollment(ctx, sectionenrollmentdb.InsertSectionEnrollmentParams{
			EnrollmentID: enrollmentID,
			SectionID:    pgtype.UUID{Bytes: p.SectionID, Valid: true},
		})
	}
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	return result, nil
}

// WithdrawSection transitions an in_progress inscription to withdrawn.
// Returns ErrNotFound when the inscription is absent, already withdrawn, or soft-deleted.
func (r *postgresRepository) WithdrawSection(ctx context.Context, id uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	row, err := r.q.WithdrawSectionEnrollment(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	return row, nil
}

// GetSectionEnrollment returns a live inscription by id.
func (r *postgresRepository) GetSectionEnrollment(ctx context.Context, id uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	row, err := r.q.GetSectionEnrollmentByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	return row, nil
}

// GetOwnSectionEnrollment returns a live inscription by id (same as GetSectionEnrollment;
// ownership is enforced by the service after the fetch).
func (r *postgresRepository) GetOwnSectionEnrollment(ctx context.Context, id uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	return r.GetSectionEnrollment(ctx, id)
}

// ListSectionEnrollments returns live inscriptions filtered by optional criteria.
func (r *postgresRepository) ListSectionEnrollments(ctx context.Context, f ListSectionEnrollmentsFilter) ([]sectionenrollmentdb.SectionEnrollment, error) {
	params := sectionenrollmentdb.ListSectionEnrollmentsParams{}
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
func (r *postgresRepository) ListOwnSectionEnrollments(ctx context.Context, studentID uuid.UUID) ([]sectionenrollmentdb.SectionEnrollment, error) {
	rows, err := r.q.ListOwnSectionEnrollments(ctx, pgtype.UUID{Bytes: studentID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// SetSectionEnrollmentOutcomeTx transitions a section_enrollment to passed or failed within
// the caller-owned transaction tx. It executes the owned SetSectionEnrollmentOutcome query
// over the given tx using the WithTx pattern, preserving the single-tx atomicity boundary.
// Zero rows returned → ErrInvalidTransition (withdrawn source or invalid target).
func (r *postgresRepository) SetSectionEnrollmentOutcomeTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, outcome string, finalGrade pgtype.Numeric) (sectionenrollmentdb.SectionEnrollment, error) {
	q := sectionenrollmentdb.New(tx)
	return setOutcomeWithQuerier(ctx, q, id, outcome, finalGrade)
}

// setOutcomeWithQuerier is the querier-level implementation of the outcome transition.
// Separated from SetSectionEnrollmentOutcomeTx so that it can be unit-tested with a fake querier
// without requiring a real pgx.Tx.
func setOutcomeWithQuerier(ctx context.Context, q sectionenrollmentdb.Querier, id uuid.UUID, outcome string, finalGrade pgtype.Numeric) (sectionenrollmentdb.SectionEnrollment, error) {
	row, err := q.SetSectionEnrollmentOutcome(ctx, sectionenrollmentdb.SetSectionEnrollmentOutcomeParams{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		Status:     outcome,
		FinalGrade: finalGrade,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Zero rows: source is withdrawn, or target is not passed/failed.
			return sectionenrollmentdb.SectionEnrollment{}, fmt.Errorf("%w: invalid transition to %q or source is withdrawn", ErrInvalidTransition, outcome)
		}
		return sectionenrollmentdb.SectionEnrollment{}, TranslatePgError(err)
	}
	return row, nil
}

// newFakeRepository constructs a repository backed by the given Querier without a
// pool (the pool is only used for EnrollSectionTx, which opens a real transaction).
// Used exclusively in unit tests that exercise non-transactional repository methods.
func newFakeRepository(q sectionenrollmentdb.Querier) Repository {
	return &postgresRepository{q: q, pool: nil}
}
