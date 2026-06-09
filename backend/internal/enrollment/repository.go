package enrollment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/enrollment/enrollmentdb"
)

// Repository provides data access for the enrollment domain.
type Repository interface {
	// CreateEnrollmentTx runs the full quota-checked create/revive in one transaction
	// with a FOR UPDATE lock on the program_quotas row. Returns ErrQuotaNotFound when no
	// live quota row exists, ErrQuotaFull when capacity is exhausted, and ErrAlreadyExists
	// when a live pending or paid enrollment already exists for the key.
	CreateEnrollmentTx(ctx context.Context, p CreateEnrollmentParams, actor *uuid.UUID) (enrollmentdb.Enrollment, error)

	// MarkEnrollmentPaid transitions status pending → paid and sets paid_at.
	// Returns ErrNotFound when absent and ErrInvalidTransition for any non-pending source state.
	MarkEnrollmentPaid(ctx context.Context, id uuid.UUID, actor *uuid.UUID) (enrollmentdb.Enrollment, error)

	// CancelEnrollment transitions status pending|paid → cancelled.
	// Returns ErrNotFound when absent and ErrInvalidTransition when already cancelled.
	CancelEnrollment(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error

	// GetEnrollment returns a live enrollment by id. Soft-deleted rows return ErrNotFound.
	GetEnrollment(ctx context.Context, id uuid.UUID) (enrollmentdb.Enrollment, error)

	// ListEnrollments returns live enrollments matching the optional filter.
	ListEnrollments(ctx context.Context, f ListEnrollmentsFilter) ([]enrollmentdb.Enrollment, error)

	// ListOwnEnrollments returns all live enrollments for the given student.
	ListOwnEnrollments(ctx context.Context, studentID uuid.UUID) ([]enrollmentdb.Enrollment, error)
}

// CreateEnrollmentParams holds the validated inputs for a new enrollment.
type CreateEnrollmentParams struct {
	StudentID uuid.UUID
	ProgramID uuid.UUID
	Year      int32
}

// ListEnrollmentsFilter holds optional filter parameters for ListEnrollments.
// A nil pointer means the filter is not applied.
type ListEnrollmentsFilter struct {
	StudentID *uuid.UUID
	ProgramID *uuid.UUID
	Year      *int32
	Status    *string
}

// postgresRepository is the production implementation backed by a sqlc Querier and a
// connection pool used exclusively by CreateEnrollmentTx.
type postgresRepository struct {
	q    enrollmentdb.Querier
	pool *pgxpool.Pool
}

// Compile-time proof that *postgresRepository satisfies the Repository interface.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by the given sqlc Querier.
// pool is required by CreateEnrollmentTx, which opens a transaction; pass the same
// pool that was used to create the Querier.
func NewPostgresRepository(q enrollmentdb.Querier, pool *pgxpool.Pool) Repository {
	return &postgresRepository{q: q, pool: pool}
}

// CreateEnrollmentTx runs the quota-checked create/revive in a single READ COMMITTED
// transaction. Lock order: program_quotas row first (FOR UPDATE), then the enrollment
// key row — this single, consistent order prevents deadlocks.
func (r *postgresRepository) CreateEnrollmentTx(ctx context.Context, p CreateEnrollmentParams, actor *uuid.UUID) (enrollmentdb.Enrollment, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return enrollmentdb.Enrollment{}, TranslatePgError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := enrollmentdb.New(tx)
	pgProgram := pgtype.UUID{Bytes: p.ProgramID, Valid: true}
	pgStudent := pgtype.UUID{Bytes: p.StudentID, Valid: true}

	// 1. Lock the program_quotas row. No live row → ErrQuotaNotFound (fail-closed).
	capacity, err := q.LockProgramQuotaForYear(ctx, enrollmentdb.LockProgramQuotaForYearParams{
		ProgramID: pgProgram,
		Year:      p.Year,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return enrollmentdb.Enrollment{}, ErrQuotaNotFound
	}
	if err != nil {
		return enrollmentdb.Enrollment{}, TranslatePgError(err)
	}

	// 2. Detect any existing row on the unique key (locked, any status including soft-deleted).
	existing, keyErr := q.GetEnrollmentByKeyForUpdate(ctx, enrollmentdb.GetEnrollmentByKeyForUpdateParams{
		StudentID: pgStudent,
		ProgramID: pgProgram,
		Year:      p.Year,
	})
	isRevival := false
	switch {
	case keyErr == nil:
		// Row exists. Live pending/paid → already enrolled.
		if existing.Status != "cancelled" && !existing.DeletedAt.Valid {
			return enrollmentdb.Enrollment{}, ErrAlreadyExists
		}
		// Cancelled or soft-deleted → revival candidate.
		isRevival = true
	case errors.Is(keyErr, pgx.ErrNoRows):
		// No row for this key — fresh insert path.
	default:
		return enrollmentdb.Enrollment{}, TranslatePgError(keyErr)
	}

	// 3. Count occupied seats under the lock. Revival also consumes a seat.
	n, err := q.CountActiveEnrollments(ctx, enrollmentdb.CountActiveEnrollmentsParams{
		ProgramID: pgProgram,
		Year:      p.Year,
	})
	if err != nil {
		return enrollmentdb.Enrollment{}, TranslatePgError(err)
	}
	if n >= int64(capacity) {
		return enrollmentdb.Enrollment{}, ErrQuotaFull
	}

	// 4. Insert or revive.
	var row enrollmentdb.Enrollment
	if isRevival {
		row, err = q.ReviveEnrollment(ctx, enrollmentdb.ReviveEnrollmentParams{
			ID:        existing.ID,
			UpdatedBy: optionalUUID(actor),
		})
	} else {
		row, err = q.InsertEnrollment(ctx, enrollmentdb.InsertEnrollmentParams{
			StudentID: pgStudent,
			ProgramID: pgProgram,
			Year:      p.Year,
			CreatedBy: optionalUUID(actor),
			UpdatedBy: optionalUUID(actor),
		})
	}
	if err != nil {
		return enrollmentdb.Enrollment{}, TranslatePgError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return enrollmentdb.Enrollment{}, TranslatePgError(err)
	}
	return row, nil
}

// MarkEnrollmentPaid transitions status pending → paid. When the UPDATE returns 0 rows
// (because the row is absent or not pending), a pre-fetch distinguishes the two error cases.
func (r *postgresRepository) MarkEnrollmentPaid(ctx context.Context, id uuid.UUID, actor *uuid.UUID) (enrollmentdb.Enrollment, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}

	row, err := r.q.MarkEnrollmentPaid(ctx, enrollmentdb.MarkEnrollmentPaidParams{
		ID:        pgID,
		UpdatedBy: optionalUUID(actor),
	})
	if err == nil {
		return row, nil
	}

	// 0-row update: distinguish absent from wrong-state via pre-fetch.
	if errors.Is(err, pgx.ErrNoRows) {
		existing, fetchErr := r.q.GetEnrollment(ctx, pgID)
		if fetchErr != nil {
			return enrollmentdb.Enrollment{}, TranslatePgError(fetchErr)
		}
		// Row exists but status is not pending → wrong-state rejection.
		_ = existing
		return enrollmentdb.Enrollment{}, fmt.Errorf("%w: current status is %q", ErrInvalidTransition, existing.Status)
	}
	return enrollmentdb.Enrollment{}, TranslatePgError(err)
}

// CancelEnrollment transitions status pending|paid → cancelled. When 0 rows are affected,
// a pre-fetch distinguishes absent from already-cancelled.
func (r *postgresRepository) CancelEnrollment(ctx context.Context, id uuid.UUID, actor *uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}

	n, err := r.q.CancelEnrollment(ctx, enrollmentdb.CancelEnrollmentParams{
		ID:        pgID,
		UpdatedBy: optionalUUID(actor),
	})
	if err != nil {
		return TranslatePgError(err)
	}
	if n > 0 {
		return nil
	}

	// 0 rows affected: distinguish absent from already-cancelled via pre-fetch.
	existing, fetchErr := r.q.GetEnrollment(ctx, pgID)
	if fetchErr != nil {
		return TranslatePgError(fetchErr)
	}
	// Row exists with status 'cancelled' → invalid transition.
	_ = existing
	return fmt.Errorf("%w: current status is %q", ErrInvalidTransition, existing.Status)
}

// GetEnrollment returns a live enrollment by id. Soft-deleted rows return ErrNotFound.
func (r *postgresRepository) GetEnrollment(ctx context.Context, id uuid.UUID) (enrollmentdb.Enrollment, error) {
	row, err := r.q.GetEnrollment(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return enrollmentdb.Enrollment{}, TranslatePgError(err)
	}
	return row, nil
}

// ListEnrollments returns live enrollments filtered by optional criteria.
func (r *postgresRepository) ListEnrollments(ctx context.Context, f ListEnrollmentsFilter) ([]enrollmentdb.Enrollment, error) {
	params := enrollmentdb.ListEnrollmentsParams{}
	if f.StudentID != nil {
		params.StudentID = pgtype.UUID{Bytes: *f.StudentID, Valid: true}
	}
	if f.ProgramID != nil {
		params.ProgramID = pgtype.UUID{Bytes: *f.ProgramID, Valid: true}
	}
	if f.Year != nil {
		params.Year = pgtype.Int4{Int32: *f.Year, Valid: true}
	}
	if f.Status != nil {
		params.Status = pgtype.Text{String: *f.Status, Valid: true}
	}
	rows, err := r.q.ListEnrollments(ctx, params)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// ListOwnEnrollments returns all live enrollments for the given student.
func (r *postgresRepository) ListOwnEnrollments(ctx context.Context, studentID uuid.UUID) ([]enrollmentdb.Enrollment, error) {
	rows, err := r.q.ListOwnEnrollments(ctx, pgtype.UUID{Bytes: studentID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// optionalUUID converts a *uuid.UUID to pgtype.UUID, returning an invalid (null) value
// when the pointer is nil (system or background operations with no actor).
func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}
