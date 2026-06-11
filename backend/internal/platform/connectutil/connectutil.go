// Package connectutil provides request-boundary helpers for Connect RPC handlers.
package connectutil

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

// ParseUUID parses s as a UUID. On failure it returns a Connect
// CodeInvalidArgument error with the message "invalid UUID".
//
// This helper is for transport-layer parsing in handler methods; it does NOT
// apply field-specific context to the error message. Use it when the call site
// already makes the failing field unambiguous (e.g. a single-UUID request).
func ParseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid UUID"))
	}
	return id, nil
}
