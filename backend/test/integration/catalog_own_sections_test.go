package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1/catalogv1connect"
)

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

		// Create academic period with unique year
		year := 5000 + int32(time.Now().UnixNano()%500) + int32(i)*3
		pResp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
			Year:      year,
			Term:      1,
			StartDate: "5000-03-01",
			EndDate:   "5000-07-31",
		}), adminSID))
		if err != nil {
			// Retry with different year
			pResp, err = client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
				Year:      year + 500,
				Term:      2,
				StartDate: "5500-08-01",
				EndDate:   "5500-12-31",
			}), adminSID))
			if err != nil {
				t.Fatalf("seedTeacherWithSections: CreateAcademicPeriod %d: %v", i, err)
			}
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
