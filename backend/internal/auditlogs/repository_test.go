package auditlogs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Compile-time guard: *postgresRepository must satisfy Repository.
// This test will fail to compile until repository.go defines both types.
var _ Repository = (*postgresRepository)(nil)

// TestToListAuditLogsParams covers the pure translation from ListParams to
// auditlogsdb.ListAuditLogsParams — no database required.
func TestToListAuditLogsParams_RequiredFieldsAndRowLimit(t *testing.T) {
	t.Parallel()

	entityID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	in := ListParams{
		Entity:   "grades",
		EntityID: entityID,
		RowLimit: 21,
	}

	got := toListAuditLogsParams(in)

	if got.Entity != "grades" {
		t.Errorf("Entity = %q, want %q", got.Entity, "grades")
	}
	if got.EntityID.Bytes != entityID {
		t.Errorf("EntityID.Bytes = %v, want %v", got.EntityID.Bytes, entityID)
	}
	if !got.EntityID.Valid {
		t.Error("EntityID.Valid must be true when EntityID is set")
	}
	if got.RowLimit != 21 {
		t.Errorf("RowLimit = %d, want 21", got.RowLimit)
	}
}

func TestToListAuditLogsParams_NilOptionals_ProduceInvalidPgTypes(t *testing.T) {
	t.Parallel()

	in := ListParams{
		Entity:      "grades",
		EntityID:    uuid.New(),
		ActorID:     nil,
		CreatedFrom: nil,
		CreatedTo:   nil,
		PageToken:   nil,
		RowLimit:    1,
	}

	got := toListAuditLogsParams(in)

	if got.ActorID.Valid {
		t.Errorf("ActorID.Valid = true, want false for nil ActorID")
	}
	if got.ActorID != (pgtype.UUID{}) {
		t.Errorf("ActorID must be zero value for nil ActorID, got %+v", got.ActorID)
	}
	if got.CreatedFrom.Valid {
		t.Errorf("CreatedFrom.Valid = true, want false for nil CreatedFrom")
	}
	if got.CreatedTo.Valid {
		t.Errorf("CreatedTo.Valid = true, want false for nil CreatedTo")
	}
	if got.PageToken.Valid {
		t.Errorf("PageToken.Valid = true, want false for nil PageToken")
	}
}

func TestToListAuditLogsParams_SetActorID_CarriesBytes(t *testing.T) {
	t.Parallel()

	actorID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	in := ListParams{
		Entity:   "grades",
		EntityID: uuid.New(),
		ActorID:  &actorID,
		RowLimit: 1,
	}

	got := toListAuditLogsParams(in)

	if !got.ActorID.Valid {
		t.Error("ActorID.Valid must be true when ActorID is set")
	}
	if got.ActorID.Bytes != actorID {
		t.Errorf("ActorID.Bytes = %v, want %v", got.ActorID.Bytes, actorID)
	}
}

func TestToListAuditLogsParams_SetPageToken_CarriesBytes(t *testing.T) {
	t.Parallel()

	token := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	in := ListParams{
		Entity:    "grades",
		EntityID:  uuid.New(),
		PageToken: &token,
		RowLimit:  1,
	}

	got := toListAuditLogsParams(in)

	if !got.PageToken.Valid {
		t.Error("PageToken.Valid must be true when PageToken is set")
	}
	if got.PageToken.Bytes != token {
		t.Errorf("PageToken.Bytes = %v, want %v", got.PageToken.Bytes, token)
	}
}

func TestToListAuditLogsParams_SetCreatedFrom_CarriesTime(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	in := ListParams{
		Entity:      "grades",
		EntityID:    uuid.New(),
		CreatedFrom: &ts,
		RowLimit:    1,
	}

	got := toListAuditLogsParams(in)

	if !got.CreatedFrom.Valid {
		t.Error("CreatedFrom.Valid must be true when CreatedFrom is set")
	}
	if !got.CreatedFrom.Time.Equal(ts) {
		t.Errorf("CreatedFrom.Time = %v, want %v", got.CreatedFrom.Time, ts)
	}
}

func TestToListAuditLogsParams_SetCreatedTo_CarriesTime(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 3, 20, 18, 0, 0, 0, time.UTC)
	in := ListParams{
		Entity:    "grades",
		EntityID:  uuid.New(),
		CreatedTo: &ts,
		RowLimit:  1,
	}

	got := toListAuditLogsParams(in)

	if !got.CreatedTo.Valid {
		t.Error("CreatedTo.Valid must be true when CreatedTo is set")
	}
	if !got.CreatedTo.Time.Equal(ts) {
		t.Errorf("CreatedTo.Time = %v, want %v", got.CreatedTo.Time, ts)
	}
}

func TestToListAuditLogsParams_AllOptionals_SetAndValid(t *testing.T) {
	t.Parallel()

	actorID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	token := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	entityID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")

	in := ListParams{
		Entity:      "enrollment",
		EntityID:    entityID,
		ActorID:     &actorID,
		CreatedFrom: &from,
		CreatedTo:   &to,
		PageToken:   &token,
		RowLimit:    51,
	}

	got := toListAuditLogsParams(in)

	if got.Entity != "enrollment" {
		t.Errorf("Entity = %q, want %q", got.Entity, "enrollment")
	}
	if got.EntityID.Bytes != entityID || !got.EntityID.Valid {
		t.Errorf("EntityID mismatch: got %+v", got.EntityID)
	}
	if got.ActorID.Bytes != actorID || !got.ActorID.Valid {
		t.Errorf("ActorID mismatch: got %+v", got.ActorID)
	}
	if !got.CreatedFrom.Time.Equal(from) || !got.CreatedFrom.Valid {
		t.Errorf("CreatedFrom mismatch: got %+v", got.CreatedFrom)
	}
	if !got.CreatedTo.Time.Equal(to) || !got.CreatedTo.Valid {
		t.Errorf("CreatedTo mismatch: got %+v", got.CreatedTo)
	}
	if got.PageToken.Bytes != token || !got.PageToken.Valid {
		t.Errorf("PageToken mismatch: got %+v", got.PageToken)
	}
	if got.RowLimit != 51 {
		t.Errorf("RowLimit = %d, want 51", got.RowLimit)
	}
}
