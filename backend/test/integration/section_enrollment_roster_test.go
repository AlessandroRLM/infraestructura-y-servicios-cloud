package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
)

// seedRosterFixture creates a teacher, a section assigned to that teacher, and n
// section_enrollments for distinct students. Each student is a real user with the
// "student" role so that the enrollment can be inserted via the enrollments table.
//
// Returns: teacherSID, sectionID, []sectionEnrollmentID, adminSID, and a cleanup func.
func seedRosterFixture(t *testing.T, email string, n int) (teacherSID, sectionID string, seIDs []string, adminSID string, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	// Create teacher.
	teacherID, teacherSID := seedTeacherProfile(t, "roster-teacher-"+email)

	// Admin session for section/teacher assignment.
	adminSID = catalogSeedAdminSession(t, "roster-admin-"+email)

	// Create a course, period, and section using raw SQL (simpler than going through RPC
	// since seedTeacherWithSections already uses the catalog client path and we need the
	// section_id for direct section_enrollment inserts).
	var courseID, periodID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO courses (code, name, credits) VALUES ($1, $2, $3) RETURNING id`,
		"ROST-CRS-"+uniqueSuffix(t), "Roster Test Course", 3,
	).Scan(&courseID); err != nil {
		t.Fatalf("seedRosterFixture: insert course: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM courses WHERE id = $1`, courseID)
	})

	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO academic_periods (year, term, start_date, end_date) VALUES ($1, $2, $3, $4) RETURNING id`,
		seAcademicPeriodYearCounter.Add(1), 1, "4000-03-01", "4000-07-31",
	).Scan(&periodID); err != nil {
		t.Fatalf("seedRosterFixture: insert period: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM academic_periods WHERE id = $1`, periodID)
	})

	var secID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO sections (course_id, academic_period_id, capacity) VALUES ($1, $2, $3) RETURNING id`,
		courseID, periodID, 50,
	).Scan(&secID); err != nil {
		t.Fatalf("seedRosterFixture: insert section: %v", err)
	}
	sectionID = secID.String()
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pgxPool.Exec(c, `DELETE FROM section_teachers WHERE section_id = $1`, secID)
		_, _ = pgxPool.Exec(c, `DELETE FROM sections WHERE id = $1`, secID)
	})

	// Assign teacher to section.
	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO section_teachers (section_id, teacher_id) VALUES ($1, $2)`,
		secID, teacherID,
	); err != nil {
		t.Fatalf("seedRosterFixture: insert section_teacher: %v", err)
	}

	// Create n students, each with a paid enrollment and a section_enrollment.
	for i := 0; i < n; i++ {
		studentID, _ := seedUserWithSession(t, "roster-student-"+uuid.New().String()[:8]+"@test.local", "student")
		// enrollments.student_id references student_profiles(user_id), not users(id).
		seedStudentProfile(t, studentID, 4000)

		// Insert paid enrollment for this student.
		var programID uuid.UUID
		if err := pgxPool.QueryRow(ctx,
			`INSERT INTO programs (code, name) VALUES ($1, $2) RETURNING id`,
			"RPROG-"+uniqueSuffix(t), "Roster Prog",
		).Scan(&programID); err != nil {
			t.Fatalf("seedRosterFixture: insert program: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, programID)
		})

		var enrollID uuid.UUID
		if err := pgxPool.QueryRow(ctx,
			`INSERT INTO enrollments (student_id, program_id, year, status) VALUES ($1, $2, $3, 'paid') RETURNING id`,
			studentID, programID, 4000,
		).Scan(&enrollID); err != nil {
			t.Fatalf("seedRosterFixture: insert enrollment: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, enrollID)
		})

		var seID uuid.UUID
		if err := pgxPool.QueryRow(ctx,
			`INSERT INTO section_enrollments (enrollment_id, section_id, status) VALUES ($1, $2, 'in_progress') RETURNING id`,
			enrollID, secID,
		).Scan(&seID); err != nil {
			t.Fatalf("seedRosterFixture: insert section_enrollment: %v", err)
		}
		seIDs = append(seIDs, seID.String())
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, seID)
		})
	}

	cleanup = func() {} // individual cleanups registered above
	return teacherSID, sectionID, seIDs, adminSID, cleanup
}

// --- Cross-teacher isolation (negative test — strengthens S-07) ---

// TestListSectionRosterForTeacher_CrossTeacherIsolation verifies two properties:
//
//  1. Teacher B calling ListSectionRosterForTeacher on teacher A's section receives an
//     empty page (anti-leak), even though B has their OWN section with enrollments (so B
//     has section_teachers rows and the EXISTS guard is exercised against the correct target).
//
//  2. Teacher B calling ListSectionRosterForTeacher on their OWN section returns a
//     non-empty page — confirming the anti-leak in (1) is not a false positive.
func TestListSectionRosterForTeacher_CrossTeacherIsolation(t *testing.T) {
	// Seed teacher A with a section containing 3 students.
	teacherASID, sectionIDA, _, _, _ := seedRosterFixture(t, "iso-teacher-a@test.local", 3)

	// Seed teacher B with their OWN section containing 2 students.
	teacherBSID, sectionIDB, seIDsB, _, _ := seedRosterFixture(t, "iso-teacher-b@test.local", 2)

	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	// (1) Teacher B queries teacher A's section — must get empty (anti-leak).
	respLeak, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionIDA,
			PageSize:  20,
		},
	), teacherBSID))
	if err != nil {
		t.Fatalf("cross-teacher anti-leak: expected OK, got %v", err)
	}
	if len(respLeak.Msg.GetSectionEnrollments()) != 0 {
		t.Errorf("cross-teacher anti-leak: teacher B got %d rows for teacher A's section, want 0",
			len(respLeak.Msg.GetSectionEnrollments()))
	}

	// (2) Teacher B queries their OWN section — must return B's enrollments (non-empty).
	respOwn, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionIDB,
			PageSize:  20,
		},
	), teacherBSID))
	if err != nil {
		t.Fatalf("teacher B own roster: %v", err)
	}
	if len(respOwn.Msg.GetSectionEnrollments()) != len(seIDsB) {
		t.Errorf("teacher B own roster: got %d rows, want %d",
			len(respOwn.Msg.GetSectionEnrollments()), len(seIDsB))
	}

	// Confirm teacher A cannot see teacher B's section either (symmetry).
	respSymmetric, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionIDB,
			PageSize:  20,
		},
	), teacherASID))
	if err != nil {
		t.Fatalf("cross-teacher symmetry: expected OK, got %v", err)
	}
	if len(respSymmetric.Msg.GetSectionEnrollments()) != 0 {
		t.Errorf("cross-teacher symmetry: teacher A got %d rows for teacher B's section, want 0",
			len(respSymmetric.Msg.GetSectionEnrollments()))
	}
}

// --- S-06: Teacher gets roster for a section they teach (with student_id) ---

// TestListSectionRosterForTeacher_HappyPath verifies that a teacher who teaches section_A
// receives all 5 in_progress enrollments, each with non-empty student_id, enrollment_id,
// section_id, status, and registered_at; final_grade is empty for in_progress rows.
func TestListSectionRosterForTeacher_HappyPath(t *testing.T) {
	teacherSID, sectionID, seIDs, _, _ := seedRosterFixture(t, "happy@test.local", 5)
	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	resp, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionID,
			PageSize:  20,
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher: %v", err)
	}

	rows := resp.Msg.GetSectionEnrollments()
	if len(rows) != 5 {
		t.Fatalf("ListSectionRosterForTeacher: got %d rows, want 5", len(rows))
	}

	// Verify all seeded section_enrollment IDs are present.
	gotIDs := make(map[string]bool)
	for _, r := range rows {
		gotIDs[r.GetId()] = true
	}
	for _, id := range seIDs {
		if !gotIDs[id] {
			t.Errorf("section_enrollment %s missing from roster", id)
		}
	}

	// Verify fields are present.
	for _, r := range rows {
		if r.GetStudentId() == "" {
			t.Errorf("row %s: student_id is empty", r.GetId())
		}
		if r.GetEnrollmentId() == "" {
			t.Errorf("row %s: enrollment_id is empty", r.GetId())
		}
		if r.GetSectionId() != sectionID {
			t.Errorf("row %s: section_id = %q, want %q", r.GetId(), r.GetSectionId(), sectionID)
		}
		if r.GetStatus() != "in_progress" {
			t.Errorf("row %s: status = %q, want in_progress", r.GetId(), r.GetStatus())
		}
		if r.GetRegisteredAt() == "" {
			t.Errorf("row %s: registered_at is empty", r.GetId())
		}
	}

	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("next_page_token = %q, want empty", resp.Msg.GetNextPageToken())
	}
}

// --- S-07: Teacher requests roster for a section they do NOT teach (anti-leak) ---

// TestListSectionRosterForTeacher_OutOfScope_EmptyNotError verifies that a teacher who
// is NOT in section_teachers for the requested section receives an empty page and OK (not 403).
func TestListSectionRosterForTeacher_OutOfScope_EmptyNotError(t *testing.T) {
	// Create a teacher with NO section_teachers rows for the target section.
	_, teacherSID := seedTeacherProfile(t, "roster-out-of-scope@test.local")

	// Create a separate section (different teacher owns it).
	_, otherSectionID, _, _, _ := seedRosterFixture(t, "out-of-scope-owner@test.local", 2)

	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	resp, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: otherSectionID,
			PageSize:  20,
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher (out-of-scope): expected OK, got %v", err)
	}
	if len(resp.Msg.GetSectionEnrollments()) != 0 {
		t.Errorf("ListSectionRosterForTeacher (out-of-scope): got %d rows, want 0", len(resp.Msg.GetSectionEnrollments()))
	}
	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("next_page_token = %q, want empty", resp.Msg.GetNextPageToken())
	}
}

// --- S-08: Section with no enrolled students ---

// TestListSectionRosterForTeacher_EmptySection verifies that a teacher with a section_teachers
// row but zero section_enrollments receives an empty page and OK.
func TestListSectionRosterForTeacher_EmptySection(t *testing.T) {
	// seedRosterFixture with n=0 = teacher owns section but no students.
	teacherSID, sectionID, _, _, _ := seedRosterFixture(t, "empty-section@test.local", 0)
	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	resp, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionID,
			PageSize:  20,
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher (empty section): %v", err)
	}
	if len(resp.Msg.GetSectionEnrollments()) != 0 {
		t.Errorf("got %d rows, want 0", len(resp.Msg.GetSectionEnrollments()))
	}
}

// --- S-09: Withdrawn enrollment appears in roster and is flagged ---

// TestListSectionRosterForTeacher_WithdrawnEnrollmentPresent verifies that withdrawn
// enrollments are included in the roster with status="withdrawn".
func TestListSectionRosterForTeacher_WithdrawnEnrollmentPresent(t *testing.T) {
	// Seed 3 in_progress + 1 withdrawn.
	teacherSID, sectionID, seIDs, _, _ := seedRosterFixture(t, "withdrawn@test.local", 3)
	ctx := context.Background()

	// Insert one withdrawn section_enrollment manually.
	var programID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO programs (code, name) VALUES ($1, $2) RETURNING id`,
		"RWPROG-"+uniqueSuffix(t), "Roster Withdrawn Prog",
	).Scan(&programID); err != nil {
		t.Fatalf("insert program: %v", err)
	}
	t.Cleanup(func() { _, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, programID) })

	withdrawnStudentID, _ := seedUserWithSession(t, "roster-withdrawn-student-"+uuid.New().String()[:8]+"@test.local", "student")
	// enrollments.student_id references student_profiles(user_id).
	seedStudentProfile(t, withdrawnStudentID, 4001)

	var enrollID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO enrollments (student_id, program_id, year, status) VALUES ($1, $2, $3, 'paid') RETURNING id`,
		withdrawnStudentID, programID, 4001,
	).Scan(&enrollID); err != nil {
		t.Fatalf("insert enrollment: %v", err)
	}
	t.Cleanup(func() { _, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, enrollID) })

	var withdrawnSEID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO section_enrollments (enrollment_id, section_id, status) VALUES ($1, $2, 'withdrawn') RETURNING id`,
		enrollID, sectionID,
	).Scan(&withdrawnSEID); err != nil {
		t.Fatalf("insert withdrawn section_enrollment: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, withdrawnSEID)
	})

	client := newSectionEnrollmentClient(nil)

	resp, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionID,
			PageSize:  20,
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher (withdrawn): %v", err)
	}

	rows := resp.Msg.GetSectionEnrollments()
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4 (3 in_progress + 1 withdrawn)", len(rows))
	}

	// Confirm the withdrawn row is present with correct status.
	var foundWithdrawn bool
	for _, r := range rows {
		if r.GetId() == withdrawnSEID.String() {
			if r.GetStatus() != "withdrawn" {
				t.Errorf("withdrawn row status = %q, want withdrawn", r.GetStatus())
			}
			foundWithdrawn = true
		}
	}
	if !foundWithdrawn {
		t.Errorf("withdrawn section_enrollment %s not found in roster", withdrawnSEID)
	}
	_ = seIDs
}

// --- S-10: Roster pagination boundary ---

// TestListSectionRosterForTeacher_PaginationBoundary verifies that 45 enrollments paginate
// correctly across two pages.
func TestListSectionRosterForTeacher_PaginationBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping pagination boundary test in short mode")
	}
	teacherSID, sectionID, seIDs, _, _ := seedRosterFixture(t, "paginate@test.local", 45)
	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	// First page.
	resp1, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionID,
			PageSize:  20,
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(resp1.Msg.GetSectionEnrollments()) != 20 {
		t.Fatalf("page 1: got %d rows, want 20", len(resp1.Msg.GetSectionEnrollments()))
	}
	token := resp1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token is empty, expected a token")
	}

	// Second page.
	resp2, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionID,
			PageSize:  25,
			PageToken: token,
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(resp2.Msg.GetSectionEnrollments()) != 25 {
		t.Fatalf("page 2: got %d rows, want 25", len(resp2.Msg.GetSectionEnrollments()))
	}
	if resp2.Msg.GetNextPageToken() != "" {
		t.Errorf("page 2: next_page_token = %q, want empty", resp2.Msg.GetNextPageToken())
	}

	// Verify union = all 45, no duplicates.
	allIDs := make(map[string]bool)
	for _, r := range resp1.Msg.GetSectionEnrollments() {
		allIDs[r.GetId()] = true
	}
	for _, r := range resp2.Msg.GetSectionEnrollments() {
		if allIDs[r.GetId()] {
			t.Errorf("duplicate section_enrollment %s across pages", r.GetId())
		}
		allIDs[r.GetId()] = true
	}
	if len(allIDs) != 45 {
		t.Errorf("union: %d unique rows, want 45", len(allIDs))
	}
	for _, id := range seIDs {
		if !allIDs[id] {
			t.Errorf("seeded section_enrollment %s missing from combined pages", id)
		}
	}
}

// --- S-15: page_size clamping (above maximum) ---

// TestListSectionRosterForTeacher_PageSizeClamped verifies that page_size=500 (above max 200)
// is clamped to 200 — the RPC still succeeds.
func TestListSectionRosterForTeacher_PageSizeClamped(t *testing.T) {
	teacherSID, sectionID, _, _, _ := seedRosterFixture(t, "clamp@test.local", 1)
	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	resp, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionID,
			PageSize:  500,
		},
	), teacherSID))
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher (clamped): %v", err)
	}
	if len(resp.Msg.GetSectionEnrollments()) > 200 {
		t.Errorf("got %d rows, want ≤200 (clamped)", len(resp.Msg.GetSectionEnrollments()))
	}
}

// --- S-16: Invalid section_id UUID ---

// TestListSectionRosterForTeacher_InvalidSectionID verifies that a malformed section_id
// returns CodeInvalidArgument.
func TestListSectionRosterForTeacher_InvalidSectionID(t *testing.T) {
	_, teacherSID := seedTeacherProfile(t, "roster-invalid-uuid@test.local")
	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	_, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: "not-a-uuid",
			PageSize:  20,
		},
	), teacherSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// --- S-18: Student cannot call ListSectionRosterForTeacher ---

// TestListSectionRosterForTeacher_StudentPermissionDenied verifies that a student
// (no section_enrollment.view_teaching) receives CodePermissionDenied.
func TestListSectionRosterForTeacher_StudentPermissionDenied(t *testing.T) {
	_, studentSID := seedUserWithSession(t, "roster-student-denied@test.local", "student")
	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	_, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: uuid.New().String(),
			PageSize:  20,
		},
	), studentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// --- S-11: Teacher cannot call ListSectionEnrollments (regression guard) ---

// TestListSectionEnrollments_TeacherPermissionDenied verifies that a teacher
// (no enrollment.manage) receives CodePermissionDenied on ListSectionEnrollments.
func TestListSectionEnrollments_TeacherPermissionDenied(t *testing.T) {
	_, teacherSID := seedTeacherProfile(t, "roster-teacher-listse-denied@test.local")
	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	_, err := client.ListSectionEnrollments(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionEnrollmentsRequest{},
	), teacherSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// --- Admin bypass integration tests (T10) ---

// TestListSectionRosterForTeacher_AdminSeesFullRoster verifies that an admin (enrollment.manage)
// calling ListSectionRosterForTeacher on a section they do NOT teach receives the full roster,
// bypassing the section_teachers EXISTS guard.
func TestListSectionRosterForTeacher_AdminSeesFullRoster(t *testing.T) {
	ctx := context.Background()

	// Seed teacher A with a section containing 3 students. adminSID is the catalog admin
	// returned by seedRosterFixture — it has enrollment.manage via the CROSS JOIN admin grant.
	_, sectionID, seIDs, adminSID, _ := seedRosterFixture(t, "admin-full-roster@test.local", 3)

	client := newSectionEnrollmentClient(nil)

	// Admin queries roster for a section they do NOT teach.
	resp, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionID,
			PageSize:  20,
		},
	), adminSID))
	if err != nil {
		t.Fatalf("ListSectionRosterForTeacher (admin full roster): %v", err)
	}

	rows := resp.Msg.GetSectionEnrollments()
	if len(rows) != 3 {
		t.Fatalf("admin bypass: got %d rows, want 3", len(rows))
	}

	gotIDs := make(map[string]bool)
	for _, r := range rows {
		gotIDs[r.GetId()] = true
	}
	for _, id := range seIDs {
		if !gotIDs[id] {
			t.Errorf("admin bypass: section_enrollment %s missing — admin should see full roster", id)
		}
	}

	// Verify fields are populated.
	for _, r := range rows {
		if r.GetStudentId() == "" {
			t.Errorf("admin bypass: row %s has empty student_id", r.GetId())
		}
		if r.GetSectionId() != sectionID {
			t.Errorf("admin bypass: row %s section_id = %q, want %q", r.GetId(), r.GetSectionId(), sectionID)
		}
	}
}

// TestListSectionRosterForTeacher_TeacherSeesMembershipScoped verifies that a teacher
// calling ListSectionRosterForTeacher on their OWN section receives only their scoped
// roster (not all section_enrollments in the system) even when another section exists.
func TestListSectionRosterForTeacher_TeacherSeesMembershipScoped(t *testing.T) {
	ctx := context.Background()

	// Seed teacher A with 2 enrollments.
	teacherASID, sectionIDA, seIDsA, _, _ := seedRosterFixture(t, "teacher-scoped-a@test.local", 2)

	// Seed teacher B with 3 enrollments in a different section — teacher A must not see these.
	_, sectionIDB, _, _, _ := seedRosterFixture(t, "teacher-scoped-b@test.local", 3)

	client := newSectionEnrollmentClient(nil)

	// Teacher A queries their own section.
	resp, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionIDA,
			PageSize:  20,
		},
	), teacherASID))
	if err != nil {
		t.Fatalf("teacher scoped: %v", err)
	}

	rows := resp.Msg.GetSectionEnrollments()
	if len(rows) != len(seIDsA) {
		t.Fatalf("teacher scoped: got %d rows, want %d", len(rows), len(seIDsA))
	}

	// Teacher A must not see any row from section B.
	// We verify by confirming all returned section_ids belong to section A.
	for _, r := range rows {
		if r.GetSectionId() != sectionIDA {
			t.Errorf("teacher scoped: row %s has section_id %q, expected %q (own section)",
				r.GetId(), r.GetSectionId(), sectionIDA)
		}
	}
	_ = sectionIDB
}

// TestListSectionRosterForTeacher_TeacherEmptyNoLeak verifies that a teacher who does NOT
// teach the requested section receives an empty page with OK — no PermissionDenied, no
// existence disclosure.
func TestListSectionRosterForTeacher_TeacherEmptyNoLeak(t *testing.T) {
	// Create a fresh teacher with no section_teachers rows.
	_, outTeacherSID := seedTeacherProfile(t, "roster-noleak-out@test.local")

	// Seed another teacher's section with 2 enrollments (the target).
	_, targetSectionID, _, _, _ := seedRosterFixture(t, "roster-noleak-owner@test.local", 2)

	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	resp, err := client.ListSectionRosterForTeacher(ctx, withSID(connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: targetSectionID,
			PageSize:  20,
		},
	), outTeacherSID))
	if err != nil {
		t.Fatalf("teacher no-leak: expected OK, got %v", err)
	}
	if len(resp.Msg.GetSectionEnrollments()) != 0 {
		t.Errorf("teacher no-leak: got %d rows, want 0 (anti-leak)",
			len(resp.Msg.GetSectionEnrollments()))
	}
	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("teacher no-leak: next_page_token = %q, want empty", resp.Msg.GetNextPageToken())
	}
}

// TestListSectionRosterForTeacher_UnauthenticatedReturnsError verifies that a request
// with no session cookie is rejected with CodeUnauthenticated.
func TestListSectionRosterForTeacher_UnauthenticatedReturnsError(t *testing.T) {
	// Seed a section so we have a real section_id (not a random UUID that might cause
	// a different code path).
	_, sectionID, _, _, _ := seedRosterFixture(t, "roster-unauth@test.local", 1)

	client := newSectionEnrollmentClient(nil)
	ctx := context.Background()

	// No session cookie — request without withSID.
	_, err := client.ListSectionRosterForTeacher(ctx, connect.NewRequest(
		&section_enrollmentv1.ListSectionRosterForTeacherRequest{
			SectionId: sectionID,
			PageSize:  20,
		},
	))
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}
