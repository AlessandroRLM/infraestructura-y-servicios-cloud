package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
)

// RoleLoader loads the permission set for an authenticated user.
// Implementations may query a database, cache, or policy engine.
type RoleLoader interface {
	Load(ctx context.Context, userID uuid.UUID) (authz.PermissionSet, error)
}

// NoopRoleLoader is a RoleLoader that always returns an empty PermissionSet.
// It is the default used by the auth interceptor when no role-based authorization
// is configured for a service.
type NoopRoleLoader struct{}

// Load returns an empty PermissionSet and no error.
func (NoopRoleLoader) Load(_ context.Context, _ uuid.UUID) (authz.PermissionSet, error) {
	return authz.PermissionSet{}, nil
}
