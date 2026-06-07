// Package server provides the HTTP server, routing, and middleware for the API.
package server

import (
	"connectrpc.com/connect"
)

// Chain converts the provided unary interceptor functions into a slice of
// connect.HandlerOption values suitable for passing to NewXxxHandler constructors.
// Callers that want no interceptors (e.g. health) call Chain() with no arguments.
func Chain(interceptors ...connect.UnaryInterceptorFunc) []connect.HandlerOption {
	opts := make([]connect.HandlerOption, 0, len(interceptors))
	for _, fn := range interceptors {
		opts = append(opts, connect.WithInterceptors(fn))
	}
	return opts
}
