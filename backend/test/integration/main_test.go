package integration_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/authdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/health"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/logging"
	platformredis "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/redis"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
	migrations "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/migrations"
)

var (
	baseURL     string
	dbDSN       string
	dbPool      interface{ Ping(context.Context) error }
	redisPinger db.Pinger
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "failed to terminate postgres container: %v\n", err) //nolint:errcheck
		}
	}()

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get postgres DSN: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	dbDSN = pgDSN
	if err := os.Setenv("DATABASE_URL", pgDSN); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set DATABASE_URL: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	if err := os.Setenv("DB_MAX_CONNS", "5"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set DB_MAX_CONNS: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}

	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start redis container: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	defer func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "failed to terminate redis container: %v\n", err) //nolint:errcheck
		}
	}()

	redisAddr, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get redis endpoint: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	redisURL := "redis://" + redisAddr
	if err := os.Setenv("REDIS_URL", redisURL); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set REDIS_URL: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}

	pool, err := db.NewPool(ctx, pgDSN, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create postgres pool: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	defer pool.Close()
	dbPool = pool

	if err := db.Migrate(pgDSN, migrations.FS); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}

	rPinger, err := platformredis.NewPinger(redisURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create redis pinger: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	redisPinger = rPinger

	// Auth wiring — mirrors cmd/api/main.go so WU3 integration tests can call auth endpoints.
	redisClient, err := platformredis.NewClient(redisURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create redis client for auth: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	testCfg := config.Config{
		BcryptCost:    4, // fast for tests
		SessionTTL:    time.Hour,
		ResetTokenTTL: 15 * time.Minute,
		AppEnv:        "test",
		CookieSecure:  false,
	}
	sessionStore := session.NewRedisStore(redisClient)
	roleLoader := auth.NoopRoleLoader{}
	authInterceptor := auth.NewSessionInterceptor(sessionStore, roleLoader, testCfg)
	queries := authdb.New(pool)
	repo := auth.NewPostgresRepository(queries)
	svc := auth.NewService(repo, sessionStore, roleLoader, testCfg)
	authHandler := auth.NewHandler(svc, testCfg)
	authOpts := server.Chain(authInterceptor)
	authReg := func(mux *http.ServeMux) {
		auth.Register(mux, authHandler, authOpts...)
	}

	log := logging.New(slog.LevelError) // suppress output in tests
	srv := server.New(log, pool, rPinger, health.Register, authReg)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to bind random port: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	baseURL = "http://" + listener.Addr().String()
	srv.Addr = listener.Addr().String()

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err) //nolint:errcheck
		}
	}()

	waitForServer(listener.Addr().String(), 5*time.Second)

	code := m.Run()

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "server shutdown error: %v\n", err) //nolint:errcheck
	}

	os.Exit(code)
}

// waitForServer polls the address until it accepts TCP connections or the timeout expires.
func waitForServer(addr string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			if err := conn.Close(); err != nil {
				_ = err
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
