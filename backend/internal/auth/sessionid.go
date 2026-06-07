// Package auth implements the session-based authentication slice.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// NewSessionID generates a cryptographically random 256-bit (32-byte) session
// identifier encoded as URL-safe base64 with no padding (43 characters).
func NewSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: failed to generate session id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
