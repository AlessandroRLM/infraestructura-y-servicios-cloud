package integration_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
)

// seedSectionEnrollmentDirect inserts a section_enrollment row directly into the DB,
// bypassing the service layer. The caller must have already established a paid enrollment
// for (enrollmentID, sectionID) before calling this. The unique constraint on
// (enrollment_id, section_id) means each pair may only be inserted once.
// Returns the section_enrollment id string and registers a cleanup.
func seedSectionEnrollmentDirect(t *testing.T, enrollmentID, sectionID string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := pgxPool.QueryRow(ctx,
		`INSERT INTO section_enrollments (enrollment_id, section_id, status)
		 VALUES ($1, $2, 'in_progress') RETURNING id`,
		enrollmentID, sectionID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedSectionEnrollmentDirect: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, id)
	})
	return id
}

// seedSEBundle creates a self-contained enrollment + section_enrollment bundle for a student.
// It allocates a fresh program, links the given courseID and periodID to that program, creates
// a paid enrollment for the student, and inserts a section_enrollment row for the given section.
// This gives each student one enrollment per program, satisfying the unique(student_id, program_id, year)
// constraint even when called many times for the same student.
// Returns the (enrollmentID, seID) pair and registers all cleanups.
func seedSEBundle(t *testing.T, studentID uuid.UUID, year int32, courseID, sectionID string) (enrollmentID, seID string) {
	t.Helper()
	ctx := context.Background()

	// Fresh program per bundle to avoid the unique(student_id, program_id, year) constraint.
	var programID string
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO programs (code, name) VALUES ($1, $2) RETURNING id`,
		"SE-PAG-PROG-"+uniqueSuffix(t), "SE Pag Program",
	).Scan(&programID); err != nil {
		t.Fatalf("seedSEBundle: insert program: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, programID)
	})

	// Link course to program.
	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO program_courses (program_id, course_id) VALUES ($1, $2)`,
		programID, courseID,
	); err != nil {
		t.Fatalf("seedSEBundle: link course: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM program_courses WHERE program_id = $1 AND course_id = $2`, programID, courseID)
	})

	enrollmentID, cleanupEnroll := seedPaidEnrollment(t, studentID.String(), programID, year)
	_ = cleanupEnroll

	seID = seedSectionEnrollmentDirect(t, enrollmentID, sectionID)
	return enrollmentID, seID
}

// setupSEPaginationFixture builds a complete fixture for ListSectionEnrollments pagination:
//   - one course + academic_period (open window) + section (capacity >= count)
//   - `count` students each with a unique paid enrollment (via seedSEBundle) + one SE
//
// Returns (sectionID, adminSID, seIDs, cleanup).
func setupSEPaginationFixture(t *testing.T, adminEmail, suffix string, count int, capacity int32) (sectionID, adminSID string, seIDs []string, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	_, adminSID = seedUserWithSession(t, adminEmail, "admin")

	_, courseID, cleanupCatalog := seedProgramWithCourse(t)
	periodID, year, cleanupPeriod := seedAcademicPeriodWithWindow(t, true, false)
	sectionID, cleanupSection := seedSection(t, courseID, periodID, capacity)

	seIDs = make([]string, 0, count)
	for i := 0; i < count; i++ {
		email := fmt.Sprintf("se-pag-%s-s%02d@se-pag.test", suffix, i)
		studentID := seedUserWithRole(t, email, "student")

		// Ensure student_profiles row for the FK constraint.
		_, spErr := pgxPool.Exec(ctx,
			`INSERT INTO student_profiles (user_id, admission_year)
			 VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
			studentID, year,
		)
		if spErr != nil {
			t.Fatalf("setupSEPaginationFixture: ensure student_profiles: %v", spErr)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, studentID)
		})

		_, seID := seedSEBundle(t, studentID, year, courseID, sectionID)
		seIDs = append(seIDs, seID)
	}

	cleanup = func() {
		cleanupSection()
		cleanupPeriod()
		cleanupCatalog()
	}
	return sectionID, adminSID, seIDs, cleanup
}

// seedOwnSEFixture builds a fixture for ListOwnSectionEnrollments pagination:
// one student, one shared section, `count` SEs via unique programs (one per SE).
// Returns (studentSID, seIDs, cleanup).
func seedOwnSEFixture(t *testing.T, studentEmail, suffix string, count int) (studentID uuid.UUID, studentSID string, seIDs []string, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	studentID, studentSID = seedUserWithSession(t, studentEmail, "student")

	_, courseID, cleanupCatalog := seedProgramWithCourse(t)
	periodID, year, cleanupPeriod := seedAcademicPeriodWithWindow(t, true, false)
	sectionID, cleanupSection := seedSection(t, courseID, periodID, int32(count+5))

	// Ensure student_profiles row.
	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO student_profiles (user_id, admission_year) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
		studentID, year,
	); err != nil {
		t.Fatalf("seedOwnSEFixture: ensure student_profiles: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, studentID)
	})

	seIDs = make([]string, 0, count)
	for i := 0; i < count; i++ {
		_, seID := seedSEBundle(t, studentID, year, courseID, sectionID)
		seIDs = append(seIDs, seID)
	}

	cleanup = func() {
		cleanupSection()
		cleanupPeriod()
		cleanupCatalog()
	}
	return studentID, studentSID, seIDs, cleanup
}

// --- ListSectionEnrollments pagination ---

// TestSectionEnrollment_ListSectionEnrollments_FirstPage seeds >20 inscriptions and
// asserts the first page returns page_size items and a non-empty next_page_token.
func TestSectionEnrollment_ListSectionEnrollments_FirstPage(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, adminSID, _, cleanup := setupSEPaginationFixture(t,
		"se-pag-fp-admin@se-pagfp.test", suffix, 25, 50)
	defer cleanup()

	client := newSectionEnrollmentClient(nil)
	req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		SectionId: sectionID,
		PageSize:  20,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListSectionEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListSectionEnrollments: %v", err)
	}
	if len(resp.Msg.GetSectionEnrollments()) != 20 {
		t.Errorf("got %d inscriptions, want 20", len(resp.Msg.GetSectionEnrollments()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty when more pages exist")
	}
}

// TestSectionEnrollment_ListSectionEnrollments_SecondPage verifies no overlap or gap
// across two pages.
func TestSectionEnrollment_ListSectionEnrollments_SecondPage(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, adminSID, seIDs, cleanup := setupSEPaginationFixture(t,
		"se-pag-sp-admin@se-pagsp.test", suffix, 25, 50)
	defer cleanup()

	seededIDs := make(map[string]struct{}, len(seIDs))
	for _, id := range seIDs {
		seededIDs[id] = struct{}{}
	}

	client := newSectionEnrollmentClient(nil)

	// Page 1.
	req1 := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		SectionId: sectionID,
		PageSize:  20,
	})
	req1.Header().Set("Cookie", "sid="+adminSID)
	p1, err := client.ListSectionEnrollments(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty")
	}

	page1IDs := make(map[string]struct{})
	for _, se := range p1.Msg.GetSectionEnrollments() {
		page1IDs[se.GetId()] = struct{}{}
	}

	// Page 2.
	req2 := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		SectionId: sectionID,
		PageSize:  20,
		PageToken: token,
	})
	req2.Header().Set("Cookie", "sid="+adminSID)
	p2, err := client.ListSectionEnrollments(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	// Verify no duplicates.
	for _, se := range p2.Msg.GetSectionEnrollments() {
		if _, dup := page1IDs[se.GetId()]; dup {
			t.Errorf("duplicate se_id %s across pages", se.GetId())
		}
	}

	// Verify all seeded IDs appear across both pages.
	allReturned := make(map[string]struct{})
	for _, se := range p1.Msg.GetSectionEnrollments() {
		allReturned[se.GetId()] = struct{}{}
	}
	for _, se := range p2.Msg.GetSectionEnrollments() {
		allReturned[se.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded se_id %s missing from paginated results", id)
		}
	}
}

// TestSectionEnrollment_ListSectionEnrollments_LastPageEmptyToken walks to exhaustion
// and verifies the final page has an empty next_page_token.
func TestSectionEnrollment_ListSectionEnrollments_LastPageEmptyToken(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "se-pag-lp-admin@se-paglp.test", "admin")
	client := newSectionEnrollmentClient(nil)

	var token string
	for {
		req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
			PageSize:  200,
			PageToken: token,
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		resp, err := client.ListSectionEnrollments(ctx, req)
		if err != nil {
			t.Fatalf("ListSectionEnrollments walk: %v", err)
		}
		token = resp.Msg.GetNextPageToken()
		if token == "" {
			break
		}
	}
}

// TestSectionEnrollment_ListSectionEnrollments_ClampZero verifies page_size=0 is
// clamped to 20.
func TestSectionEnrollment_ListSectionEnrollments_ClampZero(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, adminSID, _, cleanup := setupSEPaginationFixture(t,
		"se-pag-cz-admin@se-pagcz.test", suffix, 25, 50)
	defer cleanup()

	client := newSectionEnrollmentClient(nil)
	req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		SectionId: sectionID,
		PageSize:  0,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSectionEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListSectionEnrollments: %v", err)
	}
	if len(resp.Msg.GetSectionEnrollments()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetSectionEnrollments()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty (more pages exist)")
	}
}

// TestSectionEnrollment_ListSectionEnrollments_InvalidToken verifies a malformed
// page_token returns CodeInvalidArgument.
func TestSectionEnrollment_ListSectionEnrollments_InvalidToken(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "se-pag-inv-admin@se-paginv.test", "admin")
	client := newSectionEnrollmentClient(nil)

	req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		PageToken: "not-a-uuid",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListSectionEnrollments(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSectionEnrollment_ListSectionEnrollments_FilterSectionIDPreserved verifies that
// section_id filter still applies alongside pagination.
func TestSectionEnrollment_ListSectionEnrollments_FilterSectionIDPreserved(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]

	// Section 1 with 3 inscriptions.
	sectionID1, adminSID, seIDs1, cleanup1 := setupSEPaginationFixture(t,
		"se-pag-fsec-admin@se-pagfsec.test", suffix, 3, 10)
	defer cleanup1()

	// Section 2: a separate section seeded with a different course/period.
	_, courseID2, cleanupCat2 := seedProgramWithCourse(t)
	defer cleanupCat2()
	periodID2, year2, cleanupPeriod2 := seedAcademicPeriodWithWindow(t, true, false)
	defer cleanupPeriod2()
	sectionID2, cleanupSec2 := seedSection(t, courseID2, periodID2, 5)
	defer cleanupSec2()

	// Decoy student with their own paid enrollment in section 2.
	decoyStudentID := seedUserWithRole(t, "se-pag-fsec-decoy@se-pagfsec.test", "student")
	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO student_profiles (user_id, admission_year) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
		decoyStudentID, year2,
	); err != nil {
		t.Fatalf("ensure student_profiles for decoy: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, decoyStudentID)
	})
	_, decoySEID := seedSEBundle(t, decoyStudentID, year2, courseID2, sectionID2)

	client := newSectionEnrollmentClient(nil)
	req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		SectionId: sectionID1,
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSectionEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListSectionEnrollments(section_id): %v", err)
	}

	foundSection1SE := false
	for _, se := range resp.Msg.GetSectionEnrollments() {
		if se.GetSectionId() != sectionID1 {
			t.Errorf("se %s has section_id=%s, want %s", se.GetId(), se.GetSectionId(), sectionID1)
		}
		if se.GetId() == decoySEID {
			t.Errorf("decoy se %s from section2 leaked into section1-filtered result", decoySEID)
		}
		if se.GetId() == seIDs1[0] {
			foundSection1SE = true
		}
	}
	if !foundSection1SE {
		t.Errorf("seeded se %s from section1 missing from section_id-filtered result", seIDs1[0])
	}
}

// TestSectionEnrollment_ListSectionEnrollments_FilterEnrollmentIDPreserved verifies
// enrollment_id filter still applies alongside pagination.
func TestSectionEnrollment_ListSectionEnrollments_FilterEnrollmentIDPreserved(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]

	_, adminSID := seedUserWithSession(t, "se-pag-fenr-admin@se-pagfenr.test", "admin")
	_, courseID, cleanupCat := seedProgramWithCourse(t)
	defer cleanupCat()
	periodID, year, cleanupPeriod := seedAcademicPeriodWithWindow(t, true, false)
	defer cleanupPeriod()
	sectionID, cleanupSection := seedSection(t, courseID, periodID, 10)
	defer cleanupSection()

	// Seed 3 students, each with their own enrollment and SE.
	var targetEnrollmentID, targetSEID string
	for i := 0; i < 3; i++ {
		email := fmt.Sprintf("se-pag-fenr-%s-s%d@se-pagfenr.test", suffix, i)
		studentID := seedUserWithRole(t, email, "student")
		if _, err := pgxPool.Exec(ctx,
			`INSERT INTO student_profiles (user_id, admission_year) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
			studentID, year,
		); err != nil {
			t.Fatalf("ensure student_profiles: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, studentID)
		})
		enrollmentID, seID := seedSEBundle(t, studentID, year, courseID, sectionID)
		if i == 0 {
			targetEnrollmentID = enrollmentID
			targetSEID = seID
		}
	}

	client := newSectionEnrollmentClient(nil)
	req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		EnrollmentId: targetEnrollmentID,
		PageSize:     200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSectionEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListSectionEnrollments(enrollment_id): %v", err)
	}

	found := false
	for _, se := range resp.Msg.GetSectionEnrollments() {
		if se.GetEnrollmentId() != targetEnrollmentID {
			t.Errorf("se %s has enrollment_id=%s, want %s", se.GetId(), se.GetEnrollmentId(), targetEnrollmentID)
		}
		if se.GetId() == targetSEID {
			found = true
		}
	}
	if !found {
		t.Errorf("seeded se %s for target enrollment missing from result", targetSEID)
	}
}

// TestSectionEnrollment_ListSectionEnrollments_FilterStatusPreserved verifies status
// filter still applies alongside pagination.
func TestSectionEnrollment_ListSectionEnrollments_FilterStatusPreserved(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, adminSID, seIDs, cleanup := setupSEPaginationFixture(t,
		"se-pag-fst-admin@se-pagfst.test", suffix, 2, 10)
	defer cleanup()

	// Mark the first SE as withdrawn directly.
	if _, err := pgxPool.Exec(ctx,
		`UPDATE section_enrollments SET status='withdrawn', updated_at=now() WHERE id=$1`,
		seIDs[0],
	); err != nil {
		t.Fatalf("mark withdrawn: %v", err)
	}
	inProgressSEID := seIDs[1]

	client := newSectionEnrollmentClient(nil)
	req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		SectionId: sectionID,
		Status:    "in_progress",
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSectionEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListSectionEnrollments(status): %v", err)
	}

	found := false
	for _, se := range resp.Msg.GetSectionEnrollments() {
		if se.GetStatus() != "in_progress" {
			t.Errorf("se %s has status=%s, want in_progress", se.GetId(), se.GetStatus())
		}
		if se.GetId() == seIDs[0] {
			t.Errorf("withdrawn se %s leaked into in_progress-filtered result", seIDs[0])
		}
		if se.GetId() == inProgressSEID {
			found = true
		}
	}
	if !found {
		t.Errorf("in_progress se %s missing from status-filtered result", inProgressSEID)
	}
}

// TestSectionEnrollment_ListSectionEnrollments_IDDescOrder verifies items are ordered
// by id DESC.
func TestSectionEnrollment_ListSectionEnrollments_IDDescOrder(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	sectionID, adminSID, _, cleanup := setupSEPaginationFixture(t,
		"se-pag-ord-admin@se-pagord.test", suffix, 5, 10)
	defer cleanup()

	client := newSectionEnrollmentClient(nil)
	req := connect.NewRequest(&section_enrollmentv1.ListSectionEnrollmentsRequest{
		SectionId: sectionID,
		PageSize:  20,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListSectionEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListSectionEnrollments: %v", err)
	}
	ses := resp.Msg.GetSectionEnrollments()
	for i := 1; i < len(ses); i++ {
		if ses[i-1].GetId() <= ses[i].GetId() {
			t.Errorf("ses[%d].id=%s >= ses[%d].id=%s (want DESC order)",
				i-1, ses[i-1].GetId(), i, ses[i].GetId())
		}
	}
}

// --- ListOwnSectionEnrollments pagination ---

// TestSectionEnrollment_ListOwnSectionEnrollments_SelfScopeAndPagination verifies
// that ListOwnSectionEnrollments returns only the caller's own inscriptions and
// respects pagination.
func TestSectionEnrollment_ListOwnSectionEnrollments_SelfScopeAndPagination(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]

	studentID, studentSID, ownSEIDs, cleanup := seedOwnSEFixture(t,
		fmt.Sprintf("se-pag-own-s@se-pagonw-%s.test", suffix), suffix, 3)
	defer cleanup()

	// Decoy SE: a different student, same section setup.
	otherSuffix := uuid.New().String()[:8]
	_, _, otherSEIDs, cleanupOther := seedOwnSEFixture(t,
		fmt.Sprintf("se-pag-own-other@se-pagonw-%s.test", otherSuffix), otherSuffix, 1)
	defer cleanupOther()
	decoySeID := otherSEIDs[0]

	_ = studentID
	client := newSectionEnrollmentClient(nil)
	req := connect.NewRequest(&section_enrollmentv1.ListOwnSectionEnrollmentsRequest{
		PageSize: 200,
	})
	req.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.ListOwnSectionEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListOwnSectionEnrollments: %v", err)
	}

	// The decoy SE must never appear.
	for _, se := range resp.Msg.GetSectionEnrollments() {
		if se.GetId() == decoySeID {
			t.Errorf("decoy se %s for other student leaked", decoySeID)
		}
	}

	// All own SEs must appear.
	returned := make(map[string]struct{})
	for _, se := range resp.Msg.GetSectionEnrollments() {
		returned[se.GetId()] = struct{}{}
	}
	for _, id := range ownSEIDs {
		if _, ok := returned[id]; !ok {
			t.Errorf("own se %s missing from ListOwnSectionEnrollments result", id)
		}
	}
}

// TestSectionEnrollment_ListOwnSectionEnrollments_ClampZero verifies page_size=0 is
// clamped to 20.
func TestSectionEnrollment_ListOwnSectionEnrollments_ClampZero(t *testing.T) {
	suffix := uuid.New().String()[:8]
	_, studentSID, _, cleanup := seedOwnSEFixture(t,
		fmt.Sprintf("se-pag-owncz@se-pagowncz-%s.test", suffix), suffix, 25)
	defer cleanup()

	client := newSectionEnrollmentClient(nil)
	req := connect.NewRequest(&section_enrollmentv1.ListOwnSectionEnrollmentsRequest{
		PageSize: 0,
	})
	req.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.ListOwnSectionEnrollments(context.Background(), req)
	if err != nil {
		t.Fatalf("ListOwnSectionEnrollments: %v", err)
	}
	if len(resp.Msg.GetSectionEnrollments()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetSectionEnrollments()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty (more pages exist)")
	}
}

// TestSectionEnrollment_ListOwnSectionEnrollments_InvalidToken verifies a malformed
// page_token returns CodeInvalidArgument.
func TestSectionEnrollment_ListOwnSectionEnrollments_InvalidToken(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	_, studentSID := seedUserWithSession(t,
		fmt.Sprintf("se-pag-owninv@se-pagowninv-%s.test", suffix), "student")
	client := newSectionEnrollmentClient(nil)

	req := connect.NewRequest(&section_enrollmentv1.ListOwnSectionEnrollmentsRequest{
		PageToken: "not-a-uuid",
	})
	req.Header().Set("Cookie", "sid="+studentSID)
	_, err := client.ListOwnSectionEnrollments(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSectionEnrollment_ListOwnSectionEnrollments_Pagination verifies walking pages
// returns all own inscriptions without overlap or gap.
func TestSectionEnrollment_ListOwnSectionEnrollments_Pagination(t *testing.T) {
	suffix := uuid.New().String()[:8]
	_, studentSID, seIDs, cleanup := seedOwnSEFixture(t,
		fmt.Sprintf("se-pag-ownpag@se-pagownpag-%s.test", suffix), suffix, 25)
	defer cleanup()

	seededIDs := make(map[string]struct{}, len(seIDs))
	for _, id := range seIDs {
		seededIDs[id] = struct{}{}
	}

	client := newSectionEnrollmentClient(nil)

	// Page 1.
	req1 := connect.NewRequest(&section_enrollmentv1.ListOwnSectionEnrollmentsRequest{PageSize: 20})
	req1.Header().Set("Cookie", "sid="+studentSID)
	p1, err := client.ListOwnSectionEnrollments(context.Background(), req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty (25 SEs seeded)")
	}

	page1IDs := make(map[string]struct{})
	for _, se := range p1.Msg.GetSectionEnrollments() {
		page1IDs[se.GetId()] = struct{}{}
	}

	// Page 2.
	req2 := connect.NewRequest(&section_enrollmentv1.ListOwnSectionEnrollmentsRequest{
		PageSize:  20,
		PageToken: token,
	})
	req2.Header().Set("Cookie", "sid="+studentSID)
	p2, err := client.ListOwnSectionEnrollments(context.Background(), req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	for _, se := range p2.Msg.GetSectionEnrollments() {
		if _, dup := page1IDs[se.GetId()]; dup {
			t.Errorf("duplicate se_id %s across pages", se.GetId())
		}
	}

	allReturned := make(map[string]struct{})
	for _, se := range p1.Msg.GetSectionEnrollments() {
		allReturned[se.GetId()] = struct{}{}
	}
	for _, se := range p2.Msg.GetSectionEnrollments() {
		allReturned[se.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded se %s missing from paginated ListOwnSectionEnrollments", id)
		}
	}
}
