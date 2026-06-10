package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

// testDBURL and testRedisURL are placeholder URLs used only in unit tests.
// They are not real credentials.
//
//nolint:gosec // G101: test fixture — not a real credential.
const (
	testDBURL    = "postgres://testuser:testpass@localhost:5432/testdb"
	testRedisURL = "redis://localhost:6379"
)

// baseAuthEnv provides the minimum auth env vars required by Load().
var baseAuthEnv = map[string]string{
	"DATABASE_URL":      testDBURL,
	"REDIS_URL":         testRedisURL,
	"SESSION_TTL":       "3600s",
	"BCRYPT_COST":       "4",
	"RESET_TOKEN_TTL":   "900s",
	"REPORTS_CACHE_TTL": "5m",
}

func TestLoad(t *testing.T) {
	// Note: t.Parallel() is intentionally omitted at the top level because subtests
	// use t.Setenv, which mutates shared environment state and cannot run in parallel.

	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		wantCfg *config.Config
	}{
		{
			name: "valid_all_vars",
			env: func() map[string]string {
				m := make(map[string]string, len(baseAuthEnv)+4)
				for k, v := range baseAuthEnv {
					m[k] = v
				}
				m["DB_MAX_CONNS"] = "5"
				m["HTTP_ADDR"] = ":9090"
				m["LOG_LEVEL"] = "warn"
				m["COOKIE_SECURE"] = "true"
				m["APP_ENV"] = "staging"
				return m
			}(),
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      5,
				HTTPAddr:        ":9090",
				LogLevel:        slog.LevelWarn,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      4,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
				CookieSecure:    true,
				AppEnv:          "staging",
			},
		},
		{
			name: "missing_database_url",
			env: map[string]string{
				"REDIS_URL":       testRedisURL,
				"SESSION_TTL":     "3600s",
				"BCRYPT_COST":     "4",
				"RESET_TOKEN_TTL": "900s",
			},
			wantErr: true,
		},
		{
			name: "missing_redis_url",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"SESSION_TTL":     "3600s",
				"BCRYPT_COST":     "4",
				"RESET_TOKEN_TTL": "900s",
			},
			wantErr: true,
		},
		{
			name: "db_max_conns_absent_defaults_to_10",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "4",
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      4,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
			},
		},
		{
			name: "db_max_conns_zero_is_error",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"REDIS_URL":       testRedisURL,
				"DB_MAX_CONNS":    "0",
				"SESSION_TTL":     "3600s",
				"BCRYPT_COST":     "4",
				"RESET_TOKEN_TTL": "900s",
			},
			wantErr: true,
		},
		{
			name: "db_max_conns_negative_is_error",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"REDIS_URL":       testRedisURL,
				"DB_MAX_CONNS":    "-1",
				"SESSION_TTL":     "3600s",
				"BCRYPT_COST":     "4",
				"RESET_TOKEN_TTL": "900s",
			},
			wantErr: true,
		},
		{
			name: "http_addr_absent_defaults_to_8080",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "4",
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      4,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
			},
		},
		{
			name: "log_level_absent_defaults_to_info",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "4",
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      4,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
			},
		},
		// Auth config tests — Phase 1
		{
			name: "missing_session_ttl_is_error",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"REDIS_URL":       testRedisURL,
				"BCRYPT_COST":     "4",
				"RESET_TOKEN_TTL": "900s",
			},
			wantErr: true,
		},
		{
			name: "session_ttl_zero_is_error",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"REDIS_URL":       testRedisURL,
				"SESSION_TTL":     "0s",
				"BCRYPT_COST":     "4",
				"RESET_TOKEN_TTL": "900s",
			},
			wantErr: true,
		},
		{
			name: "missing_reset_token_ttl_is_error",
			env: map[string]string{
				"DATABASE_URL": testDBURL,
				"REDIS_URL":    testRedisURL,
				"SESSION_TTL":  "3600s",
				"BCRYPT_COST":  "4",
			},
			wantErr: true,
		},
		{
			name: "reset_token_ttl_zero_is_error",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"REDIS_URL":       testRedisURL,
				"SESSION_TTL":     "3600s",
				"BCRYPT_COST":     "4",
				"RESET_TOKEN_TTL": "0s",
			},
			wantErr: true,
		},
		{
			name: "bcrypt_cost_absent_defaults_to_12",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      12,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
			},
		},
		{
			name: "cookie_secure_defaults_to_false",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "4",
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      4,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
				CookieSecure:    false,
			},
		},
		{
			name: "app_env_absent_defaults_to_empty",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "4",
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      4,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
				AppEnv:          "",
			},
		},
		// BCRYPT_COST range validation
		{
			name: "bcrypt_cost_below_min_is_error",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"REDIS_URL":       testRedisURL,
				"SESSION_TTL":     "3600s",
				"BCRYPT_COST":     "3", // bcrypt.MinCost is 4
				"RESET_TOKEN_TTL": "900s",
			},
			wantErr: true,
		},
		{
			name: "bcrypt_cost_above_max_is_error",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"REDIS_URL":       testRedisURL,
				"SESSION_TTL":     "3600s",
				"BCRYPT_COST":     "32", // bcrypt.MaxCost is 31
				"RESET_TOKEN_TTL": "900s",
			},
			wantErr: true,
		},
		{
			name: "bcrypt_cost_at_min_is_valid",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "4", // bcrypt.MinCost
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      4,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
			},
		},
		{
			name: "bcrypt_cost_at_max_is_valid",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "31", // bcrypt.MaxCost
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      31,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
			},
		},
		// Reports cache TTL tests
		{
			name: "missing_reports_cache_ttl_is_error",
			env: map[string]string{
				"DATABASE_URL":    testDBURL,
				"REDIS_URL":       testRedisURL,
				"SESSION_TTL":     "3600s",
				"BCRYPT_COST":     "4",
				"RESET_TOKEN_TTL": "900s",
				// REPORTS_CACHE_TTL intentionally absent
			},
			wantErr: true,
		},
		{
			name: "reports_cache_ttl_zero_is_error",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "4",
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "0s",
			},
			wantErr: true,
		},
		{
			name: "reports_cache_ttl_5m_is_valid",
			env: map[string]string{
				"DATABASE_URL":      testDBURL,
				"REDIS_URL":         testRedisURL,
				"SESSION_TTL":       "3600s",
				"BCRYPT_COST":       "4",
				"RESET_TOKEN_TTL":   "900s",
				"REPORTS_CACHE_TTL": "5m",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL:     testDBURL,
				RedisURL:        testRedisURL,
				DBMaxConns:      10,
				HTTPAddr:        ":8080",
				LogLevel:        slog.LevelInfo,
				SessionTTL:      3600 * time.Second,
				BcryptCost:      4,
				ResetTokenTTL:   900 * time.Second,
				ReportsCacheTTL: 5 * time.Minute,
			},
		},
	}

	allKeys := []string{
		"DATABASE_URL", "REDIS_URL", "DB_MAX_CONNS", "HTTP_ADDR", "LOG_LEVEL",
		"SESSION_TTL", "BCRYPT_COST", "RESET_TOKEN_TTL", "COOKIE_SECURE", "APP_ENV",
		"REPORTS_CACHE_TTL",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range allKeys {
				t.Setenv(key, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := config.Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
				return
			}

			if tt.wantCfg == nil {
				return
			}

			if cfg.DatabaseURL != tt.wantCfg.DatabaseURL {
				t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, tt.wantCfg.DatabaseURL)
			}
			if cfg.RedisURL != tt.wantCfg.RedisURL {
				t.Errorf("RedisURL = %q, want %q", cfg.RedisURL, tt.wantCfg.RedisURL)
			}
			if cfg.DBMaxConns != tt.wantCfg.DBMaxConns {
				t.Errorf("DBMaxConns = %d, want %d", cfg.DBMaxConns, tt.wantCfg.DBMaxConns)
			}
			if cfg.HTTPAddr != tt.wantCfg.HTTPAddr {
				t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, tt.wantCfg.HTTPAddr)
			}
			if cfg.LogLevel != tt.wantCfg.LogLevel {
				t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, tt.wantCfg.LogLevel)
			}
			if cfg.SessionTTL != tt.wantCfg.SessionTTL {
				t.Errorf("SessionTTL = %v, want %v", cfg.SessionTTL, tt.wantCfg.SessionTTL)
			}
			if cfg.BcryptCost != tt.wantCfg.BcryptCost {
				t.Errorf("BcryptCost = %d, want %d", cfg.BcryptCost, tt.wantCfg.BcryptCost)
			}
			if cfg.ResetTokenTTL != tt.wantCfg.ResetTokenTTL {
				t.Errorf("ResetTokenTTL = %v, want %v", cfg.ResetTokenTTL, tt.wantCfg.ResetTokenTTL)
			}
			if cfg.CookieSecure != tt.wantCfg.CookieSecure {
				t.Errorf("CookieSecure = %v, want %v", cfg.CookieSecure, tt.wantCfg.CookieSecure)
			}
			if cfg.AppEnv != tt.wantCfg.AppEnv {
				t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, tt.wantCfg.AppEnv)
			}
			if cfg.ReportsCacheTTL != tt.wantCfg.ReportsCacheTTL {
				t.Errorf("ReportsCacheTTL = %v, want %v", cfg.ReportsCacheTTL, tt.wantCfg.ReportsCacheTTL)
			}
		})
	}
}
