package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	gradesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1"
)

// TestGradesOwnProgramFilter_S_B3b verifies that ListOwnGrades filters grades by the
// student's enrollment program (enrollments.program_id), NOT by catalog membership
// (program_courses). This is the S-B3b spec requirement.
//
// Setup:
//   - Course C linked to two programs P1 and P2 via program_courses.
//   - Student has a paid enrollment under P1 (not P2).
//   - One section of C, one section_enrollment, one grade.
//
// Assertions:
//   - program_id=P1  → returns the grade (student enrolled under P1).
//   - program_id=P2  → returns nothing (student NOT enrolled under P2, even though C ∈ P2).
//   - no program filter → returns the grade.
func TestGradesOwnProgramFilter_S_B3b(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedGradesAdminSID(t, "s-b3b")

	// Seed a course and a primary program P1. seedProgramWithCourse links C → P1.
	p1ID, courseID, cleanupP1 := seedProgramWithCourse(t)
	t.Cleanup(cleanupP1)

	// Create a second program P2 and also link C → P2 (catalog membership only).
	var p2ID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO programs (code, name) VALUES ($1, $2) RETURNING id`,
		"S-B3B-P2-"+uniqueSuffix(t), "S-B3b Program 2",
	).Scan(&p2ID); err != nil {
		t.Fatalf("insert p2: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, p2ID)
	})

	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO program_courses (program_id, course_id) VALUES ($1, $2)`,
		p2ID, courseID,
	); err != nil {
		t.Fatalf("link course to p2: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM program_courses WHERE program_id = $1 AND course_id = $2`, p2ID, courseID)
	})

	// Academic period (open enrollment window for admin enroll path).
	periodID, periodYear, cleanupPeriod := seedAcademicPeriodWithWindow(t, true, false)
	t.Cleanup(cleanupPeriod)

	// Section of course C in this period.
	sectionID, cleanupSection := seedSection(t, courseID, periodID, 30)
	t.Cleanup(cleanupSection)

	// Teacher for grading.
	_, teacherSID := gradesSeedTeacherWithSession(t, "s-b3b-teacher", sectionID)

	// Evaluation scheme.
	evals := seedEvaluationScheme(t, courseID, []string{"1.0"}, adminSID)

	// Student enrolled under P1 only (not P2).
	studentID, studentSID := seedUserWithSession(t, "s-b3b-student-"+uniqueSuffix(t)+"@grades.test", "student")
	seedStudentProfile(t, studentID, periodYear)

	// Paid enrollment under P1.
	enrollmentID, cleanupEnroll := seedPaidEnrollment(t, studentID.String(), p1ID, periodYear)
	t.Cleanup(cleanupEnroll)

	// Section enrollment via admin.
	seClient := newSectionEnrollmentClient(nil)
	se, err := seEnrollAdmin(ctx, seClient, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("admin section enroll: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, se.GetId())
	})

	// Record a grade.
	g := seedGrade(t, evals[0].GetId(), se.GetId(), "5.5", teacherSID)
	_ = g

	client := newGradesClient(nil)

	// Case 1: filter by P1 → must return the grade.
	respP1, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{
		ProgramId: &p1ID,
	}), studentSID))
	if err != nil {
		t.Fatalf("ListOwnGrades program_id=P1: %v", err)
	}
	if len(respP1.Msg.GetGrades()) != 1 {
		t.Fatalf("program_id=P1: got %d grades, want 1", len(respP1.Msg.GetGrades()))
	}
	// program_id on the P1-filtered result must equal P1 (enrollment's program, not the filter param).
	if gotP1ProgramID := respP1.Msg.GetGrades()[0].GetProgramId(); gotP1ProgramID != p1ID {
		t.Errorf("program_id=P1 filter: OwnGrade.program_id = %q, want %q", gotP1ProgramID, p1ID)
	}

	// Case 2: filter by P2 → must return nothing (student enrolled under P1, not P2).
	p2IDStr := p2ID.String()
	respP2, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{
		ProgramId: &p2IDStr,
	}), studentSID))
	if err != nil {
		t.Fatalf("ListOwnGrades program_id=P2: %v", err)
	}
	if len(respP2.Msg.GetGrades()) != 0 {
		t.Errorf("program_id=P2: got %d grades, want 0 (student not enrolled under P2)", len(respP2.Msg.GetGrades()))
	}

	// Case 3: no program filter → must return the grade.
	respAll, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{}), studentSID))
	if err != nil {
		t.Fatalf("ListOwnGrades no filter: %v", err)
	}
	if len(respAll.Msg.GetGrades()) != 1 {
		t.Errorf("no filter: got %d grades, want 1", len(respAll.Msg.GetGrades()))
	}

	// Case 4: verify program_id on the returned row equals P1 (from enrollment, not filter param).
	if len(respAll.Msg.GetGrades()) > 0 {
		gotProgramID := respAll.Msg.GetGrades()[0].GetProgramId()
		if gotProgramID != p1ID {
			t.Errorf("OwnGrade.program_id = %q, want %q", gotProgramID, p1ID)
		}
	}
}

// TestGradesOwnProgramFilter_PeriodFilter verifies that the academic_period_id filter
// returns grades for the matching period and nothing for a different period.
func TestGradesOwnProgramFilter_PeriodFilter(t *testing.T) {
	ctx := context.Background()

	_, adminSID := seedGradesAdminSID(t, "period-filter")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "period-filter-teacher", fix.SectionID)

	// Seed a grade in fix.PeriodID.
	g := seedGrade(t, evals[0].GetId(), fix.SectionEnrollmentID, "4.0", teacherSID)
	_ = g

	// Seed a second, distinct academic period (no section/grade in it).
	otherPeriodID, _, cleanupOtherPeriod := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(cleanupOtherPeriod)

	client := newGradesClient(nil)

	// Matching period → returns the grade.
	respMatch, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{
		AcademicPeriodId: &fix.PeriodID,
	}), fix.StudentSID))
	if err != nil {
		t.Fatalf("ListOwnGrades period=matching: %v", err)
	}
	if len(respMatch.Msg.GetGrades()) != 1 {
		t.Errorf("period=matching: got %d grades, want 1", len(respMatch.Msg.GetGrades()))
	}

	// Non-matching period → returns nothing.
	respOther, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{
		AcademicPeriodId: &otherPeriodID,
	}), fix.StudentSID))
	if err != nil {
		t.Fatalf("ListOwnGrades period=other: %v", err)
	}
	if len(respOther.Msg.GetGrades()) != 0 {
		t.Errorf("period=other: got %d grades, want 0", len(respOther.Msg.GetGrades()))
	}
}
