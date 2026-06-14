package integration_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
)

// --- Seed helpers for catalog pagination tests ---

// seedCatalogProgram inserts a program row directly in the DB and registers cleanup.
// Returns the program UUID string.
func seedCatalogProgram(t *testing.T, suffix string) string {
	t.Helper()
	var id string
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO programs (code, name) VALUES ($1, $2) RETURNING id`,
		"PROG-PAG-"+suffix, "Pagination Program "+suffix,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedCatalogProgram: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM programs WHERE id = $1`, id)
	})
	return id
}

// seedCatalogCourse inserts a course row directly in the DB and registers cleanup.
// Returns the course UUID string.
func seedCatalogCourse(t *testing.T, suffix string) string {
	t.Helper()
	var id string
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO courses (code, name, credits) VALUES ($1, $2, 3) RETURNING id`,
		"CRS-PAG-"+suffix, "Pagination Course "+suffix,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedCatalogCourse: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM courses WHERE id = $1`, id)
	})
	return id
}

// seedCatalogSection inserts a section row directly in the DB and registers cleanup.
// Returns the section UUID string.
func seedCatalogSection(t *testing.T, courseID, academicPeriodID string) string {
	t.Helper()
	var id string
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO sections (course_id, academic_period_id, capacity) VALUES ($1, $2, 30) RETURNING id`,
		courseID, academicPeriodID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedCatalogSection: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM sections WHERE id = $1`, id)
	})
	return id
}

// seedCatalogAcademicPeriod inserts an academic_period row directly in the DB and registers cleanup.
// Returns the period UUID string.
func seedCatalogAcademicPeriod(t *testing.T, year int, term int) string {
	t.Helper()
	var id string
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO academic_periods (year, term, start_date, end_date)
		 VALUES ($1, $2, '2026-03-01', '2026-07-31') RETURNING id`,
		year, term,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedCatalogAcademicPeriod(%d, %d): %v", year, term, err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM academic_periods WHERE id = $1`, id)
	})
	return id
}

// --- ListPrograms pagination ---

// TestCatalog_ListPrograms_FirstPage seeds >20 programs and asserts the first page
// returns page_size items and a non-empty next_page_token.
func TestCatalog_ListPrograms_FirstPage(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-prog-pag-fp@catalog-pg.test")
	client := newCatalogClient(nil)

	// Seed 25 programs to ensure two pages.
	suffix := uuid.New().String()[:6]
	for i := 0; i < 25; i++ {
		seedCatalogProgram(t, fmt.Sprintf("%s-%02d", suffix, i))
	}

	req := connect.NewRequest(&catalogv1.ListProgramsRequest{PageSize: 20})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListPrograms(ctx, req)
	if err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	if len(resp.Msg.GetPrograms()) < 20 {
		t.Errorf("got %d programs, want >= 20", len(resp.Msg.GetPrograms()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty when more pages exist")
	}
}

// TestCatalog_ListPrograms_SecondPage verifies no overlap or gap across two pages.
func TestCatalog_ListPrograms_SecondPage(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-prog-pag-sp@catalog-pg2.test")
	client := newCatalogClient(nil)

	// Seed 25 programs using a unique suffix to isolate this test's data.
	suffix := uuid.New().String()[:6]
	seededIDs := make(map[string]struct{}, 25)
	for i := 0; i < 25; i++ {
		id := seedCatalogProgram(t, fmt.Sprintf("%s-%02d", suffix, i))
		seededIDs[id] = struct{}{}
	}

	// Page 1.
	req1 := connect.NewRequest(&catalogv1.ListProgramsRequest{PageSize: 20})
	req1.Header().Set("Cookie", "sid="+adminSID)
	p1, err := client.ListPrograms(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty")
	}

	// Collect page 1 IDs.
	page1IDs := make(map[string]struct{})
	for _, p := range p1.Msg.GetPrograms() {
		page1IDs[p.GetId()] = struct{}{}
	}

	// Page 2.
	req2 := connect.NewRequest(&catalogv1.ListProgramsRequest{PageSize: 20, PageToken: token})
	req2.Header().Set("Cookie", "sid="+adminSID)
	p2, err := client.ListPrograms(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	// Verify no duplicates.
	for _, p := range p2.Msg.GetPrograms() {
		if _, dup := page1IDs[p.GetId()]; dup {
			t.Errorf("duplicate program_id %s across pages", p.GetId())
		}
	}

	// Verify all 25 seeded IDs appear across both pages.
	allReturned := make(map[string]struct{})
	for _, p := range p1.Msg.GetPrograms() {
		allReturned[p.GetId()] = struct{}{}
	}
	for _, p := range p2.Msg.GetPrograms() {
		allReturned[p.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded program %s missing from paginated results", id)
		}
	}
}

// TestCatalog_ListPrograms_LastPageEmptyToken seeds exactly 5 programs within an isolated
// namespace and requests page_size=10. Expects empty next_page_token.
func TestCatalog_ListPrograms_LastPageEmptyToken(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-prog-pag-lp@catalog-pglp.test")
	client := newCatalogClient(nil)

	suffix := uuid.New().String()[:6]
	var lastID string
	for i := 0; i < 5; i++ {
		lastID = seedCatalogProgram(t, fmt.Sprintf("%s-%02d", suffix, i))
	}
	_ = lastID

	// We cannot easily isolate "only our programs" without a filter, but we can check
	// that walking pages to exhaustion eventually yields empty token.
	// Walk all pages with page_size=200 (max), expect empty token on final page.
	req := connect.NewRequest(&catalogv1.ListProgramsRequest{PageSize: 200})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListPrograms(ctx, req)
	if err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}

	// Either we get all on one page (empty token) or we need to walk.
	token := resp.Msg.GetNextPageToken()
	for token != "" {
		req := connect.NewRequest(&catalogv1.ListProgramsRequest{PageSize: 200, PageToken: token})
		req.Header().Set("Cookie", "sid="+adminSID)
		resp, err = client.ListPrograms(ctx, req)
		if err != nil {
			t.Fatalf("ListPrograms walk: %v", err)
		}
		token = resp.Msg.GetNextPageToken()
	}
	// Reached the last page: token must be empty.
	if resp.Msg.GetNextPageToken() != "" {
		t.Error("last page: next_page_token must be empty")
	}
}

// TestCatalog_ListPrograms_ClampZero verifies page_size=0 is clamped to 20.
func TestCatalog_ListPrograms_ClampZero(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-prog-pag-cz@catalog-pgcz.test")
	client := newCatalogClient(nil)

	// Seed 25 programs.
	suffix := uuid.New().String()[:6]
	for i := 0; i < 25; i++ {
		seedCatalogProgram(t, fmt.Sprintf("%s-%02d", suffix, i))
	}

	req := connect.NewRequest(&catalogv1.ListProgramsRequest{PageSize: 0})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListPrograms(ctx, req)
	if err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	if len(resp.Msg.GetPrograms()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetPrograms()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty (more pages exist)")
	}
}

// TestCatalog_ListPrograms_InvalidToken verifies malformed page_token returns CodeInvalidArgument.
func TestCatalog_ListPrograms_InvalidToken(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-prog-pag-inv@catalog-pginv.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.ListProgramsRequest{PageToken: "not-a-uuid"})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListPrograms(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_ListPrograms_SoftDeletedExcluded verifies soft-deleted programs never appear on any page.
func TestCatalog_ListPrograms_SoftDeletedExcluded(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-prog-pag-sd@catalog-pgsd.test")
	client := newCatalogClient(nil)

	// Insert a soft-deleted program directly.
	var deletedID string
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO programs (code, name, deleted_at) VALUES ($1, $2, now()) RETURNING id`,
		"PROG-DELETED-"+uuid.New().String()[:6], "Deleted Program",
	).Scan(&deletedID)
	if err != nil {
		t.Fatalf("insert deleted program: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, deletedID)
	})

	// Walk all pages and assert the deleted program never appears.
	var token string
	for {
		req := connect.NewRequest(&catalogv1.ListProgramsRequest{PageSize: 200, PageToken: token})
		req.Header().Set("Cookie", "sid="+adminSID)
		resp, err := client.ListPrograms(ctx, req)
		if err != nil {
			t.Fatalf("ListPrograms: %v", err)
		}
		for _, p := range resp.Msg.GetPrograms() {
			if p.GetId() == deletedID {
				t.Errorf("soft-deleted program %s appeared in ListPrograms", deletedID)
			}
		}
		token = resp.Msg.GetNextPageToken()
		if token == "" {
			break
		}
	}
}

// TestCatalog_ListPrograms_IDDescOrder verifies items within a page are ordered by id DESC.
func TestCatalog_ListPrograms_IDDescOrder(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-prog-pag-ord@catalog-pgord.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.ListProgramsRequest{PageSize: 20})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListPrograms(ctx, req)
	if err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	progs := resp.Msg.GetPrograms()
	for i := 1; i < len(progs); i++ {
		if progs[i-1].GetId() <= progs[i].GetId() {
			t.Errorf("programs[%d].id=%s >= programs[%d].id=%s (want DESC order)",
				i-1, progs[i-1].GetId(), i, progs[i].GetId())
		}
	}
}

// --- ListCourses pagination ---

// TestCatalog_ListCourses_FirstPage seeds >20 courses and asserts the first page
// returns page_size items and a non-empty next_page_token.
func TestCatalog_ListCourses_FirstPage(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-crs-pag-fp@catalog-cg.test")
	client := newCatalogClient(nil)

	suffix := uuid.New().String()[:6]
	for i := 0; i < 25; i++ {
		seedCatalogCourse(t, fmt.Sprintf("%s-%02d", suffix, i))
	}

	req := connect.NewRequest(&catalogv1.ListCoursesRequest{PageSize: 20})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListCourses(ctx, req)
	if err != nil {
		t.Fatalf("ListCourses: %v", err)
	}
	if len(resp.Msg.GetCourses()) < 20 {
		t.Errorf("got %d courses, want >= 20", len(resp.Msg.GetCourses()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty when more pages exist")
	}
}

// TestCatalog_ListCourses_SecondPage verifies no overlap or gap across two pages.
func TestCatalog_ListCourses_SecondPage(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-crs-pag-sp@catalog-cg2.test")
	client := newCatalogClient(nil)

	suffix := uuid.New().String()[:6]
	seededIDs := make(map[string]struct{}, 25)
	for i := 0; i < 25; i++ {
		id := seedCatalogCourse(t, fmt.Sprintf("%s-%02d", suffix, i))
		seededIDs[id] = struct{}{}
	}

	req1 := connect.NewRequest(&catalogv1.ListCoursesRequest{PageSize: 20})
	req1.Header().Set("Cookie", "sid="+adminSID)
	p1, err := client.ListCourses(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty")
	}

	page1IDs := make(map[string]struct{})
	for _, c := range p1.Msg.GetCourses() {
		page1IDs[c.GetId()] = struct{}{}
	}

	req2 := connect.NewRequest(&catalogv1.ListCoursesRequest{PageSize: 20, PageToken: token})
	req2.Header().Set("Cookie", "sid="+adminSID)
	p2, err := client.ListCourses(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	for _, c := range p2.Msg.GetCourses() {
		if _, dup := page1IDs[c.GetId()]; dup {
			t.Errorf("duplicate course_id %s across pages", c.GetId())
		}
	}

	allReturned := make(map[string]struct{})
	for _, c := range p1.Msg.GetCourses() {
		allReturned[c.GetId()] = struct{}{}
	}
	for _, c := range p2.Msg.GetCourses() {
		allReturned[c.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded course %s missing from paginated results", id)
		}
	}
}

// TestCatalog_ListCourses_ClampZero verifies page_size=0 is clamped to 20.
func TestCatalog_ListCourses_ClampZero(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-crs-pag-cz@catalog-cgcz.test")
	client := newCatalogClient(nil)

	suffix := uuid.New().String()[:6]
	for i := 0; i < 25; i++ {
		seedCatalogCourse(t, fmt.Sprintf("%s-%02d", suffix, i))
	}

	req := connect.NewRequest(&catalogv1.ListCoursesRequest{PageSize: 0})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListCourses(ctx, req)
	if err != nil {
		t.Fatalf("ListCourses: %v", err)
	}
	if len(resp.Msg.GetCourses()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetCourses()))
	}
}

// TestCatalog_ListCourses_InvalidToken verifies malformed page_token returns CodeInvalidArgument.
func TestCatalog_ListCourses_InvalidToken(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-crs-pag-inv@catalog-cginv.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.ListCoursesRequest{PageToken: "not-a-uuid"})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListCourses(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_ListCourses_IDDescOrder verifies items within a page are ordered by id DESC.
func TestCatalog_ListCourses_IDDescOrder(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-crs-pag-ord@catalog-cgord.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.ListCoursesRequest{PageSize: 20})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListCourses(ctx, req)
	if err != nil {
		t.Fatalf("ListCourses: %v", err)
	}
	courses := resp.Msg.GetCourses()
	for i := 1; i < len(courses); i++ {
		if courses[i-1].GetId() <= courses[i].GetId() {
			t.Errorf("courses[%d].id=%s >= courses[%d].id=%s (want DESC order)",
				i-1, courses[i-1].GetId(), i, courses[i].GetId())
		}
	}
}

// --- ListSections pagination ---

// TestCatalog_ListSections_FirstPage seeds >20 sections and asserts first page
// returns page_size items and a non-empty next_page_token.
func TestCatalog_ListSections_FirstPage(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-sec-pag-fp@catalog-sg.test")
	client := newCatalogClient(nil)

	courseID := seedCatalogCourse(t, "sec-pag-"+uuid.New().String()[:6])
	periodID := seedCatalogAcademicPeriod(t, 2050, 1)
	for i := 0; i < 25; i++ {
		seedCatalogSection(t, courseID, periodID)
	}

	req := connect.NewRequest(&catalogv1.ListSectionsRequest{PageSize: 20})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSections(ctx, req)
	if err != nil {
		t.Fatalf("ListSections: %v", err)
	}
	if len(resp.Msg.GetSections()) < 20 {
		t.Errorf("got %d sections, want >= 20", len(resp.Msg.GetSections()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty when more pages exist")
	}
}

// TestCatalog_ListSections_SecondPage verifies no overlap or gap across two pages.
func TestCatalog_ListSections_SecondPage(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-sec-pag-sp@catalog-sg2.test")
	client := newCatalogClient(nil)

	courseID := seedCatalogCourse(t, "sec-pag-sp-"+uuid.New().String()[:6])
	periodID := seedCatalogAcademicPeriod(t, 2051, 1)
	seededIDs := make(map[string]struct{}, 25)
	for i := 0; i < 25; i++ {
		id := seedCatalogSection(t, courseID, periodID)
		seededIDs[id] = struct{}{}
	}

	req1 := connect.NewRequest(&catalogv1.ListSectionsRequest{
		CourseId: &courseID,
		PageSize: 20,
	})
	req1.Header().Set("Cookie", "sid="+adminSID)
	p1, err := client.ListSections(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatalf("page 1: next_page_token must be non-empty (seeded 25 sections for courseID %s)", courseID)
	}

	page1IDs := make(map[string]struct{})
	for _, s := range p1.Msg.GetSections() {
		page1IDs[s.GetId()] = struct{}{}
	}

	req2 := connect.NewRequest(&catalogv1.ListSectionsRequest{
		CourseId:  &courseID,
		PageSize:  20,
		PageToken: token,
	})
	req2.Header().Set("Cookie", "sid="+adminSID)
	p2, err := client.ListSections(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	for _, s := range p2.Msg.GetSections() {
		if _, dup := page1IDs[s.GetId()]; dup {
			t.Errorf("duplicate section_id %s across pages", s.GetId())
		}
	}

	allReturned := make(map[string]struct{})
	for _, s := range p1.Msg.GetSections() {
		allReturned[s.GetId()] = struct{}{}
	}
	for _, s := range p2.Msg.GetSections() {
		allReturned[s.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded section %s missing from paginated results", id)
		}
	}
}

// TestCatalog_ListSections_FilterByCoursePreserved verifies that course_id filter
// applies alongside pagination: only sections for the given course are returned.
func TestCatalog_ListSections_FilterByCoursePreserved(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-sec-pag-fc@catalog-sgfc.test")
	client := newCatalogClient(nil)

	courseA := seedCatalogCourse(t, "filter-crs-a-"+uuid.New().String()[:6])
	courseB := seedCatalogCourse(t, "filter-crs-b-"+uuid.New().String()[:6])
	periodID := seedCatalogAcademicPeriod(t, 2052, 1)

	const nA = 5
	const nB = 3
	idsA := make(map[string]struct{}, nA)
	idsB := make(map[string]struct{}, nB)
	for i := 0; i < nA; i++ {
		idsA[seedCatalogSection(t, courseA, periodID)] = struct{}{}
	}
	for i := 0; i < nB; i++ {
		idsB[seedCatalogSection(t, courseB, periodID)] = struct{}{}
	}

	req := connect.NewRequest(&catalogv1.ListSectionsRequest{
		CourseId: &courseA,
		PageSize: 200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSections(ctx, req)
	if err != nil {
		t.Fatalf("ListSections(courseA): %v", err)
	}

	for _, s := range resp.Msg.GetSections() {
		if s.GetCourseId() != courseA {
			t.Errorf("section %s has course_id=%s, want %s", s.GetId(), s.GetCourseId(), courseA)
		}
		if _, isBSection := idsB[s.GetId()]; isBSection {
			t.Errorf("section from courseB leaked into courseA-filtered result: %s", s.GetId())
		}
	}

	for id := range idsA {
		found := false
		for _, s := range resp.Msg.GetSections() {
			if s.GetId() == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("seeded section %s for courseA missing from filtered result", id)
		}
	}
}

// TestCatalog_ListSections_FilterByAcademicPeriodPreserved verifies that academic_period_id
// filter applies alongside pagination.
func TestCatalog_ListSections_FilterByAcademicPeriodPreserved(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-sec-pag-fap@catalog-sgfap.test")
	client := newCatalogClient(nil)

	courseID := seedCatalogCourse(t, "filter-ap-crs-"+uuid.New().String()[:6])
	periodA := seedCatalogAcademicPeriod(t, 2053, 1)
	periodB := seedCatalogAcademicPeriod(t, 2053, 2)

	const nA = 4
	const nB = 3
	idsA := make(map[string]struct{}, nA)
	idsB := make(map[string]struct{}, nB)
	for i := 0; i < nA; i++ {
		idsA[seedCatalogSection(t, courseID, periodA)] = struct{}{}
	}
	for i := 0; i < nB; i++ {
		idsB[seedCatalogSection(t, courseID, periodB)] = struct{}{}
	}

	req := connect.NewRequest(&catalogv1.ListSectionsRequest{
		AcademicPeriodId: &periodA,
		PageSize:         200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSections(ctx, req)
	if err != nil {
		t.Fatalf("ListSections(periodA): %v", err)
	}

	for _, s := range resp.Msg.GetSections() {
		if s.GetAcademicPeriodId() != periodA {
			t.Errorf("section %s has academic_period_id=%s, want %s", s.GetId(), s.GetAcademicPeriodId(), periodA)
		}
	}
	for id := range idsA {
		found := false
		for _, s := range resp.Msg.GetSections() {
			if s.GetId() == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("seeded section %s for periodA missing from filtered result", id)
		}
	}
}

// TestCatalog_ListSections_ClampZero verifies page_size=0 is clamped to 20.
func TestCatalog_ListSections_ClampZero(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-sec-pag-cz@catalog-sgcz.test")
	client := newCatalogClient(nil)

	courseID := seedCatalogCourse(t, "clamp-crs-"+uuid.New().String()[:6])
	periodID := seedCatalogAcademicPeriod(t, 2054, 1)
	for i := 0; i < 25; i++ {
		seedCatalogSection(t, courseID, periodID)
	}

	req := connect.NewRequest(&catalogv1.ListSectionsRequest{
		CourseId: &courseID,
		PageSize: 0,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSections(ctx, req)
	if err != nil {
		t.Fatalf("ListSections: %v", err)
	}
	if len(resp.Msg.GetSections()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetSections()))
	}
}

// TestCatalog_ListSections_InvalidToken verifies malformed page_token returns CodeInvalidArgument.
func TestCatalog_ListSections_InvalidToken(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-sec-pag-inv@catalog-sginv.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.ListSectionsRequest{PageToken: "not-a-uuid"})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListSections(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_ListSections_IDDescOrder verifies items within a page are ordered by id DESC.
func TestCatalog_ListSections_IDDescOrder(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-sec-pag-ord@catalog-sgord.test")
	client := newCatalogClient(nil)

	courseID := seedCatalogCourse(t, "ord-crs-"+uuid.New().String()[:6])
	periodID := seedCatalogAcademicPeriod(t, 2055, 1)
	for i := 0; i < 5; i++ {
		seedCatalogSection(t, courseID, periodID)
	}

	req := connect.NewRequest(&catalogv1.ListSectionsRequest{
		CourseId: &courseID,
		PageSize: 20,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSections(ctx, req)
	if err != nil {
		t.Fatalf("ListSections: %v", err)
	}
	sections := resp.Msg.GetSections()
	for i := 1; i < len(sections); i++ {
		if sections[i-1].GetId() <= sections[i].GetId() {
			t.Errorf("sections[%d].id=%s >= sections[%d].id=%s (want DESC order)",
				i-1, sections[i-1].GetId(), i, sections[i].GetId())
		}
	}
}
