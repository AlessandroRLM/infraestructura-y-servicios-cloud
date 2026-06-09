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

func (f *fakeRepository) ListEnrollments(_ context.Context, _ ListEnrollmentsFilter) ([]enrollmentdb.Enrollment, error) {
	f.listCalled = true
	return f.listResult, f.listErr
}

func (f *fakeRepository) ListOwnEnrollments(_ context.Context, _ uuid.UUID) ([]enrollmentdb.Enrollment, error) {
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

	_, err := svc.ListOwnEnrollments(ctx)
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

	_, err := svc.ListOwnEnrollments(context.Background())
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
