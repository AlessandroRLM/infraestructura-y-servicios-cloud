package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
)

// seedAuditLogs inserts n audit_logs rows for the given entity+entity_id directly via SQL.
// Returns the seeded row IDs (as UUID strings, oldest-first by insertion order).
// Registers t.Cleanup to delete the rows (FK-safe: audit_logs has no children).
func seedAuditLogs(t *testing.T, entity string, entityID uuid.UUID, n int) []string {
	t.Helper()
	ctx := context.Background()

	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		var id uuid.UUID
		err := pgxPool.QueryRow(ctx,
			`INSERT INTO audit_logs (action, entity, entity_id, created_at)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			fmt.Sprintf("test.action.%d", i),
			entity,
			entityID,
			// Spread timestamps to ensure distinct ordering.
			time.Now().UTC().Add(time.Duration(i)*time.Millisecond),
		).Scan(&id)
		if err != nil {
			t.Fatalf("seedAuditLogs: insert row %d: %v", i, err)
		}
		ids = append(ids, id.String())
	}

	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM audit_logs WHERE entity = $1 AND entity_id = $2`,
			entity, entityID,
		)
	})

	return ids
}

// listAuditLogsAdmin is a convenience wrapper that calls ListAuditLogs as admin.
func listAuditLogsAdmin(
	t *testing.T,
	ctx context.Context,
	adminSID string,
	entity string,
	entityID uuid.UUID,
	pageSize int32,
	pageToken string,
) *auditlogsv1.ListAuditLogsResponse {
	t.Helper()
	client := newAuditLogsClient(nil)
	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:    entity,
		EntityId:  entityID.String(),
		PageSize:  pageSize,
		PageToken: pageToken,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListAuditLogs(ctx, req)
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	return resp.Msg
}

// TestAuditLogs_Pagination_FullDescWalkToExhaustion seeds N rows and walks all pages
// using next_page_token until it is empty. Asserts: union = N rows, no duplicates,
// no gaps, DESC id order within each page, last page has empty next_page_token.
func TestAuditLogs_Pagination_FullDescWalkToExhaustion(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-pag-walk@audit.test", "admin")

	entityID := uuid.New()
	const total = 55
	seedAuditLogs(t, "grades", entityID, total)

	collected := make(map[string]struct{})
	var pageToken string
	pagesWalked := 0

	for {
		resp := listAuditLogsAdmin(t, ctx, adminSID, "grades", entityID, 20, pageToken)
		pagesWalked++

		if pagesWalked > total+1 {
			t.Fatal("infinite pagination loop detected")
		}

		// Check DESC order within the page.
		for i := 1; i < len(resp.Logs); i++ {
			prev := resp.Logs[i-1].Id
			curr := resp.Logs[i].Id
			if prev <= curr {
				t.Errorf("page %d: logs[%d].id=%s >= logs[%d].id=%s (want DESC order)",
					pagesWalked, i-1, prev, i, curr)
			}
		}

		// Collect IDs and check for duplicates.
		for _, log := range resp.Logs {
			if _, exists := collected[log.Id]; exists {
				t.Errorf("duplicate log id %s on page %d", log.Id, pagesWalked)
			}
			collected[log.Id] = struct{}{}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	if len(collected) != total {
		t.Errorf("total collected = %d, want %d", len(collected), total)
	}
}

// TestAuditLogs_Pagination_PageBoundariesExact seeds 25 rows with page_size=20.
// Asserts: page 1 has 20 rows + non-empty token; page 2 has 5 rows + empty token.
func TestAuditLogs_Pagination_PageBoundariesExact(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-pag-exact@audit.test", "admin")

	entityID := uuid.New()
	seedAuditLogs(t, "grades", entityID, 25)

	// Page 1.
	page1 := listAuditLogsAdmin(t, ctx, adminSID, "grades", entityID, 20, "")
	if len(page1.Logs) != 20 {
		t.Errorf("page 1: got %d rows, want 20", len(page1.Logs))
	}
	if page1.NextPageToken == "" {
		t.Error("page 1: next_page_token must be non-empty (more rows exist)")
	}

	// Page 2 using token from page 1.
	page2 := listAuditLogsAdmin(t, ctx, adminSID, "grades", entityID, 20, page1.NextPageToken)
	if len(page2.Logs) != 5 {
		t.Errorf("page 2: got %d rows, want 5", len(page2.Logs))
	}
	if page2.NextPageToken != "" {
		t.Errorf("page 2: next_page_token must be empty (last page), got %q", page2.NextPageToken)
	}
}

// TestAuditLogs_Pagination_EmptyResult_Returns200EmptyList verifies that a valid request
// for an entity+entity_id with zero rows returns HTTP 200 with an empty list and empty token.
// CodeNotFound must NOT be returned.
func TestAuditLogs_Pagination_EmptyResult_Returns200EmptyList(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-pag-empty@audit.test", "admin")

	// Use a random entity_id with no seeded rows.
	entityID := uuid.New()
	resp := listAuditLogsAdmin(t, ctx, adminSID, "sections", entityID, 20, "")

	if len(resp.Logs) != 0 {
		t.Errorf("expected empty logs list, got %d rows", len(resp.Logs))
	}
	if resp.NextPageToken != "" {
		t.Errorf("expected empty next_page_token, got %q", resp.NextPageToken)
	}
}

// TestAuditLogs_Pagination_PageSizeUnset_Clamps20 seeds 30 rows and calls with page_size=0.
// Asserts exactly 20 rows returned (clamped to minimum) and non-empty next_page_token.
func TestAuditLogs_Pagination_PageSizeUnset_Clamps20(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-pag-clamp20@audit.test", "admin")

	entityID := uuid.New()
	seedAuditLogs(t, "grades", entityID, 30)

	resp := listAuditLogsAdmin(t, ctx, adminSID, "grades", entityID, 0, "") // 0 = unset
	if len(resp.Logs) != 20 {
		t.Errorf("page_size=0 → expected 20 rows (clamped to min), got %d", len(resp.Logs))
	}
	if resp.NextPageToken == "" {
		t.Error("expected non-empty next_page_token (more rows exist after clamped page)")
	}
}

// TestAuditLogs_Pagination_PageSizeAbove200_Clamps200 seeds 250 rows and calls with page_size=999.
// Asserts exactly 200 rows returned (clamped to maximum) and non-empty next_page_token.
func TestAuditLogs_Pagination_PageSizeAbove200_Clamps200(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-pag-clamp200@audit.test", "admin")

	entityID := uuid.New()
	seedAuditLogs(t, "grades", entityID, 250)

	resp := listAuditLogsAdmin(t, ctx, adminSID, "grades", entityID, 999, "")
	if len(resp.Logs) != 200 {
		t.Errorf("page_size=999 → expected 200 rows (clamped to max), got %d", len(resp.Logs))
	}
	if resp.NextPageToken == "" {
		t.Error("expected non-empty next_page_token (more rows exist after clamped page)")
	}
}

// TestAuditLogs_Pagination_Scenario23_CursorStabilityUnderInserts is a deterministic
// sequential test that verifies cursor stability when new rows are inserted between pages.
//
//  1. Seed N "old" rows for entity+entity_id.
//  2. Request page 1 → receive up to page_size rows + next_page_token=C.
//  3. INSERT M "new" rows for the SAME entity+entity_id (their UUIDv7 ids will be > all old ids).
//  4. Request page 2 with page_token=C.
//  5. Assert: no overlap with page 1; no old rows skipped; new rows NOT visible on page 2
//     (because ORDER BY id DESC with id < C excludes ids > C).
func TestAuditLogs_Pagination_Scenario23_CursorStabilityUnderInserts(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-pag-cursor-stable@audit.test", "admin")

	entityID := uuid.New()
	const oldCount = 25
	const pageSize = 20
	const newCount = 5

	// Step 1: seed old rows.
	seedAuditLogs(t, "grades", entityID, oldCount)

	// Step 2: page 1 — expect 20 rows + a cursor.
	client := newAuditLogsClient(nil)
	req1 := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: entityID.String(),
		PageSize: pageSize,
	})
	req1.Header().Set("Cookie", "sid="+adminSID)
	resp1, err := client.ListAuditLogs(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(resp1.Msg.Logs) != pageSize {
		t.Fatalf("page 1: expected %d rows, got %d", pageSize, len(resp1.Msg.Logs))
	}
	cursor := resp1.Msg.NextPageToken
	if cursor == "" {
		t.Fatal("page 1: expected non-empty next_page_token")
	}

	// Collect page 1 IDs.
	page1IDs := make(map[string]struct{}, pageSize)
	for _, log := range resp1.Msg.Logs {
		page1IDs[log.Id] = struct{}{}
	}

	// Step 3: insert new rows AFTER page 1 is received.
	// These rows have higher UUIDv7 ids than the old rows and appear logically BEFORE page 1
	// in DESC order. They must NOT appear on page 2 (id < cursor).
	newIDs := seedAuditLogs(t, "grades", entityID, newCount)

	// Step 4: page 2 with cursor from page 1.
	req2 := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:    "grades",
		EntityId:  entityID.String(),
		PageSize:  pageSize,
		PageToken: cursor,
	})
	req2.Header().Set("Cookie", "sid="+adminSID)
	resp2, err := client.ListAuditLogs(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	// Step 5a: no overlap between page 1 and page 2.
	for _, log := range resp2.Msg.Logs {
		if _, inPage1 := page1IDs[log.Id]; inPage1 {
			t.Errorf("cursor stability violation: row %s appears on both page 1 and page 2", log.Id)
		}
	}

	// Step 5b: none of the newly inserted rows should appear on page 2.
	newIDSet := make(map[string]struct{}, len(newIDs))
	for _, id := range newIDs {
		newIDSet[id] = struct{}{}
	}
	for _, log := range resp2.Msg.Logs {
		if _, isNew := newIDSet[log.Id]; isNew {
			t.Errorf("cursor stability violation: newly inserted row %s leaked onto page 2", log.Id)
		}
	}

	// Step 5c: page 2 should have exactly the remaining old rows (oldCount - pageSize = 5).
	const expectedPage2Rows = oldCount - pageSize
	if len(resp2.Msg.Logs) != expectedPage2Rows {
		t.Errorf("page 2: expected %d rows (remaining old batch), got %d", expectedPage2Rows, len(resp2.Msg.Logs))
	}
	if resp2.Msg.NextPageToken != "" {
		t.Errorf("page 2: expected empty next_page_token (last page of old batch), got %q", resp2.Msg.NextPageToken)
	}
}
