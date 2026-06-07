// Package main is the composition root for the API server.
// It wires configuration, logging, database, cache, migrations, and HTTP server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	migrations "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/migrations"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/authdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/health"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/logging"
	platformredis "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/redis"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac/rbacdb"
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
	roleLoader := rbac.NewPostgresRoleLoader(rbacdb.New(pool))
	authInterceptor := auth.NewSessionInterceptor(sessionStore, roleLoader, cfg)

	// exempt lists procedures that bypass the authz interceptor entirely.
	// Public procedures (no session) and authenticated-but-no-permission procedures
	// (Logout) belong here. Every other procedure must appear in policies below or
	// it will be denied automatically (fail-closed).
	exempt := map[string]struct{}{
		authv1connect.AuthServiceLoginProcedure:                {},
		authv1connect.AuthServiceRequestPasswordResetProcedure: {},
		authv1connect.AuthServiceConfirmPasswordResetProcedure: {},
		authv1connect.AuthServiceLogoutProcedure:               {},
	}

	// policies maps each protected procedure to a PolicyFunc. An empty map is valid
	// while no procedure requires a permission; domain slices add entries as they land.
	// Any procedure not in exempt and not in policies is denied (fail-closed).
	policies := map[string]authz.PolicyFunc{}

	authzInterceptor := auth.NewAuthzInterceptor(exempt, policies)

	// Auth handler (repository → service → Connect handler).
	queries := authdb.New(pool)
	repo := auth.NewPostgresRepository(queries)
	svc := auth.NewService(repo, sessionStore, roleLoader, cfg)
	authHandler := auth.NewHandler(svc, cfg)

	// Interceptor options for the auth service endpoint.
	authOpts := server.Chain(authInterceptor, authzInterceptor)

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
