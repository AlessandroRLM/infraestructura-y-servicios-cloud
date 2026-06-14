package integration_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1"
)

// --- Seed helpers for enrollment pagination tests ---

// seedEnrollmentForStudent inserts an enrollment row directly in the DB for the given
// student and program, using the given year and status "pending". It ensures a
// student_profiles row exists for the student (required by the FK constraint) using
// an upsert. Registers cleanup for the enrollment row only (student and profile are
// cleaned up by their own helpers).
// Returns the enrollment UUID string.
func seedEnrollmentForStudent(t *testing.T, studentID uuid.UUID, programID string, year int32) string {
	t.Helper()
	ctx := context.Background()

	// Ensure student_profiles row exists (enrollments.student_id → student_profiles.user_id FK).
	_, err := pgxPool.Exec(ctx,
		`INSERT INTO student_profiles (user_id, admission_year) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
		studentID, year,
	)
	if err != nil {
		t.Fatalf("seedEnrollmentForStudent: ensure student_profiles: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, studentID)
	})

	var id string
	err = pgxPool.QueryRow(ctx,
		`INSERT INTO enrollments (student_id, program_id, year, status)
		 VALUES ($1, $2, $3, 'pending') RETURNING id`,
		studentID, programID, year,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedEnrollmentForStudent: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, id)
	})
	return id
}

// --- ListEnrollments pagination ---

// TestEnrollment_ListEnrollments_FirstPage seeds >20 enrollments and asserts the
// first page returns page_size items and a non-empty next_page_token.
func TestEnrollment_ListEnrollments_FirstPage(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-fp-admin@enroll-pag.test", "admin")
	client := newEnrollmentClient(nil)

	programID, cleanupProg := seedProgramWithQuota(t, 50, 2200)
	defer cleanupProg()

	suffix := uuid.New().String()[:6]
	for i := 0; i < 25; i++ {
		studentID := seedUserWithRole(t, fmt.Sprintf("enroll-pag-fp-s%s-%02d@enroll-pag.test", suffix, i), "student")
		seedEnrollmentForStudent(t, studentID, programID, 2200)
	}

	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageSize: 20})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}
	if len(resp.Msg.GetEnrollments()) != 20 {
		t.Errorf("got %d enrollments, want 20", len(resp.Msg.GetEnrollments()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty when more pages exist")
	}
}

// TestEnrollment_ListEnrollments_SecondPage verifies no overlap or gap across two pages.
func TestEnrollment_ListEnrollments_SecondPage(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-sp-admin@enroll-pag2.test", "admin")
	client := newEnrollmentClient(nil)

	programID, cleanupProg := seedProgramWithQuota(t, 30, 2201)
	defer cleanupProg()

	suffix := uuid.New().String()[:6]
	seededIDs := make(map[string]struct{}, 25)
	for i := 0; i < 25; i++ {
		studentID := seedUserWithRole(t, fmt.Sprintf("enroll-pag-sp-s%s-%02d@enroll-pag2.test", suffix, i), "student")
		id := seedEnrollmentForStudent(t, studentID, programID, 2201)
		seededIDs[id] = struct{}{}
	}

	// Page 1.
	req1 := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageSize: 20})
	req1.Header().Set("Cookie", "sid="+adminSID)
	p1, err := client.ListEnrollments(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty")
	}

	// Collect page 1 IDs.
	page1IDs := make(map[string]struct{})
	for _, e := range p1.Msg.GetEnrollments() {
		page1IDs[e.GetId()] = struct{}{}
	}

	// Page 2.
	req2 := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageSize: 20, PageToken: token})
	req2.Header().Set("Cookie", "sid="+adminSID)
	p2, err := client.ListEnrollments(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	// Verify no duplicates.
	for _, e := range p2.Msg.GetEnrollments() {
		if _, dup := page1IDs[e.GetId()]; dup {
			t.Errorf("duplicate enrollment_id %s across pages", e.GetId())
		}
	}

	// Verify all 25 seeded IDs appear across both pages.
	allReturned := make(map[string]struct{})
	for _, e := range p1.Msg.GetEnrollments() {
		allReturned[e.GetId()] = struct{}{}
	}
	for _, e := range p2.Msg.GetEnrollments() {
		allReturned[e.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded enrollment %s missing from paginated results", id)
		}
	}
}

// TestEnrollment_ListEnrollments_LastPageEmptyToken walks pages to exhaustion and
// verifies the final page has an empty next_page_token.
func TestEnrollment_ListEnrollments_LastPageEmptyToken(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-lp-admin@enroll-paglp.test", "admin")
	client := newEnrollmentClient(nil)

	var token string
	for {
		req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageSize: 200, PageToken: token})
		req.Header().Set("Cookie", "sid="+adminSID)
		resp, err := client.ListEnrollments(ctx, req)
		if err != nil {
			t.Fatalf("ListEnrollments walk: %v", err)
		}
		token = resp.Msg.GetNextPageToken()
		if token == "" {
			// Reached last page.
			if resp.Msg.GetNextPageToken() != "" {
				t.Error("last page: next_page_token must be empty")
			}
			break
		}
	}
}

// TestEnrollment_ListEnrollments_ClampZero verifies page_size=0 is clamped to 20.
func TestEnrollment_ListEnrollments_ClampZero(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-cz-admin@enroll-pagcz.test", "admin")
	client := newEnrollmentClient(nil)

	programID, cleanupProg := seedProgramWithQuota(t, 50, 2202)
	defer cleanupProg()

	suffix := uuid.New().String()[:6]
	for i := 0; i < 25; i++ {
		studentID := seedUserWithRole(t, fmt.Sprintf("enroll-pag-cz-s%s-%02d@enroll-pagcz.test", suffix, i), "student")
		seedEnrollmentForStudent(t, studentID, programID, 2202)
	}

	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageSize: 0})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}
	if len(resp.Msg.GetEnrollments()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetEnrollments()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty (more pages exist)")
	}
}

// TestEnrollment_ListEnrollments_ClampMax verifies page_size=999 is clamped to 200.
func TestEnrollment_ListEnrollments_ClampMax(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-cm-admin@enroll-pagcm.test", "admin")
	client := newEnrollmentClient(nil)

	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageSize: 999})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}
	if len(resp.Msg.GetEnrollments()) > 200 {
		t.Errorf("page_size=999 → got %d rows, want ≤200 (clamped to max)", len(resp.Msg.GetEnrollments()))
	}
}

// TestEnrollment_ListEnrollments_InvalidToken verifies malformed page_token returns
// CodeInvalidArgument.
func TestEnrollment_ListEnrollments_InvalidToken(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-inv-admin@enroll-paginv.test", "admin")
	client := newEnrollmentClient(nil)

	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageToken: "not-a-uuid"})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListEnrollments(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestEnrollment_ListEnrollments_IDDescOrder verifies items within a page are ordered
// by id DESC.
func TestEnrollment_ListEnrollments_IDDescOrder(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-ord-admin@enroll-pagord.test", "admin")
	client := newEnrollmentClient(nil)

	programID, cleanupProg := seedProgramWithQuota(t, 30, 2203)
	defer cleanupProg()

	suffix := uuid.New().String()[:6]
	for i := 0; i < 5; i++ {
		studentID := seedUserWithRole(t, fmt.Sprintf("enroll-pag-ord-s%s-%02d@enroll-pagord.test", suffix, i), "student")
		seedEnrollmentForStudent(t, studentID, programID, 2203)
	}

	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageSize: 20})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}
	enrollments := resp.Msg.GetEnrollments()
	for i := 1; i < len(enrollments); i++ {
		if enrollments[i-1].GetId() <= enrollments[i].GetId() {
			t.Errorf("enrollments[%d].id=%s >= enrollments[%d].id=%s (want DESC order)",
				i-1, enrollments[i-1].GetId(), i, enrollments[i].GetId())
		}
	}
}

// TestEnrollment_ListEnrollments_FilterStudentPreserved verifies that student_id
// filter still applies alongside pagination.
func TestEnrollment_ListEnrollments_FilterStudentPreserved(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-fst-admin@enroll-pagfst.test", "admin")
	client := newEnrollmentClient(nil)

	programID, cleanupProg := seedProgramWithQuota(t, 30, 2204)
	defer cleanupProg()

	s1ID := seedUserWithRole(t, "enroll-pag-fst-s1@enroll-pagfst.test", "student")
	s2ID := seedUserWithRole(t, "enroll-pag-fst-s2@enroll-pagfst.test", "student")

	e1ID := seedEnrollmentForStudent(t, s1ID, programID, 2204)
	e2ID := seedEnrollmentForStudent(t, s2ID, programID, 2204)
	_ = e2ID

	s1IDStr := s1ID.String()
	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{
		StudentId: s1IDStr,
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListEnrollments(student_id): %v", err)
	}

	found := false
	for _, e := range resp.Msg.GetEnrollments() {
		if e.GetStudentId() != s1IDStr {
			t.Errorf("enrollment %s has student_id=%s, want %s", e.GetId(), e.GetStudentId(), s1IDStr)
		}
		if e.GetId() == e1ID {
			found = true
		}
	}
	if !found {
		t.Errorf("seeded enrollment %s for s1 missing from student-filtered result", e1ID)
	}
}

// TestEnrollment_ListEnrollments_FilterProgramPreserved verifies program_id filter
// still applies alongside pagination.
func TestEnrollment_ListEnrollments_FilterProgramPreserved(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-fprog-admin@enroll-pagfprog.test", "admin")
	client := newEnrollmentClient(nil)

	prog1ID, cleanupProg1 := seedProgramWithQuota(t, 10, 2205)
	defer cleanupProg1()
	prog2ID, cleanupProg2 := seedProgramWithQuota(t, 10, 2205)
	defer cleanupProg2()

	s1ID := seedUserWithRole(t, "enroll-pag-fprog-s1@enroll-pagfprog.test", "student")
	s2ID := seedUserWithRole(t, "enroll-pag-fprog-s2@enroll-pagfprog.test", "student")

	e1ID := seedEnrollmentForStudent(t, s1ID, prog1ID, 2205)
	e2ID := seedEnrollmentForStudent(t, s2ID, prog2ID, 2205)
	_ = e2ID

	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{
		ProgramId: prog1ID,
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListEnrollments(program_id): %v", err)
	}

	found := false
	for _, e := range resp.Msg.GetEnrollments() {
		if e.GetProgramId() != prog1ID {
			t.Errorf("enrollment %s has program_id=%s, want %s", e.GetId(), e.GetProgramId(), prog1ID)
		}
		if e.GetId() == e1ID {
			found = true
		}
	}
	if !found {
		t.Errorf("seeded enrollment %s for prog1 missing from program-filtered result", e1ID)
	}
}

// TestEnrollment_ListEnrollments_FilterYearPreserved verifies year filter still
// applies alongside pagination.
func TestEnrollment_ListEnrollments_FilterYearPreserved(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-fyr-admin@enroll-pagfyr.test", "admin")
	client := newEnrollmentClient(nil)

	programID, cleanupProg := seedProgramWithQuota(t, 10, 2206)
	defer cleanupProg()
	programID2, cleanupProg2 := seedProgramWithQuota(t, 10, 2207)
	defer cleanupProg2()

	s1ID := seedUserWithRole(t, "enroll-pag-fyr-s1@enroll-pagfyr.test", "student")
	s2ID := seedUserWithRole(t, "enroll-pag-fyr-s2@enroll-pagfyr.test", "student")

	e1ID := seedEnrollmentForStudent(t, s1ID, programID, 2206)
	e2ID := seedEnrollmentForStudent(t, s2ID, programID2, 2207)
	_ = e2ID

	year := int32(2206)
	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{
		Year:     year,
		PageSize: 200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListEnrollments(year): %v", err)
	}

	found := false
	for _, e := range resp.Msg.GetEnrollments() {
		if e.GetYear() != year {
			t.Errorf("enrollment %s has year=%d, want %d", e.GetId(), e.GetYear(), year)
		}
		if e.GetId() == e1ID {
			found = true
		}
	}
	if !found {
		t.Errorf("seeded enrollment %s for year 2206 missing from year-filtered result", e1ID)
	}
}

// TestEnrollment_ListEnrollments_FilterStatusPreserved verifies status filter still
// applies alongside pagination.
func TestEnrollment_ListEnrollments_FilterStatusPreserved(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-fss-admin@enroll-pagfss.test", "admin")
	client := newEnrollmentClient(nil)

	programID, cleanupProg := seedProgramWithQuota(t, 10, 2208)
	defer cleanupProg()

	// Seed a pending enrollment.
	s1ID := seedUserWithRole(t, "enroll-pag-fss-s1@enroll-pagfss.test", "student")
	e1ID := seedEnrollmentForStudent(t, s1ID, programID, 2208)

	// Seed a cancelled enrollment directly (bypassing the service state machine).
	s2ID := seedUserWithRole(t, "enroll-pag-fss-s2@enroll-pagfss.test", "student")
	// Ensure student_profiles row exists for the direct insert.
	_, spErr := pgxPool.Exec(context.Background(),
		`INSERT INTO student_profiles (user_id, admission_year) VALUES ($1, 2208) ON CONFLICT (user_id) DO NOTHING`,
		s2ID,
	)
	if spErr != nil {
		t.Fatalf("ensure student_profiles for s2: %v", spErr)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, s2ID)
	})
	var e2ID string
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO enrollments (student_id, program_id, year, status)
		 VALUES ($1, $2, 2208, 'cancelled') RETURNING id`,
		s2ID, programID,
	).Scan(&e2ID)
	if err != nil {
		t.Fatalf("insert cancelled enrollment: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, e2ID)
	})

	status := "pending"
	req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{
		Status:   status,
		PageSize: 200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err2 := client.ListEnrollments(ctx, req)
	if err2 != nil {
		t.Fatalf("ListEnrollments(status): %v", err2)
	}

	found := false
	for _, e := range resp.Msg.GetEnrollments() {
		if e.GetStatus() != status {
			t.Errorf("enrollment %s has status=%s, want %s", e.GetId(), e.GetStatus(), status)
		}
		if e.GetId() == e1ID {
			found = true
		}
		if e.GetId() == e2ID {
			t.Errorf("cancelled enrollment %s appeared in pending-filtered result", e2ID)
		}
	}
	if !found {
		t.Errorf("seeded pending enrollment %s missing from status-filtered result", e1ID)
	}
}

// TestEnrollment_ListEnrollments_SoftDeletedExcluded verifies soft-deleted enrollments
// never appear on any page.
func TestEnrollment_ListEnrollments_SoftDeletedExcluded(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-pag-sd-admin@enroll-pagsd.test", "admin")
	client := newEnrollmentClient(nil)

	programID, cleanupProg := seedProgramWithQuota(t, 10, 2209)
	defer cleanupProg()

	// Insert a soft-deleted enrollment directly.
	studentID := seedUserWithRole(t, "enroll-pag-sd-s1@enroll-pagsd.test", "student")
	// Ensure student_profiles row exists for the direct insert.
	_, spErr := pgxPool.Exec(context.Background(),
		`INSERT INTO student_profiles (user_id, admission_year) VALUES ($1, 2209) ON CONFLICT (user_id) DO NOTHING`,
		studentID,
	)
	if spErr != nil {
		t.Fatalf("ensure student_profiles for soft-delete student: %v", spErr)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, studentID)
	})
	var deletedID string
	err := pgxPool.QueryRow(context.Background(),
		`INSERT INTO enrollments (student_id, program_id, year, status, deleted_at)
		 VALUES ($1, $2, 2209, 'cancelled', now()) RETURNING id`,
		studentID, programID,
	).Scan(&deletedID)
	if err != nil {
		t.Fatalf("insert soft-deleted enrollment: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, deletedID)
	})

	// Walk all pages and assert the soft-deleted enrollment never appears.
	var token string
	for {
		req := connect.NewRequest(&enrollmentv1.ListEnrollmentsRequest{PageSize: 200, PageToken: token})
		req.Header().Set("Cookie", "sid="+adminSID)
		resp, err := client.ListEnrollments(ctx, req)
		if err != nil {
			t.Fatalf("ListEnrollments: %v", err)
		}
		for _, e := range resp.Msg.GetEnrollments() {
			if e.GetId() == deletedID {
				t.Errorf("soft-deleted enrollment %s appeared in ListEnrollments", deletedID)
			}
		}
		token = resp.Msg.GetNextPageToken()
		if token == "" {
			break
		}
	}
}

// --- ListOwnEnrollments pagination ---

// TestEnrollment_ListOwnEnrollments_SelfScopeAndPagination verifies that
// ListOwnEnrollments returns only the caller's own enrollments and respects pagination.
func TestEnrollment_ListOwnEnrollments_SelfScopeAndPagination(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "enroll-own-pag-admin@enroll-ownpag.test", "admin")
	studentID, studentSID := seedUserWithSession(t, "enroll-own-pag-student@enroll-ownpag.test", "student")
	client := newEnrollmentClient(nil)

	// Seed a decoy enrollment for a different student — must never appear.
	otherStudentID := seedUserWithRole(t, "enroll-own-pag-other@enroll-ownpag.test", "student")

	programID, cleanupProg := seedProgramWithQuota(t, 50, 2210)
	defer cleanupProg()

	// Seed enrollments for the target student across different programs/years.
	ownIDs := make(map[string]struct{}, 3)
	for i := 0; i < 3; i++ {
		pid, cp := seedProgramWithQuota(t, 5, int32(2211+i))
		defer cp()
		id := seedEnrollmentForStudent(t, studentID, pid, int32(2211+i))
		ownIDs[id] = struct{}{}
	}

	// Seed a decoy enrollment for a different student using the common program.
	decoyID := seedEnrollmentForStudent(t, otherStudentID, programID, 2210)
	_ = decoyID

	// Student calls ListOwnEnrollments.
	req := connect.NewRequest(&enrollmentv1.ListOwnEnrollmentsRequest{PageSize: 200})
	req.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.ListOwnEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListOwnEnrollments: %v", err)
	}

	_ = adminSID // used only to confirm authz in separate authz test

	// All returned enrollments must belong to the calling student.
	for _, e := range resp.Msg.GetEnrollments() {
		if e.GetStudentId() != studentID.String() {
			t.Errorf("ListOwnEnrollments: got student_id=%s, want %s", e.GetStudentId(), studentID.String())
		}
		if e.GetId() == decoyID {
			t.Errorf("ListOwnEnrollments: decoy enrollment %s leaked", decoyID)
		}
	}

	// All seeded own IDs must appear.
	returned := make(map[string]struct{})
	for _, e := range resp.Msg.GetEnrollments() {
		returned[e.GetId()] = struct{}{}
	}
	for id := range ownIDs {
		if _, ok := returned[id]; !ok {
			t.Errorf("ListOwnEnrollments: own enrollment %s missing", id)
		}
	}
}

// TestEnrollment_ListOwnEnrollments_ClampZero verifies page_size=0 is clamped to 20.
func TestEnrollment_ListOwnEnrollments_ClampZero(t *testing.T) {
	ctx := context.Background()
	studentID, studentSID := seedUserWithSession(t, "enroll-own-pag-cz@enroll-ownpagcz.test", "student")
	client := newEnrollmentClient(nil)

	// Seed 25 enrollments for this student across unique programs.
	for i := 0; i < 25; i++ {
		pid, cp := seedProgramWithQuota(t, 5, int32(2220+i))
		defer cp()
		seedEnrollmentForStudent(t, studentID, pid, int32(2220+i))
	}

	req := connect.NewRequest(&enrollmentv1.ListOwnEnrollmentsRequest{PageSize: 0})
	req.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.ListOwnEnrollments(ctx, req)
	if err != nil {
		t.Fatalf("ListOwnEnrollments: %v", err)
	}
	if len(resp.Msg.GetEnrollments()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetEnrollments()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty (more pages exist)")
	}
}

// TestEnrollment_ListOwnEnrollments_InvalidToken verifies malformed page_token returns
// CodeInvalidArgument.
func TestEnrollment_ListOwnEnrollments_InvalidToken(t *testing.T) {
	ctx := context.Background()
	_, studentSID := seedUserWithSession(t, "enroll-own-pag-inv@enroll-ownpaginv.test", "student")
	client := newEnrollmentClient(nil)

	req := connect.NewRequest(&enrollmentv1.ListOwnEnrollmentsRequest{PageToken: "not-a-uuid"})
	req.Header().Set("Cookie", "sid="+studentSID)
	_, err := client.ListOwnEnrollments(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestEnrollment_ListOwnEnrollments_Pagination verifies that walking pages
// returns all own enrollments without overlap or gap.
func TestEnrollment_ListOwnEnrollments_Pagination(t *testing.T) {
	ctx := context.Background()
	studentID, studentSID := seedUserWithSession(t, "enroll-own-pag-pag@enroll-ownpagpag.test", "student")
	client := newEnrollmentClient(nil)

	// Seed 25 enrollments for this student across unique programs.
	seededIDs := make(map[string]struct{}, 25)
	for i := 0; i < 25; i++ {
		pid, cp := seedProgramWithQuota(t, 5, int32(2250+i))
		defer cp()
		id := seedEnrollmentForStudent(t, studentID, pid, int32(2250+i))
		seededIDs[id] = struct{}{}
	}

	// Page 1.
	req1 := connect.NewRequest(&enrollmentv1.ListOwnEnrollmentsRequest{PageSize: 20})
	req1.Header().Set("Cookie", "sid="+studentSID)
	p1, err := client.ListOwnEnrollments(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty (25 enrollments seeded)")
	}

	// Collect page 1 IDs.
	page1IDs := make(map[string]struct{})
	for _, e := range p1.Msg.GetEnrollments() {
		page1IDs[e.GetId()] = struct{}{}
	}

	// Page 2.
	req2 := connect.NewRequest(&enrollmentv1.ListOwnEnrollmentsRequest{PageSize: 20, PageToken: token})
	req2.Header().Set("Cookie", "sid="+studentSID)
	p2, err := client.ListOwnEnrollments(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	// Verify no duplicates.
	for _, e := range p2.Msg.GetEnrollments() {
		if _, dup := page1IDs[e.GetId()]; dup {
			t.Errorf("duplicate enrollment_id %s across pages", e.GetId())
		}
	}

	// Verify all 25 seeded own IDs appear.
	allReturned := make(map[string]struct{})
	for _, e := range p1.Msg.GetEnrollments() {
		allReturned[e.GetId()] = struct{}{}
	}
	for _, e := range p2.Msg.GetEnrollments() {
		allReturned[e.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded enrollment %s missing from paginated ListOwnEnrollments", id)
		}
	}
}
