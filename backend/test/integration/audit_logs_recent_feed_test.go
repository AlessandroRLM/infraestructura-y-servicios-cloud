package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
)

// listRecentAuditLogs calls ListRecentAuditLogs as admin with the given parameters.
func listRecentAuditLogs(
	t *testing.T,
	ctx context.Context,
	adminSID string,
	actorID string,
	createdFrom string,
	createdTo string,
	pageSize int32,
	pageToken string,
) *auditlogsv1.ListRecentAuditLogsResponse {
	t.Helper()
	client := newAuditLogsClient(nil)
	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		ActorId:     actorID,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		PageSize:    pageSize,
		PageToken:   pageToken,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListRecentAuditLogs(ctx, req)
	if err != nil {
		t.Fatalf("ListRecentAuditLogs: %v", err)
	}
	return resp.Msg
}

// TestRecentAuditLogs_Feed_NewestFirst verifies that the global feed returns rows
// in created_at DESC order (newest rows appear first).
func TestRecentAuditLogs_Feed_NewestFirst(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-order@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	now := time.Now().UTC()
	older := now.Add(-2 * time.Second)
	newer := now

	// Insert older first, then newer — response must return newer first.
	olderID := seedAuditLogWithActor(t, "grades", entityID, nil, "older.action", older)
	newerID := seedAuditLogWithActor(t, "grades", entityID, nil, "newer.action", newer)

	resp := listRecentAuditLogs(t, ctx, adminSID, "", "", "", 200, "")

	// Find the two seeded rows in the response.
	var gotOlder, gotNewer int
	var olderPos, newerPos int
	for i, log := range resp.Logs {
		if log.Id == olderID {
			gotOlder++
			olderPos = i
		}
		if log.Id == newerID {
			gotNewer++
			newerPos = i
		}
	}

	if gotNewer == 0 || gotOlder == 0 {
		t.Fatalf("expected both seeded rows in response (newer=%d, older=%d)", gotNewer, gotOlder)
	}
	// newerPos must be < olderPos (newer appears earlier in the DESC result).
	if newerPos >= olderPos {
		t.Errorf("newer row at position %d, older row at position %d; want newer first (DESC order)",
			newerPos, olderPos)
	}
}

// TestRecentAuditLogs_Feed_GlobalScope verifies that the feed returns rows from
// multiple distinct entities (not scoped to a single entity+entity_id).
func TestRecentAuditLogs_Feed_GlobalScope(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-global@audit.test", "admin")

	entityIDA := uuid.New()
	entityIDB := uuid.New()
	cleanupAuditLogs(t, "sections", entityIDA)
	cleanupAuditLogs(t, "enrollments", entityIDB)

	now := time.Now().UTC()
	idA := seedAuditLogWithActor(t, "sections", entityIDA, nil, "section.action", now)
	idB := seedAuditLogWithActor(t, "enrollments", entityIDB, nil, "enrollment.action", now.Add(time.Millisecond))

	resp := listRecentAuditLogs(t, ctx, adminSID, "", "", "", 200, "")

	gotA, gotB := false, false
	for _, log := range resp.Logs {
		if log.Id == idA {
			gotA = true
		}
		if log.Id == idB {
			gotB = true
		}
	}
	if !gotA || !gotB {
		t.Errorf("global feed must contain rows from both entities: gotA=%v gotB=%v", gotA, gotB)
	}
}

// TestRecentAuditLogs_Feed_ActorFilter_OnlyMatchingRows seeds rows for actor A, actor B,
// and NULL actor; filters by actor A; asserts only A rows are returned.
func TestRecentAuditLogs_Feed_ActorFilter_OnlyMatchingRows(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-actor@audit.test", "admin")

	actorA, _ := seedUserWithSession(t, "recent-feed-actor-a@audit.test", "admin")
	actorB, _ := seedUserWithSession(t, "recent-feed-actor-b@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	now := time.Now().UTC()
	idA1 := seedAuditLogWithActor(t, "grades", entityID, &actorA, "grade.update", now)
	idA2 := seedAuditLogWithActor(t, "grades", entityID, &actorA, "grade.update", now.Add(time.Millisecond))
	_ = seedAuditLogWithActor(t, "grades", entityID, &actorB, "grade.update", now.Add(2*time.Millisecond))
	_ = seedAuditLogWithActor(t, "grades", entityID, nil, "grade.update", now.Add(3*time.Millisecond))

	resp := listRecentAuditLogs(t, ctx, adminSID, actorA.String(), "", "", 200, "")

	// Verify that ALL returned rows (from this entity) belong to actorA.
	for _, log := range resp.Logs {
		if log.EntityId == entityID.String() {
			if log.ActorId != actorA.String() {
				t.Errorf("actor filter violation: row %s has actor_id=%s, want %s",
					log.Id, log.ActorId, actorA.String())
			}
		}
	}

	// Both actorA rows must be present.
	found := make(map[string]bool)
	for _, log := range resp.Logs {
		found[log.Id] = true
	}
	if !found[idA1] || !found[idA2] {
		t.Errorf("both actor A rows must be present: idA1=%v idA2=%v", found[idA1], found[idA2])
	}
}

// TestRecentAuditLogs_Feed_NoActorFilter_IncludesNullActor verifies that without an
// actor_id filter, rows with NULL actor are included in the global feed.
func TestRecentAuditLogs_Feed_NoActorFilter_IncludesNullActor(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-nullactor@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	nullID := seedAuditLogWithActor(t, "grades", entityID, nil, "system.action", time.Now().UTC())

	resp := listRecentAuditLogs(t, ctx, adminSID, "", "", "", 200, "")

	found := false
	for _, log := range resp.Logs {
		if log.Id == nullID {
			found = true
			if log.ActorId != "" {
				t.Errorf("NULL actor must map to empty string, got %q", log.ActorId)
			}
		}
	}
	if !found {
		t.Error("NULL-actor row must appear in global feed when no actor_id filter is set")
	}
}

// TestRecentAuditLogs_Feed_DateRange_LowerBound verifies that created_from=T
// returns only rows where created_at >= T.
func TestRecentAuditLogs_Feed_DateRange_LowerBound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-from@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	base := time.Now().UTC().Truncate(time.Second)
	before := base.Add(-5 * time.Second)
	boundary := base

	beforeID := seedAuditLogWithActor(t, "grades", entityID, nil, "before", before)
	atID := seedAuditLogWithActor(t, "grades", entityID, nil, "at.boundary", boundary)
	afterID := seedAuditLogWithActor(t, "grades", entityID, nil, "after", boundary.Add(time.Second))

	resp := listRecentAuditLogs(t, ctx, adminSID, "", boundary.Format(time.RFC3339), "", 200, "")

	found := map[string]bool{}
	for _, log := range resp.Logs {
		found[log.Id] = true
	}

	if found[beforeID] {
		t.Error("row before created_from boundary must be excluded")
	}
	if !found[atID] {
		t.Error("row exactly at created_from boundary must be included (inclusive)")
	}
	if !found[afterID] {
		t.Error("row after created_from boundary must be included")
	}
}

// TestRecentAuditLogs_Feed_DateRange_ClosedBothEnds verifies that both created_from and
// created_to are inclusive.
func TestRecentAuditLogs_Feed_DateRange_ClosedBothEnds(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-range@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	base := time.Now().UTC().Truncate(time.Second)
	t1 := base
	t2 := base.Add(6 * time.Second)

	beforeID := seedAuditLogWithActor(t, "grades", entityID, nil, "before", t1.Add(-2*time.Second))
	atT1ID := seedAuditLogWithActor(t, "grades", entityID, nil, "at.t1", t1)
	midID := seedAuditLogWithActor(t, "grades", entityID, nil, "mid", t1.Add(3*time.Second))
	atT2ID := seedAuditLogWithActor(t, "grades", entityID, nil, "at.t2", t2)
	afterID := seedAuditLogWithActor(t, "grades", entityID, nil, "after", t2.Add(2*time.Second))

	resp := listRecentAuditLogs(t, ctx, adminSID, "",
		t1.Format(time.RFC3339), t2.Format(time.RFC3339), 200, "")

	found := map[string]bool{}
	for _, log := range resp.Logs {
		found[log.Id] = true
	}

	if found[beforeID] {
		t.Error("row before t1 must be excluded")
	}
	if !found[atT1ID] {
		t.Error("row at t1 must be included (inclusive lower bound)")
	}
	if !found[midID] {
		t.Error("row between t1 and t2 must be included")
	}
	if !found[atT2ID] {
		t.Error("row at t2 must be included (inclusive upper bound)")
	}
	if found[afterID] {
		t.Error("row after t2 must be excluded")
	}
}

// TestRecentAuditLogs_Feed_MalformedActorID_ReturnsInvalidArgument verifies that a
// non-UUID actor_id is rejected with CodeInvalidArgument.
func TestRecentAuditLogs_Feed_MalformedActorID_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-badactor@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		ActorId: "not-a-uuid",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListRecentAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestRecentAuditLogs_Feed_MalformedCreatedFrom_ReturnsInvalidArgument verifies that a
// non-RFC3339 created_from is rejected with CodeInvalidArgument.
func TestRecentAuditLogs_Feed_MalformedCreatedFrom_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-badfrom@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		CreatedFrom: "not-a-date",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListRecentAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestRecentAuditLogs_Feed_MalformedPageToken_ReturnsInvalidArgument verifies that a
// malformed page_token is rejected with CodeInvalidArgument.
func TestRecentAuditLogs_Feed_MalformedPageToken_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "recent-feed-badtoken@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListRecentAuditLogsRequest{
		PageToken: "not-a-cursor",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListRecentAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}
