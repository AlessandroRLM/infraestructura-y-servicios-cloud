package integration_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

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
	// user_profiles is required by the acta query (INNER JOIN user_profiles).
	seedUserProfile(t, studentID, "Ana")

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
		// user_profiles not required by occupancy query but seed for consistency.
		seedUserProfile(t, studentID, fmt.Sprintf("Stu%d", i))
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

// TestReports_Content_ProgramSummaryReport_YearOutOfRange verifies that years outside
// [2000, 2100] return CodeInvalidArgument before any cache or DB access.
func TestReports_Content_ProgramSummaryReport_YearOutOfRange(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-content-year-range@reports.test", "admin")
	client := newReportsClient(nil)

	nonExistentProgramID := "00000000-0000-0000-0000-000000000099"

	t.Run("year_1999", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetProgramSummaryReportRequest{
			ProgramId: nonExistentProgramID,
			Year:      1999,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		_, err := client.GetProgramSummaryReport(ctx, req)
		assertConnectCode(t, err, connect.CodeInvalidArgument)
	})

	t.Run("year_2101", func(t *testing.T) {
		req := connect.NewRequest(&reportsv1.GetProgramSummaryReportRequest{
			ProgramId: nonExistentProgramID,
			Year:      2101,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		_, err := client.GetProgramSummaryReport(ctx, req)
		assertConnectCode(t, err, connect.CodeInvalidArgument)
	})
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
	// user_profiles not strictly required by FichaForStudent but good practice.
	seedUserProfile(t, studentID, "Maria")

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
	// student_name must be populated from user_profiles join.
	if resp.Msg.StudentName == "" {
		t.Error("expected StudentName to be populated, got empty string")
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

// TestReports_Content_SectionGradeReport_TruncationAt500 verifies that when a section has
// 501 enrolled students, GetSectionGradeReport returns exactly 500 rows and truncated==true.
// It also verifies that a section with fewer students reports truncated==false.
func TestReports_Content_SectionGradeReport_TruncationAt500(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-truncation-admin@reports.test", "admin")

	// Seed a program+course (no quota needed for this test).
	programID, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)

	// High-capacity section to hold 501 enrollments.
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 600)
	t.Cleanup(sectionCleanup)

	// Run suffix for unique emails and national IDs.
	runSuffix := uuid.New().String()[:8]

	// Bulk-insert 501 users, user_profiles, student_profiles, enrollments, and
	// section_enrollments using generate_series to avoid N+1 round trips.
	// Each INSERT uses the series index and run suffix to guarantee uniqueness.
	_, err := pgxPool.Exec(ctx, `
		INSERT INTO users (email, password_hash)
		SELECT
			'trunc-' || $1 || '-' || gs || '@reports.test',
			'$2a$10$placeholder'
		FROM generate_series(1, 501) AS gs
	`, runSuffix)
	if err != nil {
		t.Fatalf("bulk insert users: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM users WHERE email LIKE 'trunc-'||$1||'-%@reports.test'`, runSuffix)
	})

	_, err = pgxPool.Exec(ctx, `
		INSERT INTO user_profiles (user_id, given_names, last_name_paternal, national_id_type, national_id)
		SELECT
			u.id,
			'Student' || gs,
			'Trunc',
			'RUT',
			'TRUNC-' || $1 || '-' || gs
		FROM generate_series(1, 501) AS gs
		JOIN users u ON u.email = 'trunc-' || $1 || '-' || gs || '@reports.test'
	`, runSuffix)
	if err != nil {
		t.Fatalf("bulk insert user_profiles: %v", err)
	}

	_, err = pgxPool.Exec(ctx, `
		INSERT INTO student_profiles (user_id, admission_year)
		SELECT u.id, $2
		FROM generate_series(1, 501) AS gs
		JOIN users u ON u.email = 'trunc-' || $1 || '-' || gs || '@reports.test'
	`, runSuffix, periodYear)
	if err != nil {
		t.Fatalf("bulk insert student_profiles: %v", err)
	}

	_, err = pgxPool.Exec(ctx, `
		INSERT INTO enrollments (student_id, program_id, year, status)
		SELECT u.id, $2::uuid, $3, 'paid'
		FROM generate_series(1, 501) AS gs
		JOIN users u ON u.email = 'trunc-' || $1 || '-' || gs || '@reports.test'
	`, runSuffix, programID, periodYear)
	if err != nil {
		t.Fatalf("bulk insert enrollments: %v", err)
	}

	_, err = pgxPool.Exec(ctx, `
		INSERT INTO section_enrollments (enrollment_id, section_id)
		SELECT en.id, $2::uuid
		FROM generate_series(1, 501) AS gs
		JOIN users u ON u.email = 'trunc-' || $1 || '-' || gs || '@reports.test'
		JOIN enrollments en ON en.student_id = u.id AND en.program_id = $3::uuid AND en.year = $4
	`, runSuffix, sectionID, programID, periodYear)
	if err != nil {
		t.Fatalf("bulk insert section_enrollments: %v", err)
	}
	// Cleanup in FK-safe order (section_enrollments → enrollments; users cascade profiles).
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pgxPool.Exec(c, `
			DELETE FROM section_enrollments
			WHERE section_id = $1::uuid
			  AND enrollment_id IN (
			      SELECT en.id FROM enrollments en
			      JOIN users u ON u.id = en.student_id
			      WHERE u.email LIKE 'trunc-'||$2||'-%@reports.test'
			  )
		`, sectionID, runSuffix)
		_, _ = pgxPool.Exec(c, `
			DELETE FROM enrollments
			WHERE student_id IN (
			    SELECT id FROM users WHERE email LIKE 'trunc-'||$1||'-%@reports.test'
			)
		`, runSuffix)
	})

	testRedisClient.Del(ctx, "report:section_grades:"+sectionID)

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: sectionID})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.GetSectionGradeReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Rows) != 500 {
		t.Errorf("expected exactly 500 rows (cap), got %d", len(resp.Msg.Rows))
	}
	if !resp.Msg.Truncated {
		t.Error("expected Truncated=true for 501 students, got false")
	}

	// Non-truncated: the empty section from earlier tests serves as baseline — use a fresh
	// empty section to assert truncated==false.
	emptySectionID, emptyCleanup := seedSection(t, courseID, periodID, 10)
	t.Cleanup(emptyCleanup)
	testRedisClient.Del(ctx, "report:section_grades:"+emptySectionID)
	req2 := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: emptySectionID})
	req2.Header().Set("Cookie", "sid="+adminSID)
	resp2, err := client.GetSectionGradeReport(ctx, req2)
	if err != nil {
		t.Fatalf("non-truncated section: unexpected error: %v", err)
	}
	if resp2.Msg.Truncated {
		t.Error("expected Truncated=false for empty section, got true")
	}
}

// TestReports_Content_SoftDelete_Awareness verifies that soft-deleted section_enrollments,
// sections, and students are excluded from report results.
func TestReports_Content_SoftDelete_Awareness(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-softdelete-admin@reports.test", "admin")

	t.Run("acta_excludes_soft_deleted_SE", func(t *testing.T) {
		programID, courseID, programCleanup := seedProgramWithCourse(t)
		t.Cleanup(programCleanup)
		periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
		t.Cleanup(periodCleanup)
		sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
		t.Cleanup(sectionCleanup)

		// Seed 2 students.
		studentA, _ := seedUserWithSession(t, fmt.Sprintf("reports-sd-acta-a-%s@reports.test", uuid.New().String()[:8]), "student")
		studentB, _ := seedUserWithSession(t, fmt.Sprintf("reports-sd-acta-b-%s@reports.test", uuid.New().String()[:8]), "student")
		seedStudentProfile(t, studentA, periodYear)
		seedStudentProfile(t, studentB, periodYear)
		seedUserProfile(t, studentA, "AliveStudent")
		seedUserProfile(t, studentB, "DeletedStudent")

		enrollA, cleanA := seedPaidEnrollment(t, studentA.String(), programID, periodYear)
		t.Cleanup(cleanA)
		enrollB, cleanB := seedPaidEnrollment(t, studentB.String(), programID, periodYear)
		t.Cleanup(cleanB)

		seClient := newSectionEnrollmentClient(nil)
		seA, err := seEnrollAdmin(ctx, seClient, adminSID, enrollA, sectionID)
		if err != nil {
			t.Fatalf("enroll studentA: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, seA.GetId())
		})
		seB, err := seEnrollAdmin(ctx, seClient, adminSID, enrollB, sectionID)
		if err != nil {
			t.Fatalf("enroll studentB: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, seB.GetId())
		})

		// Soft-delete studentB's section_enrollment.
		if _, err := pgxPool.Exec(ctx,
			`UPDATE section_enrollments SET deleted_at = now() WHERE id = $1`, seB.GetId(),
		); err != nil {
			t.Fatalf("soft-delete SE: %v", err)
		}

		testRedisClient.Del(ctx, "report:section_grades:"+sectionID)

		client := newReportsClient(nil)
		req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: sectionID})
		req.Header().Set("Cookie", "sid="+adminSID)

		resp, err := client.GetSectionGradeReport(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Only studentA's row must appear (soft-deleted SE excluded).
		if len(resp.Msg.Rows) != 1 {
			t.Errorf("expected 1 row after SE soft-delete, got %d", len(resp.Msg.Rows))
		}
		if len(resp.Msg.Rows) == 1 && resp.Msg.Rows[0].StudentId != studentA.String() {
			t.Errorf("expected studentA %s in acta, got %s", studentA.String(), resp.Msg.Rows[0].StudentId)
		}
	})

	t.Run("occupancy_excludes_soft_deleted_section", func(t *testing.T) {
		_, courseID, programCleanup := seedProgramWithCourse(t)
		t.Cleanup(programCleanup)
		periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
		t.Cleanup(periodCleanup)

		liveID, liveCleanup := seedSection(t, courseID, periodID, 30)
		t.Cleanup(liveCleanup)

		var deletedSectionID uuid.UUID
		if err := pgxPool.QueryRow(ctx,
			`INSERT INTO sections (course_id, academic_period_id, capacity) VALUES ($1, $2, 30) RETURNING id`,
			courseID, periodID,
		).Scan(&deletedSectionID); err != nil {
			t.Fatalf("insert section to delete: %v", err)
		}
		// Soft-delete immediately.
		if _, err := pgxPool.Exec(ctx,
			`UPDATE sections SET deleted_at = now() WHERE id = $1`, deletedSectionID,
		); err != nil {
			t.Fatalf("soft-delete section: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM sections WHERE id = $1`, deletedSectionID)
		})

		testRedisClient.Del(ctx, "report:section_occupancy:"+periodID)

		client := newReportsClient(nil)
		req := connect.NewRequest(&reportsv1.GetSectionOccupancyReportRequest{AcademicPeriodId: periodID})
		req.Header().Set("Cookie", "sid="+adminSID)

		resp, err := client.GetSectionOccupancyReport(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The soft-deleted section must NOT appear; the live section must.
		foundDeleted := false
		foundLive := false
		for _, r := range resp.Msg.Rows {
			if r.SectionId == deletedSectionID.String() {
				foundDeleted = true
			}
			if r.SectionId == liveID {
				foundLive = true
			}
		}
		if foundDeleted {
			t.Errorf("soft-deleted section %s must not appear in occupancy report", deletedSectionID)
		}
		if !foundLive {
			t.Errorf("live section %s must appear in occupancy report", liveID)
		}
	})

	t.Run("ficha_excludes_soft_deleted_SE", func(t *testing.T) {
		programID, courseID, programCleanup := seedProgramWithCourse(t)
		t.Cleanup(programCleanup)
		periodID, periodYear, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
		t.Cleanup(periodCleanup)
		sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
		t.Cleanup(sectionCleanup)

		studentID, _ := seedUserWithSession(t, fmt.Sprintf("reports-sd-ficha-%s@reports.test", uuid.New().String()[:8]), "student")
		seedStudentProfile(t, studentID, periodYear)
		seedUserProfile(t, studentID, "FichaStudent")

		enrollmentID, enrollCleanup := seedPaidEnrollment(t, studentID.String(), programID, periodYear)
		t.Cleanup(enrollCleanup)

		seClient := newSectionEnrollmentClient(nil)
		se, err := seEnrollAdmin(ctx, seClient, adminSID, enrollmentID, sectionID)
		if err != nil {
			t.Fatalf("enroll student: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, se.GetId())
		})

		// Soft-delete the section_enrollment.
		if _, err := pgxPool.Exec(ctx,
			`UPDATE section_enrollments SET deleted_at = now() WHERE id = $1`, se.GetId(),
		); err != nil {
			t.Fatalf("soft-delete SE: %v", err)
		}

		testRedisClient.Del(ctx, "report:student_record:"+studentID.String())

		client := newReportsClient(nil)
		req := connect.NewRequest(&reportsv1.GetStudentRecordReportRequest{StudentId: studentID.String()})
		req.Header().Set("Cookie", "sid="+adminSID)

		resp, err := client.GetStudentRecordReport(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Soft-deleted SE must produce zero rows in the ficha.
		if len(resp.Msg.Rows) != 0 {
			t.Errorf("expected 0 rows in ficha after SE soft-delete, got %d", len(resp.Msg.Rows))
		}
	})
}

// TestReports_Content_OccupancyReport_ZeroCapacitySection verifies that a section with
// capacity == 0 yields fill_percentage == "0.00" (no division-by-zero panic).
func TestReports_Content_OccupancyReport_ZeroCapacitySection(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-zerocap-admin@reports.test", "admin")

	_, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)

	// Seed a section with capacity = 0.
	zeroCapSectionID, zeroCapCleanup := seedSection(t, courseID, periodID, 0)
	t.Cleanup(zeroCapCleanup)

	testRedisClient.Del(ctx, "report:section_occupancy:"+periodID)

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionOccupancyReportRequest{AcademicPeriodId: periodID})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.GetSectionOccupancyReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found *reportsv1.SectionOccupancyRow
	for _, r := range resp.Msg.Rows {
		if r.SectionId == zeroCapSectionID {
			found = r
			break
		}
	}
	if found == nil {
		t.Fatalf("zero-capacity section %s not found in occupancy rows", zeroCapSectionID)
	}
	if found.FillPercentage != "0.00" {
		t.Errorf("zero-capacity fill_percentage = %q, want %q", found.FillPercentage, "0.00")
	}
	if found.Capacity != 0 {
		t.Errorf("zero-capacity section capacity = %d, want 0", found.Capacity)
	}
}

// TestReports_Content_ProgramSummaryReport_OverQuota verifies that when enrolled_count
// exceeds quota_capacity, available_seats is clamped to 0 (never negative).
func TestReports_Content_ProgramSummaryReport_OverQuota(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-overquota-admin@reports.test", "admin")

	// Program with quota capacity = 3, then seed 4 paid (non-cancelled) enrollments.
	programID, programCleanup := seedProgramWithQuota(t, 3, 2025)
	t.Cleanup(programCleanup)

	runSuffix := uuid.New().String()[:8]
	for i := 0; i < 4; i++ {
		email := fmt.Sprintf("reports-overquota-s%d-%s@reports.test", i, runSuffix)
		studentID, _ := seedUserWithSession(t, email, "student")
		seedStudentProfile(t, studentID, 2025)
		_, enrollCleanup := seedPaidEnrollment(t, studentID.String(), programID, 2025)
		t.Cleanup(enrollCleanup)
	}

	testRedisClient.Del(ctx, fmt.Sprintf("report:program_enrollment:%s:%d", programID, 2025))

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetProgramSummaryReportRequest{
		ProgramId: programID,
		Year:      2025,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.GetProgramSummaryReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Rows) == 0 {
		t.Fatal("expected at least 1 row in program summary, got 0")
	}

	row := resp.Msg.Rows[0]
	if row.EnrolledCount != 4 {
		t.Errorf("EnrolledCount = %d, want 4", row.EnrolledCount)
	}
	if row.QuotaCapacity != 3 {
		t.Errorf("QuotaCapacity = %d, want 3", row.QuotaCapacity)
	}
	// available_seats must be clamped to 0, never negative.
	if row.AvailableSeats != 0 {
		t.Errorf("AvailableSeats = %d, want 0 (clamped when over quota)", row.AvailableSeats)
	}
	// fill_percentage > 100% is allowed per the contract (only available_seats is clamped).
	// The value must be parseable as a non-zero decimal.
	if row.FillPercentage == "" || row.FillPercentage == "0.00" {
		t.Errorf("FillPercentage = %q, expected a non-zero percentage for over-quota scenario", row.FillPercentage)
	}
}
