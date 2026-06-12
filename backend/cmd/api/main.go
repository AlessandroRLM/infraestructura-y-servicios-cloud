// Package main is the composition root for the API server.
// It wires configuration, logging, database, cache, migrations, and HTTP server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	migrations "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/migrations"

	auditlogsv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/audit_logs/v1/auditlogsv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/auth/v1/authv1connect"
	catalogv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1/catalogv1connect"
	enrollmentv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1/enrollmentv1connect"
	gradesv1connect "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1/gradesv1connect"
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

	// Metrics registry and domain counters.
	metricsReg := platformmetrics.New()
	seMetrics := metricsReg.SectionEnrollmentMetrics()
	redInterceptor := metricsReg.RPCInterceptor()

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
		authv1connect.AuthServiceGetSessionProcedure:           {},
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
	}

	authzInterceptor := auth.NewAuthzInterceptor(exempt, policies)

	// concurrencyLimiter caps inscription transactions at floor(DBMaxConns*0.6) to
	// protect the connection pool from stampede exhaustion. With default DBMaxConns=10
	// the cap is 6; saturated requests return CodeResourceExhausted immediately.
	seLimiter := sectionenrollment.NewConcurrencyLimiter(cfg.DBMaxConns, seMetrics)
	seLimiterInterceptor := sectionenrollment.NewConcurrencyLimitInterceptor(seLimiter)

	// Auth handler (repository → service → Connect handler).
	queries := authdb.New(pool)
	repo := auth.NewPostgresRepository(queries)
	svc := auth.NewService(repo, sessionStore, roleLoader, cfg)
	authHandler := auth.NewHandler(svc, cfg)

	// Interceptor options shared across all service endpoints.
	// The RED interceptor is OUTERMOST so that all requests — including rejected ones
	// (unauthenticated, authz denied, limiter saturated) — are counted with their codes.
	authOpts := server.Chain(redInterceptor, authInterceptor, authzInterceptor)
	// seOpts prepends the admission limiter between RED and auth for the section_enrollment
	// service. The limiter is procedure-aware: it only gates EnrollOwnSection and EnrollSection;
	// list/get/withdraw procedures are passed through without acquiring a slot.
	seOpts := server.Chain(redInterceptor, seLimiterInterceptor, authInterceptor, authzInterceptor)

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

	// Enrollment handler (enrollmentdb.Querier → repository → service → Connect handler).
	enrollmentQueries := enrollmentdb.New(pool)
	enrollmentRepo := enrollment.NewPostgresRepository(enrollmentQueries, pool)
	enrollmentSvc := enrollment.NewService(enrollmentRepo)
	enrollmentHandler := enrollment.NewHandler(enrollmentSvc)

	enrollmentReg := func(mux *http.ServeMux) {
		enrollment.Register(mux, enrollmentHandler, authOpts...)
	}

	// Section enrollment handler (sectionenrollmentdb.Querier → repository → service → Connect handler).
	// enrollOpts prepends the admission limiter for the two enroll procedures; the limiter
	// inspects the procedure name and is a no-op for non-enroll procedures registered on the
	// same handler. This keeps a single handler registration while enforcing per-procedure limits.
	seQueries := sectionenrollmentdb.New(pool)
	seRepo := sectionenrollment.NewPostgresRepository(seQueries, pool, seMetrics)
	seSvc := sectionenrollment.NewService(seRepo)
	seHandler := sectionenrollment.NewHandler(seSvc)

	sectionEnrollmentReg := func(mux *http.ServeMux) {
		sectionenrollment.Register(mux, seHandler, seOpts...)
	}

	// Grades handler (gradesdb.Querier → repository → service → Connect handler).
	// seRepo provides SetSectionEnrollmentOutcomeTx for the atomic outcome mediation.
	gradesQueries := gradesdb.New(pool)
	gradesRepo := grades.NewPostgresRepository(gradesQueries, pool, seRepo)
	gradesSvc := grades.NewService(gradesRepo)
	gradesHandler := grades.NewHandler(gradesSvc)

	gradesReg := func(mux *http.ServeMux) {
		grades.Register(mux, gradesHandler, authOpts...)
	}

	// Reports handler (reportsdb.Querier → repository → service → Connect handler).
	// Reuses the existing redisClient (no new Redis connection).
	reportsQueries := reportsdb.New(pool)
	reportsRepo := reports.NewPostgresRepository(reportsQueries)
	reportsCache := reports.NewRedisCache(redisClient)
	reportsSvc := reports.NewService(reportsRepo, reportsCache, cfg.ReportsCacheTTL)
	reportsHandler := reports.NewHandler(reportsSvc)

	reportsReg := func(mux *http.ServeMux) {
		reports.Register(mux, reportsHandler, authOpts...)
	}

	// Audit logs handler (auditlogsdb.Querier → repository → service → Connect handler).
	// No Redis cache — freshness over speed for admin-only low-volume reads.
	auditQueries := auditlogsdb.New(pool)
	auditRepo := auditlogs.NewPostgresRepository(auditQueries)
	auditSvc := auditlogs.NewService(auditRepo)
	auditHandler := auditlogs.NewHandler(auditSvc)

	auditLogsReg := func(mux *http.ServeMux) {
		auditlogs.Register(mux, auditHandler, authOpts...)
	}

	// Redis pinger for the readyz handler.
	redisPinger, err := platformredis.NewPinger(cfg.RedisURL)
	if err != nil {
		log.Error("failed to create redis pinger", "err", err)
		os.Exit(1)
	}

	// metricsHandlerReg registers the /metrics endpoint on the mux. The handler
	// enforces X-Metrics-Token authentication and is NOT a Connect procedure.
	metricsHandlerReg := func(mux *http.ServeMux) {
		mux.Handle("/metrics", metricsReg.Handler(cfg.MetricsAuthToken))
	}

	srv := server.New(log, pool, redisPinger,
		health.Register,
		authReg,
		profilesReg,
		catalogReg,
		enrollmentReg,
		sectionEnrollmentReg,
		gradesReg,
		reportsReg,
		auditLogsReg,
		metricsHandlerReg,
	)
	srv.Addr = cfg.HTTPAddr

	log.Info("starting server", "addr", cfg.HTTPAddr)
	server.RunWithGracefulShutdown(ctx, srv)
}
