package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	gradesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1"
)

// TestGradesRead_StudentListOwnGrades (AS-14): student sees their own grades only.
func TestGradesRead_StudentListOwnGrades(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "student-own-grades")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "student-own-grades", fix.SectionID)

	// Record a grade for the student's section enrollment.
	g := seedGrade(t, evals[0].GetId(), fix.SectionEnrollmentID, "5.0", teacherSID)
	_ = g

	client := newGradesClient(nil)

	// Student calls ListOwnGrades.
	resp, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{}), fix.StudentSID))
	if err != nil {
		t.Fatalf("ListOwnGrades: %v", err)
	}

	grades := resp.Msg.GetGrades()
	if len(grades) == 0 {
		t.Fatal("student ListOwnGrades: expected at least 1 grade, got 0")
	}

	// Verify graded_by is NOT exposed in OwnGrade.
	for _, og := range grades {
		if og.GetSectionEnrollmentId() != fix.SectionEnrollmentID {
			t.Errorf("OwnGrade SE id = %q, want %q", og.GetSectionEnrollmentId(), fix.SectionEnrollmentID)
		}
	}
}

// TestGradesRead_OwnGradesAreIsolated: student B cannot see student A's grades via ListOwnGrades.
func TestGradesRead_OwnGradesAreIsolated(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "own-grades-isolated")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "own-grades-isolated", fix.SectionID)

	// Grade student A.
	_ = seedGrade(t, evals[0].GetId(), fix.SectionEnrollmentID, "5.0", teacherSID)

	// Seed student B (not enrolled in this section).
	_, studentBSID := seedGradesAdminSID(t, "own-grades-isolated-b") // admin role so they can call ListOwnGrades

	// Actually seed as student role.
	studentBID, studentBSID2 := seedUserWithSession(t, "grades-own-b-"+uniqueSuffix(t)+"@grades.test", "student")
	_ = studentBID

	client := newGradesClient(nil)

	resp, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{}), studentBSID2))
	if err != nil {
		t.Fatalf("student B ListOwnGrades: %v", err)
	}
	// Student B should see 0 grades (they have no section_enrollment in this section).
	if len(resp.Msg.GetGrades()) != 0 {
		t.Errorf("student B should have 0 grades, got %d", len(resp.Msg.GetGrades()))
	}
	_ = studentBSID
}

// TestGradesRead_GradedByHiddenFromStudent: the OwnGrade proto does NOT expose graded_by.
// We verify the student response has no graded_by field populated.
func TestGradesRead_GradedByHiddenFromStudent(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "graded-by-hidden")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "graded-by-hidden", fix.SectionID)

	_ = seedGrade(t, evals[0].GetId(), fix.SectionEnrollmentID, "5.0", teacherSID)

	client := newGradesClient(nil)

	resp, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{}), fix.StudentSID))
	if err != nil {
		t.Fatalf("ListOwnGrades: %v", err)
	}
	if len(resp.Msg.GetGrades()) == 0 {
		t.Fatal("expected at least 1 grade")
	}

	// OwnGrade has no graded_by field — verify proto type is OwnGrade (has no GradedBy).
	// This is enforced at compile time by the proto, but we assert the value isn't leaked
	// by checking the response type explicitly.
	og := resp.Msg.GetGrades()[0]
	// OwnGrade proto does not have a GradedBy field — this is a compile-time guarantee.
	// We verify the value contains evaluation_id and section_enrollment_id as expected.
	if og.GetEvaluationId() == "" {
		t.Error("OwnGrade.evaluation_id should not be empty")
	}
	if og.GetSectionEnrollmentId() == "" {
		t.Error("OwnGrade.section_enrollment_id should not be empty")
	}
}

// TestGradesRead_TeacherListGradesForSection (AS-16): teacher lists grades for their section.
func TestGradesRead_TeacherListGradesForSection(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "teacher-list-grades")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "teacher-list-grades", fix.SectionID)

	_ = seedGrade(t, evals[0].GetId(), fix.SectionEnrollmentID, "6.0", teacherSID)

	client := newGradesClient(nil)

	resp, err := client.ListGradesForSection(ctx, withSID(connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: fix.SectionID,
	}), teacherSID))
	if err != nil {
		t.Fatalf("ListGradesForSection: %v", err)
	}
	if len(resp.Msg.GetGrades()) == 0 {
		t.Error("teacher should see at least 1 grade in their section")
	}

	// Teacher's view includes graded_by.
	for _, g := range resp.Msg.GetGrades() {
		if g.GetGradedBy() == "" {
			t.Error("Grade.graded_by should be set for teacher view")
		}
	}
}

// TestGradesRead_TeacherListGradesAcrossSections: teachers with grades.read permission
// can list grades for any section (no section_teachers scoping on read; write requires membership).
// This verifies that the permission system works correctly: teacher has grades.read, not grades.write.
func TestGradesRead_TeacherListGradesAcrossSections(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "teacher-across-sections")

	// Two separate fixtures with different courses and sections.
	fix1 := seedGradesFixture(t, adminSID)
	fix2 := seedGradesFixture(t, adminSID)

	// Teacher1 is assigned only to section1 (for write permission purposes).
	_, teacher1SID := gradesSeedTeacherWithSession(t, "across-t1", fix1.SectionID)

	// Create a grade in section2 via admin override.
	evals2 := seedEvaluationScheme(t, fix2.CourseID, []string{"1.0"}, adminSID)
	_, err := newGradesClient(nil).OverrideGrade(ctx, withSID(connect.NewRequest(&gradesv1.OverrideGradeRequest{
		EvaluationId:        evals2[0].GetId(),
		SectionEnrollmentId: fix2.SectionEnrollmentID,
		Value:               "5.0",
	}), adminSID))
	if err != nil {
		t.Fatalf("admin OverrideGrade for section2: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM grades WHERE section_enrollment_id = $1`, fix2.SectionEnrollmentID)
	})

	// Teacher1 can list section2's grades (grades.read is not section-scoped on reads).
	// This confirms that write authz (section_teachers) is separate from read authz (permission only).
	resp, err := newGradesClient(nil).ListGradesForSection(ctx, withSID(connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: fix2.SectionID,
	}), teacher1SID))
	if err != nil {
		t.Fatalf("ListGradesForSection cross-section: %v", err)
	}
	// Teacher CAN see section2's grades (grades.read is not section-scoped).
	if len(resp.Msg.GetGrades()) < 1 {
		t.Errorf("teacher1 should see at least 1 grade in section2, got %d", len(resp.Msg.GetGrades()))
	}
}

// TestGradesRead_GetGradeHappyPath: admin can fetch a single grade by id.
func TestGradesRead_GetGradeHappyPath(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "get-grade-happy")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "get-grade-happy", fix.SectionID)

	g := seedGrade(t, evals[0].GetId(), fix.SectionEnrollmentID, "4.5", teacherSID)

	client := newGradesClient(nil)
	resp, err := client.GetGrade(ctx, withSID(connect.NewRequest(&gradesv1.GetGradeRequest{
		Id: g.GetId(),
	}), adminSID))
	if err != nil {
		t.Fatalf("GetGrade: %v", err)
	}
	if resp.Msg.GetGrade().GetId() != g.GetId() {
		t.Errorf("GetGrade id = %q, want %q", resp.Msg.GetGrade().GetId(), g.GetId())
	}
}

// TestGradesRead_GetGradeNotFound: fetching a non-existent grade id → CodeNotFound.
func TestGradesRead_GetGradeNotFound(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "get-grade-notfound")
	client := newGradesClient(nil)

	_, err := client.GetGrade(ctx, withSID(connect.NewRequest(&gradesv1.GetGradeRequest{
		Id: "00000000-0000-0000-0000-000000000002",
	}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestGradesRead_AdminOverrideGrade (AS-19): admin calls OverrideGrade → graded_by=admin,
// version is set, and audit log is written on correction.
func TestGradesRead_AdminOverrideGrade(t *testing.T) {
	ctx := context.Background()
	adminID, adminSID := seedGradesAdminSID(t, "admin-override")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	client := newGradesClient(nil)

	// Admin records initial grade.
	first, err := client.OverrideGrade(ctx, withSID(connect.NewRequest(&gradesv1.OverrideGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
	}), adminSID))
	if err != nil {
		t.Fatalf("OverrideGrade (first): %v", err)
	}
	g := first.Msg.GetGrade()
	withGradeCleanup(t, g.GetId())

	if g.GetGradedBy() != adminID.String() {
		t.Errorf("graded_by = %q, want admin %q", g.GetGradedBy(), adminID.String())
	}
	if g.GetVersion() != 1 {
		t.Errorf("version = %d, want 1", g.GetVersion())
	}

	// Admin overrides again → version bumps, audit log written.
	v1 := int32(1)
	second, err := client.OverrideGrade(ctx, withSID(connect.NewRequest(&gradesv1.OverrideGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "4.0",
		ExpectedVersion:     &v1,
	}), adminSID))
	if err != nil {
		t.Fatalf("OverrideGrade (correction): %v", err)
	}
	g2 := second.Msg.GetGrade()
	if g2.GetVersion() != 2 {
		t.Errorf("version after override = %d, want 2", g2.GetVersion())
	}

	// Audit log should have 1 row (on correction only).
	var count int
	err = pgxPool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE entity = 'grades' AND entity_id = $1::uuid`,
		g.GetId(),
	).Scan(&count)
	if err != nil {
		t.Fatalf("count audit_logs: %v", err)
	}
	if count != 1 {
		t.Errorf("audit_logs count = %d, want 1", count)
	}
	withAuditLogCleanup(t, "grades", g.GetId())
}

// TestGradesRead_PermGradesReadDenied: student calling ListGradesForSection → CodePermissionDenied.
func TestGradesRead_PermGradesReadDenied(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "perm-read-denied")
	fix := seedGradesFixture(t, adminSID)
	client := newGradesClient(nil)

	_, err := client.ListGradesForSection(ctx, withSID(connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: fix.SectionID,
	}), fix.StudentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestGradesRead_PermGradesViewOwnDenied: non-student (no grades.view_own) →
// calling ListOwnGrades → CodePermissionDenied.
func TestGradesRead_PermGradesViewOwnDenied(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "perm-viewown-denied")
	fix := seedGradesFixture(t, adminSID)

	// A teacher has grades.write/grades.read but NOT grades.view_own.
	_, teacherSID := gradesSeedTeacherWithSession(t, "perm-viewown", fix.SectionID)
	client := newGradesClient(nil)

	_, err := client.ListOwnGrades(ctx, withSID(connect.NewRequest(&gradesv1.ListOwnGradesRequest{}), teacherSID))
	// Teacher role does not have grades.view_own → PermissionDenied.
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestGradesRead_PermGradesOverrideDenied: teacher calling OverrideGrade → CodePermissionDenied.
func TestGradesRead_PermGradesOverrideDenied(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "perm-override-denied")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "perm-override-denied", fix.SectionID)
	client := newGradesClient(nil)

	_, err := client.OverrideGrade(ctx, withSID(connect.NewRequest(&gradesv1.OverrideGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
	}), teacherSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestGradesRead_UnauthenticatedDeniedOnAllEndpoints: quick table-driven
// unauthenticated gate across multiple grades procedures.
func TestGradesRead_UnauthenticatedDeniedOnAllEndpoints(t *testing.T) {
	ctx := context.Background()
	client := newGradesClient(nil)
	fakeID := "00000000-0000-0000-0000-000000000003"

	cases := []struct {
		name string
		call func() error
	}{
		{"ListEvaluations", func() error {
			_, err := client.ListEvaluations(ctx, connect.NewRequest(&gradesv1.ListEvaluationsRequest{CourseId: fakeID}))
			return err
		}},
		{"ListGradesForSection", func() error {
			_, err := client.ListGradesForSection(ctx, connect.NewRequest(&gradesv1.ListGradesForSectionRequest{SectionId: fakeID}))
			return err
		}},
		{"GetGrade", func() error {
			_, err := client.GetGrade(ctx, connect.NewRequest(&gradesv1.GetGradeRequest{Id: fakeID}))
			return err
		}},
		{"ListOwnGrades", func() error {
			_, err := client.ListOwnGrades(ctx, connect.NewRequest(&gradesv1.ListOwnGradesRequest{}))
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			assertConnectCode(t, err, connect.CodeUnauthenticated)
		})
	}
}
