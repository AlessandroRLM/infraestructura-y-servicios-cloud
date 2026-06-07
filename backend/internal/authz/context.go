package authz

import "context"

// permissionsKey is an unexported type used as the context key for a PermissionSet.
// Using a private type prevents collisions with keys from other packages.
type permissionsKey struct{}

// WithPermissions stores the given PermissionSet in ctx, replacing any previously
// stored set. The value is retrieved by PermissionsFromContext.
func WithPermissions(ctx context.Context, s PermissionSet) context.Context {
	return context.WithValue(ctx, permissionsKey{}, s)
}

// PermissionsFromContext retrieves the PermissionSet stored by WithPermissions.
// The boolean is false when no set has been stored (unauthenticated or public routes).
func PermissionsFromContext(ctx context.Context) (PermissionSet, bool) {
	s, ok := ctx.Value(permissionsKey{}).(PermissionSet)
	return s, ok
}
