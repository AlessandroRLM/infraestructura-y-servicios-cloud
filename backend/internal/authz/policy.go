package authz

import "context"

// Decision is the result of a Policy evaluation.
type Decision struct {
	// Allowed is true when the caller is permitted to proceed.
	Allowed bool
}

// Policy evaluates whether a caller may perform an action identified by a Permission.
// Implementations receive the full request context, allowing them to inspect the caller's
// identity, permission set, or any other context value.
type Policy interface {
	Evaluate(ctx context.Context, required Permission) Decision
}

// PermissionPolicy is a Policy that allows a request when the caller's PermissionSet
// (stored in context by the session interceptor) contains the required Permission.
// An absent or empty set is treated as denied.
type PermissionPolicy struct{}

// Evaluate reads the caller's PermissionSet from ctx and returns Allowed = set.Has(required).
// When no set is stored in the context the result is Allowed = false.
func (PermissionPolicy) Evaluate(ctx context.Context, required Permission) Decision {
	set, ok := PermissionsFromContext(ctx)
	if !ok {
		return Decision{Allowed: false}
	}
	return Decision{Allowed: set.Has(required)}
}
