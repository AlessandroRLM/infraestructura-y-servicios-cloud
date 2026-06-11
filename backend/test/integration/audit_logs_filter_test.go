package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
)

// seedAuditLogWithActor inserts a single audit_log row with the given actor_id (may be nil for NULL).
// actorID MUST reference a valid users.id due to the FK constraint on audit_logs.actor_id.
// Use seedUserWithSession or seedUserWithRole to obtain a valid UUID.
func seedAuditLogWithActor(t *testing.T, entity string, entityID uuid.UUID, actorID *uuid.UUID, action string, createdAt time.Time) string {
	t.Helper()
	ctx := context.Background()

	var id uuid.UUID
	var err error
	if actorID != nil {
		err = pgxPool.QueryRow(ctx,
			`INSERT INTO audit_logs (action, entity, entity_id, actor_id, created_at)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			action, entity, entityID, *actorID, createdAt,
		).Scan(&id)
	} else {
		err = pgxPool.QueryRow(ctx,
			`INSERT INTO audit_logs (action, entity, entity_id, created_at)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			action, entity, entityID, createdAt,
		).Scan(&id)
	}
	if err != nil {
		t.Fatalf("seedAuditLogWithActor: %v", err)
	}
	return id.String()
}

// seedAuditLogWithDetail inserts a single audit_log row with a JSONB detail.
func seedAuditLogWithDetail(t *testing.T, entity string, entityID uuid.UUID, detail string) string {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pgxPool.QueryRow(ctx,
		`INSERT INTO audit_logs (action, entity, entity_id, detail) VALUES ($1, $2, $3, $4::jsonb) RETURNING id`,
		"test.detail", entity, entityID, detail,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedAuditLogWithDetail: %v", err)
	}
	return id.String()
}

// seedAuditLogNullDetail inserts an audit_log row with a NULL detail column.
func seedAuditLogNullDetail(t *testing.T, entity string, entityID uuid.UUID) string {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pgxPool.QueryRow(ctx,
		`INSERT INTO audit_logs (action, entity, entity_id) VALUES ($1, $2, $3) RETURNING id`,
		"test.null.detail", entity, entityID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedAuditLogNullDetail: %v", err)
	}
	return id.String()
}

// cleanupAuditLogs registers a t.Cleanup to delete all audit_logs for entity+entityID.
func cleanupAuditLogs(t *testing.T, entity string, entityID uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM audit_logs WHERE entity = $1 AND entity_id = $2`,
			entity, entityID,
		)
	})
}

// listAuditLogsFiltered calls ListAuditLogs with all optional filters.
func listAuditLogsFiltered(
	t *testing.T,
	ctx context.Context,
	adminSID string,
	entity string,
	entityID uuid.UUID,
	actorID string,
	createdFrom string,
	createdTo string,
) *auditlogsv1.ListAuditLogsResponse {
	t.Helper()
	client := newAuditLogsClient(nil)
	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:      entity,
		EntityId:    entityID.String(),
		ActorId:     actorID,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		PageSize:    200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListAuditLogs(ctx, req)
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	return resp.Msg
}

// jsonEqual returns true if two JSON strings represent equivalent values,
// ignoring key ordering differences caused by PostgreSQL JSONB normalization.
func jsonEqual(a, b string) bool {
	var va, vb interface{}
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}
	ra, _ := json.Marshal(va)
	rb, _ := json.Marshal(vb)
	return string(ra) == string(rb)
}

// TestAuditLogs_Filter_ByActorID_ReturnsOnlyMatchingRows seeds rows for actor A,
// actor B, and NULL actor; filters by actor A; asserts only A rows are returned.
func TestAuditLogs_Filter_ByActorID_ReturnsOnlyMatchingRows(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-actor@audit.test", "admin")

	// Create two distinct real users to use as actor IDs (FK constraint).
	actorA, _ := seedUserWithSession(t, "audit-filter-actor-a@audit.test", "admin")
	actorB, _ := seedUserWithSession(t, "audit-filter-actor-b@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	now := time.Now().UTC()

	idA1 := seedAuditLogWithActor(t, "grades", entityID, &actorA, "grade.update", now)
	idA2 := seedAuditLogWithActor(t, "grades", entityID, &actorA, "grade.update", now.Add(time.Millisecond))
	_ = seedAuditLogWithActor(t, "grades", entityID, &actorB, "grade.update", now.Add(2*time.Millisecond))
	_ = seedAuditLogWithActor(t, "grades", entityID, nil, "grade.update", now.Add(3*time.Millisecond))

	resp := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityID, actorA.String(), "", "")

	if len(resp.Logs) != 2 {
		t.Errorf("expected 2 rows for actor A, got %d", len(resp.Logs))
	}
	for _, log := range resp.Logs {
		if log.Id != idA1 && log.Id != idA2 {
			t.Errorf("unexpected row id %s (not actor A)", log.Id)
		}
		if log.ActorId != actorA.String() {
			t.Errorf("row %s: actor_id = %q, want %q", log.Id, log.ActorId, actorA.String())
		}
	}
}

// TestAuditLogs_Filter_NoActorFilter_IncludesNullActorRows verifies that without an
// actor_id filter, both actor A rows and NULL-actor rows are returned.
func TestAuditLogs_Filter_NoActorFilter_IncludesNullActorRows(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-noactor@audit.test", "admin")

	// Create a real user to use as actor A (FK constraint).
	actorA, _ := seedUserWithSession(t, "audit-filter-noactor-a@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	now := time.Now().UTC()

	_ = seedAuditLogWithActor(t, "grades", entityID, &actorA, "grade.update", now)
	_ = seedAuditLogWithActor(t, "grades", entityID, nil, "grade.update", now.Add(time.Millisecond))

	// No actor_id filter — should return both actor A and NULL-actor rows.
	resp := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityID, "", "", "")

	if len(resp.Logs) != 2 {
		t.Errorf("expected 2 rows (actor A + NULL actor), got %d", len(resp.Logs))
	}
}

// TestAuditLogs_Filter_CreatedAtFrom_HalfOpenLowerBound verifies that created_from=T
// returns only rows where created_at >= T.
func TestAuditLogs_Filter_CreatedAtFrom_HalfOpenLowerBound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-from@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	base := time.Now().UTC().Truncate(time.Second)
	before := base.Add(-2 * time.Second)
	boundary := base
	after := base.Add(2 * time.Second)

	_ = seedAuditLogWithActor(t, "grades", entityID, nil, "before", before)
	idAt := seedAuditLogWithActor(t, "grades", entityID, nil, "at.boundary", boundary)
	idAfter := seedAuditLogWithActor(t, "grades", entityID, nil, "after", after)

	resp := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityID, "",
		boundary.Format(time.RFC3339), "")

	if len(resp.Logs) != 2 {
		t.Errorf("expected 2 rows (at + after boundary), got %d", len(resp.Logs))
	}
	for _, log := range resp.Logs {
		if log.Id != idAt && log.Id != idAfter {
			t.Errorf("unexpected row %s (should be filtered by created_from)", log.Id)
		}
	}
}

// TestAuditLogs_Filter_CreatedAtRange_ClosedBothEnds verifies that created_from=T1,
// created_to=T2 returns only rows where T1 <= created_at <= T2 (inclusive both ends).
func TestAuditLogs_Filter_CreatedAtRange_ClosedBothEnds(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-range@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	base := time.Now().UTC().Truncate(time.Second)
	t1 := base
	t2 := base.Add(4 * time.Second)

	_ = seedAuditLogWithActor(t, "grades", entityID, nil, "before", t1.Add(-2*time.Second))
	idT1 := seedAuditLogWithActor(t, "grades", entityID, nil, "at.t1", t1)
	idMid := seedAuditLogWithActor(t, "grades", entityID, nil, "mid", t1.Add(2*time.Second))
	idT2 := seedAuditLogWithActor(t, "grades", entityID, nil, "at.t2", t2)
	_ = seedAuditLogWithActor(t, "grades", entityID, nil, "after", t2.Add(2*time.Second))

	resp := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityID, "",
		t1.Format(time.RFC3339), t2.Format(time.RFC3339))

	if len(resp.Logs) != 3 {
		t.Errorf("expected 3 rows [t1, mid, t2], got %d", len(resp.Logs))
	}
	for _, log := range resp.Logs {
		if log.Id != idT1 && log.Id != idMid && log.Id != idT2 {
			t.Errorf("unexpected row %s (outside [T1, T2] range)", log.Id)
		}
	}
}

// TestAuditLogs_Filter_AllFilters_Combined verifies that entity+entity_id+actor_id+date
// range combined return only rows satisfying all four constraints simultaneously.
func TestAuditLogs_Filter_AllFilters_Combined(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-combined@audit.test", "admin")

	// Create two real users for actor_id (FK constraint).
	actorA, _ := seedUserWithSession(t, "audit-filter-combined-a@audit.test", "admin")
	actorB, _ := seedUserWithSession(t, "audit-filter-combined-b@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	base := time.Now().UTC().Truncate(time.Second)
	t1 := base
	t2 := base.Add(10 * time.Second)

	// Only this row satisfies ALL four constraints: actorA, inside [t1, t2].
	idTarget := seedAuditLogWithActor(t, "grades", entityID, &actorA, "match", t1.Add(2*time.Second))

	// Outside time range (actorA but too early).
	_ = seedAuditLogWithActor(t, "grades", entityID, &actorA, "early", t1.Add(-2*time.Second))
	// Wrong actor (inside range but actorB).
	_ = seedAuditLogWithActor(t, "grades", entityID, &actorB, "wrong.actor", t1.Add(2*time.Second))
	// NULL actor (inside range but no actor).
	_ = seedAuditLogWithActor(t, "grades", entityID, nil, "null.actor", t1.Add(2*time.Second))

	resp := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityID, actorA.String(),
		t1.Format(time.RFC3339), t2.Format(time.RFC3339))

	if len(resp.Logs) != 1 {
		t.Errorf("expected 1 row satisfying all filters, got %d", len(resp.Logs))
	}
	if len(resp.Logs) == 1 && resp.Logs[0].Id != idTarget {
		t.Errorf("unexpected row %s, want %s", resp.Logs[0].Id, idTarget)
	}
}

// TestAuditLogs_Filter_NullActorRow_FieldAbsentInResponse verifies that a row with
// actor_id IS NULL returns with actor_id = "" (not the zero UUID string).
func TestAuditLogs_Filter_NullActorRow_FieldAbsentInResponse(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-nullactor@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	_ = seedAuditLogWithActor(t, "grades", entityID, nil, "system.action", time.Now().UTC())

	resp := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityID, "", "", "")

	if len(resp.Logs) == 0 {
		t.Fatal("expected at least 1 row")
	}
	zeroUUID := "00000000-0000-0000-0000-000000000000"
	for _, log := range resp.Logs {
		if log.ActorId == zeroUUID {
			t.Errorf("NULL actor must map to empty string, got zero UUID")
		}
		if log.ActorId != "" {
			t.Errorf("NULL actor must map to empty string, got %q", log.ActorId)
		}
	}
}

// TestAuditLogs_Filter_MissingEntity_ReturnsInvalidArgument verifies that entity=""
// is rejected with CodeInvalidArgument.
func TestAuditLogs_Filter_MissingEntity_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-noentity@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "",
		EntityId: uuid.New().String(),
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestAuditLogs_Filter_MissingEntityID_ReturnsInvalidArgument verifies that entity_id=""
// is rejected with CodeInvalidArgument.
func TestAuditLogs_Filter_MissingEntityID_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-noid@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: "",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestAuditLogs_Filter_MalformedEntityID_ReturnsInvalidArgument verifies that a
// non-UUID entity_id is rejected with CodeInvalidArgument.
func TestAuditLogs_Filter_MalformedEntityID_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-badid@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: "not-a-uuid",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestAuditLogs_Filter_MalformedActorID_ReturnsInvalidArgument verifies that a non-empty
// invalid actor_id is rejected with CodeInvalidArgument.
func TestAuditLogs_Filter_MalformedActorID_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-badactor@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:   "grades",
		EntityId: uuid.New().String(),
		ActorId:  "bad-uuid",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestAuditLogs_Filter_MalformedPageToken_ReturnsInvalidArgument verifies that a
// non-UUID page_token is rejected with CodeInvalidArgument.
func TestAuditLogs_Filter_MalformedPageToken_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-badtoken@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:    "grades",
		EntityId:  uuid.New().String(),
		PageToken: "not-a-uuid",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestAuditLogs_Filter_MalformedCreatedAtFrom_ReturnsInvalidArgument verifies that
// a non-RFC3339 created_from string is rejected with CodeInvalidArgument.
func TestAuditLogs_Filter_MalformedCreatedAtFrom_ReturnsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-badfrom@audit.test", "admin")
	client := newAuditLogsClient(nil)

	req := connect.NewRequest(&auditlogsv1.ListAuditLogsRequest{
		Entity:      "grades",
		EntityId:    uuid.New().String(),
		CreatedFrom: "not-a-date",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListAuditLogs(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestAuditLogs_Filter_DetailNonNull_RawJSONPassthrough verifies that a row with a
// non-null detail JSONB column is returned as a valid JSON string with the same
// key-value content as the original (PostgreSQL normalizes JSONB key order).
func TestAuditLogs_Filter_DetailNonNull_RawJSONPassthrough(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-detail@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	// PostgreSQL normalizes JSONB key ordering; compare semantically.
	seedJSON := `{"old_value":"5.0","new_value":"6.0"}`
	_ = seedAuditLogWithDetail(t, "grades", entityID, seedJSON)

	resp := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityID, "", "", "")

	if len(resp.Logs) == 0 {
		t.Fatal("expected 1 row")
	}
	got := resp.Logs[0].Detail
	if got == "" {
		t.Error("detail must not be empty for a non-null JSONB row")
	}
	if !jsonEqual(got, seedJSON) {
		t.Errorf("detail JSON content mismatch:\n  got  %q\n  want %q", got, seedJSON)
	}
}

// TestAuditLogs_Filter_DetailNull_AbsentInResponse verifies that a row with detail=NULL
// returns with detail = "" (absent/empty in proto), not the literal "null".
func TestAuditLogs_Filter_DetailNull_AbsentInResponse(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-nulldetail@audit.test", "admin")

	entityID := uuid.New()
	cleanupAuditLogs(t, "grades", entityID)

	_ = seedAuditLogNullDetail(t, "grades", entityID)

	resp := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityID, "", "", "")

	if len(resp.Logs) == 0 {
		t.Fatal("expected 1 row")
	}
	if resp.Logs[0].Detail == "null" {
		t.Errorf("detail must be empty string for NULL, got literal %q", resp.Logs[0].Detail)
	}
	if resp.Logs[0].Detail != "" {
		t.Errorf("detail must be empty string for NULL detail, got %q", resp.Logs[0].Detail)
	}
}

// TestAuditLogs_Filter_CrossEntityIsolation verifies that rows for entity+entity_id=A
// are invisible when querying entity+entity_id=B (strict partition isolation).
func TestAuditLogs_Filter_CrossEntityIsolation(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "audit-filter-isolation@audit.test", "admin")

	entityIDA := uuid.New()
	entityIDB := uuid.New()
	cleanupAuditLogs(t, "grades", entityIDA)
	cleanupAuditLogs(t, "grades", entityIDB)

	now := time.Now().UTC()
	seedAuditLogs(t, "grades", entityIDA, 3)
	for i := 0; i < 2; i++ {
		_ = seedAuditLogWithActor(t, "grades", entityIDB, nil, fmt.Sprintf("b.action.%d", i), now.Add(time.Duration(i)*time.Millisecond))
	}

	// Query for entity A — must return only A rows.
	respA := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityIDA, "", "", "")
	for _, log := range respA.Logs {
		if log.EntityId != entityIDA.String() {
			t.Errorf("entity isolation: got entity_id=%s, want %s", log.EntityId, entityIDA.String())
		}
	}
	if len(respA.Logs) != 3 {
		t.Errorf("expected 3 rows for entity A, got %d", len(respA.Logs))
	}

	// Query for entity B — must return only B rows.
	respB := listAuditLogsFiltered(t, ctx, adminSID, "grades", entityIDB, "", "", "")
	if len(respB.Logs) != 2 {
		t.Errorf("expected 2 rows for entity B, got %d", len(respB.Logs))
	}
}
