package iam

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

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

// AssignRoleParams holds the parameters for role assignment.
type AssignRoleParams struct {
	// UserID is the target user's UUID.
	UserID uuid.UUID
	// RoleName is the name of the role to assign (admin, teacher, student).
	RoleName string
	// Actor is the UUID of the admin performing the operation.
	Actor uuid.UUID
}

// RevokeRoleParams holds the parameters for role revocation.
type RevokeRoleParams struct {
	// UserID is the target user's UUID.
	UserID uuid.UUID
	// RoleName is the name of the role to revoke (admin, teacher, student).
	RoleName string
	// Actor is the UUID of the admin performing the operation.
	Actor uuid.UUID
}

// Repository is the consumer-side data-access seam for the iam slice.
type Repository interface {
	// ListUsers returns up to params.RowLimit user rows for pagination.
	ListUsers(ctx context.Context, params ListUsersParams) ([]iamdb.ListUsersRow, error)
	// GetUserByID returns identity + profile columns for a single non-deleted user.
	GetUserByID(ctx context.Context, userID uuid.UUID) (iamdb.GetUserByIDRow, error)
	// GetUserRoles returns role names for a user, sorted alphabetically.
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	// AssignRole inserts a user_roles row (idempotent). Returns (rowsInserted, error).
	// rowsInserted is 0 for an idempotent re-assign, 1 for a new assignment.
	AssignRole(ctx context.Context, params AssignRoleParams) (int64, error)
	// RevokeRoleTx hard-deletes the user_roles row and writes an audit_logs entry
	// atomically inside a single transaction. Returns ErrNotFound when the user
	// does not have the specified role (0 rows deleted).
	RevokeRoleTx(ctx context.Context, params RevokeRoleParams) error
	// CountAdmins returns the current number of users holding the admin role.
	CountAdmins(ctx context.Context) (int32, error)
	// InsertAuditLog writes a role mutation audit event. Used by AssignRole
	// to record the audit outside the transaction.
	InsertAuditLog(ctx context.Context, params AuditLogParams) error
}

// AuditLogParams holds the fields for an audit_logs entry.
type AuditLogParams struct {
	// ActorID is the UUID of the user performing the action.
	ActorID uuid.UUID
	// Action is the event name (e.g. "role.assign", "role.revoke").
	Action string
	// Entity is the entity type (e.g. "users").
	Entity string
	// EntityID is the UUID of the affected entity.
	EntityID uuid.UUID
	// Detail is the JSON payload (e.g. {"role": "admin"}).
	Detail json.RawMessage
}

// postgresRepository wraps iamdb.Querier and implements Repository.
type postgresRepository struct {
	q    iamdb.Querier
	pool *pgxpool.Pool
}

// Compile-time proof that *postgresRepository satisfies Repository.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by an iamdb.Querier.
// pool is required for transactional operations (RevokeRoleTx).
func NewPostgresRepository(q iamdb.Querier, pool *pgxpool.Pool) *postgresRepository {
	return &postgresRepository{q: q, pool: pool}
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

// AssignRole inserts a user_roles row with ON CONFLICT DO NOTHING.
// Returns (rowsInserted, nil); rowsInserted == 0 means the role already existed.
func (r *postgresRepository) AssignRole(ctx context.Context, params AssignRoleParams) (int64, error) {
	n, err := r.q.AssignRole(ctx, iamdb.AssignRoleParams{
		UserID:   pgtype.UUID{Bytes: params.UserID, Valid: true},
		RoleName: params.RoleName,
	})
	if err != nil {
		return 0, TranslatePgError(err)
	}
	return n, nil
}

// RevokeRoleTx hard-deletes the user_roles row and inserts an audit_logs entry
// atomically. Returns ErrNotFound when the DELETE affects 0 rows (role not held).
func (r *postgresRepository) RevokeRoleTx(ctx context.Context, params RevokeRoleParams) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TranslatePgError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := iamdb.New(tx)

	n, err := q.RevokeRole(ctx, iamdb.RevokeRoleParams{
		UserID:   pgtype.UUID{Bytes: params.UserID, Valid: true},
		RoleName: params.RoleName,
	})
	if err != nil {
		return TranslatePgError(err)
	}
	if n == 0 {
		return fmt.Errorf("%w: user does not have role %q", ErrNotFound, params.RoleName)
	}

	detail, err := json.Marshal(map[string]string{"role": params.RoleName})
	if err != nil {
		return fmt.Errorf("iam: marshal audit detail: %w", err)
	}
	if err := q.InsertAuditLog(ctx, iamdb.InsertAuditLogParams{
		ActorID:  pgtype.UUID{Bytes: params.Actor, Valid: true},
		Action:   "role.revoke",
		Entity:   "users",
		EntityID: pgtype.UUID{Bytes: params.UserID, Valid: true},
		Detail:   detail,
	}); err != nil {
		return fmt.Errorf("iam: RevokeRoleTx audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return TranslatePgError(err)
	}
	return nil
}

// CountAdmins returns how many users currently hold the admin role.
func (r *postgresRepository) CountAdmins(ctx context.Context) (int32, error) {
	n, err := r.q.CountAdmins(ctx)
	if err != nil {
		return 0, fmt.Errorf("iam: CountAdmins: %w", err)
	}
	return n, nil
}

// InsertAuditLog writes a role mutation audit event.
func (r *postgresRepository) InsertAuditLog(ctx context.Context, params AuditLogParams) error {
	if err := r.q.InsertAuditLog(ctx, iamdb.InsertAuditLogParams{
		ActorID:  pgtype.UUID{Bytes: params.ActorID, Valid: true},
		Action:   params.Action,
		Entity:   params.Entity,
		EntityID: pgtype.UUID{Bytes: params.EntityID, Valid: true},
		Detail:   params.Detail,
	}); err != nil {
		return fmt.Errorf("iam: InsertAuditLog: %w", err)
	}
	return nil
}
