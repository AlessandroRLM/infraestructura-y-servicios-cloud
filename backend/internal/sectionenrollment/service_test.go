package sectionenrollment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment/sectionenrollmentdb"
)

// fakeRepository is a fake implementation of the Repository interface for service unit tests.
type fakeRepository struct {
	enrollTxCalled  bool
	enrollTxRow     sectionenrollmentdb.SectionEnrollment
	enrollTxErr     error
	enrollTxIsAdmin bool // captures the isAdmin flag from the last call

	withdrawCalled bool
	withdrawRow    sectionenrollmentdb.SectionEnrollment
	withdrawErr    error

	getSECalled bool
	getSERow    sectionenrollmentdb.SectionEnrollment
	getSEErr    error

	listCalled bool
	listRows   []sectionenrollmentdb.SectionEnrollment
	listErr    error

	listOwnCalled bool
	listOwnRows   []sectionenrollmentdb.SectionEnrollment
	listOwnErr    error

	getOwnCalled bool
	getOwnRow    sectionenrollmentdb.SectionEnrollment
	getOwnErr    error
}

func (f *fakeRepository) EnrollSectionTx(_ context.Context, _ EnrollSectionParams, isAdmin bool) (sectionenrollmentdb.SectionEnrollment, error) {
	f.enrollTxCalled = true
	f.enrollTxIsAdmin = isAdmin
	return f.enrollTxRow, f.enrollTxErr
}

func (f *fakeRepository) WithdrawSection(_ context.Context, _ uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	f.withdrawCalled = true
	return f.withdrawRow, f.withdrawErr
}

func (f *fakeRepository) GetSectionEnrollment(_ context.Context, _ uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	f.getSECalled = true
	return f.getSERow, f.getSEErr
}

func (f *fakeRepository) ListSectionEnrollments(_ context.Context, _ ListSectionEnrollmentsRepoParams) ([]sectionenrollmentdb.SectionEnrollment, error) {
	f.listCalled = true
	return f.listRows, f.listErr
}

func (f *fakeRepository) ListOwnSectionEnrollments(_ context.Context, _ ListOwnSectionEnrollmentsRepoParams) ([]sectionenrollmentdb.SectionEnrollment, error) {
	f.listOwnCalled = true
	return f.listOwnRows, f.listOwnErr
}

func (f *fakeRepository) GetOwnSectionEnrollment(_ context.Context, _ uuid.UUID) (sectionenrollmentdb.SectionEnrollment, error) {
	f.getOwnCalled = true
	return f.getOwnRow, f.getOwnErr
}

func (f *fakeRepository) SetSectionEnrollmentOutcomeTx(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ string, _ pgtype.Numeric) (sectionenrollmentdb.SectionEnrollment, error) {
	return sectionenrollmentdb.SectionEnrollment{}, nil
}

// contextWithUser adds a user ID to the context (mirrors auth.WithUserID).
func contextWithUser(userID uuid.UUID) context.Context {
	return auth.WithUserID(context.Background(), userID)
}

// TestService_EnrollOwnSection_NoContext verifies that EnrollOwnSection without an
// authenticated user in context returns ErrNotFound (fail-closed).
func TestService_EnrollOwnSection_NoContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.EnrollOwnSection(context.Background(), uuid.New().String(), uuid.New().String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("EnrollOwnSection(no ctx user) = %v; want ErrNotFound", err)
	}
	if repo.enrollTxCalled {
		t.Error("EnrollSectionTx must not be called when user is absent from context")
	}
}

// TestService_EnrollOwnSection_BadSectionID verifies that an invalid section UUID returns ErrInvalidInput.
func TestService_EnrollOwnSection_BadSectionID(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, err := svc.EnrollOwnSection(ctx, "not-a-uuid", uuid.New().String())
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("EnrollOwnSection(bad section UUID) = %v; want ErrInvalidInput", err)
	}
}

// TestService_EnrollOwnSection_BadProgramID verifies that an invalid program_id UUID returns ErrInvalidInput.
func TestService_EnrollOwnSection_BadProgramID(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, err := svc.EnrollOwnSection(ctx, uuid.New().String(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("EnrollOwnSection(bad program UUID) = %v; want ErrInvalidInput", err)
	}
}

// TestService_EnrollOwnSection_UsesIsAdminFalse verifies that the student self-service path
// calls EnrollSectionTx with isAdmin=false.
func TestService_EnrollOwnSection_UsesIsAdminFalse(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{enrollTxRow: newInsertedRow(uuid.New(), uuid.New(), uuid.New())}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, _ = svc.EnrollOwnSection(ctx, uuid.New().String(), uuid.New().String())

	if !repo.enrollTxCalled {
		t.Fatal("EnrollSectionTx was not called")
	}
	if repo.enrollTxIsAdmin {
		t.Error("EnrollOwnSection must call EnrollSectionTx with isAdmin=false")
	}
}

// TestService_GetOwnSectionEnrollment_OwnershipMismatch verifies that when the fetched
// inscription belongs to a different student, ErrNotFound is returned.
func TestService_GetOwnSectionEnrollment_OwnershipMismatch(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	callerID := uuid.New() // different from ownerID

	// The inscription's enrollment is owned by ownerID, but caller is callerID.
	// Service must compare enrollment.student_id to the caller's user_id.
	// Since we're using a fake repo, we control what GetOwnSectionEnrollment returns.
	// The service fetches the row then checks student ownership via a separate DB read OR
	// embeds the student_id in the SectionEnrollment row. Since SectionEnrollment has no
	// student_id, the service must check via the enrollment.
	// Per the design: the service calls ListOwnSectionEnrollments (scoped by student_id)
	// for list ops, and for get-own uses GetOwnSectionEnrollment + then checks ownership
	// by matching the caller's enrollments. The simplest approach matching the enrollment
	// pattern: fetch by id then do a separate enrollment lookup. However, per the spec
	// the service derives the student from context and must not disclose existence.
	// We test that a mismatch (repo returns a row whose enrollment doesn't match caller)
	// → ErrNotFound. We simulate this by making the repo return a row, then the service
	// is responsible for the ownership check.

	// Build a row where the enrollment belongs to a DIFFERENT user (we track via context only).
	// Since the fakeRepository returns whatever we configure, we test the service's
	// own-scope protection by verifying it only accepts rows where the enrollment's student
	// matches the context user. We configure the service to use a listOwn that returns NO rows
	// for the caller, simulating a mismatch.
	row := newInsertedRow(uuid.New(), uuid.New(), uuid.New())
	_ = ownerID
	repo := &fakeRepository{
		getOwnRow: row,
		// Service calls ListOwnSectionEnrollments scoped to caller to verify ownership.
		listOwnRows: nil, // no rows for caller → ownership mismatch
	}
	svc := NewService(repo)

	ctx := contextWithUser(callerID)
	seID := uuid.UUID(row.ID.Bytes)
	_, err := svc.GetOwnSectionEnrollment(ctx, seID.String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetOwnSectionEnrollment(mismatch) = %v; want ErrNotFound", err)
	}
}

// TestService_GetOwnSectionEnrollment_NoContext returns ErrNotFound when no user in context.
func TestService_GetOwnSectionEnrollment_NoContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.GetOwnSectionEnrollment(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetOwnSectionEnrollment(no ctx) = %v; want ErrNotFound", err)
	}
}

// TestService_ListOwnSectionEnrollments_DerivesFromContext verifies that no student_id
// is required in the call — it is always derived from the context.
func TestService_ListOwnSectionEnrollments_DerivesFromContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{listOwnRows: []sectionenrollmentdb.SectionEnrollment{}}
	svc := NewService(repo)

	callerID := uuid.New()
	ctx := contextWithUser(callerID)

	result, err := svc.ListOwnSectionEnrollments(ctx, 0, "")
	if err != nil {
		t.Fatalf("ListOwnSectionEnrollments: unexpected error %v", err)
	}
	_ = result
	if !repo.listOwnCalled {
		t.Error("ListOwnSectionEnrollments was not called on repository")
	}
}

// TestService_ListOwnSectionEnrollments_NoContext returns ErrNotFound.
func TestService_ListOwnSectionEnrollments_NoContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.ListOwnSectionEnrollments(context.Background(), 0, "")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ListOwnSectionEnrollments(no ctx) = %v; want ErrNotFound", err)
	}
}

// TestService_ListOwnSectionEnrollments_InvalidToken verifies that a malformed
// page_token returns ErrInvalidInput before touching the repository.
func TestService_ListOwnSectionEnrollments_InvalidToken(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, err := svc.ListOwnSectionEnrollments(ctx, 20, "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("ListOwnSectionEnrollments(bad token) = %v; want ErrInvalidInput", err)
	}
	if repo.listOwnCalled {
		t.Error("repo.ListOwnSectionEnrollments must not be called on invalid token")
	}
}

// TestService_ListSectionEnrollments_InvalidToken verifies that a malformed page_token
// returns ErrInvalidInput before touching the repository.
func TestService_ListSectionEnrollments_InvalidToken(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.ListSectionEnrollments(context.Background(), ListSectionEnrollmentsFilter{}, 20, "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("ListSectionEnrollments(bad token) = %v; want ErrInvalidInput", err)
	}
	if repo.listCalled {
		t.Error("repo.ListSectionEnrollments must not be called on invalid token")
	}
}

// TestService_ListSectionEnrollments_ClampMin verifies page_size ≤ 0 is clamped to 20.
func TestService_ListSectionEnrollments_ClampMin(t *testing.T) {
	t.Parallel()

	// Repo returns 21 rows (size+1 sentinel pattern).
	rows := make([]sectionenrollmentdb.SectionEnrollment, 21)
	for i := range rows {
		rows[i] = newInsertedRow(uuid.New(), uuid.New(), uuid.New())
	}
	repo := &fakeRepository{listRows: rows}
	svc := NewService(repo)

	result, err := svc.ListSectionEnrollments(context.Background(), ListSectionEnrollmentsFilter{}, 0, "")
	if err != nil {
		t.Fatalf("ListSectionEnrollments: %v", err)
	}
	if len(result.SectionEnrollments) != 20 {
		t.Errorf("clamped page size = %d, want 20", len(result.SectionEnrollments))
	}
	if result.NextPageToken == "" {
		t.Error("next_page_token must be non-empty when HasNext=true")
	}
}

// TestService_ListSectionEnrollments_LastPage verifies that an empty next_page_token
// is returned when the result fits within the page size.
func TestService_ListSectionEnrollments_LastPage(t *testing.T) {
	t.Parallel()

	rows := make([]sectionenrollmentdb.SectionEnrollment, 5)
	for i := range rows {
		rows[i] = newInsertedRow(uuid.New(), uuid.New(), uuid.New())
	}
	repo := &fakeRepository{listRows: rows}
	svc := NewService(repo)

	result, err := svc.ListSectionEnrollments(context.Background(), ListSectionEnrollmentsFilter{}, 20, "")
	if err != nil {
		t.Fatalf("ListSectionEnrollments: %v", err)
	}
	if len(result.SectionEnrollments) != 5 {
		t.Errorf("last page count = %d, want 5", len(result.SectionEnrollments))
	}
	if result.NextPageToken != "" {
		t.Errorf("last page: next_page_token = %q, want empty", result.NextPageToken)
	}
}

// TestService_EnrollSection_UsesIsAdminTrue verifies that admin EnrollSection passes
// isAdmin=true to the repository and is not window-gated at the service layer.
func TestService_EnrollSection_UsesIsAdminTrue(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{enrollTxRow: newInsertedRow(uuid.New(), uuid.New(), uuid.New())}
	svc := NewService(repo)

	// Admin context (any valid user — policies checked at handler interceptor level).
	ctx := contextWithUser(uuid.New())
	_, _ = svc.EnrollSection(ctx, uuid.New().String(), uuid.New().String())

	if !repo.enrollTxCalled {
		t.Fatal("EnrollSectionTx was not called")
	}
	if !repo.enrollTxIsAdmin {
		t.Error("EnrollSection must call EnrollSectionTx with isAdmin=true")
	}
}

// TestService_WithdrawSection_PropagatesNotFound verifies error propagation.
func TestService_WithdrawSection_PropagatesNotFound(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{withdrawErr: ErrNotFound}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, err := svc.WithdrawSection(ctx, uuid.New().String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("WithdrawSection(not found) = %v; want ErrNotFound", err)
	}
}

// TestService_EnrollSection_PaidGateChecked verifies that if the repo returns ErrNotPaid,
// it propagates as-is (the gate is applied inside the repository's transaction).
func TestService_EnrollSection_PaidGateChecked(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{enrollTxErr: ErrNotPaid}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, err := svc.EnrollSection(ctx, uuid.New().String(), uuid.New().String())
	if !errors.Is(err, ErrNotPaid) {
		t.Errorf("EnrollSection(not paid) = %v; want ErrNotPaid", err)
	}
	if !repo.enrollTxCalled {
		t.Error("EnrollSectionTx must be called even when it returns ErrNotPaid")
	}
}
