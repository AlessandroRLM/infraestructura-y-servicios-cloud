package auth

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
)

// errPermissionDenied is the underlying error for authorization failures.
var errPermissionDenied = errors.New("permission denied")

// NewAuthzInterceptor returns a Connect unary interceptor that enforces permission
// checks based on the provided required-permission map.
//
// For each incoming request it looks up req.Spec().Procedure in required:
//   - If the procedure is not in the map, the request passes through unconditionally.
//   - If the procedure is in the map, the interceptor calls policy.Evaluate with the
//     required permission. Allowed = true → pass through. Allowed = false →
//     connect.CodePermissionDenied (distinct from the session interceptor's
//     connect.CodeUnauthenticated so clients can distinguish re-login from access-denied).
//
// Inject an empty map to deploy the mechanism without enforcing any procedure yet.
// The session interceptor must run before this interceptor in the chain so that
// authz.PermissionsFromContext returns the correct set.
func NewAuthzInterceptor(required map[string]authz.Permission, policy authz.Policy) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			perm, ok := required[req.Spec().Procedure]
			if !ok {
				return next(ctx, req)
			}

			decision := policy.Evaluate(ctx, perm)
			if !decision.Allowed {
				return nil, connect.NewError(connect.CodePermissionDenied, errPermissionDenied)
			}

			return next(ctx, req)
		}
	}
}
