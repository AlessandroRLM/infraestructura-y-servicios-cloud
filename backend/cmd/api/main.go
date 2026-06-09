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
	catalogv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1/catalogv1connect"
	profilesv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1/profilesv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/authdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog/catalogdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/health"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/logging"
	platformredis "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/redis"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles/profilesdb"
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

	// policies maps each protected procedure to a PolicyFunc. Every procedure not in
	// exempt and not in this map is denied (fail-closed). Profile management procedures
	// require users.manage; the self-read procedure requires profile.view_own.
	// All catalog procedures require catalog.manage.
	policies := map[string]authz.PolicyFunc{
		profilesv1connect.ProfileServiceUpsertUserProfileProcedure:         authz.RequirePermission(authz.PermUsersManage),
		profilesv1connect.ProfileServiceGetUserProfileProcedure:            authz.RequirePermission(authz.PermUsersManage),
		profilesv1connect.ProfileServiceUpsertStudentProfileProcedure:      authz.RequirePermission(authz.PermUsersManage),
		profilesv1connect.ProfileServiceGetStudentProfileProcedure:         authz.RequirePermission(authz.PermUsersManage),
		profilesv1connect.ProfileServiceUpsertTeacherProfileProcedure:      authz.RequirePermission(authz.PermUsersManage),
		profilesv1connect.ProfileServiceGetTeacherProfileProcedure:         authz.RequirePermission(authz.PermUsersManage),
		profilesv1connect.ProfileServiceAddTeacherQualificationProcedure:   authz.RequirePermission(authz.PermUsersManage),
		profilesv1connect.ProfileServiceListTeacherQualificationsProcedure: authz.RequirePermission(authz.PermUsersManage),
		profilesv1connect.ProfileServiceGetOwnProfileProcedure:             authz.RequirePermission(authz.PermProfileViewOwn),

		// Catalog procedures — all require catalog.manage.
		catalogv1connect.CatalogServiceCreateProgramProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateProgramProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetProgramProcedure:                authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListProgramsProcedure:              authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteProgramProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateCourseProcedure:              authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateCourseProcedure:              authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetCourseProcedure:                 authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListCoursesProcedure:               authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteCourseProcedure:              authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateAcademicPeriodProcedure:      authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateAcademicPeriodProcedure:      authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetAcademicPeriodProcedure:         authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListAcademicPeriodsProcedure:       authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteAcademicPeriodProcedure:      authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateProgramQuotaProcedure:        authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateProgramQuotaProcedure:        authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetProgramQuotaProcedure:           authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListProgramQuotasProcedure:         authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteProgramQuotaProcedure:        authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceAddCourseToProgramProcedure:        authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceRemoveCourseFromProgramProcedure:   authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListProgramCoursesProcedure:        authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateSectionProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateSectionProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetSectionProcedure:                authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListSectionsProcedure:              authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteSectionProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceAssignTeacherToSectionProcedure:    authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceRemoveTeacherFromSectionProcedure:  authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListSectionTeachersProcedure:       authz.RequirePermission(authz.PermCatalogManage),
	}

	authzInterceptor := auth.NewAuthzInterceptor(exempt, policies)

	// Auth handler (repository → service → Connect handler).
	queries := authdb.New(pool)
	repo := auth.NewPostgresRepository(queries)
	svc := auth.NewService(repo, sessionStore, roleLoader, cfg)
	authHandler := auth.NewHandler(svc, cfg)

	// Interceptor options shared across all service endpoints.
	authOpts := server.Chain(authInterceptor, authzInterceptor)

	// authReg curries auth.Register into the HandlerReg signature.
	authReg := func(mux *http.ServeMux) {
		auth.Register(mux, authHandler, authOpts...)
	}

	// Profiles handler (profilesdb.Querier → repository → service → Connect handler).
	profilesQueries := profilesdb.New(pool)
	profilesRepo := profiles.NewPostgresRepository(profilesQueries)
	profilesSvc := profiles.NewService(profilesRepo)
	profilesHandler := profiles.NewHandler(profilesSvc)

	profilesReg := func(mux *http.ServeMux) {
		profiles.Register(mux, profilesHandler, authOpts...)
	}

	// Catalog handler (catalogdb.Querier → repository → service → Connect handler).
	catalogQueries := catalogdb.New(pool)
	catalogRepo := catalog.NewPostgresRepository(catalogQueries, pool)
	catalogSvc := catalog.NewService(catalogRepo)
	catalogHandler := catalog.NewHandler(catalogSvc)

	catalogReg := func(mux *http.ServeMux) {
		catalog.Register(mux, catalogHandler, authOpts...)
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
		profilesReg,
		catalogReg,
	)
	srv.Addr = cfg.HTTPAddr

	log.Info("starting server", "addr", cfg.HTTPAddr)
	server.RunWithGracefulShutdown(ctx, srv)
}
