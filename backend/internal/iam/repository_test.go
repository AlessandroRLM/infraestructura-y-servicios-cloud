package iam_test

import (
	"context"
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

// --- Repository unit tests ---

func TestRepository_ListUsers_DelegatesToQuerier(t *testing.T) {
	t.Parallel()

	expected := []iamdb.ListUsersRow{
		{ID: pgtype.UUID{Bytes: uuid.New(), Valid: true}, Email: "alice@example.com"},
	}
	q := &fakeQuerier{listUsersRows: expected}
	repo := iam.NewPostgresRepository(q)

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
	repo := iam.NewPostgresRepository(q)

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
	repo := iam.NewPostgresRepository(q)

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
	repo := iam.NewPostgresRepository(q)

	_, err := repo.GetUserByID(context.Background(), uuid.New())
	if !errors.Is(err, iam.ErrNotFound) {
		t.Errorf("GetUserByID (no rows): got %v, want ErrNotFound", err)
	}
}

func TestRepository_GetUserByID_TranslatesDBError(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{getUserByIDErr: &pgconn.PgError{Code: "23503"}}
	repo := iam.NewPostgresRepository(q)

	_, err := repo.GetUserByID(context.Background(), uuid.New())
	if !errors.Is(err, iam.ErrInvalidInput) {
		t.Errorf("GetUserByID (23503): got %v, want ErrInvalidInput", err)
	}
}

func TestRepository_GetUserRoles_DelegatesToQuerier(t *testing.T) {
	t.Parallel()

	expected := []string{"admin", "teacher"}
	q := &fakeQuerier{getUserRolesResult: expected}
	repo := iam.NewPostgresRepository(q)

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

	// When the querier returns nil (user has no roles), the repo returns an empty slice.
	q := &fakeQuerier{getUserRolesResult: nil}
	repo := iam.NewPostgresRepository(q)

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
	repo := iam.NewPostgresRepository(q)

	_, err := repo.GetUserRoles(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("GetUserRoles (db error): expected error, got nil")
	}
	if !errors.Is(err, inner) {
		t.Errorf("GetUserRoles (db error): original error not wrapped, got %v", err)
	}
}
