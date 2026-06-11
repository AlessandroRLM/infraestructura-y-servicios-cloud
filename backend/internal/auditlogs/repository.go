package auditlogs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auditlogs/auditlogsdb"
)

// ListParams holds the filtering and pagination parameters for ListAuditLogs.
type ListParams struct {
	// Entity is required — the logical resource type (e.g. "grades").
	Entity string
	// EntityID is required — the UUID of the resource instance.
	EntityID uuid.UUID
	// ActorID is optional — when non-nil, only rows matching this actor are returned.
	ActorID *uuid.UUID
	// CreatedFrom is optional — inclusive lower bound on created_at.
	CreatedFrom *time.Time
	// CreatedTo is optional — inclusive upper bound on created_at.
	CreatedTo *time.Time
	// PageToken is optional — when non-nil, only rows with id < PageToken are returned.
	PageToken *uuid.UUID
	// RowLimit is the LIMIT applied to the query (should be clampedPageSize + 1).
	RowLimit int32
}

// Repository is the consumer-side data-access seam for the audit_logs slice.
// All methods are read-only (pure SELECTs). No pool or transaction is exposed.
type Repository interface {
	// ListAuditLogs returns up to params.RowLimit rows for the given entity instance,
	// ordered newest-first. Caller is responsible for page detection and trimming.
	ListAuditLogs(ctx context.Context, params ListParams) ([]auditlogsdb.AuditLog, error)
}

// postgresRepository wraps auditlogsdb.Querier and implements Repository.
type postgresRepository struct {
	q auditlogsdb.Querier
}

// Compile-time proof that *postgresRepository satisfies Repository.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by an auditlogsdb.Querier.
func NewPostgresRepository(q auditlogsdb.Querier) *postgresRepository {
	return &postgresRepository{q: q}
}

// toListAuditLogsParams translates ListParams into the sqlc-generated
// auditlogsdb.ListAuditLogsParams. Optional pointer fields are mapped to
// pgtype.UUID / pgtype.Timestamptz with Valid=false when absent.
func toListAuditLogsParams(p ListParams) auditlogsdb.ListAuditLogsParams {
	out := auditlogsdb.ListAuditLogsParams{
		Entity:   p.Entity,
		EntityID: pgtype.UUID{Bytes: p.EntityID, Valid: true},
		RowLimit: p.RowLimit,
	}
	if p.ActorID != nil {
		out.ActorID = pgtype.UUID{Bytes: *p.ActorID, Valid: true}
	}
	if p.CreatedFrom != nil {
		out.CreatedFrom = pgtype.Timestamptz{Time: *p.CreatedFrom, Valid: true}
	}
	if p.CreatedTo != nil {
		out.CreatedTo = pgtype.Timestamptz{Time: *p.CreatedTo, Valid: true}
	}
	if p.PageToken != nil {
		out.PageToken = pgtype.UUID{Bytes: *p.PageToken, Valid: true}
	}
	return out
}

// ListAuditLogs translates ListParams to auditlogsdb.ListAuditLogsParams and executes
// the keyset query. Errors are translated via TranslatePgError.
func (r *postgresRepository) ListAuditLogs(ctx context.Context, params ListParams) ([]auditlogsdb.AuditLog, error) {
	rows, err := r.q.ListAuditLogs(ctx, toListAuditLogsParams(params))
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}
