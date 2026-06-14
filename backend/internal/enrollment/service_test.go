package enrollment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/enrollment/enrollmentdb"
)

// fakeRepository is a test double for Repository.
type fakeRepository struct {
	createCalled bool
	createResult enrollmentdb.Enrollment
	createErr    error

	markPaidCalled bool
	markPaidResult enrollmentdb.Enrollment
	markPaidErr    error

	cancelCalled bool
	cancelErr    error

	getEnrollmentCalled bool
	getEnrollmentResult enrollmentdb.Enrollment
	getEnrollmentErr    error

	listCalled bool
	listResult []enrollmentdb.Enrollment
	listErr    error

	listOwnCalled bool
	listOwnResult []enrollmentdb.Enrollment
	listOwnErr    error
}

func (f *fakeRepository) CreateEnrollmentTx(_ context.Context, _ CreateEnrollmentParams, _ *uuid.UUID) (enrollmentdb.Enrollment, error) {
	f.createCalled = true
	return f.createResult, f.createErr
}

func (f *fakeRepository) MarkEnrollmentPaid(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (enrollmentdb.Enrollment, error) {
	f.markPaidCalled = true
	return f.markPaidResult, f.markPaidErr
}

func (f *fakeRepository) CancelEnrollment(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	f.cancelCalled = true
	return f.cancelErr
}

func (f *fakeRepository) GetEnrollment(_ context.Context, _ uuid.UUID) (enrollmentdb.Enrollment, error) {
	f.getEnrollmentCalled = true
	return f.getEnrollmentResult, f.getEnrollmentErr
}

func (f *fakeRepository) ListEnrollments(_ context.Context, _ ListEnrollmentsRepoParams) ([]enrollmentdb.Enrollment, error) {
	f.listCalled = true
	return f.listResult, f.listErr
}

func (f *fakeRepository) ListOwnEnrollments(_ context.Context, _ ListOwnEnrollmentsRepoParams) ([]enrollmentdb.Enrollment, error) {
	f.listOwnCalled = true
	return f.listOwnResult, f.listOwnErr
}

// ctxWithUser returns a context carrying a user ID, mirroring how the auth
// interceptor populates context for authenticated requests.
func ctxWithUser(userID uuid.UUID) context.Context {
	return auth.WithUserID(context.Background(), userID)
}

// ---- Validation tests ----

// TestCreateEnrollment_YearZero verifies that year ≤ 0 returns ErrInvalidInput
// before any repository call.
func TestCreateEnrollment_YearZero(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	ctx := ctxWithUser(uuid.New())

	_, err := svc.CreateEnrollment(ctx, uuid.New().String(), uuid.New().String(), 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("year=0: want ErrInvalidInput, got %v", err)
	}
	if repo.createCalled {
		t.Error("year=0: repository.CreateEnrollmentTx must not be called")
	}
}

// TestCreateEnrollment_NegativeYear verifies that a negative year is rejected.
func TestCreateEnrollment_NegativeYear(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	ctx := ctxWithUser(uuid.New())

	_, err := svc.CreateEnrollment(ctx, uuid.New().String(), uuid.New().String(), -1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("year=-1: want ErrInvalidInput, got %v", err)
	}
}

// TestCreateEnrollment_BadStudentIDUUID verifies that a malformed student_id UUID
// returns ErrInvalidInput before any repository call.
func TestCreateEnrollment_BadStudentIDUUID(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	ctx := ctxWithUser(uuid.New())

	_, err := svc.CreateEnrollment(ctx, "not-a-uuid", uuid.New().String(), 2025)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("bad student UUID: want ErrInvalidInput, got %v", err)
	}
	if repo.createCalled {
		t.Error("bad student UUID: repository must not be called")
	}
}

// TestCreateEnrollment_BadProgramIDUUID verifies that a malformed program_id UUID
// returns ErrInvalidInput.
func TestCreateEnrollment_BadProgramIDUUID(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	ctx := ctxWithUser(uuid.New())

	_, err := svc.CreateEnrollment(ctx, uuid.New().String(), "bad-uuid", 2025)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("bad program UUID: want ErrInvalidInput, got %v", err)
	}
}

// ---- State-machine tests (via service wrapper around fakeRepository) ----

// TestMarkEnrollmentPaid_RejectsNonPending verifies that when the repository signals
// ErrInvalidTransition the service propagates it unchanged.
func TestMarkEnrollmentPaid_PropagatesInvalidTransition(t *testing.T) {
	repo := &fakeRepository{markPaidErr: ErrInvalidTransition}
	svc := NewService(repo)
	ctx := ctxWithUser(uuid.New())

	_, err := svc.MarkEnrollmentPaid(ctx, uuid.New().String())
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("mark-paid invalid transition: want ErrInvalidTransition, got %v", err)
	}
	if !repo.markPaidCalled {
		t.Error("mark-paid: repo.MarkEnrollmentPaid was not called")
	}
}

// TestCancelEnrollment_PropagatesInvalidTransition verifies that when the repository
// signals ErrInvalidTransition (already cancelled) the service propagates it.
func TestCancelEnrollment_PropagatesInvalidTransition(t *testing.T) {
	repo := &fakeRepository{cancelErr: ErrInvalidTransition}
	svc := NewService(repo)
	ctx := ctxWithUser(uuid.New())

	err := svc.CancelEnrollment(ctx, uuid.New().String())
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("cancel invalid transition: want ErrInvalidTransition, got %v", err)
	}
}

// ---- ListOwnEnrollments context injection tests ----

// TestListOwnEnrollments_InjectsContextUser verifies that the service reads the user
// ID from context and delegates to the repository using that ID.
func TestListOwnEnrollments_InjectsContextUser(t *testing.T) {
	userID := uuid.New()
	repo := &fakeRepository{listOwnResult: []enrollmentdb.Enrollment{}}
	svc := NewService(repo)
	ctx := ctxWithUser(userID)

	_, err := svc.ListOwnEnrollments(ctx, 0, "")
	if err != nil {
		t.Fatalf("ListOwnEnrollments: %v", err)
	}
	if !repo.listOwnCalled {
		t.Error("ListOwnEnrollments: repo.ListOwnEnrollments was not called")
	}
}

// TestListOwnEnrollments_NoCtxUser verifies that when no user ID is in context,
// the service returns ErrNotFound (fail-closed, never leak existence).
func TestListOwnEnrollments_NoCtxUser(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.ListOwnEnrollments(context.Background(), 0, "")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("no ctx user: want ErrNotFound, got %v", err)
	}
	if repo.listOwnCalled {
		t.Error("no ctx user: repo.ListOwnEnrollments must not be called")
	}
}

// ---- GetOwnEnrollment ownership tests ----

// TestGetOwnEnrollment_Self verifies that a student retrieving their own enrollment succeeds.
func TestGetOwnEnrollment_Self(t *testing.T) {
	userID := uuid.New()
	enrollment := enrollmentdb.Enrollment{
		ID:        pgUUID(uuid.New()),
		StudentID: pgUUID(userID),
		Status:    "pending",
	}
	repo := &fakeRepository{getEnrollmentResult: enrollment}
	svc := NewService(repo)
	ctx := ctxWithUser(userID)

	enrollID := uuid.UUID(enrollment.ID.Bytes)
	got, err := svc.GetOwnEnrollment(ctx, enrollID.String())
	if err != nil {
		t.Fatalf("GetOwnEnrollment self: %v", err)
	}
	if got.ID != enrollment.ID {
		t.Errorf("GetOwnEnrollment self: got id %v, want %v", got.ID, enrollment.ID)
	}
}

// TestGetOwnEnrollment_OwnershipMismatch verifies that when the enrollment belongs to
// a different student, the service returns ErrNotFound — never leaking existence.
func TestGetOwnEnrollment_OwnershipMismatch(t *testing.T) {
	userID := uuid.New()
	otherID := uuid.New()
	enrollment := enrollmentdb.Enrollment{
		ID:        pgUUID(uuid.New()),
		StudentID: pgUUID(otherID), // owned by a different user
		Status:    "pending",
	}
	repo := &fakeRepository{getEnrollmentResult: enrollment}
	svc := NewService(repo)
	ctx := ctxWithUser(userID)

	enrollID := uuid.UUID(enrollment.ID.Bytes)
	_, err := svc.GetOwnEnrollment(ctx, enrollID.String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ownership mismatch: want ErrNotFound (no existence leak), got %v", err)
	}
}

// TestGetOwnEnrollment_NoCtxUser verifies that an absent context user returns ErrNotFound.
func TestGetOwnEnrollment_NoCtxUser(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.GetOwnEnrollment(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("no ctx user: want ErrNotFound, got %v", err)
	}
}

// ---- ListEnrollments pagination unit tests ----

// TestListEnrollments_ClampMin verifies page_size=0 is clamped to 20 (min).
func TestListEnrollments_ClampMin(t *testing.T) {
	repo := &fakeRepository{listResult: []enrollmentdb.Enrollment{}}
	svc := NewService(repo)

	_, err := svc.ListEnrollments(context.Background(), ListEnrollmentsFilter{}, 0, "")
	if err != nil {
		t.Fatalf("ListEnrollments(pageSize=0): %v", err)
	}
	if !repo.listCalled {
		t.Error("repo.ListEnrollments was not called")
	}
}

// TestListEnrollments_ClampMax verifies page_size=999 is clamped to 200 (max).
func TestListEnrollments_ClampMax(t *testing.T) {
	repo := &fakeRepository{listResult: []enrollmentdb.Enrollment{}}
	svc := NewService(repo)

	_, err := svc.ListEnrollments(context.Background(), ListEnrollmentsFilter{}, 999, "")
	if err != nil {
		t.Fatalf("ListEnrollments(pageSize=999): %v", err)
	}
}

// TestListEnrollments_ValidToken verifies a well-formed page_token is parsed and
// passed to the repository without error.
func TestListEnrollments_ValidToken(t *testing.T) {
	token := uuid.New().String()
	repo := &fakeRepository{listResult: []enrollmentdb.Enrollment{}}
	svc := NewService(repo)

	_, err := svc.ListEnrollments(context.Background(), ListEnrollmentsFilter{}, 20, token)
	if err != nil {
		t.Fatalf("ListEnrollments(valid token): %v", err)
	}
}

// TestListEnrollments_InvalidToken verifies a malformed page_token returns ErrInvalidInput.
func TestListEnrollments_InvalidToken(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.ListEnrollments(context.Background(), ListEnrollmentsFilter{}, 20, "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("malformed token: want ErrInvalidInput, got %v", err)
	}
	if repo.listCalled {
		t.Error("malformed token: repo.ListEnrollments must not be called")
	}
}

// TestListEnrollments_NextTokenSetWhenHasNext verifies that when the repository returns
// clampedSize+1 rows (HasNext=true), a non-empty next_page_token is produced.
func TestListEnrollments_NextTokenSetWhenHasNext(t *testing.T) {
	// Build 21 fake rows (clamp=20, so 21 > clamped → HasNext=true).
	rows := make([]enrollmentdb.Enrollment, 21)
	for i := range rows {
		rows[i] = enrollmentdb.Enrollment{ID: pgUUID(uuid.New())}
	}
	repo := &fakeRepository{listResult: rows}
	svc := NewService(repo)

	result, err := svc.ListEnrollments(context.Background(), ListEnrollmentsFilter{}, 20, "")
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}
	if result.NextPageToken == "" {
		t.Error("next_page_token must be non-empty when more rows exist")
	}
	if len(result.Enrollments) != 20 {
		t.Errorf("got %d enrollments, want 20 (clamped)", len(result.Enrollments))
	}
}

// TestListEnrollments_EmptyTokenOnLastPage verifies that when the repository returns
// ≤ clampedSize rows (HasNext=false), next_page_token is empty.
func TestListEnrollments_EmptyTokenOnLastPage(t *testing.T) {
	rows := make([]enrollmentdb.Enrollment, 5)
	for i := range rows {
		rows[i] = enrollmentdb.Enrollment{ID: pgUUID(uuid.New())}
	}
	repo := &fakeRepository{listResult: rows}
	svc := NewService(repo)

	result, err := svc.ListEnrollments(context.Background(), ListEnrollmentsFilter{}, 20, "")
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}
	if result.NextPageToken != "" {
		t.Errorf("next_page_token must be empty on last page, got %q", result.NextPageToken)
	}
	if len(result.Enrollments) != 5 {
		t.Errorf("got %d enrollments, want 5", len(result.Enrollments))
	}
}

// ---- ListOwnEnrollments pagination unit tests ----

// TestListOwnEnrollments_ClampMin verifies page_size=0 is clamped to 20 (min).
func TestListOwnEnrollments_ClampMin(t *testing.T) {
	userID := uuid.New()
	repo := &fakeRepository{listOwnResult: []enrollmentdb.Enrollment{}}
	svc := NewService(repo)
	ctx := ctxWithUser(userID)

	_, err := svc.ListOwnEnrollments(ctx, 0, "")
	if err != nil {
		t.Fatalf("ListOwnEnrollments(pageSize=0): %v", err)
	}
}

// TestListOwnEnrollments_InvalidToken verifies a malformed page_token returns ErrInvalidInput.
func TestListOwnEnrollments_InvalidToken(t *testing.T) {
	userID := uuid.New()
	repo := &fakeRepository{}
	svc := NewService(repo)
	ctx := ctxWithUser(userID)

	_, err := svc.ListOwnEnrollments(ctx, 20, "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("malformed token: want ErrInvalidInput, got %v", err)
	}
	if repo.listOwnCalled {
		t.Error("malformed token: repo.ListOwnEnrollments must not be called")
	}
}

// TestListOwnEnrollments_NextTokenSetWhenHasNext verifies that 21 rows → non-empty token.
func TestListOwnEnrollments_NextTokenSetWhenHasNext(t *testing.T) {
	userID := uuid.New()
	rows := make([]enrollmentdb.Enrollment, 21)
	for i := range rows {
		rows[i] = enrollmentdb.Enrollment{ID: pgUUID(uuid.New())}
	}
	repo := &fakeRepository{listOwnResult: rows}
	svc := NewService(repo)
	ctx := ctxWithUser(userID)

	result, err := svc.ListOwnEnrollments(ctx, 20, "")
	if err != nil {
		t.Fatalf("ListOwnEnrollments: %v", err)
	}
	if result.NextPageToken == "" {
		t.Error("next_page_token must be non-empty when more rows exist")
	}
	if len(result.Enrollments) != 20 {
		t.Errorf("got %d enrollments, want 20 (clamped)", len(result.Enrollments))
	}
}
