package grades

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/grades/gradesdb"
)

// --- helpers ---

func rat(s string) *big.Rat {
	r, _ := new(big.Rat).SetString(s)
	return r
}

func gw(value, weight string) gradeWeight {
	return gradeWeight{value: rat(value), weight: rat(weight)}
}

func contextWithUser(userID uuid.UUID) context.Context {
	return auth.WithUserID(context.Background(), userID)
}

// --- fakeRepository ---

// fakeRepository implements Repository for service unit tests.
// All methods are no-ops unless configured; sentinel fields record whether each
// method was called so tests can assert delegation.
type fakeRepository struct {
	createSchemeCalled bool
	createSchemeEvals  []gradesdb.Evaluation
	createSchemeErr    error

	recreateSchemeCalled bool
	recreateSchemeEvals  []gradesdb.Evaluation
	recreateSchemeErr    error

	listEvalsCalled bool
	listEvalsRows   []gradesdb.Evaluation
	listEvalsErr    error

	recordGradeCalled  bool
	recordGradeResult  gradesdb.Grade
	recordGradeOutcome RecordOutcome
	recordGradeErr     error

	listForSectionCalled bool
	listForSectionRows   []gradesdb.Grade
	listForSectionErr    error

	listForSectionByTeacherCalled bool
	listForSectionByTeacherRows   []gradesdb.Grade
	listForSectionByTeacherErr    error

	getGradeCalled bool
	getGradeResult gradesdb.Grade
	getGradeErr    error

	getGradeForTeacherCalled bool
	getGradeForTeacherResult gradesdb.Grade
	getGradeForTeacherErr    error

	listOwnCalled bool
	listOwnRows   []gradesdb.Grade
	listOwnErr    error

	isTeacherCalled bool
	isTeacherResult bool
	isTeacherErr    error
}

func (f *fakeRepository) CreateEvaluationSchemeTx(_ context.Context, _ CreateSchemeParams) ([]gradesdb.Evaluation, error) {
	f.createSchemeCalled = true
	return f.createSchemeEvals, f.createSchemeErr
}

func (f *fakeRepository) RecreateEvaluationSchemeTx(_ context.Context, _ CreateSchemeParams) ([]gradesdb.Evaluation, error) {
	f.recreateSchemeCalled = true
	return f.recreateSchemeEvals, f.recreateSchemeErr
}

func (f *fakeRepository) ListEvaluations(_ context.Context, _ uuid.UUID) ([]gradesdb.Evaluation, error) {
	f.listEvalsCalled = true
	return f.listEvalsRows, f.listEvalsErr
}

func (f *fakeRepository) RecordGradeTx(_ context.Context, _ RecordGradeParams) (gradesdb.Grade, RecordOutcome, error) {
	f.recordGradeCalled = true
	return f.recordGradeResult, f.recordGradeOutcome, f.recordGradeErr
}

func (f *fakeRepository) ListGradesForSection(_ context.Context, _ uuid.UUID) ([]gradesdb.Grade, error) {
	f.listForSectionCalled = true
	return f.listForSectionRows, f.listForSectionErr
}

func (f *fakeRepository) ListGradesForSectionByTeacher(_ context.Context, _, _ uuid.UUID) ([]gradesdb.Grade, error) {
	f.listForSectionByTeacherCalled = true
	return f.listForSectionByTeacherRows, f.listForSectionByTeacherErr
}

func (f *fakeRepository) GetGrade(_ context.Context, _ uuid.UUID) (gradesdb.Grade, error) {
	f.getGradeCalled = true
	return f.getGradeResult, f.getGradeErr
}

func (f *fakeRepository) GetGradeByIDForTeacher(_ context.Context, _, _ uuid.UUID) (gradesdb.Grade, error) {
	f.getGradeForTeacherCalled = true
	return f.getGradeForTeacherResult, f.getGradeForTeacherErr
}

func (f *fakeRepository) ListOwnGrades(_ context.Context, _ uuid.UUID) ([]gradesdb.Grade, error) {
	f.listOwnCalled = true
	return f.listOwnRows, f.listOwnErr
}

func (f *fakeRepository) IsTeacherForSection(_ context.Context, _, _ uuid.UUID) (bool, error) {
	f.isTeacherCalled = true
	return f.isTeacherResult, f.isTeacherErr
}

// =====================================================================================
// computeWeightedFinal — table-driven *big.Rat exactness tests
// =====================================================================================

func TestComputeWeightedFinal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pairs       []gradeWeight
		wantRounded string
		wantPassed  bool
	}{
		{
			name:        "single weight 1.0 grade 5.0",
			pairs:       []gradeWeight{gw("5.0", "1.0")},
			wantRounded: "5.0",
			wantPassed:  true,
		},
		{
			name:        "boundary 4.0/4.0/4.0 with 0.40/0.30/0.30 scheme",
			pairs:       []gradeWeight{gw("4.0", "0.40"), gw("4.0", "0.30"), gw("4.0", "0.30")},
			wantRounded: "4.0",
			wantPassed:  true, // 4.0 >= 4.0 → passed (inclusive)
		},
		{
			name:        "3.95 rounds to 4.0 → passed (half-up)",
			pairs:       []gradeWeight{gw("5.0", "0.40"), gw("4.5", "0.30"), gw("2.0", "0.30")},
			wantRounded: "4.0",
			wantPassed:  true,
		},
		{
			name:        "3.88 rounds to 3.9 → failed",
			pairs:       []gradeWeight{gw("4.6", "0.40"), gw("3.8", "0.30"), gw("3.0", "0.30")},
			wantRounded: "3.9",
			wantPassed:  false,
		},
		{
			name:        "3.94 rounds to 3.9 → failed (does NOT round to 4.0)",
			pairs:       []gradeWeight{gw("4.9", "0.40"), gw("3.5", "0.30"), gw("3.0", "0.30")},
			wantRounded: "3.9",
			wantPassed:  false,
		},
		{
			name:        "4.05 rounds to 4.1 → passed",
			pairs:       []gradeWeight{gw("4.125", "1.0")},
			wantRounded: "4.1",
			wantPassed:  true,
		},
		{
			name:        "threshold inclusivity exactly 4.0 → passed",
			pairs:       []gradeWeight{gw("4.0", "1.0")},
			wantRounded: "4.0",
			wantPassed:  true,
		},
		{
			name:        "just below threshold 3.9 → failed",
			pairs:       []gradeWeight{gw("3.94", "1.0")},
			wantRounded: "3.9",
			wantPassed:  false,
		},
		{
			name:        "all graded 5.0 / 4.5 / 5.0 with 0.40/0.30/0.30 → 4.85 rounds 4.9 → passed",
			pairs:       []gradeWeight{gw("5.0", "0.40"), gw("4.5", "0.30"), gw("5.0", "0.30")},
			wantRounded: "4.9",
			wantPassed:  true,
		},
		{
			name:        "all graded 3.0 / 2.5 / 3.5 with 0.40/0.30/0.30 → 3.0 → failed",
			pairs:       []gradeWeight{gw("3.0", "0.40"), gw("2.5", "0.30"), gw("3.5", "0.30")},
			wantRounded: "3.0",
			wantPassed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotRounded, gotPassed := computeWeightedFinal(tt.pairs)
			if gotRounded != tt.wantRounded {
				t.Errorf("computeWeightedFinal() rounded = %q, want %q", gotRounded, tt.wantRounded)
			}
			if gotPassed != tt.wantPassed {
				t.Errorf("computeWeightedFinal() passed = %v, want %v", gotPassed, tt.wantPassed)
			}
		})
	}
}

// =====================================================================================
// validateWeights — exact big.Rat sum == 1.0, no epsilon
// =====================================================================================

func TestValidateWeights(t *testing.T) {
	t.Parallel()

	t.Run("valid scheme 0.40/0.30/0.30", func(t *testing.T) {
		t.Parallel()
		weights, err := validateWeights([]string{"0.40", "0.30", "0.30"})
		if err != nil {
			t.Fatalf("validateWeights: unexpected error %v", err)
		}
		if len(weights) != 3 {
			t.Errorf("len(weights) = %d, want 3", len(weights))
		}
	})

	t.Run("single evaluation weight 1.0 is valid (thesis case)", func(t *testing.T) {
		t.Parallel()
		weights, err := validateWeights([]string{"1.0"})
		if err != nil {
			t.Fatalf("validateWeights: unexpected error %v", err)
		}
		if len(weights) != 1 {
			t.Errorf("len(weights) = %d, want 1", len(weights))
		}
	})

	t.Run("sum 0.999 rejected — no epsilon tolerance", func(t *testing.T) {
		t.Parallel()
		_, err := validateWeights([]string{"0.333", "0.333", "0.333"})
		if !errors.Is(err, ErrSchemeIncomplete) {
			t.Errorf("validateWeights(sum 0.999) = %v; want ErrSchemeIncomplete", err)
		}
	})

	t.Run("sum 1.001 rejected", func(t *testing.T) {
		t.Parallel()
		_, err := validateWeights([]string{"0.40", "0.30", "0.301"})
		if !errors.Is(err, ErrSchemeIncomplete) {
			t.Errorf("validateWeights(sum 1.001) = %v; want ErrSchemeIncomplete", err)
		}
	})

	t.Run("weight 0.0 rejected — must be > 0", func(t *testing.T) {
		t.Parallel()
		_, err := validateWeights([]string{"0.0", "1.0"})
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("validateWeights(weight 0) = %v; want ErrInvalidInput", err)
		}
	})

	t.Run("weight > 1.0 rejected individually", func(t *testing.T) {
		t.Parallel()
		_, err := validateWeights([]string{"1.5"})
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("validateWeights(weight 1.5) = %v; want ErrInvalidInput", err)
		}
	})

	t.Run("empty weight list rejected", func(t *testing.T) {
		t.Parallel()
		_, err := validateWeights([]string{})
		if !errors.Is(err, ErrSchemeIncomplete) {
			t.Errorf("validateWeights(empty) = %v; want ErrSchemeIncomplete", err)
		}
	})

	t.Run("non-decimal string rejected", func(t *testing.T) {
		t.Parallel()
		_, err := validateWeights([]string{"abc"})
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("validateWeights(non-decimal) = %v; want ErrInvalidInput", err)
		}
	})
}

// =====================================================================================
// Service delegation — fake Repository with called bool sentinels
// =====================================================================================

func TestService_RecordGrade_DelegatesAndPropagatesError(t *testing.T) {
	t.Parallel()

	injectedErr := errors.New("repo failure")
	repo := &fakeRepository{recordGradeErr: injectedErr}
	svc := NewService(repo)

	callerID := uuid.New()
	ctx := contextWithUser(callerID)

	evalID := uuid.New().String()
	seID := uuid.New().String()

	_, _, err := svc.RecordGrade(ctx, evalID, seID, "5.0", nil)
	if !repo.recordGradeCalled {
		t.Fatal("RecordGradeTx was not called on the repository")
	}
	if !errors.Is(err, injectedErr) {
		t.Errorf("RecordGrade error = %v; want wrapped %v", err, injectedErr)
	}
}

func TestService_RecordGrade_NoContextUser_DoesNotCallRepo(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, _, err := svc.RecordGrade(context.Background(), uuid.New().String(), uuid.New().String(), "5.0", nil)
	if repo.recordGradeCalled {
		t.Error("RecordGradeTx must not be called when user is absent from context")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("RecordGrade(no user) = %v; want ErrNotFound", err)
	}
}

func TestService_RecordGrade_BadEvalID_DoesNotCallRepo(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, _, err := svc.RecordGrade(ctx, "not-a-uuid", uuid.New().String(), "5.0", nil)
	if repo.recordGradeCalled {
		t.Error("RecordGradeTx must not be called for an invalid evaluation ID")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("RecordGrade(bad eval UUID) = %v; want ErrInvalidInput", err)
	}
}

func TestService_RecordGrade_BadValue_DoesNotCallRepo(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, _, err := svc.RecordGrade(ctx, uuid.New().String(), uuid.New().String(), "7.5", nil)
	if repo.recordGradeCalled {
		t.Error("RecordGradeTx must not be called for an out-of-range grade value")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("RecordGrade(value 7.5) = %v; want ErrInvalidInput", err)
	}
}

// TestService_OutcomeSetter_ErrorCausesGradeRollback verifies that when the outcomeSetter
// returns an error, the grade write is not committed: the service surfaces the error and
// the repository is responsible for rolling back the transaction. We verify this at the
// service boundary by checking that the error propagates when the injected outcomeSetter fails.
// (The integration atomicity test verifies zero rows in the DB for the withdrawn-SE path;
// here we test the error-propagation contract via a fake repo wired to return the error.)
func TestService_OutcomeSetter_ErrorPropagates(t *testing.T) {
	t.Parallel()

	outcomeErr := errors.New("outcome write failed")
	// Repository configured to return the outcomeSetter error so we can verify propagation.
	repo := &fakeRepository{
		recordGradeErr: outcomeErr,
	}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, _, err := svc.RecordGrade(ctx, uuid.New().String(), uuid.New().String(), "5.0", nil)
	if !repo.recordGradeCalled {
		t.Fatal("RecordGradeTx must be called for a valid request")
	}
	if !errors.Is(err, outcomeErr) {
		t.Errorf("RecordGrade propagated error = %v; want %v", err, outcomeErr)
	}
}

func TestService_CreateEvaluationScheme_Delegates(t *testing.T) {
	t.Parallel()

	evals := []gradesdb.Evaluation{{Position: 1}}
	repo := &fakeRepository{createSchemeEvals: evals}
	svc := NewService(repo)

	ctx := context.Background()
	got, err := svc.CreateEvaluationScheme(ctx, uuid.New().String(), []string{"1.0"})
	if err != nil {
		t.Fatalf("CreateEvaluationScheme: unexpected error %v", err)
	}
	if !repo.createSchemeCalled {
		t.Error("CreateEvaluationSchemeTx was not called on the repository")
	}
	if len(got) != len(evals) {
		t.Errorf("len(evals) = %d, want %d", len(got), len(evals))
	}
}

func TestService_CreateEvaluationScheme_InvalidWeights_DoesNotCallRepo(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := context.Background()
	_, err := svc.CreateEvaluationScheme(ctx, uuid.New().String(), []string{"0.50", "0.40"})
	if repo.createSchemeCalled {
		t.Error("CreateEvaluationSchemeTx must not be called when weights are invalid")
	}
	if !errors.Is(err, ErrSchemeIncomplete) {
		t.Errorf("CreateEvaluationScheme(bad weights) = %v; want ErrSchemeIncomplete", err)
	}
}

func TestService_OverrideGrade_Delegates(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, _, _ = svc.OverrideGrade(ctx, uuid.New().String(), uuid.New().String(), "4.0", nil)

	if !repo.recordGradeCalled {
		t.Error("OverrideGrade must delegate to RecordGradeTx")
	}
}

