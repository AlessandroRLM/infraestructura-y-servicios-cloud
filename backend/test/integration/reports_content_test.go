package integration_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	reportsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1"
)

// TestReports_Content_SectionGradeReport_EmptySection verifies that an admin calling
// GetSectionGradeReport on an existing section with no enrollments returns an empty response
// (not an error).
func TestReports_Content_SectionGradeReport_EmptySection(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-empty@reports.test", "admin")

	_, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	// Clear cache to force DB hit.
	testRedisClient.Del(ctx, "report:section_grades:"+sectionID)

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: sectionID})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.GetSectionGradeReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error for empty section: %v", err)
	}
	if resp.Msg.SectionId != sectionID {
		t.Errorf("SectionId=%s, want %s", resp.Msg.SectionId, sectionID)
	}
	if len(resp.Msg.Rows) != 0 {
		t.Errorf("expected 0 rows for empty section, got %d", len(resp.Msg.Rows))
	}
	if resp.Msg.Truncated {
		t.Error("expected Truncated=false for empty section")
	}
}

// TestReports_Content_GetSectionGradeReport_WithEnrollment verifies that enrolled students
// appear in the grade report acta.
func TestReports_Content_GetSectionGradeReport_WithEnrollment(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-acta-admin@reports.test", "admin")

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	studentID, _ := seedUserWithSession(t, "reports-content-acta-student@reports.test", "student")
	seedStudentProfile(t, studentID, periodYear)

	enrollmentID, enrollCleanup := seedPaidEnrollment(t, studentID.String(), programID, periodYear)
	t.Cleanup(enrollCleanup)

	// Admin section-enroll the student.
	seClient := newSectionEnrollmentClient(nil)
	se, err := seEnrollAdmin(ctx, seClient, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("section enroll: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, se.GetId())
	})

	testRedisClient.Del(ctx, "report:section_grades:"+sectionID)

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: sectionID})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.GetSectionGradeReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The student should appear in the report.
	if len(resp.Msg.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(resp.Msg.Rows))
	}
	if resp.Msg.Rows[0].StudentId != studentID.String() {
		t.Errorf("got StudentId=%s, want %s", resp.Msg.Rows[0].StudentId, studentID.String())
	}
}

// TestReports_Content_NonExistentSection_CodeNotFound verifies that requesting a report
// for a section UUID that does not exist returns CodeNotFound.
func TestReports_Content_NonExistentSection_CodeNotFound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-notfound@reports.test", "admin")

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
		SectionId: "00000000-0000-0000-0000-000000000099",
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.GetSectionGradeReport(ctx, req)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestReports_Content_MalformedUUID_CodeInvalidArgument verifies that a malformed
// UUID in the request body returns CodeInvalidArgument (not CodeNotFound or Internal).
func TestReports_Content_MalformedUUID_CodeInvalidArgument(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-baduuid@reports.test", "admin")

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
		SectionId: "not-a-valid-uuid",
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.GetSectionGradeReport(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestReports_Content_OccupancyReport_ActiveSeatCount verifies that the occupancy report
// correctly counts only active (non-withdrawn) section enrollments.
func TestReports_Content_OccupancyReport_ActiveSeatCount(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-occupancy-admin@reports.test", "admin")

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	// Enroll two students.
	for i := 0; i < 2; i++ {
		studentID, _ := seedUserWithSession(t, fmt.Sprintf("reports-occupancy-s%d@reports.test", i), "student")
		seedStudentProfile(t, studentID, periodYear)
		enrollmentID, enrollCleanup := seedPaidEnrollment(t, studentID.String(), programID, periodYear)
		t.Cleanup(enrollCleanup)
		seClient := newSectionEnrollmentClient(nil)
		se, err := seEnrollAdmin(ctx, seClient, adminSID, enrollmentID, sectionID)
		if err != nil {
			t.Fatalf("student %d section enroll: %v", i, err)
		}
		idx := i // capture
		seID := se.GetId()
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, seID)
			_ = idx
		})
	}

	testRedisClient.Del(ctx, "report:section_occupancy:"+periodID)

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionOccupancyReportRequest{
		AcademicPeriodId: periodID,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.GetSectionOccupancyReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the target section row.
	var found *reportsv1.SectionOccupancyRow
	for _, r := range resp.Msg.Rows {
		if r.SectionId == sectionID {
			found = r
			break
		}
	}
	if found == nil {
		t.Fatalf("section %s not found in occupancy report rows", sectionID)
	}
	if found.ActiveSeatCount != 2 {
		t.Errorf("ActiveSeatCount = %d, want 2", found.ActiveSeatCount)
	}
	if found.Capacity != 30 {
		t.Errorf("Capacity = %d, want 30", found.Capacity)
	}
}

// TestReports_Content_ProgramSummaryReport_NotFound verifies that requesting a summary
// for a nonexistent program returns CodeNotFound.
func TestReports_Content_ProgramSummaryReport_NotFound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-prog-notfound@reports.test", "admin")

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetProgramSummaryReportRequest{
		ProgramId: "00000000-0000-0000-0000-000000000099",
		Year:      2025,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.GetProgramSummaryReport(ctx, req)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestReports_Content_StudentRecordReport_NotFound verifies that requesting a record
// for a nonexistent student returns CodeNotFound.
func TestReports_Content_StudentRecordReport_NotFound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-student-notfound@reports.test", "admin")

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetStudentRecordReportRequest{
		StudentId: "00000000-0000-0000-0000-000000000099",
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.GetStudentRecordReport(ctx, req)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestReports_Content_StudentRecordReport_WithHistory verifies that a student with
// an enrollment history appears in the ficha report with the correct course data.
func TestReports_Content_StudentRecordReport_WithHistory(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-ficha-admin@reports.test", "admin")

	programID, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	studentID, _ := seedUserWithSession(t, "reports-content-ficha-student@reports.test", "student")
	seedStudentProfile(t, studentID, periodYear)

	enrollmentID, enrollCleanup := seedPaidEnrollment(t, studentID.String(), programID, periodYear)
	t.Cleanup(enrollCleanup)

	seClient := newSectionEnrollmentClient(nil)
	se, err := seEnrollAdmin(ctx, seClient, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("section enroll: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, se.GetId())
	})

	testRedisClient.Del(ctx, "report:student_record:"+studentID.String())

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetStudentRecordReportRequest{
		StudentId: studentID.String(),
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.GetStudentRecordReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.StudentId != studentID.String() {
		t.Errorf("StudentId=%s, want %s", resp.Msg.StudentId, studentID.String())
	}
	if len(resp.Msg.Rows) == 0 {
		t.Fatal("expected at least 1 row in student record, got 0")
	}

	// Verify the section is referenced.
	found := false
	for _, r := range resp.Msg.Rows {
		if r.SectionId == sectionID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("sectionID %s not found in student record rows", sectionID)
	}
}
