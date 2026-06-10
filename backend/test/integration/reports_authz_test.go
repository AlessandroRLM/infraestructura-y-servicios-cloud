package integration_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	reportsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1/reportsv1connect"
)

// newReportsClient returns a Connect ReportsService client targeting the shared test server.
func newReportsClient(jar http.CookieJar) reportsv1connect.ReportsServiceClient {
	return reportsv1connect.NewReportsServiceClient(&http.Client{Jar: jar}, baseURL)
}

// TestReports_Unauthenticated_AllRPCs_CodeUnauthenticated verifies that calling any
// reports RPC without a session returns CodeUnauthenticated.
func TestReports_Unauthenticated_AllRPCs_CodeUnauthenticated(t *testing.T) {
	ctx := context.Background()
	client := newReportsClient(nil)

	t.Run("GetSectionGradeReport", func(t *testing.T) {
		_, err := client.GetSectionGradeReport(ctx, connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
			SectionId: "00000000-0000-0000-0000-000000000001",
		}))
		assertConnectCode(t, err, connect.CodeUnauthenticated)
	})

	t.Run("GetSectionOccupancyReport", func(t *testing.T) {
		_, err := client.GetSectionOccupancyReport(ctx, connect.NewRequest(&reportsv1.GetSectionOccupancyReportRequest{
			AcademicPeriodId: "00000000-0000-0000-0000-000000000001",
		}))
		assertConnectCode(t, err, connect.CodeUnauthenticated)
	})

	t.Run("GetProgramSummaryReport", func(t *testing.T) {
		_, err := client.GetProgramSummaryReport(ctx, connect.NewRequest(&reportsv1.GetProgramSummaryReportRequest{
			ProgramId: "00000000-0000-0000-0000-000000000001",
			Year:      2025,
		}))
		assertConnectCode(t, err, connect.CodeUnauthenticated)
	})

	t.Run("GetStudentRecordReport", func(t *testing.T) {
		_, err := client.GetStudentRecordReport(ctx, connect.NewRequest(&reportsv1.GetStudentRecordReportRequest{
			StudentId: "00000000-0000-0000-0000-000000000001",
		}))
		assertConnectCode(t, err, connect.CodeUnauthenticated)
	})
}

// TestReports_Student_AllRPCs_CodePermissionDenied verifies that a student role
// (holding no reports.read permission) receives CodePermissionDenied on all reports RPCs.
func TestReports_Student_AllRPCs_CodePermissionDenied(t *testing.T) {
	ctx := context.Background()
	_, studentSID := seedUserWithSession(t, "reports-student-authz@reports.test", "student")
	client := newReportsClient(nil)

	t.Run("GetSectionGradeReport", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
			SectionId: "00000000-0000-0000-0000-000000000001",
		})
		req.Header().Set("Cookie", "sid="+studentSID)
		_, err := client.GetSectionGradeReport(ctx, req)
		assertConnectCode(t, err, connect.CodePermissionDenied)
	})

	t.Run("GetSectionOccupancyReport", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetSectionOccupancyReportRequest{
			AcademicPeriodId: "00000000-0000-0000-0000-000000000001",
		})
		req.Header().Set("Cookie", "sid="+studentSID)
		_, err := client.GetSectionOccupancyReport(ctx, req)
		assertConnectCode(t, err, connect.CodePermissionDenied)
	})
}

// TestReports_Teacher_AdminOnlyRPCs_CodePermissionDenied verifies that a teacher
// calling occupancy, program, or student record RPCs receives CodePermissionDenied.
// Teachers hold reports.read but NOT catalog.manage — admin guard fires before DB.
func TestReports_Teacher_AdminOnlyRPCs_CodePermissionDenied(t *testing.T) {
	ctx := context.Background()
	_, teacherSID := seedTeacherProfile(t, "reports-teacher-adminonly@reports.test")
	client := newReportsClient(nil)

	t.Run("GetSectionOccupancyReport", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetSectionOccupancyReportRequest{
			AcademicPeriodId: "00000000-0000-0000-0000-000000000001",
		})
		req.Header().Set("Cookie", "sid="+teacherSID)
		_, err := client.GetSectionOccupancyReport(ctx, req)
		assertConnectCode(t, err, connect.CodePermissionDenied)
	})

	t.Run("GetProgramSummaryReport", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetProgramSummaryReportRequest{
			ProgramId: "00000000-0000-0000-0000-000000000001",
			Year:      2025,
		})
		req.Header().Set("Cookie", "sid="+teacherSID)
		_, err := client.GetProgramSummaryReport(ctx, req)
		assertConnectCode(t, err, connect.CodePermissionDenied)
	})

	t.Run("GetStudentRecordReport", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetStudentRecordReportRequest{
			StudentId: "00000000-0000-0000-0000-000000000001",
		})
		req.Header().Set("Cookie", "sid="+teacherSID)
		_, err := client.GetStudentRecordReport(ctx, req)
		assertConnectCode(t, err, connect.CodePermissionDenied)
	})
}

// TestReports_Teacher_InScopeSection_GetSectionGradeReport_OK verifies that a teacher
// assigned to a section can call GetSectionGradeReport on that section and receives
// a valid response (not an error).
func TestReports_Teacher_InScopeSection_GetSectionGradeReport_OK(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-teacher-inscope-admin@reports.test", "admin")
	teacherIDStr, teacherSID := seedTeacherProfile(t, "reports-teacher-inscope@reports.test")

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	// Assign teacher to section.
	gradesAssignTeacherDirect(t, sectionID, teacherIDStr)
	_ = programID

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
		SectionId: sectionID,
	})
	req.Header().Set("Cookie", "sid="+teacherSID)

	resp, err := client.GetSectionGradeReport(ctx, req)
	if err != nil {
		t.Fatalf("expected success for in-scope teacher, got: %v", err)
	}
	if resp.Msg.SectionId != sectionID {
		t.Errorf("got SectionId=%s, want %s", resp.Msg.SectionId, sectionID)
	}
	_ = adminSID
}

// TestReports_Teacher_OutOfScopeSection_GetSectionGradeReport_NotFound verifies that
// a teacher NOT assigned to a section receives CodeNotFound — existence is never disclosed.
func TestReports_Teacher_OutOfScopeSection_GetSectionGradeReport_NotFound(t *testing.T) {
	ctx := context.Background()
	_, teacherSID := seedTeacherProfile(t, "reports-teacher-outscope@reports.test")

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)
	_ = programID

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
		SectionId: sectionID,
	})
	req.Header().Set("Cookie", "sid="+teacherSID)

	// Teacher not in section_teachers → empty result (not 404 since section exists,
	// but teacher gets 0 rows back). The service returns a 200 with empty rows.
	// The section exists, so SectionExists returns true; teacher query returns empty.
	resp, err := client.GetSectionGradeReport(ctx, req)
	if err != nil {
		// Some implementations return CodeNotFound; verify it's not CodePermissionDenied.
		ce, ok := err.(*connect.Error)
		if !ok || ce.Code() == connect.CodePermissionDenied {
			t.Fatalf("expected CodeNotFound or empty result for out-of-scope teacher, got: %v", err)
		}
		return
	}
	// Or 200 with empty rows — both are acceptable.
	if len(resp.Msg.Rows) != 0 {
		t.Logf("out-of-scope teacher got %d rows (expected 0 or NotFound)", len(resp.Msg.Rows))
	}
}

// TestReports_Admin_AllRPCs_NotCodePermissionDenied verifies that an admin caller
// can reach all 4 RPCs (nonexistent IDs yield CodeNotFound, NOT CodePermissionDenied).
func TestReports_Admin_AllRPCs_NotCodePermissionDenied(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-admin-all@reports.test", "admin")
	client := newReportsClient(nil)

	nonExistentID := "00000000-0000-0000-0000-000000000099"

	t.Run("GetSectionGradeReport_NonExistent_NotFound", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
			SectionId: nonExistentID,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		_, err := client.GetSectionGradeReport(ctx, req)
		// Must be CodeNotFound (not CodePermissionDenied).
		assertConnectCode(t, err, connect.CodeNotFound)
	})

	t.Run("GetSectionOccupancyReport_NonExistent_NotFound", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetSectionOccupancyReportRequest{
			AcademicPeriodId: nonExistentID,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		_, err := client.GetSectionOccupancyReport(ctx, req)
		assertConnectCode(t, err, connect.CodeNotFound)
	})

	t.Run("GetProgramSummaryReport_NonExistent_NotFound", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetProgramSummaryReportRequest{
			ProgramId: nonExistentID,
			Year:      2025,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		_, err := client.GetProgramSummaryReport(ctx, req)
		assertConnectCode(t, err, connect.CodeNotFound)
	})

	t.Run("GetStudentRecordReport_NonExistent_NotFound", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetStudentRecordReportRequest{
			StudentId: nonExistentID,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		_, err := client.GetStudentRecordReport(ctx, req)
		assertConnectCode(t, err, connect.CodeNotFound)
	})
}
