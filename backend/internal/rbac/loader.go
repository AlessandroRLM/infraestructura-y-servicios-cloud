// Package rbac provides the database-backed role and permission loader.
package rbac

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac/rbacdb"
)

// PostgresRoleLoader implements auth.RoleLoader by querying the database for all
// permissions assigned to a user via their role memberships.
type PostgresRoleLoader struct {
	q rbacdb.Querier
}

// NewPostgresRoleLoader constructs a PostgresRoleLoader backed by the given Querier.
func NewPostgresRoleLoader(q rbacdb.Querier) PostgresRoleLoader {
	return PostgresRoleLoader{q: q}
}

// Load executes the permission-fetch query for userID and returns the resulting
// PermissionSet. An empty set (not an error) is returned when the user has no roles.
func (l PostgresRoleLoader) Load(ctx context.Context, userID uuid.UUID) (authz.PermissionSet, error) {
	pgID := pgtype.UUID{Bytes: userID, Valid: true}

	codes, err := l.q.GetPermissionsForUser(ctx, pgID)
	if err != nil {
		return nil, fmt.Errorf("rbac: load permissions for user %s: %w", userID, err)
	}

	perms := make([]authz.Permission, 0, len(codes))
	for _, code := range codes {
		perms = append(perms, authz.Permission(code))
	}
	return authz.NewPermissionSet(perms), nil
}
