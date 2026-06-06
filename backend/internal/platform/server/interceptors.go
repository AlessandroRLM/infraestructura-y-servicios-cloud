// Package server provides the HTTP server, routing, and middleware for the API.
package server

import "connectrpc.com/connect"

// Chain returns the ordered Connect interceptor slice applied to all handlers.
// Auth, authz, and audit interceptors will be appended here in future slices.
func Chain() []connect.Interceptor {
	return nil
}
