package iam

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam/iamdb"
)

// ListUsersParams holds the parameters for the ListUsers repository method.
type ListUsersParams struct {
	// Query is an optional search string matched against email and display_name (ILIKE).
	Query *string
	// PageToken is the exclusive upper-bound UUID cursor for keyset pagination.
	PageToken *uuid.UUID
	// RowLimit is the LIMIT applied to the query (should be clampedPageSize + 1).
	RowLimit int32
}

// Repository is the consumer-side data-access seam for the iam slice.
// Phase 1 exposes read methods only; mutation methods are added in Phase 2.
type Repository interface {
	// ListUsers returns up to params.RowLimit user rows for pagination.
	ListUsers(ctx context.Context, params ListUsersParams) ([]iamdb.ListUsersRow, error)
	// GetUserByID returns identity + profile columns for a single non-deleted user.
	GetUserByID(ctx context.Context, userID uuid.UUID) (iamdb.GetUserByIDRow, error)
	// GetUserRoles returns role names for a user, sorted alphabetically.
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
}

// postgresRepository wraps iamdb.Querier and implements Repository.
type postgresRepository struct {
	q iamdb.Querier
}

// Compile-time proof that *postgresRepository satisfies Repository.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by an iamdb.Querier.
func NewPostgresRepository(q iamdb.Querier) *postgresRepository {
	return &postgresRepository{q: q}
}

// ListUsers translates ListUsersParams to iamdb.ListUsersParams and executes the query.
// Errors are translated via TranslatePgError.
func (r *postgresRepository) ListUsers(ctx context.Context, params ListUsersParams) ([]iamdb.ListUsersRow, error) {
	p := iamdb.ListUsersParams{
		RowLimit: params.RowLimit,
	}
	if params.PageToken != nil {
		p.PageToken = pgtype.UUID{Bytes: *params.PageToken, Valid: true}
	}
	if params.Query != nil {
		p.Query = pgtype.Text{String: *params.Query, Valid: true}
	}
	rows, err := r.q.ListUsers(ctx, p)
	if err != nil {
		return nil, TranslatePgError(err)
	}
	return rows, nil
}

// GetUserByID fetches identity + profile for a single non-deleted user.
// Returns ErrNotFound when no matching row exists.
func (r *postgresRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (iamdb.GetUserByIDRow, error) {
	row, err := r.q.GetUserByID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return iamdb.GetUserByIDRow{}, TranslatePgError(err)
	}
	return row, nil
}

// GetUserRoles returns role names for a user, sorted alphabetically.
func (r *postgresRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	roles, err := r.q.GetUserRoles(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("iam: GetUserRoles: %w", err)
	}
	if roles == nil {
		return []string{}, nil
	}
	return roles, nil
}
