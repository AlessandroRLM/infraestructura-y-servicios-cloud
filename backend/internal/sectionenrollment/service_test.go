package sectionenrollment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
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

	rosterCalled bool
	rosterParams ListSectionRosterForTeacherRepoParams
	rosterRows   []sectionenrollmentdb.ListSectionRosterForTeacherRow
	rosterErr    error
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

func (f *fakeRepository) ListSectionRosterForTeacher(_ context.Context, p ListSectionRosterForTeacherRepoParams) ([]sectionenrollmentdb.ListSectionRosterForTeacherRow, error) {
	f.rosterCalled = true
	f.rosterParams = p
	return f.rosterRows, f.rosterErr
}

func (f *fakeRepository) ListSectionRosterForTeacherAll(_ context.Context, _ ListSectionRosterForTeacherAllRepoParams) ([]sectionenrollmentdb.ListSectionRosterForTeacherRow, error) {
	return f.rosterRows, f.rosterErr
}

// contextWithUser adds a user ID to the context (mirrors auth.WithUserID).
func contextWithUser(userID uuid.UUID) context.Context {
	return auth.WithUserID(context.Background(), userID)
}

// TestService_EnrollOwnSection_NoContext verifies that EnrollOwnSection without an
// authenticated user in context returns ErrUnauthenticated (fail-closed).
func TestService_EnrollOwnSection_NoContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.EnrollOwnSection(context.Background(), uuid.New().String(), uuid.New().String())
	if !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("EnrollOwnSection(no ctx user) = %v; want ErrUnauthenticated", err)
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

// TestService_GetOwnSectionEnrollment_NoContext returns ErrUnauthenticated when no user in context.
func TestService_GetOwnSectionEnrollment_NoContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.GetOwnSectionEnrollment(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("GetOwnSectionEnrollment(no ctx) = %v; want ErrUnauthenticated", err)
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

// TestService_ListOwnSectionEnrollments_NoContext returns ErrUnauthenticated.
func TestService_ListOwnSectionEnrollments_NoContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.ListOwnSectionEnrollments(context.Background(), 0, "")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("ListOwnSectionEnrollments(no ctx) = %v; want ErrUnauthenticated", err)
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

// --- ListSectionRosterForTeacher service unit tests ---

// TestService_ListSectionRosterForTeacher_NoContext verifies that ListSectionRosterForTeacher
// without an authenticated user in context returns ErrUnauthenticated.
func TestService_ListSectionRosterForTeacher_NoContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.ListSectionRosterForTeacher(context.Background(), uuid.New().String(), 20, "")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("ListSectionRosterForTeacher(no ctx) = %v; want ErrUnauthenticated", err)
	}
	if repo.rosterCalled {
		t.Error("repo.ListSectionRosterForTeacher must not be called when user is absent from context")
	}
}

// TestService_ListSectionRosterForTeacher_BadSectionID verifies that a malformed section_id
// returns ErrInvalidInput before touching the repository.
func TestService_ListSectionRosterForTeacher_BadSectionID(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := contextWithUser(uuid.New())
	_, err := svc.ListSectionRosterForTeacher(ctx, "not-a-uuid", 20, "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("ListSectionRosterForTeacher(bad section UUID) = %v; want ErrInvalidInput", err)
	}
	if repo.rosterCalled {
		t.Error("repo.ListSectionRosterForTeacher must not be called on invalid section_id")
	}
}

// TestService_ListSectionRosterForTeacher_PassesCallerIDFromContext verifies that the teacher_id
// passed to the repository equals the session user ID, not any request-supplied value.
// (ListSectionRosterForTeacher accepts no teacher_id param — it is always derived from context.)
func TestService_ListSectionRosterForTeacher_PassesCallerIDFromContext(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	sectionID := uuid.New()

	repo := &fakeRepository{}
	svc := NewService(repo)

	ctx := contextWithUser(callerID)
	_, err := svc.ListSectionRosterForTeacher(ctx, sectionID.String(), 20, "")
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher: unexpected error %v", err)
	}

	if !repo.rosterCalled {
		t.Fatal("repo.ListSectionRosterForTeacher was not called")
	}
	if repo.rosterParams.TeacherID != callerID {
		t.Errorf("repo received TeacherID = %v, want %v (session caller)", repo.rosterParams.TeacherID, callerID)
	}
	if repo.rosterParams.SectionID != sectionID {
		t.Errorf("repo received SectionID = %v, want %v", repo.rosterParams.SectionID, sectionID)
	}
}

// --- ListSectionRosterForTeacher admin-bypass discriminator unit tests ---

// fakeRepositoryWithRosterBypass extends fakeRepository with call sentinels for the
// admin-bypass path so tests can assert WHICH repo method was invoked.
type fakeRepositoryWithRosterBypass struct {
	fakeRepository

	// rosterAllCalled is true when ListSectionRosterForTeacherAll was invoked.
	rosterAllCalled bool
	// rosterAllParams captures the params passed to ListSectionRosterForTeacherAll.
	rosterAllParams ListSectionRosterForTeacherAllRepoParams
	// rosterAllRows is the slice returned by ListSectionRosterForTeacherAll.
	rosterAllRows []sectionenrollmentdb.ListSectionRosterForTeacherRow
	// rosterAllErr is the error returned by ListSectionRosterForTeacherAll.
	rosterAllErr error

	// rosterScopedCalled is true when ListSectionRosterForTeacher was invoked.
	rosterScopedCalled bool
}

func (f *fakeRepositoryWithRosterBypass) ListSectionRosterForTeacherAll(_ context.Context, p ListSectionRosterForTeacherAllRepoParams) ([]sectionenrollmentdb.ListSectionRosterForTeacherRow, error) {
	f.rosterAllCalled = true
	f.rosterAllParams = p
	return f.rosterAllRows, f.rosterAllErr
}

// Override ListSectionRosterForTeacher so tests track calls independently.
func (f *fakeRepositoryWithRosterBypass) ListSectionRosterForTeacher(_ context.Context, p ListSectionRosterForTeacherRepoParams) ([]sectionenrollmentdb.ListSectionRosterForTeacherRow, error) {
	f.rosterScopedCalled = true
	f.fakeRepository.rosterCalled = true
	f.fakeRepository.rosterParams = p
	return f.fakeRepository.rosterRows, f.fakeRepository.rosterErr
}

// TestListSectionRosterForTeacher_AdminBypass_CallsAllVariant verifies that a caller
// holding PermEnrollmentManage is routed to ListSectionRosterForTeacherAll and NOT to
// the scoped ListSectionRosterForTeacher.
func TestListSectionRosterForTeacher_AdminBypass_CallsAllVariant(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)
	ctx = authz.WithPermissions(ctx, authz.NewPermissionSet([]authz.Permission{
		authz.PermSectionEnrollmentViewTeaching,
		authz.PermEnrollmentManage,
	}))

	repo := &fakeRepositoryWithRosterBypass{}
	svc := NewService(repo)

	sectionID := uuid.New()
	_, err := svc.ListSectionRosterForTeacher(ctx, sectionID.String(), 20, "")
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher (admin): unexpected error: %v", err)
	}
	if !repo.rosterAllCalled {
		t.Error("ListSectionRosterForTeacher (admin): ListSectionRosterForTeacherAll was NOT called")
	}
	if repo.rosterScopedCalled {
		t.Error("ListSectionRosterForTeacher (admin): scoped method was called — should NOT be for admin")
	}
}

// TestListSectionRosterForTeacher_TeacherPath_CallsScopedVariant verifies that a caller
// holding PermSectionEnrollmentViewTeaching but NOT PermEnrollmentManage is routed to
// the scoped ListSectionRosterForTeacher.
func TestListSectionRosterForTeacher_TeacherPath_CallsScopedVariant(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)
	ctx = authz.WithPermissions(ctx, authz.NewPermissionSet([]authz.Permission{
		authz.PermSectionEnrollmentViewTeaching,
	}))

	repo := &fakeRepositoryWithRosterBypass{}
	svc := NewService(repo)

	_, err := svc.ListSectionRosterForTeacher(ctx, uuid.New().String(), 20, "")
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher (teacher): unexpected error: %v", err)
	}
	if repo.rosterAllCalled {
		t.Error("ListSectionRosterForTeacher (teacher): admin variant was called — should NOT be for teacher")
	}
	if !repo.rosterScopedCalled {
		t.Error("ListSectionRosterForTeacher (teacher): scoped method was NOT called")
	}
}

// TestListSectionRosterForTeacher_TeacherEmpty_NoLeak verifies that a teacher with 0
// enrollments in the section returns an empty result with no error (anti-leak).
func TestListSectionRosterForTeacher_TeacherEmpty_NoLeak(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	ctx := auth.WithUserID(context.Background(), actorID)
	ctx = authz.WithPermissions(ctx, authz.NewPermissionSet([]authz.Permission{
		authz.PermSectionEnrollmentViewTeaching,
	}))

	repo := &fakeRepositoryWithRosterBypass{
		fakeRepository: fakeRepository{
			rosterRows: []sectionenrollmentdb.ListSectionRosterForTeacherRow{},
		},
	}
	svc := NewService(repo)

	result, err := svc.ListSectionRosterForTeacher(ctx, uuid.New().String(), 20, "")
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher (empty teacher): unexpected error: %v", err)
	}
	if len(result.Rows) != 0 {
		t.Errorf("got %d rows, want 0", len(result.Rows))
	}
}

// TestListSectionRosterForTeacher_EntryGate_StillPermSectionEnrollmentViewTeaching verifies
// that the entry gate constant has not changed — any swap to enrollment.manage would lock
// out teachers who hold only section_enrollment.view_teaching.
func TestListSectionRosterForTeacher_EntryGate_StillPermSectionEnrollmentViewTeaching(t *testing.T) {
	t.Parallel()

	const wantGate = "section_enrollment.view_teaching"
	got := string(authz.PermSectionEnrollmentViewTeaching)
	if got != wantGate {
		t.Errorf("entry-gate constant changed: got %q, want %q — teachers would be locked out", got, wantGate)
	}

	const wantBypass = "enrollment.manage"
	gotBypass := string(authz.PermEnrollmentManage)
	if gotBypass != wantBypass {
		t.Errorf("bypass constant changed: got %q, want %q", gotBypass, wantBypass)
	}
}

// TestListSectionRosterForTeacher_Unauthenticated_Guard verifies that ErrUnauthenticated
// is returned when no user is in the context — no repo call should occur.
func TestListSectionRosterForTeacher_Unauthenticated_Guard(t *testing.T) {
	t.Parallel()

	ctx := context.Background() // no user in context

	repo := &fakeRepositoryWithRosterBypass{}
	svc := NewService(repo)

	_, err := svc.ListSectionRosterForTeacher(ctx, uuid.New().String(), 20, "")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("got %v, want ErrUnauthenticated", err)
	}
	if repo.rosterAllCalled || repo.rosterScopedCalled {
		t.Error("a repo method was called — should have been rejected before any DB call")
	}
}
