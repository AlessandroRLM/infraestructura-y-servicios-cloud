package auth

import (
	"context"

	"github.com/google/uuid"
)

// RoleLoader loads the role list for an authenticated user.
// Implementations may query a database, cache, or policy engine.
type RoleLoader interface {
	Load(ctx context.Context, userID uuid.UUID) ([]string, error)
}

// NoopRoleLoader is a RoleLoader that always returns an empty role slice.
// It is the default used by the auth interceptor when no role-based authorization
// is required.
type NoopRoleLoader struct{}

// Load returns an empty slice and no error.
func (NoopRoleLoader) Load(_ context.Context, _ uuid.UUID) ([]string, error) {
	return nil, nil
}
