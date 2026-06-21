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

// fakeRepositoryRecent extends fakeRepository to also stub ListRecentAuditLogs.
type fakeRepositoryRecent struct {
	fakeRepository

	recentCalled    bool
	recentRows      []auditlogsdb.AuditLog
	recentErr       error
	gotRecentParams ListRecentParams
}

func (f *fakeRepositoryRecent) ListAuditLogs(ctx context.Context, params ListParams) ([]auditlogsdb.AuditLog, error) {
	return f.fakeRepository.ListAuditLogs(ctx, params)
}

func (f *fakeRepositoryRecent) ListRecentAuditLogs(ctx context.Context, params ListRecentParams) ([]auditlogsdb.AuditLog, error) {
	f.recentCalled = true
	f.gotRecentParams = params
	return f.recentRows, f.recentErr
}

// TestService_ListRecentAuditLogs_Delegates verifies that a well-formed request delegates
// to the repo with correct filters and converts rows to proto.
func TestService_ListRecentAuditLogs_Delegates(t *testing.T) {
	t.Parallel()

	row := makeAuditLogRow(uuid.New())
	repo := &fakeRepositoryRecent{recentRows: []auditlogsdb.AuditLog{row}}
	svc := NewService(repo)

	resp, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.recentCalled {
		t.Error("repo.ListRecentAuditLogs must be called")
	}
	if len(resp.Logs) != 1 {
		t.Errorf("expected 1 log row, got %d", len(resp.Logs))
	}
}

// TestService_ListRecentAuditLogs_PageSizeUnset_ClampsTo20 verifies that an unset (zero)
// page_size causes the repo to receive row_limit = 21 (minimum 20 + 1).
func TestService_ListRecentAuditLogs_PageSizeUnset_ClampsTo20(t *testing.T) {
	t.Parallel()

	repo := &fakeRepositoryRecent{}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotRecentParams.RowLimit != 21 {
		t.Errorf("RowLimit = %d, want 21 (clamp 0→20, +1 lookahead)", repo.gotRecentParams.RowLimit)
	}
}

// TestService_ListRecentAuditLogs_PageSizeAbove200_ClampsTo200 verifies that an oversized
// page_size is clamped to 200 (row_limit = 201).
func TestService_ListRecentAuditLogs_PageSizeAbove200_ClampsTo200(t *testing.T) {
	t.Parallel()

	repo := &fakeRepositoryRecent{}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 999,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotRecentParams.RowLimit != 201 {
		t.Errorf("RowLimit = %d, want 201 (clamp 999→200, +1 lookahead)", repo.gotRecentParams.RowLimit)
	}
}

// TestService_ListRecentAuditLogs_HasNextPage_TrimsExtraRow verifies that when the repo
// returns pageSize+1 rows, the response has exactly pageSize rows and next_page_token is set.
func TestService_ListRecentAuditLogs_HasNextPage_TrimsExtraRow(t *testing.T) {
	t.Parallel()

	// page_size = 20 → repo returns 21 rows → response has 20, token = last retained row
	rows := makeRows(21)
	repo := &fakeRepositoryRecent{recentRows: rows}
	svc := NewService(repo)

	resp, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Logs) != 20 {
		t.Errorf("expected 20 logs (trimmed), got %d", len(resp.Logs))
	}
	if resp.NextPageToken == "" {
		t.Error("next_page_token must be non-empty when there is a next page")
	}
}

// TestService_ListRecentAuditLogs_LastPage_EmptyToken verifies that when the repo returns
// ≤ pageSize rows, next_page_token is an empty string.
func TestService_ListRecentAuditLogs_LastPage_EmptyToken(t *testing.T) {
	t.Parallel()

	rows := makeRows(20)
	repo := &fakeRepositoryRecent{recentRows: rows}
	svc := NewService(repo)

	resp, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextPageToken != "" {
		t.Errorf("NextPageToken = %q, want empty (last page)", resp.NextPageToken)
	}
}

// TestService_ListRecentAuditLogs_EmptyActorID_NilInParams verifies that an empty actor_id
// string causes ActorID to be nil in the repo params (no actor filter).
func TestService_ListRecentAuditLogs_EmptyActorID_NilInParams(t *testing.T) {
	t.Parallel()

	repo := &fakeRepositoryRecent{}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		ActorId:  "",
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotRecentParams.ActorID != nil {
		t.Errorf("ActorID must be nil for empty actor_id string, got %v", repo.gotRecentParams.ActorID)
	}
}

// TestService_ListRecentAuditLogs_SetActorID_PassedToRepo verifies that a valid actor_id
// UUID string is parsed and passed as non-nil to the repo.
func TestService_ListRecentAuditLogs_SetActorID_PassedToRepo(t *testing.T) {
	t.Parallel()

	actorID := uuid.New()
	repo := &fakeRepositoryRecent{}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		ActorId:  actorID.String(),
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotRecentParams.ActorID == nil {
		t.Fatal("ActorID must be non-nil for valid actor_id")
	}
	if *repo.gotRecentParams.ActorID != actorID {
		t.Errorf("ActorID = %v, want %v", *repo.gotRecentParams.ActorID, actorID)
	}
}

// TestService_ListRecentAuditLogs_BadActorID_ReturnsErrInvalidInput verifies that a
// non-UUID actor_id returns ErrInvalidInput without calling the repo.
func TestService_ListRecentAuditLogs_BadActorID_ReturnsErrInvalidInput(t *testing.T) {
	t.Parallel()

	repo := &fakeRepositoryRecent{}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		ActorId: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected error for bad actor_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
	if repo.recentCalled {
		t.Error("repo must NOT be called when actor_id is malformed")
	}
}

// TestService_ListRecentAuditLogs_BadCreatedFrom_ReturnsErrInvalidInput verifies that a
// non-RFC3339 created_from string returns ErrInvalidInput.
func TestService_ListRecentAuditLogs_BadCreatedFrom_ReturnsErrInvalidInput(t *testing.T) {
	t.Parallel()

	repo := &fakeRepositoryRecent{}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		CreatedFrom: "not-a-date",
	})
	if err == nil {
		t.Fatal("expected error for bad created_from, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
	if repo.recentCalled {
		t.Error("repo must NOT be called when created_from is malformed")
	}
}

// TestService_ListRecentAuditLogs_BadCreatedTo_ReturnsErrInvalidInput verifies that a
// non-RFC3339 created_to string returns ErrInvalidInput.
func TestService_ListRecentAuditLogs_BadCreatedTo_ReturnsErrInvalidInput(t *testing.T) {
	t.Parallel()

	repo := &fakeRepositoryRecent{}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		CreatedTo: "not-a-date",
	})
	if err == nil {
		t.Fatal("expected error for bad created_to, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
	if repo.recentCalled {
		t.Error("repo must NOT be called when created_to is malformed")
	}
}

// TestService_ListRecentAuditLogs_BadPageToken_ReturnsErrInvalidInput verifies that an
// invalid (non-parseable) page_token returns ErrInvalidInput.
func TestService_ListRecentAuditLogs_BadPageToken_ReturnsErrInvalidInput(t *testing.T) {
	t.Parallel()

	repo := &fakeRepositoryRecent{}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageToken: "not-a-valid-cursor",
	})
	if err == nil {
		t.Fatal("expected error for bad page_token, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
	if repo.recentCalled {
		t.Error("repo must NOT be called when page_token is malformed")
	}
}

// TestService_ListRecentAuditLogs_ValidPageToken_ParsedIntoCursor verifies that a valid
// page_token produced by the service can be round-tripped into cursor params.
func TestService_ListRecentAuditLogs_ValidPageToken_ParsedIntoCursor(t *testing.T) {
	t.Parallel()

	// Produce a page_token from a first-page response.
	rows := makeRows(21) // 21 rows → has next page
	// Set known created_at on last retained row (index 19).
	knownTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	rows[19].CreatedAt = pgtype.Timestamptz{Time: knownTime, Valid: true}

	repo := &fakeRepositoryRecent{recentRows: rows}
	svc := NewService(repo)

	resp, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextPageToken == "" {
		t.Fatal("expected non-empty next_page_token")
	}

	// Now use the token in a second request — should parse without error.
	_, err = svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize:  20,
		PageToken: resp.NextPageToken,
	})
	if err != nil {
		t.Fatalf("valid next_page_token must be accepted in subsequent request: %v", err)
	}
	// Cursor must be passed to repo.
	if repo.gotRecentParams.CursorTs == nil {
		t.Error("cursor_ts must be set in repo params when page_token is provided")
	}
	if repo.gotRecentParams.CursorID == nil {
		t.Error("cursor_id must be set in repo params when page_token is provided")
	}
}

// TestService_ListRecentAuditLogs_RepoError_Propagates verifies that a repo error is
// propagated without attempting proto conversion.
func TestService_ListRecentAuditLogs_RepoError_Propagates(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db failure")
	repo := &fakeRepositoryRecent{recentErr: repoErr}
	svc := NewService(repo)

	_, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error to propagate, got: %v", err)
	}
}

// TestService_ListRecentAuditLogs_NullActor_AbsentInResponse verifies that a row with
// ActorID.Valid = false is mapped to actor_id = "" in the response.
func TestService_ListRecentAuditLogs_NullActor_AbsentInResponse(t *testing.T) {
	t.Parallel()

	row := makeAuditLogRow(uuid.New())
	row.ActorID = pgtype.UUID{Valid: false}
	repo := &fakeRepositoryRecent{recentRows: []auditlogsdb.AuditLog{row}}
	svc := NewService(repo)

	resp, err := svc.ListRecentAuditLogs(context.Background(), &auditlogsv1.ListRecentAuditLogsRequest{
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Logs) == 0 {
		t.Fatal("expected at least 1 log row")
	}
	actorID := resp.Logs[0].ActorId
	if actorID == "00000000-0000-0000-0000-000000000000" {
		t.Error("actor_id must be empty string for NULL actor, got zero UUID")
	}
	if actorID != "" {
		t.Errorf("actor_id must be empty string for NULL actor, got %q", actorID)
	}
}
