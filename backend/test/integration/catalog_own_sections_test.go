package integration_test

import (
	"context"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1/catalogv1connect"
)

// ownSectionsYearCounter generates collision-free year values for academic period seeds
// in catalog_own_sections_test.go. Starts at 10000 to avoid overlap with all other test
// helpers (which use years ≤ 9000).
var ownSectionsYearCounter atomic.Int32

func init() {
	ownSectionsYearCounter.Store(10000)
}

// newCatalogClientNoJar returns a catalog client with no cookie jar (manual cookie header).
// Alias to the existing newCatalogClient helper in catalog_authz_test.go.
func newCatalogClientPlain() catalogv1connect.CatalogServiceClient {
	return newCatalogClient(nil)
}

// seedTeacherWithSections creates a teacher and N sections assigned to that teacher.
// Returns (teacherID, teacherSID, []sectionID, adminSID, cleanup func).
func seedTeacherWithSections(t *testing.T, email string, n int) (teacherID, teacherSID string, sectionIDs []string, adminSID string, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	adminSID = catalogSeedAdminSession(t, "own-sections-admin-"+email)
	teacherID, teacherSID = seedTeacherProfile(t, email)

	client := newCatalogClient(nil)

	for i := 0; i < n; i++ {
		// Create course
		cResp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
			Code:    "OWN-CRS-" + uuid.New().String()[:8],
			Name:    "Own Section Course",
			Credits: 3,
		}), adminSID))
		if err != nil {
			t.Fatalf("seedTeacherWithSections: CreateCourse %d: %v", i, err)
		}
		courseID := cResp.Msg.GetId()

		// Create academic period with a collision-free year derived from the atomic counter.
		// Each call increments the global counter, guaranteeing uniqueness across concurrent tests.
		year := ownSectionsYearCounter.Add(1)
		pResp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
			Year:      year,
			Term:      1,
			StartDate: "9000-03-01",
			EndDate:   "9000-07-31",
		}), adminSID))
		if err != nil {
			t.Fatalf("seedTeacherWithSections: CreateAcademicPeriod %d: %v", i, err)
		}
		periodID := pResp.Msg.GetId()

		// Create section
		sResp, err := client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
			CourseId:         courseID,
			AcademicPeriodId: periodID,
			SeatCapacity:     30,
		}), adminSID))
		if err != nil {
			t.Fatalf("seedTeacherWithSections: CreateSection %d: %v", i, err)
		}
		sectionID := sResp.Msg.GetId()
		sectionIDs = append(sectionIDs, sectionID)

		// Assign teacher to section
		_, err = client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
			SectionId: sectionID,
			TeacherId: teacherID,
		}), adminSID))
		if err != nil {
			t.Fatalf("seedTeacherWithSections: AssignTeacherToSection %d: %v", i, err)
		}

		// Register per-resource cleanup
		cid, pid, sid := courseID, periodID, sectionID
		t.Cleanup(func() {
			c := context.Background()
			_, _ = pgxPool.Exec(c, `DELETE FROM section_teachers WHERE section_id = $1`, sid)
			_, _ = pgxPool.Exec(c, `DELETE FROM sections WHERE id = $1`, sid)
			_, _ = pgxPool.Exec(c, `DELETE FROM academic_periods WHERE id = $1`, pid)
			_, _ = pgxPool.Exec(c, `DELETE FROM courses WHERE id = $1`, cid)
		})
	}

	cleanup = func() {} // individual cleanups registered above
	return teacherID, teacherSID, sectionIDs, adminSID, cleanup
}

// --- Cross-teacher isolation (negative test) ---

// TestListOwnSections_TeacherCannotSeeOtherTeacherSections verifies that teacher A's
// ListOwnSections returns ONLY A's section IDs (none of B's), and teacher B's call
// returns ONLY B's section IDs (none of A's). Each teacher is assigned to DISTINCT sections.
func TestListOwnSections_TeacherCannotSeeOtherTeacherSections(t *testing.T) {
	_, teacherASID, sectionIDsA, _, _ := seedTeacherWithSections(t, "isolation-teacher-a@test.local", 2)
	_, teacherBSID, sectionIDsB, _, _ := seedTeacherWithSections(t, "isolation-teacher-b@test.local", 2)

	client := newCatalogClientPlain()
	ctx := context.Background()

	// Teacher A must see only A's sections.
	respA, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}), teacherASID))
	if err != nil {
		t.Fatalf("ListOwnSections (teacher A): %v", err)
	}
	gotA := make(map[string]bool)
	for _, s := range respA.Msg.GetSections() {
		gotA[s.GetId()] = true
	}
	for _, id := range sectionIDsA {
		if !gotA[id] {
			t.Errorf("teacher A: own section %s missing from response", id)
		}
	}
	for _, id := range sectionIDsB {
		if gotA[id] {
			t.Errorf("teacher A: sees teacher B's section %s — cross-teacher leak", id)
		}
	}

	// Teacher B must see only B's sections.
	respB, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}), teacherBSID))
	if err != nil {
		t.Fatalf("ListOwnSections (teacher B): %v", err)
	}
	gotB := make(map[string]bool)
	for _, s := range respB.Msg.GetSections() {
		gotB[s.GetId()] = true
	}
	for _, id := range sectionIDsB {
		if !gotB[id] {
			t.Errorf("teacher B: own section %s missing from response", id)
		}
	}
	for _, id := range sectionIDsA {
		if gotB[id] {
			t.Errorf("teacher B: sees teacher A's section %s — cross-teacher leak", id)
		}
	}
}

// --- S-01: Teacher lists own sections, happy path ---

// TestListOwnSections_TeacherSeedsThreeSections verifies a teacher with 3 section_teachers rows
// receives exactly those 3 sections, all with enriched fields, and no next_page_token.
func TestListOwnSections_TeacherSeedsThreeSections(t *testing.T) {
	_, teacherSID, sectionIDs, _, _ := seedTeacherWithSections(t, "own-sect-happy@test.local", 3)
	client := newCatalogClientPlain()
	ctx := context.Background()

	resp, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}), teacherSID))
	if err != nil {
		t.Fatalf("ListOwnSections: %v", err)
	}

	got := resp.Msg.GetSections()
	if len(got) != 3 {
		t.Fatalf("ListOwnSections: got %d sections, want 3", len(got))
	}

	// Verify all seeded sectionIDs are present.
	gotIDs := make(map[string]bool)
	for _, s := range got {
		gotIDs[s.GetId()] = true
	}
	for _, id := range sectionIDs {
		if !gotIDs[id] {
			t.Errorf("ListOwnSections: section %s missing from response", id)
		}
	}

	// Verify enriched fields are non-empty.
	for _, s := range got {
		if s.GetCourseCode() == "" {
			t.Errorf("section %s: course_code is empty", s.GetId())
		}
		if s.GetCourseName() == "" {
			t.Errorf("section %s: course_name is empty", s.GetId())
		}
		if s.GetPeriodYear() == 0 {
			t.Errorf("section %s: period_year is 0", s.GetId())
		}
		if s.GetSeatCapacity() == 0 {
			t.Errorf("section %s: seat_capacity is 0", s.GetId())
		}
	}

	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("ListOwnSections: next_page_token = %q, want empty", resp.Msg.GetNextPageToken())
	}
}

// --- S-02: Teacher with zero sections ---

// TestListOwnSections_TeacherWithNoSections verifies that a teacher with no section_teachers
// rows receives an empty page and OK (not an error).
func TestListOwnSections_TeacherWithNoSections(t *testing.T) {
	_, teacherSID := seedTeacherProfile(t, "own-sect-empty@test.local")
	client := newCatalogClientPlain()
	ctx := context.Background()

	resp, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}), teacherSID))
	if err != nil {
		t.Fatalf("ListOwnSections (no sections): %v", err)
	}
	if len(resp.Msg.GetSections()) != 0 {
		t.Errorf("ListOwnSections (no sections): got %d sections, want 0", len(resp.Msg.GetSections()))
	}
	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("ListOwnSections (no sections): next_page_token = %q, want empty", resp.Msg.GetNextPageToken())
	}
}

// --- S-03: Admin gets empty result (no section_teachers rows) ---

// TestListOwnSections_AdminNoTeachingRows verifies that an admin (who holds section.view_teaching
// via the seed grant) with no section_teachers rows receives an empty page and OK.
func TestListOwnSections_AdminNoTeachingRows(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "own-sect-admin@test.local", "admin")
	client := newCatalogClientPlain()
	ctx := context.Background()

	resp, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}), adminSID))
	if err != nil {
		t.Fatalf("ListOwnSections (admin): %v", err)
	}
	if len(resp.Msg.GetSections()) != 0 {
		t.Errorf("ListOwnSections (admin): got %d sections, want 0", len(resp.Msg.GetSections()))
	}
}

// --- S-14: page_size clamping (below minimum) ---

// TestListOwnSections_PageSizeClamped verifies that page_size=5 (below minimum 20)
// is clamped to 20 — the RPC still succeeds.
func TestListOwnSections_PageSizeClamped(t *testing.T) {
	_, teacherSID, _, _, _ := seedTeacherWithSections(t, "own-sect-clamp@test.local", 1)
	client := newCatalogClientPlain()
	ctx := context.Background()

	resp, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 5, // below minimum 20 — must be clamped
	}), teacherSID))
	if err != nil {
		t.Fatalf("ListOwnSections (clamped page_size): %v", err)
	}
	// Confirm at most 20 rows (in practice 1 here — just confirm no error)
	if len(resp.Msg.GetSections()) > 20 {
		t.Errorf("ListOwnSections (clamped): got %d rows, want ≤20", len(resp.Msg.GetSections()))
	}
}

// --- S-04: Pagination boundary (45 sections) ---

// TestListOwnSections_PaginationBoundary verifies that 45 sections paginate correctly across two pages.
func TestListOwnSections_PaginationBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping pagination boundary test in short mode")
	}
	_, teacherSID, sectionIDs, adminSID, _ := seedTeacherWithSections(t, "own-sect-paginate@test.local", 45)
	client := newCatalogClientPlain()
	ctx := context.Background()

	// First page: page_size=20
	resp1, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}), teacherSID))
	if err != nil {
		t.Fatalf("ListOwnSections page 1: %v", err)
	}
	if len(resp1.Msg.GetSections()) != 20 {
		t.Fatalf("page 1: got %d sections, want 20", len(resp1.Msg.GetSections()))
	}
	token := resp1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token is empty, expected a token")
	}

	// Second page: page_size=25 with token — fetches all 25 remaining sections in one request.
	resp2, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize:  25,
		PageToken: token,
	}), teacherSID))
	if err != nil {
		t.Fatalf("ListOwnSections page 2: %v", err)
	}
	if len(resp2.Msg.GetSections()) != 25 {
		t.Fatalf("page 2: got %d sections, want 25", len(resp2.Msg.GetSections()))
	}
	if resp2.Msg.GetNextPageToken() != "" {
		t.Errorf("page 2: next_page_token = %q, want empty", resp2.Msg.GetNextPageToken())
	}

	// Verify union = all 45 sections, no duplicates.
	allIDs := make(map[string]bool)
	for _, s := range resp1.Msg.GetSections() {
		allIDs[s.GetId()] = true
	}
	for _, s := range resp2.Msg.GetSections() {
		if allIDs[s.GetId()] {
			t.Errorf("pagination: duplicate section %s across pages", s.GetId())
		}
		allIDs[s.GetId()] = true
	}
	if len(allIDs) != 45 {
		t.Errorf("pagination union: %d unique sections, want 45", len(allIDs))
	}
	for _, id := range sectionIDs {
		if !allIDs[id] {
			t.Errorf("pagination: seeded section %s missing from combined pages", id)
		}
	}
	_ = adminSID
}

// --- S-17: Unauthenticated ---

// TestListOwnSections_Unauthenticated verifies that a request with no session is rejected
// with CodeUnauthenticated.
func TestListOwnSections_Unauthenticated(t *testing.T) {
	client := newCatalogClientPlain()
	ctx := context.Background()

	// No cookie header.
	_, err := client.ListOwnSections(ctx, connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}))
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// --- S-18: Student permission denied ---

// TestListOwnSections_StudentPermissionDenied verifies that a student (no section.view_teaching)
// receives CodePermissionDenied.
func TestListOwnSections_StudentPermissionDenied(t *testing.T) {
	_, studentSID := seedUserWithSession(t, "own-sect-student@test.local", "student")
	client := newCatalogClientPlain()
	ctx := context.Background()

	_, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}), studentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// --- S-05: Teacher cannot call ListSections (regression guard) ---

// TestListSections_TeacherPermissionDenied verifies that a teacher (no catalog.manage)
// receives CodePermissionDenied on ListSections.
func TestListSections_TeacherPermissionDenied(t *testing.T) {
	_, teacherSID := seedTeacherProfile(t, "own-sect-teacher-listsect@test.local")
	client := newCatalogClientPlain()
	ctx := context.Background()

	_, err := client.ListSections(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionsRequest{}), teacherSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// --- Admin bypass integration tests ---

// TestListOwnSections_AdminSeesAll verifies that an admin (catalog.manage) calling
// ListOwnSections receives ALL sections, including sections the admin does not teach.
func TestListOwnSections_AdminSeesAll(t *testing.T) {
	ctx := context.Background()

	// Seed teacher A with 2 sections the admin does NOT teach.
	_, _, sectionIDsA, adminSID, _ := seedTeacherWithSections(t, "admin-sees-all-a@test.local", 2)

	// Seed teacher B with 1 more section the admin does NOT teach.
	_, _, sectionIDsB, _, _ := seedTeacherWithSections(t, "admin-sees-all-b@test.local", 1)

	client := newCatalogClientPlain()

	// Admin queries ListOwnSections — must return all sections (not just own).
	resp, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 200,
	}), adminSID))
	if err != nil {
		t.Fatalf("ListOwnSections (admin sees all): %v", err)
	}

	gotIDs := make(map[string]bool)
	for _, s := range resp.Msg.GetSections() {
		gotIDs[s.GetId()] = true
	}

	// All seeded sections must be present in the admin's result.
	allExpected := append(sectionIDsA, sectionIDsB...)
	for _, id := range allExpected {
		if !gotIDs[id] {
			t.Errorf("admin bypass: section %s missing — admin should see all sections", id)
		}
	}

	// Verify enriched fields are populated for admin results too.
	for _, s := range resp.Msg.GetSections() {
		if s.GetCourseCode() == "" {
			t.Errorf("admin: section %s has empty course_code", s.GetId())
		}
		if s.GetCourseName() == "" {
			t.Errorf("admin: section %s has empty course_name", s.GetId())
		}
		if s.GetPeriodYear() == 0 {
			t.Errorf("admin: section %s has zero period_year", s.GetId())
		}
	}
}

// TestListOwnSections_TeacherSeesOwn verifies that a teacher calling ListOwnSections
// receives only their own sections (not all sections) even when other sections exist.
func TestListOwnSections_TeacherSeesOwn(t *testing.T) {
	ctx := context.Background()

	// Seed teacher A with 2 sections.
	_, teacherASID, sectionIDsA, _, _ := seedTeacherWithSections(t, "teacher-sees-own-a@test.local", 2)

	// Seed teacher B with 3 OTHER sections to confirm teacher A does not see them.
	_, _, sectionIDsB, _, _ := seedTeacherWithSections(t, "teacher-sees-own-b@test.local", 3)

	client := newCatalogClientPlain()

	resp, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 200,
	}), teacherASID))
	if err != nil {
		t.Fatalf("ListOwnSections (teacher sees own): %v", err)
	}

	gotIDs := make(map[string]bool)
	for _, s := range resp.Msg.GetSections() {
		gotIDs[s.GetId()] = true
	}

	// Teacher A must see their own sections.
	for _, id := range sectionIDsA {
		if !gotIDs[id] {
			t.Errorf("teacher A: own section %s missing", id)
		}
	}

	// Teacher A must NOT see teacher B's sections.
	for _, id := range sectionIDsB {
		if gotIDs[id] {
			t.Errorf("teacher A: sees teacher B's section %s — cross-teacher leak", id)
		}
	}
}

// TestListOwnSections_TeacherEmptyNoLeak verifies that a teacher with no section_teachers rows
// returns an empty page with OK — no PermissionDenied, no existence leak.
func TestListOwnSections_TeacherEmptyNoLeak(t *testing.T) {
	_, teacherSID := seedTeacherProfile(t, "teacher-empty-no-leak@test.local")
	client := newCatalogClientPlain()
	ctx := context.Background()

	resp, err := client.ListOwnSections(ctx, withSID(connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}), teacherSID))
	if err != nil {
		t.Fatalf("ListOwnSections (teacher empty no-leak): %v", err)
	}
	if len(resp.Msg.GetSections()) != 0 {
		t.Errorf("ListOwnSections (teacher empty): got %d sections, want 0", len(resp.Msg.GetSections()))
	}
}

// TestListOwnSections_UnauthenticatedReturnsError verifies that a request with no auth header
// returns an unauthenticated error.
func TestListOwnSections_UnauthenticatedReturnsError(t *testing.T) {
	client := newCatalogClientPlain()
	ctx := context.Background()

	// No session cookie — unauthenticated.
	_, err := client.ListOwnSections(ctx, connect.NewRequest(&catalogv1.ListOwnSectionsRequest{
		PageSize: 20,
	}))
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}
