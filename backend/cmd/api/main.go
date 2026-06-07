// Package main is the composition root for the API server.
// It wires configuration, logging, database, cache, migrations, and HTTP server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	migrations "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/migrations"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/authdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/health"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/logging"
	platformredis "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/redis"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	log := logging.New(cfg.LogLevel)
	slog.SetDefault(log)

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		log.Error("failed to create postgres pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Error("postgres ping failed", "err", err)
		os.Exit(1)
	}

	if err := db.Migrate(cfg.DatabaseURL, migrations.FS); err != nil {
		log.Error("migration failed", "err", err)
		os.Exit(1)
	}

	redisClient, err := platformredis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Error("failed to create redis client", "err", err)
		os.Exit(1)
	}

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error("redis ping failed", "err", err)
		os.Exit(1)
	}

	// Auth dependencies.
	sessionStore := session.NewRedisStore(redisClient)
	roleLoader := auth.NoopRoleLoader{}
	authInterceptor := auth.NewSessionInterceptor(sessionStore, roleLoader, cfg)

	// Auth handler (repository → service → Connect handler).
	queries := authdb.New(pool)
	repo := auth.NewPostgresRepository(queries)
	svc := auth.NewService(repo, sessionStore, roleLoader, cfg)
	authHandler := auth.NewHandler(svc, cfg)

	// Interceptor options for the auth service endpoint.
	authOpts := server.Chain(authInterceptor)

	// authReg curries auth.Register into the HandlerReg signature.
	authReg := func(mux *http.ServeMux) {
		auth.Register(mux, authHandler, authOpts...)
	}

	// Redis pinger for the readyz handler.
	redisPinger, err := platformredis.NewPinger(cfg.RedisURL)
	if err != nil {
		log.Error("failed to create redis pinger", "err", err)
		os.Exit(1)
	}

	srv := server.New(log, pool, redisPinger,
		health.Register,
		authReg,
	)
	srv.Addr = cfg.HTTPAddr

	log.Info("starting server", "addr", cfg.HTTPAddr)
	server.RunWithGracefulShutdown(ctx, srv)
}
