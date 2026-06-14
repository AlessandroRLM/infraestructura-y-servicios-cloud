package iam_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam/iamdb"
)

// fakeQuerier is a stub implementing iamdb.Querier for repository unit tests.
// All methods default to returning zero values unless their corresponding
// field is set.
type fakeQuerier struct {
	listUsersRows   []iamdb.ListUsersRow
	listUsersErr    error
	listUsersCalled bool

	getUserByIDRow    iamdb.GetUserByIDRow
	getUserByIDErr    error
	getUserByIDCalled bool

	getUserRolesResult []string
	getUserRolesErr    error
	getUserRolesCalled bool

	assignRoleResult int64
	assignRoleErr    error
	assignRoleCalled bool

	revokeRoleResult int64
	revokeRoleErr    error
	revokeRoleCalled bool

	countAdminsResult int32
	countAdminsErr    error
	countAdminsCalled bool

	insertAuditLogErr    error
	insertAuditLogCalled bool
}

// Compile-time check: fakeQuerier must implement iamdb.Querier.
var _ iamdb.Querier = (*fakeQuerier)(nil)

func (f *fakeQuerier) ListUsers(_ context.Context, _ iamdb.ListUsersParams) ([]iamdb.ListUsersRow, error) {
	f.listUsersCalled = true
	return f.listUsersRows, f.listUsersErr
}

func (f *fakeQuerier) GetUserByID(_ context.Context, _ pgtype.UUID) (iamdb.GetUserByIDRow, error) {
	f.getUserByIDCalled = true
	return f.getUserByIDRow, f.getUserByIDErr
}

func (f *fakeQuerier) GetUserRoles(_ context.Context, _ pgtype.UUID) ([]string, error) {
	f.getUserRolesCalled = true
	return f.getUserRolesResult, f.getUserRolesErr
}

func (f *fakeQuerier) AssignRole(_ context.Context, _ iamdb.AssignRoleParams) (int64, error) {
	f.assignRoleCalled = true
	return f.assignRoleResult, f.assignRoleErr
}

func (f *fakeQuerier) RevokeRole(_ context.Context, _ iamdb.RevokeRoleParams) (int64, error) {
	f.revokeRoleCalled = true
	return f.revokeRoleResult, f.revokeRoleErr
}

func (f *fakeQuerier) CountAdmins(_ context.Context) (int32, error) {
	f.countAdminsCalled = true
	return f.countAdminsResult, f.countAdminsErr
}

func (f *fakeQuerier) InsertAuditLog(_ context.Context, _ iamdb.InsertAuditLogParams) error {
	f.insertAuditLogCalled = true
	return f.insertAuditLogErr
}

// newTestRepo constructs a repository with a nil pool (safe for non-transactional tests).
// Pool is nil because these tests only exercise non-transactional repository methods.
func newTestRepo(q *fakeQuerier) iam.Repository {
	return iam.NewPostgresRepository(q, nil)
}

// --- Repository unit tests (read side) ---

func TestRepository_ListUsers_DelegatesToQuerier(t *testing.T) {
	t.Parallel()

	expected := []iamdb.ListUsersRow{
		{ID: pgtype.UUID{Bytes: uuid.New(), Valid: true}, Email: "alice@example.com"},
	}
	q := &fakeQuerier{listUsersRows: expected}
	repo := newTestRepo(q)

	rows, err := repo.ListUsers(context.Background(), iam.ListUsersParams{RowLimit: 21})
	if err != nil {
		t.Fatalf("ListUsers: unexpected error: %v", err)
	}
	if !q.listUsersCalled {
		t.Error("ListUsers: fakeQuerier.ListUsers was not called")
	}
	if len(rows) != len(expected) {
		t.Errorf("ListUsers: got %d rows, want %d", len(rows), len(expected))
	}
}

func TestRepository_ListUsers_TranslatesDBError(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{listUsersErr: &pgconn.PgError{Code: "23505"}}
	repo := newTestRepo(q)

	_, err := repo.ListUsers(context.Background(), iam.ListUsersParams{RowLimit: 21})
	if !errors.Is(err, iam.ErrAlreadyExists) {
		t.Errorf("ListUsers (23505): got %v, want ErrAlreadyExists", err)
	}
}

func TestRepository_GetUserByID_DelegatesToQuerier(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	expected := iamdb.GetUserByIDRow{
		ID:    pgtype.UUID{Bytes: userID, Valid: true},
		Email: "bob@example.com",
	}
	q := &fakeQuerier{getUserByIDRow: expected}
	repo := newTestRepo(q)

	row, err := repo.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserByID: unexpected error: %v", err)
	}
	if !q.getUserByIDCalled {
		t.Error("GetUserByID: fakeQuerier.GetUserByID was not called")
	}
	if row.Email != expected.Email {
		t.Errorf("GetUserByID: email = %q, want %q", row.Email, expected.Email)
	}
}

func TestRepository_GetUserByID_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{getUserByIDErr: pgx.ErrNoRows}
	repo := newTestRepo(q)

	_, err := repo.GetUserByID(context.Background(), uuid.New())
	if !errors.Is(err, iam.ErrNotFound) {
		t.Errorf("GetUserByID (no rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_GetUserByID_TranslatesDBError(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{getUserByIDErr: &pgconn.PgError{Code: "23503"}}
	repo := newTestRepo(q)

	_, err := repo.GetUserByID(context.Background(), uuid.New())
	if !errors.Is(err, iam.ErrInvalidInput) {
		t.Errorf("GetUserByID (23503): got %v, want ErrInvalidInput", err)
	}
}

func TestRepository_GetUserRoles_DelegatesToQuerier(t *testing.T) {
	t.Parallel()

	expected := []string{"admin", "teacher"}
	q := &fakeQuerier{getUserRolesResult: expected}
	repo := newTestRepo(q)

	roles, err := repo.GetUserRoles(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetUserRoles: unexpected error: %v", err)
	}
	if !q.getUserRolesCalled {
		t.Error("GetUserRoles: fakeQuerier.GetUserRoles was not called")
	}
	if len(roles) != len(expected) {
		t.Errorf("GetUserRoles: got %d roles, want %d", len(roles), len(expected))
	}
	for i, r := range roles {
		if r != expected[i] {
			t.Errorf("GetUserRoles[%d]: got %q, want %q", i, r, expected[i])
		}
	}
}

func TestRepository_GetUserRoles_NilBecomesEmpty(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{getUserRolesResult: nil}
	repo := newTestRepo(q)

	roles, err := repo.GetUserRoles(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetUserRoles: unexpected error: %v", err)
	}
	if roles == nil {
		t.Error("GetUserRoles (nil from querier): expected empty slice, got nil")
	}
	if len(roles) != 0 {
		t.Errorf("GetUserRoles (nil from querier): got %d roles, want 0", len(roles))
	}
}

func TestRepository_GetUserRoles_TranslatesDBError(t *testing.T) {
	t.Parallel()

	inner := errors.New("connection reset")
	q := &fakeQuerier{getUserRolesErr: inner}
	repo := newTestRepo(q)

	_, err := repo.GetUserRoles(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("GetUserRoles (db error): expected error, got nil")
	}
	if !errors.Is(err, inner) {
		t.Errorf("GetUserRoles (db error): original error not wrapped, got %v", err)
	}
}

// --- AssignRole repository unit tests ---

func TestRepository_AssignRole_DelegatesToQuerier(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{assignRoleResult: 1}
	repo := newTestRepo(q)

	n, err := repo.AssignRole(context.Background(), iam.AssignRoleParams{
		UserID:   uuid.New(),
		RoleName: "teacher",
		Actor:    uuid.New(),
	})
	if err != nil {
		t.Fatalf("AssignRole: unexpected error: %v", err)
	}
	if !q.assignRoleCalled {
		t.Error("AssignRole: fakeQuerier.AssignRole was not called")
	}
	if n != 1 {
		t.Errorf("AssignRole: rows = %d, want 1", n)
	}
}

func TestRepository_AssignRole_IdempotentReturnsZeroRows(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{assignRoleResult: 0} // ON CONFLICT DO NOTHING
	repo := newTestRepo(q)

	n, err := repo.AssignRole(context.Background(), iam.AssignRoleParams{
		UserID:   uuid.New(),
		RoleName: "student",
		Actor:    uuid.New(),
	})
	if err != nil {
		t.Fatalf("AssignRole (idempotent): unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("AssignRole (idempotent): rows = %d, want 0", n)
	}
}

func TestRepository_AssignRole_TranslatesDBError(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{assignRoleErr: &pgconn.PgError{Code: "23503"}}
	repo := newTestRepo(q)

	_, err := repo.AssignRole(context.Background(), iam.AssignRoleParams{
		UserID:   uuid.New(),
		RoleName: "student",
		Actor:    uuid.New(),
	})
	if !errors.Is(err, iam.ErrInvalidInput) {
		t.Errorf("AssignRole (23503): got %v, want ErrInvalidInput", err)
	}
}

// --- CountAdmins repository unit test ---

func TestRepository_CountAdmins_DelegatesToQuerier(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{countAdminsResult: 3}
	repo := newTestRepo(q)

	n, err := repo.CountAdmins(context.Background())
	if err != nil {
		t.Fatalf("CountAdmins: unexpected error: %v", err)
	}
	if !q.countAdminsCalled {
		t.Error("CountAdmins: fakeQuerier.CountAdmins was not called")
	}
	if n != 3 {
		t.Errorf("CountAdmins: got %d, want 3", n)
	}
}

// --- InsertAuditLog repository unit test ---

func TestRepository_InsertAuditLog_DelegatesToQuerier(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{}
	repo := newTestRepo(q)

	detail, _ := json.Marshal(map[string]string{"role": "admin"})
	err := repo.InsertAuditLog(context.Background(), iam.AuditLogParams{
		ActorID:  uuid.New(),
		Action:   "role.assign",
		Entity:   "users",
		EntityID: uuid.New(),
		Detail:   detail,
	})
	if err != nil {
		t.Fatalf("InsertAuditLog: unexpected error: %v", err)
	}
	if !q.insertAuditLogCalled {
		t.Error("InsertAuditLog: fakeQuerier.InsertAuditLog was not called")
	}
}
