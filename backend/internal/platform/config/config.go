// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Config holds all validated application configuration.
type Config struct {
	// HTTPAddr is the address the HTTP server listens on. Defaults to ":8080".
	HTTPAddr string
	// DatabaseURL is the direct Postgres DSN used by both the connection pool and migrations. Required.
	DatabaseURL string
	// DBMaxConns caps the pgxpool connection count per instance. Defaults to 10. Must be > 0.
	DBMaxConns int
	// RedisURL is the Redis connection URL. Required.
	RedisURL string
	// LogLevel is the slog log level. Defaults to slog.LevelInfo.
	LogLevel slog.Level
}

const (
	defaultHTTPAddr    = ":8080"
	defaultDBMaxConns  = 10
	defaultLogLevel    = slog.LevelInfo
)

// Load reads configuration from environment variables, applies defaults for optional
// variables, and returns an error when any required variable is absent or invalid.
func Load() (Config, error) {
	var cfg Config

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("config: required env var DATABASE_URL is not set")
	}
	cfg.DatabaseURL = dbURL

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return Config{}, fmt.Errorf("config: required env var REDIS_URL is not set")
	}
	cfg.RedisURL = redisURL

	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		cfg.HTTPAddr = addr
	} else {
		cfg.HTTPAddr = defaultHTTPAddr
	}

	if raw := os.Getenv("DB_MAX_CONNS"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("config: DB_MAX_CONNS must be a valid integer, got %q", raw)
		}
		if n <= 0 {
			return Config{}, fmt.Errorf("config: DB_MAX_CONNS must be > 0, got %d", n)
		}
		cfg.DBMaxConns = n
	} else {
		cfg.DBMaxConns = defaultDBMaxConns
	}

	if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		var level slog.Level
		if err := level.UnmarshalText([]byte(raw)); err != nil {
			// Unparseable level: fall back to default rather than error (per spec).
			cfg.LogLevel = defaultLogLevel
		} else {
			cfg.LogLevel = level
		}
	} else {
		cfg.LogLevel = defaultLogLevel
	}

	return cfg, nil
}
