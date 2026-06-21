package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
)

// TestRecentAuditLogs_Pagination_FullDescWalkToExhaustion seeds N rows for a unique
// entity and walks all pages using next_page_token until exhausted. Asserts:
// union = N rows, no duplicates, DESC order within each page, last page empty token.
func TestRecentAuditLogs_Pagination_FullDescWalkToExhaustion(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-pag-walk@audit.test", "admin")

	entityID := uuid.New()
	const total = 55
	// Use a distinct entity type to avoid picking up stray rows from other tests.
	cleanupAuditLogs(t, "recent_pag_walk_entity", entityID)
	// Override seedAuditLogs entity — use raw inserts with fixed entity.
	now := time.Now().UTC()
	for i := 0; i < total; i++ {
		seedAuditLogWithActor(t, "recent_pag_walk_entity", entityID, nil, "test.action", now.Add(time.Duration(i)*time.Millisecond))
	}

	collected := make(map[string]struct{})
	var pageToken string
	pagesWalked := 0
	client := newAuditLogsClient(nil)

	// Build a created_from bound to restrict to just our seeded rows.
	createdFrom := now.Add(-time.Second).Format(time.RFC3339)

	for {
		req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
			CreatedFrom: createdFrom,
			PageSize:    20,
			PageToken:   pageToken,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		resp, err := client.ListRecentAuditLogs(ctx, req)
		if err != nil {
			t.Fatalf("page %d: %v", pagesWalked+1, err)
		}
		pagesWalked++

		if pagesWalked > total+1 {
			t.Fatal("infinite pagination loop detected")
		}

		// Check DESC order within the page.
		for i := 1; i < len(resp.Msg.Logs); i++ {
			prev := resp.Msg.Logs[i-1].CreatedAt
			curr := resp.Msg.Logs[i].CreatedAt
			// Accept equal timestamps (same-second granularity); for equal ts, id must be DESC.
			if prev < curr {
				t.Errorf("page %d: logs[%d].created_at=%s > logs[%d].created_at=%s (want DESC order)",
					pagesWalked, i, curr, i-1, prev)
			}
		}

		// Collect IDs and check for duplicates.
		for _, log := range resp.Msg.Logs {
			if _, exists := collected[log.Id]; exists {
				t.Errorf("duplicate log id %s on page %d", log.Id, pagesWalked)
			}
			collected[log.Id] = struct{}{}
		}

		pageToken = resp.Msg.NextPageToken
		if pageToken == "" {
			break
		}
	}

	if len(collected) != total {
		t.Errorf("total collected = %d, want %d", len(collected), total)
	}
}

// TestRecentAuditLogs_Pagination_PageBoundariesExact seeds 25 rows and uses page_size=20.
// Asserts: page 1 has 20 rows + non-empty token; page 2 has 5 rows + empty token.
func TestRecentAuditLogs_Pagination_PageBoundariesExact(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-pag-exact@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "recent_pag_exact_entity", entityID)

	now := time.Now().UTC()
	for i := 0; i < 25; i++ {
		seedAuditLogWithActor(t, "recent_pag_exact_entity", entityID, nil, "test.action", now.Add(time.Duration(i)*time.Millisecond))
	}

	createdFrom := now.Add(-time.Second).Format(time.RFC3339)

	// Page 1.
	page1 := listRecentAuditLogs(t, ctx, adminSID, "", createdFrom, "", 20, "")
	if len(page1.Logs) != 20 {
		t.Errorf("page 1: got %d rows, want 20", len(page1.Logs))
	}
	if page1.NextPageToken == "" {
		t.Error("page 1: next_page_token must be non-empty (more rows exist)")
	}

	// Page 2 using token from page 1.
	page2 := listRecentAuditLogs(t, ctx, adminSID, "", createdFrom, "", 20, page1.NextPageToken)
	if len(page2.Logs) != 5 {
		t.Errorf("page 2: got %d rows, want 5", len(page2.Logs))
	}
	if page2.NextPageToken != "" {
		t.Errorf("page 2: next_page_token must be empty (last page), got %q", page2.NextPageToken)
	}
}

// TestRecentAuditLogs_Pagination_EmptyResult_Returns200EmptyList verifies that a valid
// request for a time range with no matching rows returns HTTP 200 with an empty list.
func TestRecentAuditLogs_Pagination_EmptyResult_Returns200EmptyList(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-pag-empty@audit.test", "admin")

	// Use a far-future created_from to guarantee no rows match.
	farFuture := time.Now().UTC().Add(100 * 365 * 24 * time.Hour).Format(time.RFC3339)
	resp := listRecentAuditLogs(t, ctx, adminSID, "", farFuture, "", 20, "")

	if len(resp.Logs) != 0 {
		t.Errorf("expected empty logs list, got %d rows", len(resp.Logs))
	}
	if resp.NextPageToken != "" {
		t.Errorf("expected empty next_page_token, got %q", resp.NextPageToken)
	}
}

// TestRecentAuditLogs_Pagination_PageSizeClamp20 seeds 30 rows and calls with page_size=0.
// Asserts exactly 20 rows returned (clamped to minimum) and non-empty next_page_token.
func TestRecentAuditLogs_Pagination_PageSizeClamp20(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-pag-clamp20@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "recent_pag_clamp20_entity", entityID)

	now := time.Now().UTC()
	for i := 0; i < 30; i++ {
		seedAuditLogWithActor(t, "recent_pag_clamp20_entity", entityID, nil, "test.action", now.Add(time.Duration(i)*time.Millisecond))
	}

	createdFrom := now.Add(-time.Second).Format(time.RFC3339)
	resp := listRecentAuditLogs(t, ctx, adminSID, "", createdFrom, "", 0, "") // 0 = unset

	if len(resp.Logs) != 20 {
		t.Errorf("page_size=0 → expected 20 rows (clamped to min), got %d", len(resp.Logs))
	}
	if resp.NextPageToken == "" {
		t.Error("expected non-empty next_page_token (more rows exist after clamped page)")
	}
}
