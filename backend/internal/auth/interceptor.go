package auth

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

// publicProcedures lists the procedures that do not require an authenticated session.
var publicProcedures = map[string]struct{}{
	authv1connect.AuthServiceLoginProcedure:                {},
	authv1connect.AuthServiceRequestPasswordResetProcedure: {},
	authv1connect.AuthServiceConfirmPasswordResetProcedure: {},
}

var (
	errSessionMissing = errors.New("session cookie required")
	errSessionInvalid = errors.New("session expired or invalid")
)

// NewSessionInterceptor returns a Connect interceptor that authenticates each
// request from its session cookie. Procedures in the public allowlist (login and
// the password-reset pair) pass through unauthenticated. For every other procedure
// it validates the "sid" cookie against the session store, renewing the session's
// sliding expiry on each call, and rejects the request with CodeUnauthenticated when
// the cookie is missing, expired, or invalid. On success it adds the authenticated
// user's identity to the context and loads their roles.
func NewSessionInterceptor(store session.Store, loader RoleLoader, cfg config.Config) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure
			if _, public := publicProcedures[procedure]; public {
				return next(ctx, req)
			}

			sid := parseSID(req.Header())
			if sid == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errSessionMissing)
			}

			sess, err := store.Touch(ctx, sid, cfg.SessionTTL)
			if err != nil {
				if errors.Is(err, session.ErrNotFound) {
					return nil, connect.NewError(connect.CodeUnauthenticated, errSessionInvalid)
				}
				slog.ErrorContext(ctx, "session store touch failure", "error", err)
				return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
			}

			ctx = WithUserID(ctx, sess.UserID)

			perms, err := loader.Load(ctx, sess.UserID)
			if err != nil {
				slog.ErrorContext(ctx, "role loader failure", "error", err)
				return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
			}
			ctx = authz.WithPermissions(ctx, perms)

			return next(ctx, req)
		}
	}
}

