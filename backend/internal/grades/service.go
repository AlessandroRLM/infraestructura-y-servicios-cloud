package grades

import (
	"context"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/grades/gradesdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/pagination"
)

const (
	// gradesPageSizeMin is the minimum effective page size for paginated grade list operations.
	gradesPageSizeMin = 20
	// gradesPageSizeMax is the maximum effective page size for paginated grade list operations.
	gradesPageSizeMax = 200
)

// gradesClamp is the shared page-size clamp for grades list operations.
var gradesClamp = pagination.Clamp{Min: gradesPageSizeMin, Max: gradesPageSizeMax}

// ListGradesForSectionResult holds the paginated result for ListGradesForSection.
type ListGradesForSectionResult struct {
	Grades        []gradesdb.Grade
	NextPageToken string
}

// ListOwnGradesResult holds the paginated result for ListOwnGrades.
type ListOwnGradesResult struct {
	Grades        []gradesdb.Grade
	NextPageToken string
}

// Service orchestrates grades business logic: UUID validation, weight-sum enforcement,
// resource-level authz checks, and delegation to the Repository.
//
// Decimal arithmetic strategy (ADR-4):
// All weighted-final computations use *big.Rat (stdlib, no external dependency).
// Grade values and evaluation weights arrive as pgtype.Numeric; they are parsed to
// *big.Rat, accumulated, and rounded HALF-UP to 1 decimal before the 4.0 threshold
// comparison. Float64 is never used for the weighted sum — it would drift at the
// 4.0 boundary and break correctness for values like 3.95 → 4.0.
//
// Rounding algorithm (round-half-up):
//  1. Multiply sum by 10.
//  2. Take floor (truncate) + add 1 if remainder >= 0.5 (i.e. multiply by 2, add 1 if ≥ 10).
//  3. Divide by 10.
//
// Result compared >= 4.0 → "passed"; < 4.0 → "failed".
//
// Dependency direction: internal/grades → internal/sectionenrollment. Never the reverse.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the given Repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateEvaluationScheme validates inputs and creates the course's evaluation scheme.
func (s *Service) CreateEvaluationScheme(ctx context.Context, courseIDStr string, weightStrs []string) ([]gradesdb.Evaluation, error) {
	courseID, err := parseServiceUUID(courseIDStr)
	if err != nil {
		return nil, err
	}

	weights, err := validateWeights(weightStrs)
	if err != nil {
		return nil, err
	}

	return s.repo.CreateEvaluationSchemeTx(ctx, CreateSchemeParams{
		CourseID: courseID,
		Weights:  weights,
	})
}

// RecreateEvaluationScheme validates inputs and replaces the course's evaluation scheme.
func (s *Service) RecreateEvaluationScheme(ctx context.Context, courseIDStr string, weightStrs []string) ([]gradesdb.Evaluation, error) {
	courseID, err := parseServiceUUID(courseIDStr)
	if err != nil {
		return nil, err
	}

	weights, err := validateWeights(weightStrs)
	if err != nil {
		return nil, err
	}

	return s.repo.RecreateEvaluationSchemeTx(ctx, CreateSchemeParams{
		CourseID: courseID,
		Weights:  weights,
	})
}

// ListEvaluations returns all live evaluations for a course.
func (s *Service) ListEvaluations(ctx context.Context, courseIDStr string) ([]gradesdb.Evaluation, error) {
	courseID, err := parseServiceUUID(courseIDStr)
	if err != nil {
		return nil, err
	}
	return s.repo.ListEvaluations(ctx, courseID)
}

// RecordGrade validates inputs, enforces resource-level authz, and records or corrects
// a grade.
func (s *Service) RecordGrade(ctx context.Context, evalIDStr, seIDStr, valueStr string, expectedVersion *int32) (gradesdb.Grade, RecordOutcome, error) {
	evalID, err := parseServiceUUID(evalIDStr)
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, err
	}
	seID, err := parseServiceUUID(seIDStr)
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, err
	}

	value, err := parseGradeValue(valueStr)
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, err
	}

	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w: no authenticated user", ErrNotFound)
	}

	// Resource-level authz (a): verify section_teachers membership.
	// We need the section_id for the SE; this is resolved inside RecordGradeTx
	// after the FOR UPDATE lock. To avoid a double-read, do the teacher check
	// inside the transaction flow via the repo's IsTeacherForSection helper.
	// The check is done separately here for clarity before the expensive tx.
	// Note: the section_id must be known; we could check after locking, but a
	// pre-check reduces lock contention on invalid authz. For now, delegate the
	// authz guard to the repo layer which has the SE row under lock.

	actor := callerID
	return s.repo.RecordGradeTx(ctx, RecordGradeParams{
		EvaluationID:        evalID,
		SectionEnrollmentID: seID,
		Value:               value,
		ExpectedVersion:     expectedVersion,
		IsOverride:          false,
		ActorID:             &actor,
	})
}

// OverrideGrade validates inputs and records or corrects a grade as an admin (no
// section_teachers check).
func (s *Service) OverrideGrade(ctx context.Context, evalIDStr, seIDStr, valueStr string, expectedVersion *int32) (gradesdb.Grade, RecordOutcome, error) {
	evalID, err := parseServiceUUID(evalIDStr)
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, err
	}
	seID, err := parseServiceUUID(seIDStr)
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, err
	}

	value, err := parseGradeValue(valueStr)
	if err != nil {
		return gradesdb.Grade{}, RecordOutcome{}, err
	}

	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return gradesdb.Grade{}, RecordOutcome{}, fmt.Errorf("%w: no authenticated user", ErrNotFound)
	}

	actor := callerID
	return s.repo.RecordGradeTx(ctx, RecordGradeParams{
		EvaluationID:        evalID,
		SectionEnrollmentID: seID,
		Value:               value,
		ExpectedVersion:     expectedVersion,
		IsOverride:          true,
		ActorID:             &actor,
	})
}

// ListGradesForSection returns a paginated page of grades scoped to a section.
// Admin callers (grades.override) receive all grades for the section.
// Teacher callers (grades.read only) receive only grades from sections where they are
// in section_teachers; out-of-scope sections return an empty list (anti-leak intact).
// pageSize is clamped to [20, 200]. pageToken must be a valid UUID string or empty.
func (s *Service) ListGradesForSection(ctx context.Context, sectionIDStr string, pageSize int32, pageToken string) (ListGradesForSectionResult, error) {
	sectionID, err := parseServiceUUID(sectionIDStr)
	if err != nil {
		return ListGradesForSectionResult{}, err
	}

	clamped := gradesClamp.Apply(pageSize)

	var tokenUUID *uuid.UUID
	if pageToken != "" {
		id, err := uuid.Parse(pageToken)
		if err != nil {
			return ListGradesForSectionResult{}, fmt.Errorf("%w: page_token is not a valid UUID: %q", ErrInvalidInput, pageToken)
		}
		tokenUUID = &id
	}

	var rows []gradesdb.Grade
	if callerIsAdmin(ctx) {
		rows, err = s.repo.ListGradesForSectionPaged(ctx, ListGradesForSectionRepoParams{
			SectionID: sectionID,
			PageToken: tokenUUID,
			RowLimit:  int32(clamped + 1),
		})
	} else {
		callerID, ok := auth.UserIDFromContext(ctx)
		if !ok {
			return ListGradesForSectionResult{}, fmt.Errorf("%w: no authenticated user", ErrNotFound)
		}
		rows, err = s.repo.ListGradesForSectionByTeacherPaged(ctx, ListGradesForSectionByTeacherRepoParams{
			SectionID: sectionID,
			TeacherID: callerID,
			PageToken: tokenUUID,
			RowLimit:  int32(clamped + 1),
		})
	}
	if err != nil {
		return ListGradesForSectionResult{}, err
	}

	page := pagination.Paginate(rows, clamped)
	nextToken := pagination.TokenOf(page, func(g gradesdb.Grade) uuid.UUID {
		return uuid.UUID(g.ID.Bytes)
	})

	return ListGradesForSectionResult{
		Grades:        page.Items,
		NextPageToken: nextToken,
	}, nil
}

// GetGrade returns a grade by id. Admin callers (grades.override) can fetch any grade.
// Teacher callers (grades.read only) are scoped to sections where they are in
// section_teachers; out-of-scope grades return ErrNotFound (never PermissionDenied).
func (s *Service) GetGrade(ctx context.Context, idStr string) (gradesdb.Grade, error) {
	id, err := parseServiceUUID(idStr)
	if err != nil {
		return gradesdb.Grade{}, err
	}

	if callerIsAdmin(ctx) {
		return s.repo.GetGrade(ctx, id)
	}

	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return gradesdb.Grade{}, fmt.Errorf("%w: no authenticated user", ErrNotFound)
	}
	return s.repo.GetGradeByIDForTeacher(ctx, id, callerID)
}

// callerIsAdmin returns true when the authenticated caller holds the grades.override
// permission, which identifies admin-level access. This determines whether read
// operations are unscoped (admin) or restricted to the caller's sections (teacher).
func callerIsAdmin(ctx context.Context) bool {
	perms, ok := authz.PermissionsFromContext(ctx)
	if !ok {
		return false
	}
	return perms.Has(authz.PermGradesOverride)
}

// ListOwnGrades returns a paginated page of grades for the authenticated student.
// pageSize is clamped to [20, 200]. pageToken must be a valid UUID string or empty.
// Student identity is derived exclusively from the context; no student_id in the request.
func (s *Service) ListOwnGrades(ctx context.Context, pageSize int32, pageToken string) (ListOwnGradesResult, error) {
	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return ListOwnGradesResult{}, fmt.Errorf("%w: no authenticated user", ErrNotFound)
	}

	clamped := gradesClamp.Apply(pageSize)

	var tokenUUID *uuid.UUID
	if pageToken != "" {
		id, err := uuid.Parse(pageToken)
		if err != nil {
			return ListOwnGradesResult{}, fmt.Errorf("%w: page_token is not a valid UUID: %q", ErrInvalidInput, pageToken)
		}
		tokenUUID = &id
	}

	rows, err := s.repo.ListOwnGradesPaged(ctx, ListOwnGradesRepoParams{
		StudentID: callerID,
		PageToken: tokenUUID,
		RowLimit:  int32(clamped + 1),
	})
	if err != nil {
		return ListOwnGradesResult{}, err
	}

	page := pagination.Paginate(rows, clamped)
	nextToken := pagination.TokenOf(page, func(g gradesdb.Grade) uuid.UUID {
		return uuid.UUID(g.ID.Bytes)
	})

	return ListOwnGradesResult{
		Grades:        page.Items,
		NextPageToken: nextToken,
	}, nil
}

// --- Arithmetic ---

// computeWeightedFinal computes the weighted sum of the provided grade/weight pairs
// using exact *big.Rat arithmetic, rounds HALF-UP to 1 decimal, and returns the
// formatted result and whether it meets the 4.0 passing threshold.
//
// Round-half-up: multiply sum by 10, add 5, floor-divide by 10 to get the rounded
// integer (×10), then format with one decimal place.
func computeWeightedFinal(pairs []gradeWeight) (rounded string, passed bool) {
	sum := new(big.Rat)
	for _, p := range pairs {
		product := new(big.Rat).Mul(p.value, p.weight)
		sum.Add(sum, product)
	}

	// Round HALF-UP to 1 decimal:
	// scaled = floor((sum * 10) + 0.5)  →  one-decimal integer
	ten := new(big.Rat).SetInt64(10)
	half := new(big.Rat).SetFrac(big.NewInt(1), big.NewInt(2))

	scaled := new(big.Rat).Mul(sum, ten)
	scaled.Add(scaled, half)

	// Floor: take numerator / denominator, truncate.
	num := scaled.Num()
	den := scaled.Denom()
	roundedInt := new(big.Int).Quo(num, den) // integer division truncates toward zero (positive → floor)

	// Build the final decimal string "X.Y".
	intPart := new(big.Int).Quo(roundedInt, big.NewInt(10))
	fracPart := new(big.Int).Mod(roundedInt, big.NewInt(10))
	if fracPart.Sign() < 0 {
		fracPart.Neg(fracPart)
	}
	rounded = fmt.Sprintf("%d.%d", intPart, fracPart)

	// Compare rounded value against 4.0 threshold using *big.Rat.
	roundedRat := new(big.Rat).SetFrac(roundedInt, big.NewInt(10))
	threshold := new(big.Rat).SetFrac(big.NewInt(4), big.NewInt(1))
	passed = roundedRat.Cmp(threshold) >= 0

	return rounded, passed
}

// --- Validation helpers ---

// validateWeights parses weight strings, validates each is in (0, 1.0], and verifies
// the sum equals exactly 1.0. Returns the weights as pgtype.Numeric values.
func validateWeights(weightStrs []string) ([]pgtype.Numeric, error) {
	if len(weightStrs) == 0 {
		return nil, fmt.Errorf("%w: at least one evaluation is required", ErrSchemeIncomplete)
	}

	sumRat := new(big.Rat)
	weights := make([]pgtype.Numeric, 0, len(weightStrs))

	for i, ws := range weightStrs {
		r, ok := new(big.Rat).SetString(ws)
		if !ok {
			return nil, fmt.Errorf("%w: weight[%d] %q is not a valid decimal", ErrInvalidInput, i, ws)
		}
		zero := new(big.Rat)
		one := new(big.Rat).SetInt64(1)
		if r.Cmp(zero) <= 0 || r.Cmp(one) > 0 {
			return nil, fmt.Errorf("%w: weight[%d] must be in (0, 1.0], got %s", ErrInvalidInput, i, ws)
		}
		sumRat.Add(sumRat, r)

		var n pgtype.Numeric
		if err := n.Scan(ws); err != nil {
			return nil, fmt.Errorf("%w: weight[%d] %q cannot be stored as NUMERIC: %v", ErrInvalidInput, i, ws, err)
		}
		weights = append(weights, n)
	}

	// Verify exact sum == 1.0.
	one := new(big.Rat).SetInt64(1)
	if sumRat.Cmp(one) != 0 {
		return nil, fmt.Errorf("%w: weights sum to %s, must equal 1.0", ErrSchemeIncomplete, sumRat.FloatString(6))
	}

	return weights, nil
}

// parseGradeValue parses a decimal string grade value and validates it is in [1.0, 7.0].
func parseGradeValue(s string) (pgtype.Numeric, error) {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return pgtype.Numeric{}, fmt.Errorf("%w: value %q is not a valid decimal", ErrInvalidInput, s)
	}
	lo := new(big.Rat).SetFrac(big.NewInt(10), big.NewInt(10)) // 1.0
	hi := new(big.Rat).SetFrac(big.NewInt(70), big.NewInt(10)) // 7.0
	if r.Cmp(lo) < 0 || r.Cmp(hi) > 0 {
		return pgtype.Numeric{}, fmt.Errorf("%w: value %s is outside [1.0, 7.0]", ErrInvalidInput, s)
	}
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	return n, nil
}

// parseServiceUUID parses a string UUID and returns ErrInvalidInput on failure.
func parseServiceUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%w: invalid id %q", ErrInvalidInput, s)
	}
	return id, nil
}
