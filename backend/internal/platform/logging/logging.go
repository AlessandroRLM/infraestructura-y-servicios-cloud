// Package logging provides a structured JSON logger backed by slog.
package logging

import (
	"log/slog"
	"os"
)

// New returns an *slog.Logger writing JSON to stdout at the given level.
// Use slog.SetDefault(New(...)) to make it the process-global logger.
func New(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
