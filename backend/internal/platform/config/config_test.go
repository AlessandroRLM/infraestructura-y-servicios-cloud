package config_test

import (
	"log/slog"
	"testing"

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
			env: map[string]string{
				"DATABASE_URL":  testDBURL,
				"REDIS_URL":     testRedisURL,
				"DB_MAX_CONNS": "5",
				"HTTP_ADDR":    ":9090",
				"LOG_LEVEL":    "warn",
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL: testDBURL,
				RedisURL:    testRedisURL,
				DBMaxConns:  5,
				HTTPAddr:    ":9090",
				LogLevel:    slog.LevelWarn,
			},
		},
		{
			name: "missing_database_url",
			env: map[string]string{
				"REDIS_URL": testRedisURL,
			},
			wantErr: true,
		},
		{
			name: "missing_redis_url",
			env: map[string]string{
				"DATABASE_URL": testDBURL,
			},
			wantErr: true,
		},
		{
			name: "db_max_conns_absent_defaults_to_10",
			env: map[string]string{
				"DATABASE_URL": testDBURL,
				"REDIS_URL":    testRedisURL,
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL: testDBURL,
				RedisURL:    testRedisURL,
				DBMaxConns:  10,
				HTTPAddr:    ":8080",
				LogLevel:    slog.LevelInfo,
			},
		},
		{
			name: "db_max_conns_zero_is_error",
			env: map[string]string{
				"DATABASE_URL":  testDBURL,
				"REDIS_URL":     testRedisURL,
				"DB_MAX_CONNS": "0",
			},
			wantErr: true,
		},
		{
			name: "db_max_conns_negative_is_error",
			env: map[string]string{
				"DATABASE_URL":  testDBURL,
				"REDIS_URL":     testRedisURL,
				"DB_MAX_CONNS": "-1",
			},
			wantErr: true,
		},
		{
			name: "http_addr_absent_defaults_to_8080",
			env: map[string]string{
				"DATABASE_URL": testDBURL,
				"REDIS_URL":    testRedisURL,
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL: testDBURL,
				RedisURL:    testRedisURL,
				DBMaxConns:  10,
				HTTPAddr:    ":8080",
				LogLevel:    slog.LevelInfo,
			},
		},
		{
			name: "log_level_absent_defaults_to_info",
			env: map[string]string{
				"DATABASE_URL": testDBURL,
				"REDIS_URL":    testRedisURL,
			},
			wantErr: false,
			wantCfg: &config.Config{
				DatabaseURL: testDBURL,
				RedisURL:    testRedisURL,
				DBMaxConns:  10,
				HTTPAddr:    ":8080",
				LogLevel:    slog.LevelInfo,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"DATABASE_URL", "REDIS_URL", "DB_MAX_CONNS", "HTTP_ADDR", "LOG_LEVEL"} {
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
		})
	}
}
