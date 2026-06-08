package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1/catalogv1connect"
)

// newCatalogClient returns a Connect CatalogService client targeting the shared test server.
func newCatalogClient(jar http.CookieJar) catalogv1connect.CatalogServiceClient {
	return catalogv1connect.NewCatalogServiceClient(&http.Client{Jar: jar}, baseURL)
}

// TestCatalog_NonAdmin_PermissionDenied verifies that a non-admin (student role)
// receives CodePermissionDenied on any catalog procedure.
func TestCatalog_NonAdmin_PermissionDenied(t *testing.T) {
	_, studentSID := seedUserWithSession(t, "catalog-student-authz@catalog.test", "student")
	client := newCatalogClient(nil)
	ctx := context.Background()

	req := connect.NewRequest(&catalogv1.ListProgramsRequest{})
	req.Header().Set("Cookie", "sid="+studentSID)

	_, err := client.ListPrograms(ctx, req)
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestCatalog_Unauthenticated_CodeUnauthenticated verifies that a caller without
// a session cookie receives CodeUnauthenticated.
func TestCatalog_Unauthenticated_CodeUnauthenticated(t *testing.T) {
	client := newCatalogClient(nil)
	ctx := context.Background()

	req := connect.NewRequest(&catalogv1.ListProgramsRequest{})
	// No Cookie header — unauthenticated.

	_, err := client.ListPrograms(ctx, req)
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestCatalog_Admin_CreateAndGetProgram verifies that an admin can create and retrieve a program.
func TestCatalog_Admin_CreateAndGetProgram(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "catalog-admin-prog@catalog.test", "admin")
	client := newCatalogClient(nil)

	createReq := connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: "PROG-AUTHZ-" + uniqueSuffix(t),
		Name: "Test Program Authz",
	})
	createReq.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.CreateProgram(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateProgram as admin: %v", err)
	}

	id := resp.Msg.GetId()
	t.Cleanup(func() {
		cleanupProgram(t, id)
	})

	getReq := connect.NewRequest(&catalogv1.GetProgramRequest{Id: id})
	getReq.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetProgram(ctx, getReq)
	if err != nil {
		t.Errorf("GetProgram as admin: %v", err)
	}
}

// TestCatalog_Admin_CreateAndGetCourse verifies admin CRUD for courses.
func TestCatalog_Admin_CreateAndGetCourse(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "catalog-admin-course@catalog.test", "admin")
	client := newCatalogClient(nil)

	createReq := connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code:    "CRS-AUTHZ-" + uniqueSuffix(t),
		Name:    "Test Course Authz",
		Credits: 3,
	})
	createReq.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.CreateCourse(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateCourse as admin: %v", err)
	}
	id := resp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, id) })

	getReq := connect.NewRequest(&catalogv1.GetCourseRequest{Id: id})
	getReq.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetCourse(ctx, getReq)
	if err != nil {
		t.Errorf("GetCourse as admin: %v", err)
	}
}

// TestCatalog_Admin_AcademicPeriod verifies admin CRUD for academic periods.
func TestCatalog_Admin_AcademicPeriod(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "catalog-admin-ap@catalog.test", "admin")
	client := newCatalogClient(nil)

	createReq := connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year:      3100,
		Term:      1,
		StartDate: "3100-03-01",
		EndDate:   "3100-07-31",
	})
	createReq.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.CreateAcademicPeriod(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateAcademicPeriod as admin: %v", err)
	}
	id := resp.Msg.GetId()
	t.Cleanup(func() { cleanupAcademicPeriod(t, id) })

	getReq := connect.NewRequest(&catalogv1.GetAcademicPeriodRequest{Id: id})
	getReq.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetAcademicPeriod(ctx, getReq)
	if err != nil {
		t.Errorf("GetAcademicPeriod as admin: %v", err)
	}
}

// uniqueSuffix generates a unique suffix for test isolation using a UUID.
func uniqueSuffix(_ *testing.T) string {
	return uuid.New().String()[:8]
}

// seedUserWithSession seeds a user with the given role, creates a Redis session,
// and returns (userID, sessionID). It reuses the rbac_interceptor_test.go helper seedUserWithRole.
func catalogSeedAdminSession(t *testing.T, email string) string {
	t.Helper()
	userID := seedUserWithRole(t, email, "admin")
	return seedSessionInRedis(t, userID, time.Hour)
}

// cleanupProgram removes a program row by id from the database.
func cleanupProgram(t *testing.T, id string) {
	t.Helper()
	_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, id)
}

// cleanupCourse removes a course row by id from the database.
func cleanupCourse(t *testing.T, id string) {
	t.Helper()
	_, _ = pgxPool.Exec(context.Background(), `DELETE FROM courses WHERE id = $1`, id)
}

// cleanupAcademicPeriod removes an academic_periods row by id.
func cleanupAcademicPeriod(t *testing.T, id string) {
	t.Helper()
	_, _ = pgxPool.Exec(context.Background(), `DELETE FROM academic_periods WHERE id = $1`, id)
}
