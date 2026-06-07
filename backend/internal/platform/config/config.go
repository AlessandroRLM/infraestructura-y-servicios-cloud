// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
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

	// SessionTTL is the sliding session expiry duration. Required; must be > 0.
	SessionTTL time.Duration
	// BcryptCost is the bcrypt work factor used when hashing passwords. Defaults to 12.
	BcryptCost int
	// ResetTokenTTL is the expiry duration for password-reset tokens. Required; must be > 0.
	ResetTokenTTL time.Duration
	// CookieSecure controls the Secure attribute on the session cookie. Defaults to false.
	CookieSecure bool
	// AppEnv identifies the runtime environment (e.g. "production"). Defaults to "".
	AppEnv string
}

const (
	defaultHTTPAddr   = ":8080"
	defaultDBMaxConns = 10
	defaultLogLevel   = slog.LevelInfo
	defaultBcryptCost = 12
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

	sessionTTL, err := requireDuration("SESSION_TTL")
	if err != nil {
		return Config{}, err
	}
	cfg.SessionTTL = sessionTTL

	if raw := os.Getenv("BCRYPT_COST"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("config: BCRYPT_COST must be a valid integer, got %q", raw)
		}
		if n < bcrypt.MinCost || n > bcrypt.MaxCost {
			return Config{}, fmt.Errorf("config: BCRYPT_COST must be between %d and %d, got %d", bcrypt.MinCost, bcrypt.MaxCost, n)
		}
		cfg.BcryptCost = n
	} else {
		cfg.BcryptCost = defaultBcryptCost
	}

	resetTokenTTL, err := requireDuration("RESET_TOKEN_TTL")
	if err != nil {
		return Config{}, err
	}
	cfg.ResetTokenTTL = resetTokenTTL

	if raw := os.Getenv("COOKIE_SECURE"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("config: COOKIE_SECURE must be a boolean, got %q", raw)
		}
		cfg.CookieSecure = v
	}

	cfg.AppEnv = os.Getenv("APP_ENV")

	return cfg, nil
}

// requireDuration reads an environment variable as a time.Duration and returns
// an error if it is absent or parses to zero.
func requireDuration(key string) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, fmt.Errorf("config: required env var %s is not set", key)
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a valid duration, got %q", key, raw)
	}
	if d <= 0 {
		return 0, fmt.Errorf("config: %s must be > 0, got %v", key, d)
	}
	return d, nil
}
