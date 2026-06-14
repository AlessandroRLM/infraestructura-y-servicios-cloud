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

	auditlogsv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1/auditlogsv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	catalogv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1/catalogv1connect"
	enrollmentv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1/enrollmentv1connect"
	gradesv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1/gradesv1connect"
	iamv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/iam/v1/iamv1connect"
	profilesv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1/profilesv1connect"
	reportsv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1/reportsv1connect"
	section_enrollmentv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1/section_enrollmentv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auditlogs"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auditlogs/auditlogsdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/authdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog/catalogdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/enrollment"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/enrollment/enrollmentdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/grades"
	gradesdb "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/grades/gradesdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/health"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam/iamdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/logging"
	platformmetrics "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/metrics"
	platformredis "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/redis"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles/profilesdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac/rbacdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/reports"
	reportsdb "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/reports/reportsdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment/sectionenrollmentdb"
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
	if err := os.Setenv("REPORTS_CACHE_TTL", "5m"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set REPORTS_CACHE_TTL: %v\n", err) //nolint:errcheck
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
		BcryptCost:       4, // fast for tests
		SessionTTL:       time.Hour,
		ResetTokenTTL:    15 * time.Minute,
		ReportsCacheTTL:  5 * time.Minute,
		AppEnv:           "test",
		CookieSecure:     false,
		MetricsAuthToken: "test-metrics-token",
	}
	testCfg := sharedCfg

	// Metrics registry for the test harness. A fresh registry ensures no counter
	// state bleeds between test runs in the same binary.
	testMetricsReg := platformmetrics.New()
	testSEMetrics := testMetricsReg.SectionEnrollmentMetrics()
	testREDInterceptor := testMetricsReg.RPCInterceptor()

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
		authv1connect.AuthServiceGetSessionProcedure:           {},
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
		profilesv1connect.ProfileServiceUpsertOwnProfileProcedure:          authz.RequirePermission(authz.PermProfileEditOwn),

		// Catalog procedures — all require catalog.manage.
		catalogv1connect.CatalogServiceCreateProgramProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateProgramProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetProgramProcedure:               authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListProgramsProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteProgramProcedure:            authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateCourseProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateCourseProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetCourseProcedure:                authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListCoursesProcedure:              authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteCourseProcedure:             authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateAcademicPeriodProcedure:     authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateAcademicPeriodProcedure:     authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetAcademicPeriodProcedure:        authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListAcademicPeriodsProcedure:      authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteAcademicPeriodProcedure:     authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceCreateProgramQuotaProcedure:       authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceUpdateProgramQuotaProcedure:       authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceGetProgramQuotaProcedure:          authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceListProgramQuotasProcedure:        authz.RequirePermission(authz.PermCatalogManage),
		catalogv1connect.CatalogServiceDeleteProgramQuotaProcedure:       authz.RequirePermission(authz.PermCatalogManage),
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

		// Enrollment management procedures — require enrollment.manage.
		enrollmentv1connect.EnrollmentServiceCreateEnrollmentProcedure:   authz.RequirePermission(authz.PermEnrollmentManage),
		enrollmentv1connect.EnrollmentServiceMarkEnrollmentPaidProcedure: authz.RequirePermission(authz.PermEnrollmentManage),
		enrollmentv1connect.EnrollmentServiceCancelEnrollmentProcedure:   authz.RequirePermission(authz.PermEnrollmentManage),
		enrollmentv1connect.EnrollmentServiceGetEnrollmentProcedure:      authz.RequirePermission(authz.PermEnrollmentManage),
		enrollmentv1connect.EnrollmentServiceListEnrollmentsProcedure:    authz.RequirePermission(authz.PermEnrollmentManage),
		// Enrollment self-view procedures — require enrollment.view_own.
		enrollmentv1connect.EnrollmentServiceListOwnEnrollmentsProcedure: authz.RequirePermission(authz.PermEnrollmentViewOwn),
		enrollmentv1connect.EnrollmentServiceGetOwnEnrollmentProcedure:   authz.RequirePermission(authz.PermEnrollmentViewOwn),

		// Section enrollment self-service procedures.
		section_enrollmentv1connect.SectionEnrollmentServiceEnrollOwnSectionProcedure:          authz.RequirePermission(authz.PermSectionsEnroll),
		section_enrollmentv1connect.SectionEnrollmentServiceListOwnSectionEnrollmentsProcedure: authz.RequirePermission(authz.PermSectionEnrollmentViewOwn),
		section_enrollmentv1connect.SectionEnrollmentServiceGetOwnSectionEnrollmentProcedure:   authz.RequirePermission(authz.PermSectionEnrollmentViewOwn),
		// Section enrollment admin procedures — require enrollment.manage.
		section_enrollmentv1connect.SectionEnrollmentServiceEnrollSectionProcedure:          authz.RequirePermission(authz.PermEnrollmentManage),
		section_enrollmentv1connect.SectionEnrollmentServiceWithdrawSectionProcedure:        authz.RequirePermission(authz.PermEnrollmentManage),
		section_enrollmentv1connect.SectionEnrollmentServiceGetSectionEnrollmentProcedure:   authz.RequirePermission(authz.PermEnrollmentManage),
		section_enrollmentv1connect.SectionEnrollmentServiceListSectionEnrollmentsProcedure: authz.RequirePermission(authz.PermEnrollmentManage),

		// Grades admin procedures — require grades.override.
		gradesv1connect.GradesServiceCreateEvaluationSchemeProcedure:   authz.RequirePermission(authz.PermGradesOverride),
		gradesv1connect.GradesServiceRecreateEvaluationSchemeProcedure: authz.RequirePermission(authz.PermGradesOverride),
		gradesv1connect.GradesServiceOverrideGradeProcedure:            authz.RequirePermission(authz.PermGradesOverride),
		// Grades teacher write procedure — require grades.write.
		gradesv1connect.GradesServiceRecordGradeProcedure: authz.RequirePermission(authz.PermGradesWrite),
		// Grades read procedures — require grades.read.
		gradesv1connect.GradesServiceListEvaluationsProcedure:      authz.RequirePermission(authz.PermGradesRead),
		gradesv1connect.GradesServiceListGradesForSectionProcedure: authz.RequirePermission(authz.PermGradesRead),
		gradesv1connect.GradesServiceGetGradeProcedure:             authz.RequirePermission(authz.PermGradesRead),
		// Grades student self-view — require grades.view_own.
		gradesv1connect.GradesServiceListOwnGradesProcedure: authz.RequirePermission(authz.PermGradesViewOwn),

		// Reports procedures — all require reports.read.
		reportsv1connect.ReportsServiceGetSectionGradeReportProcedure:     authz.RequirePermission(authz.PermReportsRead),
		reportsv1connect.ReportsServiceGetSectionOccupancyReportProcedure: authz.RequirePermission(authz.PermReportsRead),
		reportsv1connect.ReportsServiceGetProgramSummaryReportProcedure:   authz.RequirePermission(authz.PermReportsRead),
		reportsv1connect.ReportsServiceGetStudentRecordReportProcedure:    authz.RequirePermission(authz.PermReportsRead),

		// Audit logs procedure — requires audit.read.
		auditlogsv1connect.AuditLogsServiceListAuditLogsProcedure: authz.RequirePermission(authz.PermAuditRead),

		// IAM procedures — all require users.manage.
		iamv1connect.IamServiceListUsersProcedure:  authz.RequirePermission(authz.PermUsersManage),
		iamv1connect.IamServiceGetUserProcedure:    authz.RequirePermission(authz.PermUsersManage),
		iamv1connect.IamServiceAssignRoleProcedure: authz.RequirePermission(authz.PermUsersManage),
		iamv1connect.IamServiceRevokeRoleProcedure: authz.RequirePermission(authz.PermUsersManage),
	}

	authzInterceptor := auth.NewAuthzInterceptor(exempt, policies)

	// Admission limiter for the section_enrollment enroll procedures.
	// DB_MAX_CONNS is set to "5" in the test environment → cap = floor(5*0.6) = 3.
	seLimiter := sectionenrollment.NewConcurrencyLimiter(5, testSEMetrics)
	seLimiterInterceptor := sectionenrollment.NewConcurrencyLimitInterceptor(seLimiter)

	queries := authdb.New(pool)
	repo := auth.NewPostgresRepository(queries)
	svc := auth.NewService(repo, sessionStore, roleLoader, testCfg)
	authHandler := auth.NewHandler(svc, testCfg)
	// RED interceptor OUTERMOST mirrors cmd/api/main.go exactly.
	authOpts := server.Chain(testREDInterceptor, authInterceptor, authzInterceptor)
	seOpts := server.Chain(testREDInterceptor, seLimiterInterceptor, authInterceptor, authzInterceptor)
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
	catalogRepo := catalog.NewPostgresRepository(catalogQueries, pool)
	catalogSvc := catalog.NewService(catalogRepo)
	catalogHandler := catalog.NewHandler(catalogSvc)
	catalogReg := func(mux *http.ServeMux) {
		catalog.Register(mux, catalogHandler, authOpts...)
	}

	// Enrollment handler wiring — mirrors cmd/api/main.go exactly.
	enrollmentQueries := enrollmentdb.New(pool)
	enrollmentRepo := enrollment.NewPostgresRepository(enrollmentQueries, pool)
	enrollmentSvc := enrollment.NewService(enrollmentRepo)
	enrollmentHandler := enrollment.NewHandler(enrollmentSvc)
	enrollmentReg := func(mux *http.ServeMux) {
		enrollment.Register(mux, enrollmentHandler, authOpts...)
	}

	// Section enrollment handler wiring — mirrors cmd/api/main.go exactly.
	seQueries := sectionenrollmentdb.New(pool)
	seRepo := sectionenrollment.NewPostgresRepository(seQueries, pool, testSEMetrics)
	seSvc := sectionenrollment.NewService(seRepo)
	seHandler := sectionenrollment.NewHandler(seSvc)
	sectionEnrollmentReg := func(mux *http.ServeMux) {
		sectionenrollment.Register(mux, seHandler, seOpts...)
	}

	// Grades handler wiring — mirrors cmd/api/main.go exactly.
	gradesQueries := gradesdb.New(pool)
	gradesRepo := grades.NewPostgresRepository(gradesQueries, pool, seRepo)
	gradesSvc := grades.NewService(gradesRepo)
	gradesHandler := grades.NewHandler(gradesSvc)
	gradesReg := func(mux *http.ServeMux) {
		grades.Register(mux, gradesHandler, authOpts...)
	}

	// Reports handler wiring — mirrors cmd/api/main.go exactly.
	reportsQueries := reportsdb.New(pool)
	reportsRepo := reports.NewPostgresRepository(reportsQueries)
	reportsCache := reports.NewRedisCache(redisClient)
	reportsSvc := reports.NewService(reportsRepo, reportsCache, sharedCfg.ReportsCacheTTL)
	reportsHandler := reports.NewHandler(reportsSvc)
	reportsReg := func(mux *http.ServeMux) {
		reports.Register(mux, reportsHandler, authOpts...)
	}

	// Audit logs handler wiring — mirrors cmd/api/main.go exactly. No cache.
	auditQueries := auditlogsdb.New(pool)
	auditRepo := auditlogs.NewPostgresRepository(auditQueries)
	auditSvc := auditlogs.NewService(auditRepo)
	auditHandler := auditlogs.NewHandler(auditSvc)
	auditLogsReg := func(mux *http.ServeMux) {
		auditlogs.Register(mux, auditHandler, authOpts...)
	}

	// IAM handler wiring — mirrors cmd/api/main.go exactly.
	iamQueries := iamdb.New(pool)
	iamRepo := iam.NewPostgresRepository(iamQueries)
	iamSvc := iam.NewService(iamRepo)
	iamHandler := iam.NewHandler(iamSvc)
	iamReg := func(mux *http.ServeMux) {
		iam.Register(mux, iamHandler, authOpts...)
	}

	// metricsHandlerReg registers /metrics with the test token — mirrors cmd/api/main.go.
	metricsHandlerReg := func(mux *http.ServeMux) {
		mux.Handle("/metrics", testMetricsReg.Handler(sharedCfg.MetricsAuthToken))
	}

	log := logging.New(slog.LevelError) // suppress output in tests
	srv := server.New(log, pool, rPinger, health.Register, authReg, profilesReg, catalogReg, enrollmentReg, sectionEnrollmentReg, gradesReg, reportsReg, auditLogsReg, iamReg, metricsHandlerReg)

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
