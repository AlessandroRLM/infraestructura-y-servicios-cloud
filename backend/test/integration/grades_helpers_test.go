package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
	gradesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1/gradesv1connect"
	"connectrpc.com/connect"
)

// newGradesClient returns a Connect GradesService client targeting the shared test server.
func newGradesClient(jar http.CookieJar) gradesv1connect.GradesServiceClient {
	return gradesv1connect.NewGradesServiceClient(&http.Client{Jar: jar}, baseURL)
}

// gradesAdminSession seeds an admin user and returns (userID, sessionID).
func gradesAdminSession(t *testing.T, email string) (uuid.UUID, string) {
	t.Helper()
	return seedUserWithSession(t, email, "admin")
}

// gradesTeacherSession seeds a teacher user with a teacher_profile and assigns them
// to the given section. Returns (teacherID, sessionID).
func gradesTeacherSession(t *testing.T, email, sectionID string) (uuid.UUID, string) {
	t.Helper()
	adminSID := catalogSeedAdminSession(t, "grades-teacher-admin-"+email)
	teacherIDStr, teacherSID := seedTeacherProfile(t, email)

	teacherID, err := uuid.Parse(teacherIDStr)
	if err != nil {
		t.Fatalf("gradesTeacherSession: parse teacher id: %v", err)
	}

	// Assign teacher to section.
	client := newCatalogClient(nil)
	ctx := context.Background()
	_, err = client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: sectionID,
		TeacherId: teacherIDStr,
	}), adminSID))
	if err != nil {
		t.Fatalf("gradesTeacherSession: AssignTeacherToSection: %v", err)
	}

	// session cookie SID uses teacherSID
	return teacherID, teacherSID
}

// seedGradesFixture creates a full hierarchy: course, academic period, section, program,
// student enrollment, and section_enrollment. Returns a fixture ready for grade tests.
// All rows are registered for FK-safe cleanup via t.Cleanup.
type gradesFixture struct {
	CourseID            string
	SectionID           string
	ProgramID           string
	PeriodID            string
	PeriodYear          int32
	StudentID           uuid.UUID
	StudentSID          string
	EnrollmentID        string
	SectionEnrollmentID string
}

// seedGradesFixture seeds all the prerequisite rows for grades integration tests.
// adminSID must belong to an admin who can manage catalog and enrollments.
func seedGradesFixture(t *testing.T, adminSID string) gradesFixture {
	t.Helper()
	ctx := context.Background()

	// Program + course.
	programID, courseID, cleanProgram := seedProgramWithCourse(t)
	t.Cleanup(cleanProgram)

	// Academic period with an open enrollment window (student can enroll themselves if needed).
	periodID, periodYear, cleanPeriod := seedAcademicPeriodWithWindow(t, true, false)
	t.Cleanup(cleanPeriod)

	// Section.
	sectionID, cleanSection := seedSection(t, courseID, periodID, 30)
	t.Cleanup(cleanSection)

	// Student.
	studentID, studentSID := seedUserWithSession(t, "grades-student-"+uniqueSuffix(t)+"@grades.test", "student")
	seedStudentProfile(t, studentID, periodYear)

	// Enrollment (paid).
	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, studentID.String(), programID, periodYear)
	t.Cleanup(cleanEnrollment)

	// Section enrollment (admin path).
	seClient := newSectionEnrollmentClient(nil)
	se, err := seEnrollAdmin(ctx, seClient, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("seedGradesFixture: admin section enroll: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, se.GetId())
	})

	return gradesFixture{
		CourseID:            courseID,
		SectionID:           sectionID,
		ProgramID:           programID,
		PeriodID:            periodID,
		PeriodYear:          periodYear,
		StudentID:           studentID,
		StudentSID:          studentSID,
		EnrollmentID:        enrollmentID,
		SectionEnrollmentID: se.GetId(),
	}
}

// seedEvaluationScheme calls CreateEvaluationScheme for the given courseID with the
// provided weights (e.g. []string{"0.5", "0.5"}). Returns the created evaluations.
// Registers cleanup to soft-delete evaluations via raw SQL.
func seedEvaluationScheme(t *testing.T, courseID string, weights []string, adminSID string) []*gradesv1.Evaluation {
	t.Helper()
	ctx := context.Background()

	inputs := make([]*gradesv1.EvaluationInput, len(weights))
	for i, w := range weights {
		inputs[i] = &gradesv1.EvaluationInput{Weight: w}
	}

	client := newGradesClient(nil)
	resp, err := client.CreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId:    courseID,
		Evaluations: inputs,
	}), adminSID))
	if err != nil {
		t.Fatalf("seedEvaluationScheme: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`UPDATE evaluations SET deleted_at = now() WHERE course_id = $1 AND deleted_at IS NULL`,
			courseID,
		)
	})

	return resp.Msg.GetEvaluations()
}

// seedGrade calls RecordGrade and returns the recorded grade proto.
// actorSID must belong to a user with grades.write permission.
func seedGrade(t *testing.T, evaluationID, sectionEnrollmentID, value, actorSID string) *gradesv1.Grade {
	t.Helper()
	ctx := context.Background()
	client := newGradesClient(nil)

	resp, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evaluationID,
		SectionEnrollmentId: sectionEnrollmentID,
		Value:               value,
	}), actorSID))
	if err != nil {
		t.Fatalf("seedGrade: RecordGrade eval=%s se=%s value=%s: %v", evaluationID, sectionEnrollmentID, value, err)
	}

	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM grades WHERE id = $1`, resp.Msg.GetGrade().GetId())
	})

	return resp.Msg.GetGrade()
}

// seedGradesAdminSession creates an admin user and a Redis session.
// Equivalent to gradesAdminSession but returns only the session string for brevity.
func seedGradesAdminSID(t *testing.T, tag string) (uuid.UUID, string) {
	t.Helper()
	return seedUserWithSession(t, "grades-admin-"+tag+"@grades.test", "admin")
}

// withGradeCleanup registers a t.Cleanup to delete a grade by id.
func withGradeCleanup(t *testing.T, gradeID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM grades WHERE id = $1`, gradeID)
	})
}

// withAuditLogCleanup removes audit_logs rows for the given entity and entityID.
func withAuditLogCleanup(t *testing.T, entity, entityID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM audit_logs WHERE entity = $1 AND entity_id = $2::uuid`,
			entity, entityID,
		)
	})
}

// gradesSeedStudent seeds a student user with a student_profile valid for the given year.
// Returns (studentID, studentSID).
func gradesSeedStudent(t *testing.T, tag string, year int32) (uuid.UUID, string) {
	t.Helper()
	email := "grades-stu-" + tag + "-" + uniqueSuffix(t) + "@grades.test"
	id, sid := seedUserWithSession(t, email, "student")
	seedStudentProfile(t, id, year)
	return id, sid
}

// gradesAssignTeacherDirect inserts a section_teachers row directly via SQL.
// Used when a teacher profile already exists and we only need the assignment.
func gradesAssignTeacherDirect(t *testing.T, sectionID, teacherID string) {
	t.Helper()
	_, err := pgxPool.Exec(context.Background(),
		`INSERT INTO section_teachers (section_id, teacher_id) VALUES ($1::uuid, $2::uuid)
		 ON CONFLICT DO NOTHING`,
		sectionID, teacherID,
	)
	if err != nil {
		t.Fatalf("gradesAssignTeacherDirect: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM section_teachers WHERE section_id = $1::uuid AND teacher_id = $2::uuid`,
			sectionID, teacherID,
		)
	})
}

// gradesSeedTeacherWithSession creates a teacher user, inserts a teacher_profile,
// optionally assigns to a section, and returns (teacherIDStr, teacherSID).
func gradesSeedTeacherWithSession(t *testing.T, tag, sectionID string) (string, string) {
	t.Helper()
	email := "grades-teacher-" + tag + "-" + uniqueSuffix(t) + "@grades.test"
	teacherIDStr, teacherSID := seedTeacherProfile(t, email)
	if sectionID != "" {
		gradesAssignTeacherDirect(t, sectionID, teacherIDStr)
	}
	return teacherIDStr, teacherSID
}

// getSectionEnrollmentStatus reads the status and final_grade of a section_enrollment directly.
func getSectionEnrollmentStatus(t *testing.T, seID string) (status string, finalGrade *string) {
	t.Helper()
	var s string
	var fg *string
	err := pgxPool.QueryRow(context.Background(),
		`SELECT status, final_grade::text FROM section_enrollments WHERE id = $1`,
		seID,
	).Scan(&s, &fg)
	if err != nil {
		t.Fatalf("getSectionEnrollmentStatus: %v", err)
	}
	return s, fg
}

// assertSEStatus asserts the section_enrollment has the expected status and final_grade.
func assertSEStatus(t *testing.T, seID, wantStatus string, wantFinalGrade *string) {
	t.Helper()
	status, finalGrade := getSectionEnrollmentStatus(t, seID)
	if status != wantStatus {
		t.Errorf("SE status = %q, want %q", status, wantStatus)
	}
	if wantFinalGrade == nil {
		if finalGrade != nil {
			t.Errorf("SE final_grade = %q, want nil", *finalGrade)
		}
	} else {
		if finalGrade == nil {
			t.Errorf("SE final_grade = nil, want %q", *wantFinalGrade)
		} else if *finalGrade != *wantFinalGrade {
			t.Errorf("SE final_grade = %q, want %q", *finalGrade, *wantFinalGrade)
		}
	}
}

// ptr returns a pointer to the given string.
func ptr(s string) *string { return &s }

// seedTeacherSession creates a teacher session using catalogSeedAdminSession for the
// profile upsert, but returns an independent teacher session.
// Unlike gradesTeacherSession, does NOT assign teacher to a section.
func seedTeacherSession(t *testing.T, tag string) (uuid.UUID, string) {
	t.Helper()
	email := "grades-bare-teacher-" + tag + "-" + uniqueSuffix(t) + "@grades.test"
	_, teacherSID := seedTeacherProfile(t, email)

	// Parse the teacher ID from the DB.
	var teacherID uuid.UUID
	err := pgxPool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, email,
	).Scan(&teacherID)
	if err != nil {
		t.Fatalf("seedTeacherSession: resolve teacher id for %s: %v", email, err)

	}
	// Re-create session with 1h TTL (seedTeacherProfile already creates one internally)
	teacherSID = seedSessionInRedis(t, teacherID, time.Hour)
	return teacherID, teacherSID
}
