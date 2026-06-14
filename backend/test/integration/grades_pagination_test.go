package integration_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	gradesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1"
)

// seedGradesPaginationFixture builds a self-contained fixture for ListGradesForSection
// pagination tests:
//   - one course + academic period + section
//   - one teacher assigned to that section
//   - one evaluation scheme (single weight "1.0")
//   - `count` students, each with a unique paid enrollment (via seedSEBundle) + one grade
//
// Returns (sectionID, teacherSID, adminSID, gradeIDs, cleanup).
func seedGradesPaginationFixture(t *testing.T, suffix string, count int) (
	sectionID, teacherSID, adminSID string,
	gradeIDs []string,
	cleanup func(),
) {
	t.Helper()
	ctx := context.Background()

	_, adminSID = seedGradesAdminSID(t, "gpag-"+suffix)
	_, courseID, cleanupCatalog := seedProgramWithCourse(t)
	periodID, year, cleanupPeriod := seedAcademicPeriodWithWindow(t, true, false)
	sectionID, cleanupSection := seedSection(t, courseID, periodID, int32(count+5))

	// One teacher for the section.
	_, teacherSID = gradesSeedTeacherWithSession(t, "gpag-"+suffix, sectionID)

	// Create evaluation scheme with a single weight.
	evals := seedEvaluationScheme(t, courseID, []string{"1.0"}, adminSID)

	gradeIDs = make([]string, 0, count)
	for i := 0; i < count; i++ {
		email := fmt.Sprintf("gpag-%s-s%02d@gpag.test", suffix, i)
		studentID := seedUserWithRole(t, email, "student")

		// Ensure student_profiles row.
		if _, err := pgxPool.Exec(ctx,
			`INSERT INTO student_profiles (user_id, admission_year)
			 VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
			studentID, year,
		); err != nil {
			t.Fatalf("seedGradesPaginationFixture: ensure student_profiles: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, studentID)
		})

		_, seID := seedSEBundle(t, studentID, year, courseID, sectionID)

		// Record a grade for each student's SE.
		g := seedGrade(t, evals[0].GetId(), seID, "5.0", teacherSID)
		gradeIDs = append(gradeIDs, g.GetId())
	}

	cleanup = func() {
		cleanupSection()
		cleanupPeriod()
		cleanupCatalog()
	}
	return sectionID, teacherSID, adminSID, gradeIDs, cleanup
}

// seedOwnGradesFixture builds a fixture for ListOwnGrades pagination tests:
// one student, multiple section_enrollments (via unique programs per seedSEBundle),
// one grade per SE.
// Returns (studentSID, gradeIDs, cleanup).
func seedOwnGradesFixture(t *testing.T, suffix string, count int) (
	studentID uuid.UUID,
	studentSID string,
	gradeIDs []string,
	cleanup func(),
) {
	t.Helper()
	ctx := context.Background()

	studentID, studentSID = seedUserWithSession(t, fmt.Sprintf("gpag-own-%s@gpag.test", suffix), "student")
	_, adminSID := seedGradesAdminSID(t, "gpag-own-admin-"+suffix)

	_, courseID, cleanupCatalog := seedProgramWithCourse(t)
	periodID, year, cleanupPeriod := seedAcademicPeriodWithWindow(t, true, false)
	sectionID, cleanupSection := seedSection(t, courseID, periodID, int32(count+5))

	// One teacher for grading.
	_, teacherSID := gradesSeedTeacherWithSession(t, "gpag-own-teacher-"+suffix, sectionID)

	// Ensure student_profiles row.
	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO student_profiles (user_id, admission_year) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
		studentID, year,
	); err != nil {
		t.Fatalf("seedOwnGradesFixture: ensure student_profiles: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, studentID)
	})

	// One evaluation scheme per course — reused across all SEs.
	_ = adminSID
	evals := seedEvaluationScheme(t, courseID, []string{"1.0"}, adminSID)

	gradeIDs = make([]string, 0, count)
	for i := 0; i < count; i++ {
		_, seID := seedSEBundle(t, studentID, year, courseID, sectionID)
		g := seedGrade(t, evals[0].GetId(), seID, "5.0", teacherSID)
		gradeIDs = append(gradeIDs, g.GetId())
	}

	cleanup = func() {
		cleanupSection()
		cleanupPeriod()
		cleanupCatalog()
	}
	return studentID, studentSID, gradeIDs, cleanup
}

// --- ListGradesForSection pagination ---

// TestGrades_ListGradesForSection_FirstPage verifies the first page returns page_size
// items and a non-empty next_page_token when more grades exist.
func TestGrades_ListGradesForSection_FirstPage(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, _, adminSID, _, cleanup := seedGradesPaginationFixture(t, suffix, 25)
	defer cleanup()

	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: sectionID,
		PageSize:  20,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListGradesForSection(ctx, req)
	if err != nil {
		t.Fatalf("ListGradesForSection: %v", err)
	}
	if len(resp.Msg.GetGrades()) != 20 {
		t.Errorf("got %d grades, want 20", len(resp.Msg.GetGrades()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty when more pages exist")
	}
}

// TestGrades_ListGradesForSection_SecondPage verifies no overlap or gap across two pages.
func TestGrades_ListGradesForSection_SecondPage(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, _, adminSID, gradeIDs, cleanup := seedGradesPaginationFixture(t, suffix, 25)
	defer cleanup()

	seededIDs := make(map[string]struct{}, len(gradeIDs))
	for _, id := range gradeIDs {
		seededIDs[id] = struct{}{}
	}

	client := newGradesClient(nil)

	// Page 1.
	req1 := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: sectionID,
		PageSize:  20,
	})
	req1.Header().Set("Cookie", "sid="+adminSID)
	p1, err := client.ListGradesForSection(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty")
	}

	page1IDs := make(map[string]struct{})
	for _, g := range p1.Msg.GetGrades() {
		page1IDs[g.GetId()] = struct{}{}
	}

	// Page 2.
	req2 := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: sectionID,
		PageSize:  20,
		PageToken: token,
	})
	req2.Header().Set("Cookie", "sid="+adminSID)
	p2, err := client.ListGradesForSection(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	// No duplicates.
	for _, g := range p2.Msg.GetGrades() {
		if _, dup := page1IDs[g.GetId()]; dup {
			t.Errorf("duplicate grade_id %s across pages", g.GetId())
		}
	}

	// All seeded grades appear across both pages.
	allReturned := make(map[string]struct{})
	for _, g := range p1.Msg.GetGrades() {
		allReturned[g.GetId()] = struct{}{}
	}
	for _, g := range p2.Msg.GetGrades() {
		allReturned[g.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded grade_id %s missing from paginated results", id)
		}
	}
}

// TestGrades_ListGradesForSection_LastPageEmptyToken seeds a small set of grades and
// walks to exhaustion, verifying the final page has an empty next_page_token.
func TestGrades_ListGradesForSection_LastPageEmptyToken(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	// Seed 3 grades — all fit on one page of size 20.
	sectionID, _, adminSID, _, cleanup := seedGradesPaginationFixture(t, "gpag-lp-"+suffix, 3)
	defer cleanup()

	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: sectionID,
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListGradesForSection(ctx, req)
	if err != nil {
		t.Fatalf("ListGradesForSection walk: %v", err)
	}
	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("next_page_token should be empty on last page when only 3 grades seeded, got %q", resp.Msg.GetNextPageToken())
	}
}

// TestGrades_ListGradesForSection_ClampZero verifies page_size=0 is clamped to 20.
func TestGrades_ListGradesForSection_ClampZero(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, _, adminSID, _, cleanup := seedGradesPaginationFixture(t, suffix, 25)
	defer cleanup()

	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: sectionID,
		PageSize:  0,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListGradesForSection(ctx, req)
	if err != nil {
		t.Fatalf("ListGradesForSection: %v", err)
	}
	if len(resp.Msg.GetGrades()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetGrades()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty (more pages exist)")
	}
}

// TestGrades_ListGradesForSection_InvalidToken verifies a malformed page_token returns
// CodeInvalidArgument.
func TestGrades_ListGradesForSection_InvalidToken(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "gpag-inv-admin")
	client := newGradesClient(nil)

	req := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		PageToken: "not-a-uuid",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListGradesForSection(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestGrades_ListGradesForSection_SectionFilterPreserved verifies the section_id filter
// still applies alongside pagination — a teacher for section1 should not see section2 grades.
func TestGrades_ListGradesForSection_SectionFilterPreserved(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]

	// Section 1 with 3 grades.
	sectionID1, teacher1SID, _, gradeIDs1, cleanup1 := seedGradesPaginationFixture(t, suffix+"-s1", 3)
	defer cleanup1()

	// Section 2 with 1 grade — teacher1 is NOT assigned here.
	suffix2 := uuid.New().String()[:8]
	_, _, adminSID2, _, cleanup2 := seedGradesPaginationFixture(t, suffix2+"-s2", 1)
	defer cleanup2()
	_ = adminSID2

	// Teacher1 lists section1 → should see section1 grades only.
	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: sectionID1,
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+teacher1SID)
	resp, err := client.ListGradesForSection(ctx, req)
	if err != nil {
		t.Fatalf("ListGradesForSection(section_id): %v", err)
	}

	foundFirst := false
	for _, g := range resp.Msg.GetGrades() {
		if g.GetId() == gradeIDs1[0] {
			foundFirst = true
		}
	}
	if !foundFirst {
		t.Errorf("seeded grade %s from section1 missing from result", gradeIDs1[0])
	}
}

// TestGrades_ListGradesForSection_TeacherScopePreserved verifies that a teacher who is
// NOT assigned to a section gets an empty list (anti-leak: section_teachers membership).
func TestGrades_ListGradesForSection_TeacherScopePreserved(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]

	// Fixture for section1 (teacher1 assigned).
	sectionID1, _, _, _, cleanup1 := seedGradesPaginationFixture(t, suffix+"-scope-s1", 3)
	defer cleanup1()

	// A second teacher who is NOT assigned to section1.
	_, teacher2SID := gradesSeedTeacherWithSession(t, "gpag-scope-t2-"+suffix, "" /* no section */)

	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: sectionID1,
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+teacher2SID)
	resp, err := client.ListGradesForSection(ctx, req)
	if err != nil {
		t.Fatalf("ListGradesForSection out-of-scope teacher: %v", err)
	}
	// Teacher2 is not in section_teachers for section1 → empty list (not PermissionDenied).
	if len(resp.Msg.GetGrades()) != 0 {
		t.Errorf("out-of-scope teacher should get 0 grades, got %d", len(resp.Msg.GetGrades()))
	}
}

// TestGrades_ListGradesForSection_IDDescOrder verifies results are ordered by g.id DESC.
func TestGrades_ListGradesForSection_IDDescOrder(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, _, adminSID, _, cleanup := seedGradesPaginationFixture(t, suffix, 5)
	defer cleanup()

	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListGradesForSectionRequest{
		SectionId: sectionID,
		PageSize:  20,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListGradesForSection(ctx, req)
	if err != nil {
		t.Fatalf("ListGradesForSection: %v", err)
	}
	grades := resp.Msg.GetGrades()
	for i := 1; i < len(grades); i++ {
		if grades[i-1].GetId() <= grades[i].GetId() {
			t.Errorf("grades[%d].id=%s >= grades[%d].id=%s (want DESC order)",
				i-1, grades[i-1].GetId(), i, grades[i].GetId())
		}
	}
}

// --- ListOwnGrades pagination ---

// TestGrades_ListOwnGrades_SelfScope verifies that ListOwnGrades returns only the
// caller's own grades (another student's grades are not visible).
func TestGrades_ListOwnGrades_SelfScope(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	_, studentSID, ownGradeIDs, cleanup := seedOwnGradesFixture(t, suffix, 3)
	defer cleanup()

	// A second student — own fixture with different grades.
	suffix2 := uuid.New().String()[:8]
	_, _, otherGradeIDs, cleanupOther := seedOwnGradesFixture(t, suffix2, 1)
	defer cleanupOther()
	decoyGradeID := otherGradeIDs[0]

	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListOwnGradesRequest{
		PageSize: 200,
	})
	req.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.ListOwnGrades(ctx, req)
	if err != nil {
		t.Fatalf("ListOwnGrades: %v", err)
	}

	// Decoy grade must never appear.
	for _, g := range resp.Msg.GetGrades() {
		if g.GetId() == decoyGradeID {
			t.Errorf("decoy grade %s for other student leaked", decoyGradeID)
		}
	}

	// All own grades must appear.
	returned := make(map[string]struct{})
	for _, g := range resp.Msg.GetGrades() {
		returned[g.GetId()] = struct{}{}
	}
	for _, id := range ownGradeIDs {
		if _, ok := returned[id]; !ok {
			t.Errorf("own grade %s missing from ListOwnGrades result", id)
		}
	}
}

// TestGrades_ListOwnGrades_Pagination verifies walking pages returns all own grades
// without overlap or gap.
func TestGrades_ListOwnGrades_Pagination(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	_, studentSID, gradeIDs, cleanup := seedOwnGradesFixture(t, suffix, 25)
	defer cleanup()

	seededIDs := make(map[string]struct{}, len(gradeIDs))
	for _, id := range gradeIDs {
		seededIDs[id] = struct{}{}
	}

	client := newGradesClient(nil)

	// Page 1.
	req1 := connect.NewRequest(&gradesv1.ListOwnGradesRequest{PageSize: 20})
	req1.Header().Set("Cookie", "sid="+studentSID)
	p1, err := client.ListOwnGrades(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty (25 grades seeded)")
	}

	page1IDs := make(map[string]struct{})
	for _, g := range p1.Msg.GetGrades() {
		page1IDs[g.GetId()] = struct{}{}
	}

	// Page 2.
	req2 := connect.NewRequest(&gradesv1.ListOwnGradesRequest{
		PageSize:  20,
		PageToken: token,
	})
	req2.Header().Set("Cookie", "sid="+studentSID)
	p2, err := client.ListOwnGrades(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	// No duplicates.
	for _, g := range p2.Msg.GetGrades() {
		if _, dup := page1IDs[g.GetId()]; dup {
			t.Errorf("duplicate grade_id %s across pages", g.GetId())
		}
	}

	// All seeded grades appear.
	allReturned := make(map[string]struct{})
	for _, g := range p1.Msg.GetGrades() {
		allReturned[g.GetId()] = struct{}{}
	}
	for _, g := range p2.Msg.GetGrades() {
		allReturned[g.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded grade %s missing from paginated ListOwnGrades", id)
		}
	}
}

// TestGrades_ListOwnGrades_ClampZero verifies page_size=0 is clamped to 20.
func TestGrades_ListOwnGrades_ClampZero(t *testing.T) {
	suffix := uuid.New().String()[:8]
	_, studentSID, _, cleanup := seedOwnGradesFixture(t, suffix, 25)
	defer cleanup()

	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListOwnGradesRequest{PageSize: 0})
	req.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.ListOwnGrades(context.Background(), req)
	if err != nil {
		t.Fatalf("ListOwnGrades: %v", err)
	}
	if len(resp.Msg.GetGrades()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetGrades()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty (more pages exist)")
	}
}

// TestGrades_ListOwnGrades_InvalidToken verifies a malformed page_token returns
// CodeInvalidArgument.
func TestGrades_ListOwnGrades_InvalidToken(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	_, studentSID := seedUserWithSession(t, fmt.Sprintf("gpag-owninv-%s@gpag.test", suffix), "student")
	client := newGradesClient(nil)

	req := connect.NewRequest(&gradesv1.ListOwnGradesRequest{
		PageToken: "not-a-uuid",
	})
	req.Header().Set("Cookie", "sid="+studentSID)
	_, err := client.ListOwnGrades(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestGrades_ListOwnGrades_IDDescOrder verifies results are ordered by g.id DESC.
func TestGrades_ListOwnGrades_IDDescOrder(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	_, studentSID, _, cleanup := seedOwnGradesFixture(t, suffix, 5)
	defer cleanup()

	client := newGradesClient(nil)
	req := connect.NewRequest(&gradesv1.ListOwnGradesRequest{PageSize: 20})
	req.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.ListOwnGrades(ctx, req)
	if err != nil {
		t.Fatalf("ListOwnGrades: %v", err)
	}
	grades := resp.Msg.GetGrades()
	for i := 1; i < len(grades); i++ {
		if grades[i-1].GetId() <= grades[i].GetId() {
			t.Errorf("grades[%d].id=%s >= grades[%d].id=%s (want DESC order)",
				i-1, grades[i-1].GetId(), i, grades[i].GetId())
		}
	}
}
