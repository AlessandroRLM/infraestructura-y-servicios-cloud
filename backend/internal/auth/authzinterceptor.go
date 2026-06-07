package auth

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
)

// NewAuthzInterceptor returns a Connect unary interceptor that enforces authorization
// for every procedure according to the provided exempt set and policy map.
//
// Per-request routing:
//   - Procedure in exempt → pass through unconditionally. Use this for public
//     procedures (Login, RequestPasswordReset, ConfirmPasswordReset) and for
//     authenticated-but-no-permission procedures (Logout).
//   - Procedure in policies → evaluate the PolicyFunc. Allowed → pass through;
//     infra failure (Decision.Err != nil) → CodeInternal (generic, reason logged server-side);
//     denied → CodePermissionDenied with a generic "permission denied" message (reason logged).
//   - Procedure in neither → DENY with CodePermissionDenied (fail-closed). This is
//     the key guarantee: every new procedure must be consciously classified.
//
// The session interceptor must run before this interceptor so that
// authz.PermissionsFromContext returns the correct set for mapped policies.
func NewAuthzInterceptor(exempt map[string]struct{}, policies map[string]authz.PolicyFunc) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			proc := req.Spec().Procedure

			if _, ok := exempt[proc]; ok {
				return next(ctx, req)
			}

			pf, ok := policies[proc]
			if !ok {
				// Fail-closed: an unmapped procedure is denied. This forces every
				// new procedure to be explicitly added to either exempt or policies.
				slog.InfoContext(ctx, "authorization denied",
					"procedure", proc,
					"reason", "no authorization policy defined for procedure "+proc,
				)
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
			}

			subjectID, _ := UserIDFromContext(ctx)
			perms, _ := authz.PermissionsFromContext(ctx)

			ar := authz.AccessRequest{
				SubjectID:   subjectID,
				Permissions: perms,
				// ResourceOwnerID and HasResource are left at zero: the interceptor
				// operates at the procedure level and does not know the resource owner.
				// Resource-level ownership checks belong in the handler or service.
			}

			d := pf(ctx, ar)
			if !d.Allowed {
				if d.Err != nil {
					// Infra failure inside the policy: do not leak the internal error to
					// the caller; return a generic Internal code instead.
					slog.ErrorContext(ctx, "authorization policy internal error",
						"reason", d.Reason,
						"err", d.Err,
					)
					return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
				}
				// Normal policy denial: log the reason server-side, send generic message.
				slog.InfoContext(ctx, "authorization denied",
					"procedure", proc,
					"reason", d.Reason,
				)
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
			}

			return next(ctx, req)
		}
	}
}
