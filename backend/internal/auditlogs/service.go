package auditlogs

import (
	"context"
	"fmt"
	"strings"
	"time"

	auditlogsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auditlogs/auditlogsdb"
	"github.com/google/uuid"
)

const (
	// pageSizeMin is the minimum effective page size after clamping.
	pageSizeMin = 20
	// pageSizeMax is the maximum effective page size after clamping.
	pageSizeMax = 200
)

// Service implements the audit_logs domain use case: a single read-only ListAuditLogs RPC
// with validation, page-size clamping, keyset cursor assembly, and row-to-proto conversion.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the provided Repository dependency.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListAuditLogs validates the request, clamps page_size, assembles repo params,
// detects the next page, and converts rows to proto.
func (s *Service) ListAuditLogs(
	ctx context.Context,
	req *auditlogsv1.ListAuditLogsRequest,
) (*auditlogsv1.ListAuditLogsResponse, error) {
	// Validate required entity field.
	if req.GetEntity() == "" {
		return nil, fmt.Errorf("%w: entity is required", ErrInvalidInput)
	}

	// Clamp page_size: ≤0 → pageSizeMin, >pageSizeMax → pageSizeMax.
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = pageSizeMin
	} else if pageSize > pageSizeMax {
		pageSize = pageSizeMax
	}

	// Parse optional RFC3339 timestamps.
	var createdFrom, createdTo *time.Time
	if s := req.GetCreatedFrom(); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("%w: created_from is not a valid RFC3339 timestamp: %q", ErrInvalidInput, s)
		}
		createdFrom = &t
	}
	if s := req.GetCreatedTo(); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("%w: created_to is not a valid RFC3339 timestamp: %q", ErrInvalidInput, s)
		}
		createdTo = &t
	}

	// Parse optional actor_id UUID (service validates; handler also guards for non-empty).
	var actorID *uuid.UUID
	if s := req.GetActorId(); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("%w: actor_id is not a valid UUID: %q", ErrInvalidInput, s)
		}
		actorID = &id
	}

	// Parse optional page_token UUID (service validates; handler also guards for non-empty).
	var pageToken *uuid.UUID
	if s := req.GetPageToken(); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("%w: page_token is not a valid UUID: %q", ErrInvalidInput, s)
		}
		pageToken = &id
	}

	// entity_id is validated by the handler (parseUUID); parse here defensively.
	entityID, err := uuid.Parse(req.GetEntityId())
	if err != nil {
		return nil, fmt.Errorf("%w: entity_id is not a valid UUID: %q", ErrInvalidInput, req.GetEntityId())
	}

	params := ListParams{
		Entity:      req.GetEntity(),
		EntityID:    entityID,
		ActorID:     actorID,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		PageToken:   pageToken,
		RowLimit:    int32(pageSize + 1), // +1 to detect next page
	}

	rows, err := s.repo.ListAuditLogs(ctx, params)
	if err != nil {
		return nil, err
	}

	// Detect next page: if we got pageSize+1 rows, there is a next page.
	hasNextPage := len(rows) == pageSize+1
	if hasNextPage {
		rows = rows[:pageSize]
	}

	// Convert rows to proto.
	protoLogs := make([]*auditlogsv1.AuditLog, 0, len(rows))
	for _, row := range rows {
		protoLogs = append(protoLogs, auditLogToProto(row))
	}

	// Determine next_page_token: id of last retained row, or empty if no next page.
	var nextPageToken string
	if hasNextPage && len(rows) > 0 {
		nextPageToken = uuid.UUID(rows[len(rows)-1].ID.Bytes).String()
	}

	return &auditlogsv1.ListAuditLogsResponse{
		Logs:          protoLogs,
		NextPageToken: nextPageToken,
	}, nil
}

// recentCursorSep is the separator used to encode the composite keyset cursor for
// ListRecentAuditLogs. It separates the RFC3339 timestamp from the row UUID.
const recentCursorSep = "|"

// encodeRecentCursor encodes (created_at, id) as an opaque page_token string.
// Format: "<RFC3339>|<uuid>".
func encodeRecentCursor(createdAt time.Time, id uuid.UUID) string {
	return createdAt.UTC().Format(time.RFC3339Nano) + recentCursorSep + id.String()
}

// decodeRecentCursor parses an opaque page_token back into (created_at, id).
// Returns ErrInvalidInput on any parse failure.
func decodeRecentCursor(token string) (time.Time, uuid.UUID, error) {
	idx := strings.Index(token, recentCursorSep)
	if idx < 0 {
		return time.Time{}, uuid.UUID{}, fmt.Errorf("%w: page_token is not a valid cursor", ErrInvalidInput)
	}
	tsPart := token[:idx]
	idPart := token[idx+1:]

	ts, err := time.Parse(time.RFC3339Nano, tsPart)
	if err != nil {
		// Fallback: try RFC3339 (without nanoseconds).
		ts, err = time.Parse(time.RFC3339, tsPart)
		if err != nil {
			return time.Time{}, uuid.UUID{}, fmt.Errorf("%w: page_token timestamp part is invalid: %q", ErrInvalidInput, tsPart)
		}
	}

	id, err := uuid.Parse(idPart)
	if err != nil {
		return time.Time{}, uuid.UUID{}, fmt.Errorf("%w: page_token id part is not a valid UUID: %q", ErrInvalidInput, idPart)
	}

	return ts, id, nil
}

// ListRecentAuditLogs validates the request, clamps page_size, assembles repo params,
// detects the next page, and converts rows to proto. No entity scoping — global feed.
func (s *Service) ListRecentAuditLogs(
	ctx context.Context,
	req *auditlogsv1.ListRecentAuditLogsRequest,
) (*auditlogsv1.ListRecentAuditLogsResponse, error) {
	// Clamp page_size: ≤0 → pageSizeMin, >pageSizeMax → pageSizeMax.
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = pageSizeMin
	} else if pageSize > pageSizeMax {
		pageSize = pageSizeMax
	}

	// Parse optional RFC3339 timestamps.
	var createdFrom, createdTo *time.Time
	if s := req.GetCreatedFrom(); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("%w: created_from is not a valid RFC3339 timestamp: %q", ErrInvalidInput, s)
		}
		createdFrom = &t
	}
	if s := req.GetCreatedTo(); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("%w: created_to is not a valid RFC3339 timestamp: %q", ErrInvalidInput, s)
		}
		createdTo = &t
	}

	// Parse optional actor_id UUID.
	var actorID *uuid.UUID
	if s := req.GetActorId(); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("%w: actor_id is not a valid UUID: %q", ErrInvalidInput, s)
		}
		actorID = &id
	}

	// Parse optional composite keyset cursor.
	var cursorTs *time.Time
	var cursorID *uuid.UUID
	if token := req.GetPageToken(); token != "" {
		ts, id, err := decodeRecentCursor(token)
		if err != nil {
			return nil, err
		}
		cursorTs = &ts
		cursorID = &id
	}

	params := ListRecentParams{
		ActorID:     actorID,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		CursorTs:    cursorTs,
		CursorID:    cursorID,
		RowLimit:    int32(pageSize + 1), // +1 to detect next page
	}

	rows, err := s.repo.ListRecentAuditLogs(ctx, params)
	if err != nil {
		return nil, err
	}

	// Detect next page: if we got pageSize+1 rows, there is a next page.
	hasNextPage := len(rows) == pageSize+1
	if hasNextPage {
		rows = rows[:pageSize]
	}

	// Convert rows to proto.
	protoLogs := make([]*auditlogsv1.AuditLog, 0, len(rows))
	for _, row := range rows {
		protoLogs = append(protoLogs, auditLogToProto(row))
	}

	// Determine next_page_token: encode (created_at, id) of last retained row, or empty.
	var nextPageToken string
	if hasNextPage && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextPageToken = encodeRecentCursor(
			last.CreatedAt.Time.UTC(),
			uuid.UUID(last.ID.Bytes),
		)
	}

	return &auditlogsv1.ListRecentAuditLogsResponse{
		Logs:          protoLogs,
		NextPageToken: nextPageToken,
	}, nil
}

// auditLogToProto converts an auditlogsdb.AuditLog row to the proto wire message.
// Nullable fields (ActorID, Detail) are mapped to empty strings when absent.
func auditLogToProto(row auditlogsdb.AuditLog) *auditlogsv1.AuditLog {
	log := &auditlogsv1.AuditLog{
		Id:        uuid.UUID(row.ID.Bytes).String(),
		Action:    row.Action,
		Entity:    row.Entity,
		EntityId:  uuid.UUID(row.EntityID.Bytes).String(),
		CreatedAt: row.CreatedAt.Time.UTC().Format(time.RFC3339),
	}

	// ActorID: only set when valid (non-NULL). Never emit the zero UUID.
	if row.ActorID.Valid {
		log.ActorId = uuid.UUID(row.ActorID.Bytes).String()
	}

	// Detail: convert []byte to string; empty when nil or zero-length.
	if len(row.Detail) > 0 {
		log.Detail = string(row.Detail)
	}

	return log
}
