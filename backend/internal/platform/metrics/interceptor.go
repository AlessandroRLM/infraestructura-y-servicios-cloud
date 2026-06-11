package metrics

import (
	"context"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// RPCInterceptor returns a Connect unary interceptor that records RED signals (Request
// rate, Error rate, Duration) for every procedure it wraps. It is designed to be placed
// OUTERMOST in the interceptor chain so that rejected requests (unauthenticated, authz
// denied, limiter saturated) are counted with their respective status codes.
//
// Metrics recorded:
//   - academico_rpc_requests_total{service,method,code} — incremented once per call
//   - academico_rpc_duration_seconds{service,method}    — observed once per call
//
// Label derivation: service and method are parsed from req.Spec().Procedure which has the
// form "/package.ServiceName/MethodName". The code label is connect.CodeOf(err).String()
// on error and "ok" on success.
func (r *Registry) RPCInterceptor() connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			elapsed := time.Since(start).Seconds()

			service, method := parseProcedure(req.Spec().Procedure)
			code := "ok"
			if err != nil {
				code = connect.CodeOf(err).String()
			}

			r.rpcRequests.WithLabelValues(service, method, code).Inc()
			r.rpcDuration.WithLabelValues(service, method).Observe(elapsed)

			return resp, err
		})
	})
}

// parseProcedure splits a Connect procedure string of the form
// "/package.ServiceName/MethodName" into (ServiceName, MethodName).
// Malformed procedure strings fall back to "unknown".
func parseProcedure(procedure string) (service, method string) {
	// procedure format: "/com.example.v1.ServiceName/MethodName"
	// Remove leading slash, then split on "/".
	trimmed := strings.TrimPrefix(procedure, "/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return "unknown", "unknown"
	}
	// parts[0] = "com.example.v1.ServiceName", parts[1] = "MethodName"
	method = parts[1]
	// Service name is the last dot-separated segment of parts[0].
	segments := strings.Split(parts[0], ".")
	service = segments[len(segments)-1]
	return service, method
}
