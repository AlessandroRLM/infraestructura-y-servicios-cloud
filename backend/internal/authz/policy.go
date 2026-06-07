package authz

import (
	"context"

	"github.com/google/uuid"
)

// Decision is the result of a policy evaluation.
// The zero value represents a denied decision — the package is fail-closed by default.
type Decision struct {
	// Allowed is true when the caller is permitted to proceed.
	Allowed bool
	// Reason is a human-readable explanation of the decision, populated on denial
	// to aid logging and error messages surfaced to the client.
	Reason string
	// Err carries a non-nil error when the denial was caused by an unexpected
	// condition (e.g. a storage failure). Callers may wrap or log this alongside Reason.
	Err error
}

// Allow returns a Decision that permits the request.
func Allow() Decision {
	return Decision{Allowed: true}
}

// Deny returns a denied Decision with the given human-readable reason.
func Deny(reason string) Decision {
	return Decision{Allowed: false, Reason: reason}
}

// DenyErr returns a denied Decision with a reason and an underlying error. Use this
// when the denial was caused by an unexpected failure rather than a policy violation.
func DenyErr(reason string, err error) Decision {
	return Decision{Allowed: false, Reason: reason, Err: err}
}

// AccessRequest carries all information a PolicyFunc needs to make an authorization
// decision for a single incoming request.
type AccessRequest struct {
	// SubjectID is the UUID of the authenticated caller.
	SubjectID uuid.UUID
	// Permissions is the full set of permissions the caller holds, loaded from RBAC.
	Permissions PermissionSet
	// Required is the permission under evaluation. Populated by the caller site when
	// relevant; PolicyFunc implementations that close over a specific permission (e.g.
	// RequirePermission) must use their captured permission instead of this field.
	Required Permission
	// ResourceOwnerID is the UUID of the entity that owns the resource being accessed.
	// Only meaningful when HasResource is true.
	ResourceOwnerID uuid.UUID
	// HasResource indicates that ResourceOwnerID has been populated. When false,
	// ownership policies such as RequireSelf must deny.
	HasResource bool
}

// PolicyFunc is a single authorization decision unit. It receives the request context
// and an AccessRequest, and returns a Decision. Policies compose via AllOf.
type PolicyFunc func(ctx context.Context, req AccessRequest) Decision

// AllOf returns a PolicyFunc that evaluates each policy in order and returns the first
// non-allowed Decision (fail-fast). If every policy allows, it returns Allow().
// An empty AllOf denies — the package is fail-closed; callers must register at least
// one policy that explicitly allows access.
func AllOf(policies ...PolicyFunc) PolicyFunc {
	return func(ctx context.Context, req AccessRequest) Decision {
		if len(policies) == 0 {
			return Deny("no policies defined")
		}
		for _, p := range policies {
			if d := p(ctx, req); !d.Allowed {
				return d
			}
		}
		return Allow()
	}
}

// RequirePermission returns a PolicyFunc that allows the request when the caller's
// PermissionSet (from req.Permissions) contains p. It uses its captured permission p
// and ignores req.Required so that composed policies remain independent.
func RequirePermission(p Permission) PolicyFunc {
	return func(_ context.Context, req AccessRequest) Decision {
		if req.Permissions.Has(p) {
			return Allow()
		}
		return Deny("missing permission: " + string(p))
	}
}

// RequireSelf returns a PolicyFunc that allows the request when the caller is the owner
// of the resource being accessed. Both req.HasResource must be true and
// req.SubjectID must equal req.ResourceOwnerID.
//
// This is an ownership primitive for use in handlers and services where the resource
// owner is known at call time. The authz interceptor does not set ResourceOwnerID,
// so RequireSelf must not be used as an interceptor-level policy alone.
func RequireSelf() PolicyFunc {
	return func(_ context.Context, req AccessRequest) Decision {
		if req.HasResource && req.SubjectID == req.ResourceOwnerID {
			return Allow()
		}
		return Deny("resource does not belong to caller")
	}
}
