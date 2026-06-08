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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

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
	migrations "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/migrations"
)

var (
	baseURL string
	dbDSN   string
	// dbPool is the minimal Ping interface exposed to health tests.
	dbPool interface{ Ping(context.Context) error }
	// pgxPool is the concrete pool used by test helpers to run raw SQL.
	pgxPool *pgxpool.Pool
	// testRedisClient is the raw Redis client used by test helpers to inspect
	// and manipulate keys (TTL assertions, expired-session simulation, etc.).
	testRedisClient *redis.Client
	// sharedCfg is the config used for the shared test server. Tests that need
	// to assert timing values (TTL, bcrypt cost) read it directly.
	sharedCfg config.Config
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
	pgxPool = pool

	if err := db.Migrate(pgDSN, migrations.FS); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}

	rPinger, err := platformredis.NewPinger(redisURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create redis pinger: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}

	redisClient, err := platformredis.NewClient(redisURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create redis client for auth: %v\n", err) //nolint:errcheck
		os.Exit(1)
	}
	testRedisClient = redisClient
	sharedCfg = config.Config{
		BcryptCost:    4, // fast for tests
		SessionTTL:    time.Hour,
		ResetTokenTTL: 15 * time.Minute,
		AppEnv:        "test",
		CookieSecure:  false,
	}
	testCfg := sharedCfg

	sessionStore := session.NewRedisStore(redisClient)
	// Use the real Postgres role loader so that permission-based tests exercise the full chain.
	roleLoader := rbac.NewPostgresRoleLoader(rbacdb.New(pool))
	authInterceptor := auth.NewSessionInterceptor(sessionStore, roleLoader, testCfg)

	// exempt mirrors cmd/api/main.go — public and authenticated-but-no-permission procedures.
	exempt := map[string]struct{}{
		authv1connect.AuthServiceLoginProcedure:                {},
		authv1connect.AuthServiceRequestPasswordResetProcedure: {},
		authv1connect.AuthServiceConfirmPasswordResetProcedure: {},
		authv1connect.AuthServiceLogoutProcedure:               {},
	}

	// policies mirrors cmd/api/main.go — all profiles and catalog procedures are registered.
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
		catalogv1connect.CatalogServiceCreateProgramProcedure:           authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateProgramProcedure:           authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetProgramProcedure:              authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListProgramsProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteProgramProcedure:           authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateCourseProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateCourseProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetCourseProcedure:               authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListCoursesProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteCourseProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateAcademicPeriodProcedure:    authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateAcademicPeriodProcedure:    authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetAcademicPeriodProcedure:       authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListAcademicPeriodsProcedure:     authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteAcademicPeriodProcedure:    authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateProgramQuotaProcedure:      authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateProgramQuotaProcedure:      authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetProgramQuotaProcedure:         authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListProgramQuotasProcedure:       authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteProgramQuotaProcedure:      authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceAddCourseToProgramProcedure:       authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceRemoveCourseFromProgramProcedure:  authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListProgramCoursesProcedure:       authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateSectionProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateSectionProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetSectionProcedure:               authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListSectionsProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteSectionProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceAssignTeacherToSectionProcedure:   authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceRemoveTeacherFromSectionProcedure: authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListSectionTeachersProcedure:      authz.RequirePermission(authz.PermCatalogManage),
	}

	authzInterceptor := auth.NewAuthzInterceptor(exempt, policies)

	queries := authdb.New(pool)
	repo := auth.NewPostgresRepository(queries)
	svc := auth.NewService(repo, sessionStore, roleLoader, testCfg)
	authHandler := auth.NewHandler(svc, testCfg)
	authOpts := server.Chain(authInterceptor, authzInterceptor)
	authReg := func(mux *http.ServeMux) {
		auth.Register(mux, authHandler, authOpts...)
	}

	// Profiles handler wiring.
	profilesQueries := profilesdb.New(pool)
	profilesRepo := profiles.NewPostgresRepository(profilesQueries)
	profilesSvc := profiles.NewService(profilesRepo)
	profilesHandler := profiles.NewHandler(profilesSvc)
	profilesReg := func(mux *http.ServeMux) {
		profiles.Register(mux, profilesHandler, authOpts...)
	}

	// Catalog handler wiring.
	catalogQueries := catalogdb.New(pool)
	catalogRepo := catalog.NewPostgresRepository(catalogQueries)
	catalogSvc := catalog.NewService(catalogRepo)
	catalogHandler := catalog.NewHandler(catalogSvc)
	catalogReg := func(mux *http.ServeMux) {
		catalog.Register(mux, catalogHandler, authOpts...)
	}

	log := logging.New(slog.LevelError) // suppress output in tests
	srv := server.New(log, pool, rPinger, health.Register, authReg, profilesReg, catalogReg)

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
