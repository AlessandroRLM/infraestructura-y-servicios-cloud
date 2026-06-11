package auditlogs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auditlogs/auditlogsdb"
)

// fakeRepository is a test double for Repository with an explicit `called` sentinel.
type fakeRepository struct {
	called    bool
	rows      []auditlogsdb.AuditLog
	err       error
	gotParams ListParams
}

func (f *fakeRepository) ListAuditLogs(_ context.Context, params ListParams) ([]auditlogsdb.AuditLog, error) {
	f.called = true
	f.gotParams = params
	return f.rows, f.err
}

// makeAuditLogRow creates a test AuditLog row with a valid UUID and timestamp.
func makeAuditLogRow(id uuid.UUID) auditlogsdb.AuditLog {
	return auditlogsdb.AuditLog{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		ActorID:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Action:    "grade.update",
		Entity:    "grades",
		EntityID:  pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Detail:    []byte(`{"old_value":"5.0","new_value":"6.0"}`),
		CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
}

// makeRows returns n audit log rows with sequential UUIDs.
func makeRows(n int) []auditlogsdb.AuditLog {
	rows := make([]auditlogsdb.AuditLog, n)
	for i := range rows {
		rows[i] = makeAuditLogRow(uuid.New())
	}
	return rows
}

// TestService_ListAuditLogs_Delegates verifies that a well-formed request delegates
// to the repo with correct filters and converts rows to proto.
func TestService_ListAuditLogs_Delegates(t *testing.T) {
	t.Parallel()

	entityID := uuid.New()
	row := makeAuditLogRow(uuid.New())
	repo := &fakeRepository{rows: []auditlogsdb.AuditLog{row}}
	svc := NewService(repo)

	resp, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: entityID.String(),
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called {
		t.Error("repo.ListAuditLogs must be called")
	}
	if len(resp.Logs) != 1 {
		t.Errorf("expected 1 log row, got %d", len(resp.Logs))
	}
}

// TestService_ListAuditLogs_PageSizeUnset_ClampsTo20 verifies that an unset (zero) page_size
// causes the repo to receive row_limit = 21 (minimum 20 + 1).
func TestService_ListAuditLogs_PageSizeUnset_ClampsTo20(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{rows: makeRows(0)}
	svc := NewService(repo)

	_, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
		PageSize: 0, // proto zero-value = unset
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotParams.RowLimit != 21 {
		t.Errorf("RowLimit = %d, want 21 (clamp 0→20, +1 lookahead)", repo.gotParams.RowLimit)
	}
}

// TestService_ListAuditLogs_PageSizeNegative_ClampsTo20 verifies that a negative page_size
// is clamped to 20 (row_limit = 21).
func TestService_ListAuditLogs_PageSizeNegative_ClampsTo20(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{rows: makeRows(0)}
	svc := NewService(repo)

	_, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
		PageSize: -1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotParams.RowLimit != 21 {
		t.Errorf("RowLimit = %d, want 21 (clamp -1→20, +1 lookahead)", repo.gotParams.RowLimit)
	}
}

// TestService_ListAuditLogs_PageSizeAbove200_ClampsTo200 verifies that an oversized page_size
// is clamped to 200 (row_limit = 201).
func TestService_ListAuditLogs_PageSizeAbove200_ClampsTo200(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{rows: makeRows(0)}
	svc := NewService(repo)

	_, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
		PageSize: 999,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotParams.RowLimit != 201 {
		t.Errorf("RowLimit = %d, want 201 (clamp 999→200, +1 lookahead)", repo.gotParams.RowLimit)
	}
}

// TestService_ListAuditLogs_HasNextPage_TrimsExtraRow verifies that when the repo returns
// pageSize+1 rows, the response has exactly pageSize rows and next_page_token is set to the
// id of the last retained row (index pageSize-1), NOT the trimmed extra row.
func TestService_ListAuditLogs_HasNextPage_TrimsExtraRow(t *testing.T) {
	t.Parallel()

	// page_size = 20 → repo returns 21 rows → response has 20, token = row[19].ID
	rows := makeRows(21)
	repo := &fakeRepository{rows: rows}
	svc := NewService(repo)

	resp, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Logs) != 20 {
		t.Errorf("expected 20 logs (trimmed), got %d", len(resp.Logs))
	}
	// next_page_token must be the id of the last retained row (index 19)
	wantToken := uuid.UUID(rows[19].ID.Bytes).String()
	if resp.NextPageToken != wantToken {
		t.Errorf("NextPageToken = %q, want %q (last retained row id)", resp.NextPageToken, wantToken)
	}
}

// TestService_ListAuditLogs_LastPage_EmptyToken verifies that when the repo returns ≤ pageSize
// rows, next_page_token is an empty string.
func TestService_ListAuditLogs_LastPage_EmptyToken(t *testing.T) {
	t.Parallel()

	// page_size = 20 → repo returns exactly 20 rows → no next page
	rows := makeRows(20)
	repo := &fakeRepository{rows: rows}
	svc := NewService(repo)

	resp, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextPageToken != "" {
		t.Errorf("NextPageToken = %q, want empty (last page)", resp.NextPageToken)
	}
}

// TestService_ListAuditLogs_NullActor_AbsentInResponse verifies that a row with
// ActorID.Valid = false is mapped to actor_id = "" (not the zero UUID).
func TestService_ListAuditLogs_NullActor_AbsentInResponse(t *testing.T) {
	t.Parallel()

	row := makeAuditLogRow(uuid.New())
	row.ActorID = pgtype.UUID{Valid: false} // NULL actor

	repo := &fakeRepository{rows: []auditlogsdb.AuditLog{row}}
	svc := NewService(repo)

	resp, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Logs) == 0 {
		t.Fatal("expected at least 1 log row")
	}
	actorID := resp.Logs[0].ActorId
	zeroUUID := "00000000-0000-0000-0000-000000000000"
	if actorID == zeroUUID {
		t.Errorf("actor_id must be empty string for NULL actor, got zero UUID")
	}
	if actorID != "" {
		t.Errorf("actor_id must be empty string for NULL actor, got %q", actorID)
	}
}

// TestService_ListAuditLogs_NullDetail_AbsentInResponse verifies that a row with
// Detail = nil/empty maps to detail = "" (not the literal "null").
func TestService_ListAuditLogs_NullDetail_AbsentInResponse(t *testing.T) {
	t.Parallel()

	row := makeAuditLogRow(uuid.New())
	row.Detail = nil // NULL detail

	repo := &fakeRepository{rows: []auditlogsdb.AuditLog{row}}
	svc := NewService(repo)

	resp, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Logs) == 0 {
		t.Fatal("expected at least 1 log row")
	}
	detail := resp.Logs[0].Detail
	if detail == "null" {
		t.Errorf("detail must be empty string for NULL, not literal %q", detail)
	}
	if detail != "" {
		t.Errorf("detail must be empty string for NULL detail, got %q", detail)
	}
}

// TestService_ListAuditLogs_EmptyEntity_ReturnsErrInvalidInput verifies that an empty
// entity field returns ErrInvalidInput without calling the repo.
func TestService_ListAuditLogs_EmptyEntity_ReturnsErrInvalidInput(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "",
		EntityId: uuid.New().String(),
	})
	if err == nil {
		t.Fatal("expected error for empty entity, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
	if repo.called {
		t.Error("repo must NOT be called when entity is empty")
	}
}

// TestService_ListAuditLogs_BadCreatedFrom_ReturnsErrInvalidInput verifies that a
// non-RFC3339 created_from string returns ErrInvalidInput.
func TestService_ListAuditLogs_BadCreatedFrom_ReturnsErrInvalidInput(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	svc := NewService(repo)

	_, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:      "grades",
		EntityId:    uuid.New().String(),
		CreatedFrom: "not-a-date",
	})
	if err == nil {
		t.Fatal("expected error for bad created_from, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

// TestService_ListAuditLogs_RepoError_Propagates verifies that a repo error is
// propagated without attempting proto conversion.
func TestService_ListAuditLogs_RepoError_Propagates(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db failure")
	repo := &fakeRepository{err: repoErr}
	svc := NewService(repo)

	_, err := svc.ListAuditLogs(context.Background(), &auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
	})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error to propagate, got: %v", err)
	}
}
