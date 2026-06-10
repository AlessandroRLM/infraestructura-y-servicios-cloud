package integration_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	gradesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1"
)

// TestGradesRecord_InsertHappyPath (AS-5): teacher records a grade → version=1, graded_by=teacher.
func TestGradesRecord_InsertHappyPath(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "record-insert")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	teacherIDStr, teacherSID := gradesSeedTeacherWithSession(t, "record-insert", fix.SectionID)

	client := newGradesClient(nil)
	resp, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.5",
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade: %v", err)
	}

	g := resp.Msg.GetGrade()
	withGradeCleanup(t, g.GetId())

	if g.GetVersion() != 1 {
		t.Errorf("grade version = %d, want 1", g.GetVersion())
	}
	if g.GetGradedBy() != teacherIDStr {
		t.Errorf("graded_by = %q, want %q", g.GetGradedBy(), teacherIDStr)
	}
	if g.GetValue() != "5.5" {
		t.Errorf("value = %q, want 5.5", g.GetValue())
	}
	if g.GetEvaluationId() != evals[0].GetId() {
		t.Errorf("evaluation_id = %q, want %q", g.GetEvaluationId(), evals[0].GetId())
	}
}

// TestGradesRecord_Correction (AS-6): teacher corrects a grade → version bumps to 2, graded_by updated.
func TestGradesRecord_Correction(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "record-correction")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	teacherIDStr, teacherSID := gradesSeedTeacherWithSession(t, "record-correction", fix.SectionID)

	client := newGradesClient(nil)

	// First write.
	first, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "4.0",
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade (first): %v", err)
	}
	withGradeCleanup(t, first.Msg.GetGrade().GetId())

	// Correction with expected_version=1.
	v1 := int32(1)
	second, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
		ExpectedVersion:     &v1,
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade (correction): %v", err)
	}

	g := second.Msg.GetGrade()
	if g.GetVersion() != 2 {
		t.Errorf("version after correction = %d, want 2", g.GetVersion())
	}
	// numericToString preserves the DB column scale (NUMERIC(3,1) → always one decimal digit).
	if g.GetValue() != "5.0" {
		t.Errorf("value after correction = %q, want \"5.0\"", g.GetValue())
	}
	if g.GetGradedBy() != teacherIDStr {
		t.Errorf("graded_by after correction = %q, want %q", g.GetGradedBy(), teacherIDStr)
	}

	// Verify audit log was written.
	var auditCount int
	err = pgxPool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE entity = 'grades' AND entity_id = $1::uuid`,
		g.GetId(),
	).Scan(&auditCount)
	if err != nil {
		t.Fatalf("count audit_logs: %v", err)
	}
	if auditCount != 1 {
		t.Errorf("audit_logs count = %d, want 1", auditCount)
	}
	withAuditLogCleanup(t, "grades", g.GetId())
}

// TestGradesRecord_OptimisticLockConflict (AS-7): version mismatch → CodeAborted with version in detail.
func TestGradesRecord_OptimisticLockConflict(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "optlock-conflict")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "optlock-conflict", fix.SectionID)

	client := newGradesClient(nil)

	// First write (version=1).
	first, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "4.0",
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade (first): %v", err)
	}
	withGradeCleanup(t, first.Msg.GetGrade().GetId())

	// Bump with correct version → version becomes 2.
	v1 := int32(1)
	_, err = client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
		ExpectedVersion:     &v1,
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade (bump): %v", err)
	}

	// Now supply stale version=1 again → CodeAborted.
	_, err = client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "6.0",
		ExpectedVersion:     &v1,
	}), teacherSID))
	assertConnectCode(t, err, connect.CodeAborted)

	// The error message must contain the current version number.
	ce, _ := err.(*connect.Error)
	if !strings.Contains(ce.Message(), "2") {
		t.Errorf("CodeAborted message should contain current version 2, got: %q", ce.Message())
	}
}

// TestGradesRecord_NoVersionOnConflict: trying to write a second time without expected_version → CodeAborted.
func TestGradesRecord_NoVersionOnConflict(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "no-version-conflict")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "no-version-conflict", fix.SectionID)

	client := newGradesClient(nil)

	first, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "4.0",
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade (first): %v", err)
	}
	withGradeCleanup(t, first.Msg.GetGrade().GetId())

	// Second write without expected_version → CodeAborted.
	_, err = client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
	}), teacherSID))
	assertConnectCode(t, err, connect.CodeAborted)
}

// TestGradesRecord_ValueOutOfRange (AS-8): value 7.5 → CodeInvalidArgument.
func TestGradesRecord_ValueOutOfRange(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "out-of-range")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "out-of-range", fix.SectionID)

	client := newGradesClient(nil)

	_, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "7.5",
	}), teacherSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)

	_, err = client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "0.9",
	}), teacherSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestGradesRecord_AntiSelfGrade (AS-17): teacher attempts to grade their own
// section enrollment → CodePermissionDenied.
func TestGradesRecord_AntiSelfGrade(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "anti-selfgrade")

	// Create course + section.
	programID, courseID, cleanProgram := seedProgramWithCourse(t)
	t.Cleanup(cleanProgram)
	periodID, periodYear, cleanPeriod := seedAcademicPeriodWithWindow(t, true, false)
	t.Cleanup(cleanPeriod)
	sectionID, cleanSection := seedSection(t, courseID, periodID, 30)
	t.Cleanup(cleanSection)

	// Seed the teacher profile first (returns string ID and session).
	teacherIDStr, teacherSID := gradesSeedTeacherWithSession(t, "anti-selfgrade", sectionID)

	// Seed a STUDENT profile for the same user so they can be enrolled.
	// seedTeacherProfile creates teacher_profiles; we also need student_profiles.
	teacherGUID, err := uuid.Parse(teacherIDStr)
	if err != nil {
		t.Fatalf("parse teacher UUID: %v", err)
	}
	seedStudentProfile(t, teacherGUID, periodYear)

	enrollmentID, cleanEnrollment := seedPaidEnrollment(t, teacherIDStr, programID, periodYear)
	t.Cleanup(cleanEnrollment)

	seClient := newSectionEnrollmentClient(nil)
	se, err := seEnrollAdmin(ctx, seClient, adminSID, enrollmentID, sectionID)
	if err != nil {
		t.Fatalf("admin enroll teacher-student: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, se.GetId())
	})

	evals := seedEvaluationScheme(t, courseID, []string{"1.0"}, adminSID)
	client := newGradesClient(nil)

	// Teacher tries to grade their own section_enrollment → PermissionDenied.
	_, err = client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: se.GetId(),
		Value:               "7.0",
	}), teacherSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestGradesRecord_NotSectionTeacher (AS-18): a teacher not in section_teachers for
// this section → CodePermissionDenied.
func TestGradesRecord_NotSectionTeacher(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "not-section-teacher")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)

	// Teacher NOT assigned to this section.
	_, outsiderTeacherSID := gradesSeedTeacherWithSession(t, "outsider", "" /* no section */)

	client := newGradesClient(nil)

	_, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "4.0",
	}), outsiderTeacherSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestGradesRecord_StudentCannotRecord (AS-20): student role → CodePermissionDenied.
func TestGradesRecord_StudentCannotRecord(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "student-cannot-record")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	client := newGradesClient(nil)

	_, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
	}), fix.StudentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestGradesRecord_NoSchemeForCourse (AS-23): evaluation not found → CodeNotFound.
func TestGradesRecord_NoSchemeForCourse(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "no-scheme")
	fix := seedGradesFixture(t, adminSID)

	_, teacherSID := gradesSeedTeacherWithSession(t, "no-scheme", fix.SectionID)
	client := newGradesClient(nil)

	// Use a random UUID as evaluation_id — no scheme exists.
	fakeEvalID := "00000000-0000-0000-0000-000000000001"
	_, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        fakeEvalID,
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
	}), teacherSID))
	// No scheme → evaluation not found → CodeNotFound.
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestGradesRecord_CrossCourseMismatch (record-9): evaluation belongs to course C1,
// section_enrollment belongs to a section of course C2 → CodeInvalidArgument; no grade written.
func TestGradesRecord_CrossCourseMismatch(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "cross-course-mismatch")

	// Universe 1: course C1 with an evaluation scheme.
	fix1 := seedGradesFixture(t, adminSID)
	evalsC1 := seedEvaluationScheme(t, fix1.CourseID, []string{"1.0"}, adminSID)

	// Universe 2: course C2 with its own section and a student section_enrollment.
	fix2 := seedGradesFixture(t, adminSID)

	// Teacher assigned to section2 (the section from C2's universe) — has grades.write.
	_, teacherSID := gradesSeedTeacherWithSession(t, "cross-course-mismatch", fix2.SectionID)

	client := newGradesClient(nil)

	// Attempt: evaluation from C1 + section_enrollment from C2 → cross-course mismatch.
	_, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evalsC1[0].GetId(),
		SectionEnrollmentId: fix2.SectionEnrollmentID,
		Value:               "5.0",
	}), teacherSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)

	// Verify no grade was written for fix2's section_enrollment.
	var gradeCount int
	if qErr := pgxPool.QueryRow(ctx,
		`SELECT count(*) FROM grades WHERE section_enrollment_id = $1`,
		fix2.SectionEnrollmentID,
	).Scan(&gradeCount); qErr != nil {
		t.Fatalf("count grades: %v", qErr)
	}
	if gradeCount != 0 {
		t.Errorf("cross-course mismatch wrote %d grade(s), want 0", gradeCount)
	}
}

// TestGradesRecord_ConcurrentDuplicate (TDD-6): two concurrent RecordGrade calls for
// the same (evaluation, section_enrollment) → exactly one succeeds with version=1;
// the other gets CodeAborted (no expected_version on conflict).
func TestGradesRecord_ConcurrentDuplicate(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "concurrent-dup")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "concurrent-dup", fix.SectionID)

	client := newGradesClient(nil)

	var (
		mu       sync.Mutex
		success  int
		aborted  int
		otherErr []error
		gradeIDs []string
	)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
				EvaluationId:        evals[0].GetId(),
				SectionEnrollmentId: fix.SectionEnrollmentID,
				Value:               "5.0",
			}), teacherSID))

			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
				gradeIDs = append(gradeIDs, resp.Msg.GetGrade().GetId())
			} else if ce, ok := err.(*connect.Error); ok && ce.Code() == connect.CodeAborted {
				aborted++
			} else {
				otherErr = append(otherErr, err)
			}
		}()
	}
	wg.Wait()

	for _, id := range gradeIDs {
		withGradeCleanup(t, id)
	}

	if len(otherErr) > 0 {
		t.Fatalf("unexpected errors: %v", otherErr)
	}
	if success != 1 {
		t.Errorf("concurrent insert: success = %d, want 1", success)
	}
	if aborted != 1 {
		t.Errorf("concurrent insert: aborted = %d, want 1", aborted)
	}
}

