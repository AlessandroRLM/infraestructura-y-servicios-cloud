package grades

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/grades/gradesdb"
	section_enrollmentdb "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/section_enrollment/section_enrollmentdb"
)

// outcomeSetter is the consumer-side interface consumed by postgresRepository.
// It is satisfied by section_enrollment.Service (or the section_enrollment.Repository
// directly), enabling the grades layer to participate in the same database transaction
// without importing the full section_enrollment surface.
type outcomeSetter interface {
	SetSectionEnrollmentOutcomeTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, outcome string, finalGrade pgtype.Numeric) (section_enrollmentdb.SectionEnrollment, error)
}

// CreateSchemeParams holds validated inputs for CreateEvaluationSchemeTx /
// RecreateEvaluationSchemeTx.
type CreateSchemeParams struct {
	CourseID uuid.UUID
	// Weights are the validated per-position weights (summing to 1.0).
	Weights []pgtype.Numeric
}

// RecordGradeParams holds validated inputs for RecordGradeTx.
type RecordGradeParams struct {
	EvaluationID        uuid.UUID
	SectionEnrollmentID uuid.UUID
	// Value is the grade value already validated in [1.0, 7.0] as a pgtype.Numeric.
	Value pgtype.Numeric
	// ExpectedVersion is nil for a first insert. On an existing grade (conflict on insert),
	// nil means ErrConflict is returned immediately.
	ExpectedVersion *int32
	// IsOverride indicates that this is an admin override (skips section_teachers check).
	IsOverride bool
	// ActorID is the authenticated user performing the write.
	ActorID *uuid.UUID
}

// RecordOutcome carries the optional outcome that was set on the section_enrollment
// after a successful grade write.
type RecordOutcome struct {
	// OutcomeSet is true when a weighted-final was computable and SetSectionEnrollmentOutcomeTx was called.
	OutcomeSet bool
	// Outcome is "passed" or "failed" when OutcomeSet is true.
	Outcome string
	// FinalGrade is the rounded weighted-final (1 decimal) when OutcomeSet is true.
	FinalGrade string
}

// Repository is the data-access contract for the grades slice.
type Repository interface {
	// CreateEvaluationSchemeTx creates the full evaluation scheme for a course atomically.
	// Returns ErrAlreadyExists if a live scheme exists for the course.
	CreateEvaluationSchemeTx(ctx context.Context, p CreateSchemeParams) ([]gradesdb.Evaluation, error)

	// RecreateEvaluationSchemeTx atomically replaces a course's evaluation scheme.
	// Returns ErrHasDependents if any grade references the current scheme.
	RecreateEvaluationSchemeTx(ctx context.Context, p CreateSchemeParams) ([]gradesdb.Evaluation, error)

	// ListEvaluations returns all live evaluations for a course ordered by position.
	ListEvaluations(ctx context.Context, courseID uuid.UUID) ([]gradesdb.Evaluation, error)

	// RecordGradeTx performs the atomic grade upsert, weighted-final computation,
	// and optional outcome transition within a single database transaction.
	RecordGradeTx(ctx context.Context, p RecordGradeParams) (gradesdb.Grade, RecordOutcome, error)

	// ListGradesForSection returns grades for all section_enrollments in a section.
	ListGradesForSection(ctx context.Context, sectionID uuid.UUID) ([]gradesdb.Grade, error)

	// GetGrade returns a single grade by id.
	GetGrade(ctx context.Context, id uuid.UUID) (gradesdb.Grade, error)

	// ListOwnGrades returns all grades for the given student.
	ListOwnGrades(ctx context.Context, studentID uuid.UUID) ([]gradesdb.Grade, error)

	// IsTeacherForSection returns true if userID is in section_teachers for sectionID.
	IsTeacherForSection(ctx context.Context, sectionID, userID uuid.UUID) (bool, error)
}

// postgresRepository is the production implementation.
type postgresRepository struct {
	q         gradesdb.Querier
	pool      *pgxpool.Pool
	seOutcome outcomeSetter
}

// Compile-time proof that *postgresRepository satisfies Repository.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository.
// q is used for non-transactional reads; pool opens transactions for Tx methods;
// seOutcome provides the mediated section_enrollment outcome write.
func NewPostgresRepository(q gradesdb.Querier, pool *pgxpool.Pool, seOutcome outcomeSetter) Repository {
	return &postgresRepository{q: q, pool: pool, seOutcome: seOutcome}
}

// CreateEvaluationSchemeTx inserts all evaluations for a course atomically.
// Fails if any live evaluation for the course already exists.
func (r *postgresRepository) CreateEvaluationSchemeTx(ctx context.Context, p CreateSchemeParams) ([]gradesdb.Evaluation, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := gradesdb.New(tx)

	// Check for an existing live scheme.
	count, err := q.CountLiveEvaluationsForCourse(ctx, pgtype.UUID{Bytes: p.CourseID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: course already has an evaluation scheme", ErrAlreadyExists)
	}

	evals := make([]gradesdb.Evaluation, 0, len(p.Weights))
	for i, w := range p.Weights {
		eval, err := q.InsertEvaluation(ctx, gradesdb.InsertEvaluationParams{
			CourseID: pgtype.UUID{Bytes: p.CourseID, Valid: true},
			Weight:   w,
			Position: int32(i + 1),
		})
		if err != nil {
			return nil, TranslatePgError(err)
		}
		evals = append(evals, eval)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, TranslatePgError(err)
	}
	return evals, nil
}

// RecreateEvaluationSchemeTx atomically soft-deletes the old scheme and inserts the new one.
// Returns ErrHasDependents if any grade references the current evaluations.
func (r *postgresRepository) RecreateEvaluationSchemeTx(ctx context.Context, p CreateSchemeParams) ([]gradesdb.Evaluation, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := gradesdb.New(tx)

	// Gate: reject if any grade references a live evaluation for the course.
	gradeCount, err := q.CountGradesForEvaluations(ctx, pgtype.UUID{Bytes: p.CourseID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	if gradeCount > 0 {
		return nil, fmt.Errorf("%w: course has %d grade(s) referencing the current scheme", ErrHasDependents, gradeCount)
	}

	// Soft-delete the existing scheme.
	if err := q.SoftDeleteEvaluationsForCourse(ctx, pgtype.UUID{Bytes: p.CourseID, Valid: true}); err != nil {
		return nil, TranslatePgError(err)
	}

	// Insert the replacement set.
	evals := make([]gradesdb.Evaluation, 0, len(p.Weights))
	for i, w := range p.Weights {
		eval, err := q.InsertEvaluation(ctx, gradesdb.InsertEvaluationParams{
			CourseID: pgtype.UUID{Bytes: p.CourseID, Valid: true},
			Weight:   w,
			Position: int32(i + 1),
		})
		if err != nil {
			return nil, TranslatePgError(err)
		}
		evals = append(evals, eval)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, TranslatePgError(err)
	}
	return evals, nil
}

// ListEvaluations returns live evaluations for a course ordered by position.
func (r *postgresRepository) ListEvaluations(ctx context.Context, courseID uuid.UUID) ([]gradesdb.Evaluation, error) {
	rows, err := r.q.ListEvaluationsForCourse(ctx, pgtype.UUID{Bytes: courseID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// RecordGradeTx performs the atomic grade upsert + weighted-final + outcome transition.
//
// Lock order:
//  1. SET LOCAL lock_timeout='2500ms'.
//  2. SELECT section_enrollments FOR UPDATE (serialization point).
//  3. Read live evaluations for the course.
//  4. Upsert the grade (INSERT ON CONFLICT DO NOTHING; detect conflict; UPDATE with version).
//  5. Read all sibling grades for the SE.
//  6. If all evaluations graded → compute weighted final → SetSectionEnrollmentOutcomeTx.
//  7. If grade value changed → insert audit_logs row.
//  8. Commit.
func (r *postgresRepository) RecordGradeTx(ctx context.Context, p RecordGradeParams) (gradesdb.Grade, RecordOutcome, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := gradesdb.New(tx)

	// 1. Lock timeout matches the enroll hot path.
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '2500ms'"); err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
	}

	// 2. Lock section_enrollment row FOR UPDATE; resolve course_id and student_id.
	seRow, err := q.GetSectionEnrollmentForGrade(ctx, pgtype.UUID{Bytes: p.SectionEnrollmentID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w: section_enrollment not found", ErrNotFound)
		}
		return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
	}

	// Guard: withdrawn section_enrollments are rejected before any write.
	if seRow.Status == "withdrawn" {
		return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w", ErrWithdrawnSource)
	}

	// Resource-level authz (teacher path only).
	if !p.IsOverride && p.ActorID != nil {
		actorPGUUID := pgtype.UUID{Bytes: *p.ActorID, Valid: true}

		// (b) Anti-self-grading: the teacher must not be the enrolled student.
		if seRow.StudentID == actorPGUUID {
			return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w", ErrSelfGrade)
		}

		// (c) section_teachers membership check.
		isTx := gradesdb.New(tx)
		isTeacher, err := isTx.IsTeacherForSection(ctx, gradesdb.IsTeacherForSectionParams{
			SectionID: seRow.SectionID,
			TeacherID: actorPGUUID,
		})
		if err != nil {
			return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
		}
		if !isTeacher {
			return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w", ErrNotSectionTeacher)
		}
	}

	// 3. Validate evaluation belongs to the same course as the section_enrollment.
	evals, err := q.ListEvaluationsForCourse(ctx, seRow.CourseID)
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
	}

	evalPGUUID := pgtype.UUID{Bytes: p.EvaluationID, Valid: true}
	var targetEval *gradesdb.Evaluation
	for i := range evals {
		if evals[i].ID == evalPGUUID {
			targetEval = &evals[i]
			break
		}
	}
	if targetEval == nil {
		// The evaluation is not in the live scheme for this course.
		// Distinguish between "does not exist" (NotFound) and "wrong course" (CourseMismatch).
		eval, errGet := r.q.GetEvaluationByID(ctx, evalPGUUID)
		if errors.Is(errGet, pgx.ErrNoRows) || !eval.ID.Valid {
			return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w: evaluation not found", ErrNotFound)
		}
		return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w: evaluation does not belong to the section's course", ErrCourseMismatch)
	}

	actorUUID := optionalUUID(p.ActorID)

	// 4. Upsert the grade.
	var resultGrade gradesdb.Grade
	var oldValue string // for audit log (empty on first insert)
	isUpdate := false

	inserted, err := q.InsertGrade(ctx, gradesdb.InsertGradeParams{
		EvaluationID:        pgtype.UUID{Bytes: p.EvaluationID, Valid: true},
		SectionEnrollmentID: pgtype.UUID{Bytes: p.SectionEnrollmentID, Valid: true},
		GradedBy:            actorUUID,
		Value:               p.Value,
		CreatedBy:           actorUUID,
		UpdatedBy:           actorUUID,
	})
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
	}

	if inserted.ID.Valid {
		// Successful insert (first write).
		resultGrade = inserted
	} else {
		// Conflict: grade already exists. Require expected_version.
		existing, err := q.GetGradeByKey(ctx, gradesdb.GetGradeByKeyParams{
			EvaluationID:        pgtype.UUID{Bytes: p.EvaluationID, Valid: true},
			SectionEnrollmentID: pgtype.UUID{Bytes: p.SectionEnrollmentID, Valid: true},
		})
		if err != nil {
			return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
		}

		if p.ExpectedVersion == nil {
			// No version supplied — conflict; client must re-fetch.
			return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w: current version is %d", ErrConflict, existing.Version)
		}

		updated, err := q.UpdateGradeByVersion(ctx, gradesdb.UpdateGradeByVersionParams{
			ID:        existing.ID,
			Version:   *p.ExpectedVersion,
			Value:     p.Value,
			GradedBy:  actorUUID,
			UpdatedBy: actorUUID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Version mismatch — re-read the current version for the detail.
				cur, _ := r.q.GetGradeByKey(ctx, gradesdb.GetGradeByKeyParams{
					EvaluationID:        pgtype.UUID{Bytes: p.EvaluationID, Valid: true},
					SectionEnrollmentID: pgtype.UUID{Bytes: p.SectionEnrollmentID, Valid: true},
				})
				return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w: current version is %d", ErrConflict, cur.Version)
			}
			return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
		}
		oldValue = numericToString(existing.Value)
		resultGrade = updated
		isUpdate = true
	}

	// 7. Audit log on value change (update path only).
	if isUpdate && oldValue != "" {
		newValue := numericToString(resultGrade.Value)
		detail, _ := json.Marshal(map[string]string{"old_value": oldValue, "new_value": newValue})
		if err := q.InsertAuditLog(ctx, gradesdb.InsertAuditLogParams{
			ActorID:  actorUUID,
			Action:   "grade.update",
			Entity:   "grades",
			EntityID: resultGrade.ID,
			Detail:   detail,
		}); err != nil {
			return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
		}
	}

	// 5. Read all sibling grades for the SE under the tx.
	siblings, err := q.ListGradesBySectionEnrollment(ctx, pgtype.UUID{Bytes: p.SectionEnrollmentID, Valid: true})
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
	}

	// 6. Compute weighted final if all evaluations are graded.
	var outcome RecordOutcome
	if int64(len(siblings)) == int64(len(evals)) {
		// Build the gradeWeight pairs for the computation.
		pairs, err := buildGradeWeightPairs(siblings, evals)
		if err != nil {
			return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("grades: %w", err)
		}

		rounded, passed := computeWeightedFinal(pairs)
		outcomeStr := "failed"
		if passed {
			outcomeStr = "passed"
		}

		finalGradeNum := stringToNumeric(rounded)
		_, err = r.seOutcome.SetSectionEnrollmentOutcomeTx(ctx, tx, p.SectionEnrollmentID, outcomeStr, finalGradeNum)
		if err != nil {
			return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("grades: outcome write: %w", err)
		}

		outcome = RecordOutcome{
			OutcomeSet: true,
			Outcome:    outcomeStr,
			FinalGrade: rounded,
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, TranslatePgError(err)
	}
	return resultGrade, outcome, nil
}

// ListGradesForSection returns grades for all section_enrollments in a section.
func (r *postgresRepository) ListGradesForSection(ctx context.Context, sectionID uuid.UUID) ([]gradesdb.Grade, error) {
	rows, err := r.q.ListGradesForSection(ctx, pgtype.UUID{Bytes: sectionID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// GetGrade returns a single grade by id.
func (r *postgresRepository) GetGrade(ctx context.Context, id uuid.UUID) (gradesdb.Grade, error) {
	row, err := r.q.GetGradeByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return gradesdb.Grade{}, TranslatePgError(err)
	}
	return row, nil
}

// ListOwnGrades returns all grades for the given student.
func (r *postgresRepository) ListOwnGrades(ctx context.Context, studentID uuid.UUID) ([]gradesdb.Grade, error) {
	rows, err := r.q.ListOwnGrades(ctx, pgtype.UUID{Bytes: studentID, Valid: true})
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// IsTeacherForSection returns true if userID is in section_teachers for sectionID.
func (r *postgresRepository) IsTeacherForSection(ctx context.Context, sectionID, userID uuid.UUID) (bool, error) {
	ok, err := r.q.IsTeacherForSection(ctx, gradesdb.IsTeacherForSectionParams{
		SectionID: pgtype.UUID{Bytes: sectionID, Valid: true},
		TeacherID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return false, TranslatePgError(err)
	}
	return ok, nil
}

// --- Helpers ---

// optionalUUID converts *uuid.UUID to pgtype.UUID (nil → invalid).
func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// gradeWeight pairs a grade value with its evaluation weight, both as *big.Rat.
type gradeWeight struct {
	value  *big.Rat
	weight *big.Rat
}

// buildGradeWeightPairs matches each grade to its evaluation weight.
// grades and evals must all share the same section_enrollment / course_id.
func buildGradeWeightPairs(grades []gradesdb.Grade, evals []gradesdb.Evaluation) ([]gradeWeight, error) {
	evalMap := make(map[pgtype.UUID]pgtype.Numeric, len(evals))
	for _, e := range evals {
		evalMap[e.ID] = e.Weight
	}

	pairs := make([]gradeWeight, 0, len(grades))
	for _, g := range grades {
		w, ok := evalMap[g.EvaluationID]
		if !ok {
			return nil, fmt.Errorf("evaluation %s not found in scheme", g.EvaluationID.Bytes)
		}
		vRat, err := numericToRat(g.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing grade value: %w", err)
		}
		wRat, err := numericToRat(w)
		if err != nil {
			return nil, fmt.Errorf("parsing evaluation weight: %w", err)
		}
		pairs = append(pairs, gradeWeight{value: vRat, weight: wRat})
	}
	return pairs, nil
}

// numericToRat converts a pgtype.Numeric to *big.Rat.
// Uses the text-encoding path via AppendFormat.
func numericToRat(n pgtype.Numeric) (*big.Rat, error) {
	if !n.Valid {
		return nil, fmt.Errorf("null numeric")
	}
	s := numericToString(n)
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("cannot parse numeric %q as rational", s)
	}
	return r, nil
}

// numericToString returns the decimal string representation of a pgtype.Numeric.
func numericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return ""
	}
	// pgtype.Numeric stores as Int (big.Int) with Exp (scale).
	// Build the string as Int × 10^Exp.
	if n.Int == nil {
		return "0"
	}
	rat := new(big.Rat).SetInt(n.Int)
	if n.Exp != 0 {
		exp := new(big.Rat).SetFloat64(1)
		ten := big.NewInt(10)
		if n.Exp > 0 {
			e := new(big.Int).Exp(ten, big.NewInt(int64(n.Exp)), nil)
			exp.SetInt(e)
			rat.Mul(rat, exp)
		} else {
			e := new(big.Int).Exp(ten, big.NewInt(int64(-n.Exp)), nil)
			exp.SetInt(e)
			rat.Quo(rat, exp)
		}
	}
	// Format as decimal string with sufficient precision.
	f, _ := rat.Float64()
	return fmt.Sprintf("%g", f)
}

// stringToNumeric converts a formatted decimal string (e.g. "4.9") to pgtype.Numeric.
func stringToNumeric(s string) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		return pgtype.Numeric{}
	}
	return n
}
