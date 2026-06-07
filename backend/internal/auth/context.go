package auth

import (
	"context"

	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys in this package to prevent collisions.
type contextKey int

const userIDKey contextKey = iota

// WithUserID returns a new context carrying the authenticated user's ID.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFromContext extracts the user ID previously set by WithUserID.
// The second return value is false if no user ID is present.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
