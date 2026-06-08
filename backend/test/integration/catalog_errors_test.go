package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
)

// TestCatalog_DuplicateProgramCode_AlreadyExists verifies that creating a program with
// a duplicate code returns CodeAlreadyExists and leaves exactly one row in programs.
func TestCatalog_DuplicateProgramCode_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-dup-prog@catalog.test")
	client := newCatalogClient(nil)

	code := "DUP-PROG-" + uuid.New().String()[:8]

	first := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: code, Name: "First"})
	first.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.CreateProgram(ctx, first)
	if err != nil {
		t.Fatalf("CreateProgram (first): %v", err)
	}
	id := resp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, id) })

	second := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: code, Name: "Duplicate"})
	second.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.CreateProgram(ctx, second)
	assertConnectCode(t, err, connect.CodeAlreadyExists)

	// Verify only 1 row exists for this code.
	var count int
	if err := pgxPool.QueryRow(ctx, `SELECT count(*) FROM programs WHERE code = $1`, code).Scan(&count); err != nil {
		t.Fatalf("count programs: %v", err)
	}
	if count != 1 {
		t.Errorf("programs count for code %q = %d, want 1", code, count)
	}
}

// TestCatalog_DuplicateCourseCode_AlreadyExists verifies duplicate course code.
func TestCatalog_DuplicateCourseCode_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-dup-course@catalog.test")
	client := newCatalogClient(nil)

	code := "DUP-CRS-" + uuid.New().String()[:8]

	first := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: code, Name: "First", Credits: 3})
	first.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.CreateCourse(ctx, first)
	if err != nil {
		t.Fatalf("CreateCourse (first): %v", err)
	}
	id := resp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, id) })

	second := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: code, Name: "Duplicate", Credits: 4})
	second.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.CreateCourse(ctx, second)
	assertConnectCode(t, err, connect.CodeAlreadyExists)
}

// TestCatalog_DuplicateProgramCourse_AlreadyExists verifies AddCourseToProgram duplicate.
func TestCatalog_DuplicateProgramCourse_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-dup-assoc@catalog.test")
	client := newCatalogClient(nil)

	// Create program and course.
	pResp, err := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: "DUP-ASSOC-P-" + uuid.New().String()[:8], Name: "P",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	progID := pResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID) })

	cResp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code: "DUP-ASSOC-C-" + uuid.New().String()[:8], Name: "C", Credits: 3,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	courseID := cResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID) })

	// First add — OK
	if _, err := client.AddCourseToProgram(ctx, withSID(connect.NewRequest(&catalogv1.AddCourseToProgramRequest{
		ProgramId: progID, CourseId: courseID,
	}), adminSID)); err != nil {
		t.Fatalf("AddCourseToProgram (first): %v", err)
	}

	// Second add — AlreadyExists
	_, err = client.AddCourseToProgram(ctx, withSID(connect.NewRequest(&catalogv1.AddCourseToProgramRequest{
		ProgramId: progID, CourseId: courseID,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeAlreadyExists)

	// Count must be 1.
	var count int
	if err := pgxPool.QueryRow(ctx,
		`SELECT count(*) FROM program_courses WHERE program_id = $1 AND course_id = $2`,
		progID, courseID,
	).Scan(&count); err != nil {
		t.Fatalf("count program_courses: %v", err)
	}
	if count != 1 {
		t.Errorf("program_courses count = %d, want 1", count)
	}
}

// TestCatalog_BadFK_ProgramQuota_InvalidArgument verifies that creating a quota with
// a non-existent program_id returns CodeInvalidArgument.
func TestCatalog_BadFK_ProgramQuota_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-fk-quota@catalog.test")
	client := newCatalogClient(nil)

	req := withSID(connect.NewRequest(&catalogv1.CreateProgramQuotaRequest{
		ProgramId:      uuid.New().String(), // non-existent
		Year:           2025,
		AdmissionQuota: 40,
	}), adminSID)

	_, err := client.CreateProgramQuota(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// withSID adds a session cookie to a connect.Request.
func withSID[T any](req *connect.Request[T], sid string) *connect.Request[T] {
	req.Header().Set("Cookie", "sid="+sid)
	return req
}
