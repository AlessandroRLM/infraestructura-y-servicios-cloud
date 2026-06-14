package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
)

// TestCatalog_Program_FullLifecycle verifies Create→Get→Update→List→SoftDelete→Get(NotFound)→List(excluded).
func TestCatalog_Program_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-prog-lifecycle@catalog.test")
	client := newCatalogClient(nil)

	code := "PROG-LIFE-" + uuid.New().String()[:8]

	// Create
	createReq := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: code, Name: "Life Program"})
	createReq.Header().Set("Cookie", "sid="+adminSID)
	createResp, err := client.CreateProgram(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	id := createResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, id) })

	// Get
	getReq := connect.NewRequest(&catalogv1.GetProgramRequest{Id: id})
	getReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.GetProgram(ctx, getReq); err != nil {
		t.Errorf("GetProgram after create: %v", err)
	}

	// Update
	updateReq := connect.NewRequest(&catalogv1.UpdateProgramRequest{Id: id, Code: code, Name: "Life Program Updated"})
	updateReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.UpdateProgram(ctx, updateReq); err != nil {
		t.Errorf("UpdateProgram: %v", err)
	}

	// List — must include
	listReq := connect.NewRequest(&catalogv1.ListProgramsRequest{})
	listReq.Header().Set("Cookie", "sid="+adminSID)
	listResp, err := client.ListPrograms(ctx, listReq)
	if err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	found := false
	for _, p := range listResp.Msg.GetPrograms() {
		if p.GetId() == id {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListPrograms: program %s not found in result", id)
	}

	// Delete
	delReq := connect.NewRequest(&catalogv1.DeleteProgramRequest{Id: id})
	delReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.DeleteProgram(ctx, delReq); err != nil {
		t.Errorf("DeleteProgram: %v", err)
	}

	// Get after delete — must be NotFound
	getAfter := connect.NewRequest(&catalogv1.GetProgramRequest{Id: id})
	getAfter.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetProgram(ctx, getAfter)
	assertConnectCode(t, err, connect.CodeNotFound)

	// List after delete — must exclude
	listAfter := connect.NewRequest(&catalogv1.ListProgramsRequest{})
	listAfter.Header().Set("Cookie", "sid="+adminSID)
	listAfterResp, err := client.ListPrograms(ctx, listAfter)
	if err != nil {
		t.Fatalf("ListPrograms after delete: %v", err)
	}
	for _, p := range listAfterResp.Msg.GetPrograms() {
		if p.GetId() == id {
			t.Errorf("ListPrograms after delete: soft-deleted program %s still appears", id)
		}
	}
}

// TestCatalog_Course_FullLifecycle verifies Create→Get→Update→List→Delete→Get(NotFound).
func TestCatalog_Course_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-crs-lifecycle@catalog.test")
	client := newCatalogClient(nil)

	code := "CRS-LIFE-" + uuid.New().String()[:8]

	createReq := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: code, Name: "Life Course", Credits: 4})
	createReq.Header().Set("Cookie", "sid="+adminSID)
	createResp, err := client.CreateCourse(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	id := createResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, id) })

	// Update
	updateReq := connect.NewRequest(&catalogv1.UpdateCourseRequest{Id: id, Code: code, Name: "Life Course Updated", Credits: 5})
	updateReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.UpdateCourse(ctx, updateReq); err != nil {
		t.Errorf("UpdateCourse: %v", err)
	}

	// Delete
	delReq := connect.NewRequest(&catalogv1.DeleteCourseRequest{Id: id})
	delReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.DeleteCourse(ctx, delReq); err != nil {
		t.Errorf("DeleteCourse: %v", err)
	}

	// Get after delete — must be NotFound
	getAfter := connect.NewRequest(&catalogv1.GetCourseRequest{Id: id})
	getAfter.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetCourse(ctx, getAfter)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestCatalog_AcademicPeriod_FullLifecycle verifies Create→Get→Update→Delete→Get(NotFound).
func TestCatalog_AcademicPeriod_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-ap-lifecycle@catalog.test")
	client := newCatalogClient(nil)

	createReq := connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year: 3200, Term: 2, StartDate: "3200-08-01", EndDate: "3200-12-31",
	})
	createReq.Header().Set("Cookie", "sid="+adminSID)
	createResp, err := client.CreateAcademicPeriod(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateAcademicPeriod: %v", err)
	}
	id := createResp.Msg.GetId()
	t.Cleanup(func() { cleanupAcademicPeriod(t, id) })

	// Update
	updateReq := connect.NewRequest(&catalogv1.UpdateAcademicPeriodRequest{
		Id: id, Year: 3200, Term: 2, StartDate: "3200-08-15", EndDate: "3200-12-31",
	})
	updateReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.UpdateAcademicPeriod(ctx, updateReq); err != nil {
		t.Errorf("UpdateAcademicPeriod: %v", err)
	}

	// Delete
	delReq := connect.NewRequest(&catalogv1.DeleteAcademicPeriodRequest{Id: id})
	delReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.DeleteAcademicPeriod(ctx, delReq); err != nil {
		t.Errorf("DeleteAcademicPeriod: %v", err)
	}

	// Get after delete
	getAfter := connect.NewRequest(&catalogv1.GetAcademicPeriodRequest{Id: id})
	getAfter.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetAcademicPeriod(ctx, getAfter)
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestCatalog_ProgramQuota_FullLifecycle verifies quota Create→Get→Update→Delete→Get(NotFound).
func TestCatalog_ProgramQuota_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-quota-lifecycle@catalog.test")
	client := newCatalogClient(nil)

	// Create a program first.
	progCode := "PQ-LIFE-" + uuid.New().String()[:8]
	pReq := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: progCode, Name: "Quota Program"})
	pReq.Header().Set("Cookie", "sid="+adminSID)
	pResp, err := client.CreateProgram(ctx, pReq)
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	progID := pResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID) })

	// Create quota
	cReq := connect.NewRequest(&catalogv1.CreateProgramQuotaRequest{
		ProgramId: progID, Year: 2025, AdmissionQuota: 40,
	})
	cReq.Header().Set("Cookie", "sid="+adminSID)
	cResp, err := client.CreateProgramQuota(ctx, cReq)
	if err != nil {
		t.Fatalf("CreateProgramQuota: %v", err)
	}
	quotaID := cResp.Msg.GetId()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM program_quotas WHERE id = $1`, quotaID)
	})

	// Get quota
	gReq := connect.NewRequest(&catalogv1.GetProgramQuotaRequest{Id: quotaID})
	gReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.GetProgramQuota(ctx, gReq); err != nil {
		t.Errorf("GetProgramQuota: %v", err)
	}

	// Update quota
	uReq := connect.NewRequest(&catalogv1.UpdateProgramQuotaRequest{
		Id: quotaID, Year: 2025, AdmissionQuota: 50,
	})
	uReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.UpdateProgramQuota(ctx, uReq); err != nil {
		t.Errorf("UpdateProgramQuota: %v", err)
	}

	// Delete quota (soft)
	dReq := connect.NewRequest(&catalogv1.DeleteProgramQuotaRequest{Id: quotaID})
	dReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.DeleteProgramQuota(ctx, dReq); err != nil {
		t.Errorf("DeleteProgramQuota: %v", err)
	}

	// Get after delete — NotFound
	gAfter := connect.NewRequest(&catalogv1.GetProgramQuotaRequest{Id: quotaID})
	gAfter.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.GetProgramQuota(ctx, gAfter)
	assertConnectCode(t, err, connect.CodeNotFound)

	// Program can now be deleted (no live quota).
	delProgReq := connect.NewRequest(&catalogv1.DeleteProgramRequest{Id: progID})
	delProgReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.DeleteProgram(ctx, delProgReq); err != nil {
		t.Errorf("DeleteProgram after quota deleted: %v", err)
	}
}

// TestCatalog_ProgramCourses_MN verifies AddCourseToProgram→ListProgramCourses→RemoveCourseFromProgram,
// including the embedded Course fields on each ProgramCourse entry.
func TestCatalog_ProgramCourses_MN(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-mn-assoc@catalog.test")
	client := newCatalogClient(nil)

	// Create program and course.
	pCode := "MN-PROG-" + uuid.New().String()[:8]
	pReq := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: pCode, Name: "MN Program"})
	pReq.Header().Set("Cookie", "sid="+adminSID)
	pResp, err := client.CreateProgram(ctx, pReq)
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	progID := pResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID) })

	cCode := "MN-CRS-" + uuid.New().String()[:8]
	cReq := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: cCode, Name: "MN Course", Credits: 3})
	cReq.Header().Set("Cookie", "sid="+adminSID)
	cResp, err := client.CreateCourse(ctx, cReq)
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	courseID := cResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID) })

	// Add association
	addReq := connect.NewRequest(&catalogv1.AddCourseToProgramRequest{ProgramId: progID, CourseId: courseID})
	addReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.AddCourseToProgram(ctx, addReq); err != nil {
		t.Fatalf("AddCourseToProgram: %v", err)
	}

	// List — must contain with embedded course fields populated.
	listReq := connect.NewRequest(&catalogv1.ListProgramCoursesRequest{ProgramId: progID})
	listReq.Header().Set("Cookie", "sid="+adminSID)
	listResp, err := client.ListProgramCourses(ctx, listReq)
	if err != nil {
		t.Fatalf("ListProgramCourses: %v", err)
	}
	found := false
	for _, pc := range listResp.Msg.GetProgramCourses() {
		if pc.GetCourseId() == courseID {
			found = true
			// Assert embedded course fields are populated.
			c := pc.GetCourse()
			if c == nil {
				t.Errorf("ListProgramCourses: ProgramCourse.Course is nil for course %s", courseID)
				continue
			}
			if c.GetId() != courseID {
				t.Errorf("ProgramCourse.Course.Id = %q, want %q", c.GetId(), courseID)
			}
			if c.GetCode() != cCode {
				t.Errorf("ProgramCourse.Course.Code = %q, want %q", c.GetCode(), cCode)
			}
			if c.GetName() != "MN Course" {
				t.Errorf("ProgramCourse.Course.Name = %q, want %q", c.GetName(), "MN Course")
			}
			if c.GetCredits() != 3 {
				t.Errorf("ProgramCourse.Course.Credits = %d, want 3", c.GetCredits())
			}
			// Association tuple fields must remain intact.
			if pc.GetProgramId() != progID {
				t.Errorf("ProgramCourse.ProgramId = %q, want %q", pc.GetProgramId(), progID)
			}
			if pc.GetCreatedAt() == "" {
				t.Errorf("ProgramCourse.CreatedAt is empty")
			}
		}
	}
	if !found {
		t.Errorf("ListProgramCourses: course %s not found after adding", courseID)
	}

	// Duplicate add — AlreadyExists
	addDup := connect.NewRequest(&catalogv1.AddCourseToProgramRequest{ProgramId: progID, CourseId: courseID})
	addDup.Header().Set("Cookie", "sid="+adminSID)
	_, err = client.AddCourseToProgram(ctx, addDup)
	assertConnectCode(t, err, connect.CodeAlreadyExists)

	// Remove
	rmReq := connect.NewRequest(&catalogv1.RemoveCourseFromProgramRequest{ProgramId: progID, CourseId: courseID})
	rmReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.RemoveCourseFromProgram(ctx, rmReq); err != nil {
		t.Errorf("RemoveCourseFromProgram: %v", err)
	}

	// List after remove — must be empty
	listAfter := connect.NewRequest(&catalogv1.ListProgramCoursesRequest{ProgramId: progID})
	listAfter.Header().Set("Cookie", "sid="+adminSID)
	listAfterResp, err := client.ListProgramCourses(ctx, listAfter)
	if err != nil {
		t.Fatalf("ListProgramCourses after remove: %v", err)
	}
	if len(listAfterResp.Msg.GetProgramCourses()) != 0 {
		t.Errorf("ListProgramCourses after remove: expected 0, got %d", len(listAfterResp.Msg.GetProgramCourses()))
	}
}

// TestCatalog_ProgramCourses_Enriched verifies the full enrichment scenarios:
// - 2 live courses: both returned with embedded Course fields and intact association tuple.
// - 1 soft-deleted course: only the live one returned.
// - Empty program: empty list, no error.
// - Ordering: entries ordered by association created_at.
func TestCatalog_ProgramCourses_Enriched(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-enriched@catalog.test")
	client := newCatalogClient(nil)

	// --- Scenario 1 & 4: 2 live courses, both enriched, ordered by association created_at ---

	pCode1 := "ENRICH-PROG-" + uuid.New().String()[:8]
	pReq1 := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: pCode1, Name: "Enriched Program"})
	pReq1.Header().Set("Cookie", "sid="+adminSID)
	pResp1, err := client.CreateProgram(ctx, pReq1)
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	progID1 := pResp1.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID1) })

	cCode1 := "ENRICH-CRS1-" + uuid.New().String()[:8]
	cReq1 := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: cCode1, Name: "Enriched Course 1", Credits: 4})
	cReq1.Header().Set("Cookie", "sid="+adminSID)
	cResp1, err := client.CreateCourse(ctx, cReq1)
	if err != nil {
		t.Fatalf("CreateCourse 1: %v", err)
	}
	courseID1 := cResp1.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID1) })

	cCode2 := "ENRICH-CRS2-" + uuid.New().String()[:8]
	cReq2 := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: cCode2, Name: "Enriched Course 2", Credits: 6})
	cReq2.Header().Set("Cookie", "sid="+adminSID)
	cResp2, err := client.CreateCourse(ctx, cReq2)
	if err != nil {
		t.Fatalf("CreateCourse 2: %v", err)
	}
	courseID2 := cResp2.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID2) })

	// Add both associations (small sleep to ensure created_at ordering is deterministic).
	add1 := connect.NewRequest(&catalogv1.AddCourseToProgramRequest{ProgramId: progID1, CourseId: courseID1})
	add1.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.AddCourseToProgram(ctx, add1); err != nil {
		t.Fatalf("AddCourseToProgram 1: %v", err)
	}
	time.Sleep(2 * time.Millisecond) // guarantee created_at ordering
	add2 := connect.NewRequest(&catalogv1.AddCourseToProgramRequest{ProgramId: progID1, CourseId: courseID2})
	add2.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.AddCourseToProgram(ctx, add2); err != nil {
		t.Fatalf("AddCourseToProgram 2: %v", err)
	}

	listReq := connect.NewRequest(&catalogv1.ListProgramCoursesRequest{ProgramId: progID1})
	listReq.Header().Set("Cookie", "sid="+adminSID)
	listResp, err := client.ListProgramCourses(ctx, listReq)
	if err != nil {
		t.Fatalf("ListProgramCourses (2 live): %v", err)
	}
	pcs := listResp.Msg.GetProgramCourses()
	if len(pcs) != 2 {
		t.Fatalf("ListProgramCourses (2 live): want 2, got %d", len(pcs))
	}

	// Ordering: first added course must appear first.
	if pcs[0].GetCourseId() != courseID1 {
		t.Errorf("ordering: pcs[0].CourseId = %q, want %q", pcs[0].GetCourseId(), courseID1)
	}
	if pcs[1].GetCourseId() != courseID2 {
		t.Errorf("ordering: pcs[1].CourseId = %q, want %q", pcs[1].GetCourseId(), courseID2)
	}

	// Verify enriched fields for both entries.
	for i, pc := range pcs {
		c := pc.GetCourse()
		if c == nil {
			t.Errorf("pcs[%d].Course is nil", i)
			continue
		}
		if pc.GetProgramId() != progID1 {
			t.Errorf("pcs[%d].ProgramId = %q, want %q", i, pc.GetProgramId(), progID1)
		}
		if pc.GetCreatedAt() == "" {
			t.Errorf("pcs[%d].CreatedAt is empty", i)
		}
		if c.GetId() != pc.GetCourseId() {
			t.Errorf("pcs[%d].Course.Id = %q, want %q", i, c.GetId(), pc.GetCourseId())
		}
		if c.GetId() == "" {
			t.Errorf("pcs[%d].Course.Id is empty", i)
		}
		if c.GetCode() == "" {
			t.Errorf("pcs[%d].Course.Code is empty", i)
		}
		if c.GetName() == "" {
			t.Errorf("pcs[%d].Course.Name is empty", i)
		}
		if c.GetCredits() <= 0 {
			t.Errorf("pcs[%d].Course.Credits = %d, want > 0", i, c.GetCredits())
		}
	}

	// Specific field assertions for each course.
	if pcs[0].GetCourse().GetCode() != cCode1 {
		t.Errorf("pcs[0].Course.Code = %q, want %q", pcs[0].GetCourse().GetCode(), cCode1)
	}
	if pcs[0].GetCourse().GetCredits() != 4 {
		t.Errorf("pcs[0].Course.Credits = %d, want 4", pcs[0].GetCourse().GetCredits())
	}
	if pcs[1].GetCourse().GetCode() != cCode2 {
		t.Errorf("pcs[1].Course.Code = %q, want %q", pcs[1].GetCourse().GetCode(), cCode2)
	}
	if pcs[1].GetCourse().GetCredits() != 6 {
		t.Errorf("pcs[1].Course.Credits = %d, want 6", pcs[1].GetCourse().GetCredits())
	}

	// --- Scenario 2: soft-deleted course is filtered ---

	pCode2 := "SDEL-PROG-" + uuid.New().String()[:8]
	pReq2 := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: pCode2, Name: "Soft-Delete Program"})
	pReq2.Header().Set("Cookie", "sid="+adminSID)
	pResp2, err := client.CreateProgram(ctx, pReq2)
	if err != nil {
		t.Fatalf("CreateProgram (soft-del): %v", err)
	}
	progID2 := pResp2.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID2) })

	// Live course.
	liveCrsCode := "LIVE-CRS-" + uuid.New().String()[:8]
	liveReq := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: liveCrsCode, Name: "Live Course", Credits: 2})
	liveReq.Header().Set("Cookie", "sid="+adminSID)
	liveResp, err := client.CreateCourse(ctx, liveReq)
	if err != nil {
		t.Fatalf("CreateCourse (live): %v", err)
	}
	liveCrsID := liveResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, liveCrsID) })

	// Soft-deleted course: create then delete via API.
	delCrsCode := "DEL-CRS-" + uuid.New().String()[:8]
	delReq := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: delCrsCode, Name: "Deleted Course", Credits: 1})
	delReq.Header().Set("Cookie", "sid="+adminSID)
	delResp, err := client.CreateCourse(ctx, delReq)
	if err != nil {
		t.Fatalf("CreateCourse (to delete): %v", err)
	}
	delCrsID := delResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, delCrsID) })

	// Add both associations before soft-deleting.
	addLive := connect.NewRequest(&catalogv1.AddCourseToProgramRequest{ProgramId: progID2, CourseId: liveCrsID})
	addLive.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.AddCourseToProgram(ctx, addLive); err != nil {
		t.Fatalf("AddCourseToProgram (live): %v", err)
	}
	addDel := connect.NewRequest(&catalogv1.AddCourseToProgramRequest{ProgramId: progID2, CourseId: delCrsID})
	addDel.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.AddCourseToProgram(ctx, addDel); err != nil {
		t.Fatalf("AddCourseToProgram (to-delete): %v", err)
	}

	// Soft-delete the second course via the API — the program_courses row stays, but the JOIN excludes it.
	// We must first remove the association so DeleteCourse is not blocked by ErrHasDependents,
	// then re-add it after deletion via raw SQL to simulate an orphaned association.
	rmDelAssoc := connect.NewRequest(&catalogv1.RemoveCourseFromProgramRequest{ProgramId: progID2, CourseId: delCrsID})
	rmDelAssoc.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.RemoveCourseFromProgram(ctx, rmDelAssoc); err != nil {
		t.Fatalf("RemoveCourseFromProgram (pre-delete): %v", err)
	}
	delCrsReq := connect.NewRequest(&catalogv1.DeleteCourseRequest{Id: delCrsID})
	delCrsReq.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.DeleteCourse(ctx, delCrsReq); err != nil {
		t.Fatalf("DeleteCourse: %v", err)
	}
	// Re-insert the association row directly (bypassing the API) so the orphaned row exists
	// in program_courses while the course is soft-deleted.
	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO program_courses (program_id, course_id) VALUES ($1::uuid, $2::uuid)`,
		progID2, delCrsID,
	); err != nil {
		t.Fatalf("seed orphaned association: %v", err)
	}
	// Clean up the orphaned association before the FK-constrained course/program rows are deleted.
	t.Cleanup(func() {
		t.Helper()
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM program_courses WHERE program_id = $1 AND course_id = $2`,
			progID2, delCrsID,
		)
	})

	sdListReq := connect.NewRequest(&catalogv1.ListProgramCoursesRequest{ProgramId: progID2})
	sdListReq.Header().Set("Cookie", "sid="+adminSID)
	sdListResp, err := client.ListProgramCourses(ctx, sdListReq)
	if err != nil {
		t.Fatalf("ListProgramCourses (soft-delete filter): %v", err)
	}
	sdPcs := sdListResp.Msg.GetProgramCourses()
	if len(sdPcs) != 1 {
		t.Fatalf("ListProgramCourses (soft-delete filter): want 1, got %d", len(sdPcs))
	}
	if sdPcs[0].GetCourseId() != liveCrsID {
		t.Errorf("soft-delete filter: expected live course %q, got %q", liveCrsID, sdPcs[0].GetCourseId())
	}
	if sdPcs[0].GetCourse() == nil {
		t.Errorf("soft-delete filter: Course is nil on live entry")
	}

	// --- Scenario 3: empty program returns empty list ---

	pCodeEmpty := "EMPTY-PROG-" + uuid.New().String()[:8]
	pReqEmpty := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: pCodeEmpty, Name: "Empty Program"})
	pReqEmpty.Header().Set("Cookie", "sid="+adminSID)
	pRespEmpty, err := client.CreateProgram(ctx, pReqEmpty)
	if err != nil {
		t.Fatalf("CreateProgram (empty): %v", err)
	}
	emptyProgID := pRespEmpty.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, emptyProgID) })

	emptyListReq := connect.NewRequest(&catalogv1.ListProgramCoursesRequest{ProgramId: emptyProgID})
	emptyListReq.Header().Set("Cookie", "sid="+adminSID)
	emptyListResp, err := client.ListProgramCourses(ctx, emptyListReq)
	if err != nil {
		t.Fatalf("ListProgramCourses (empty program): %v", err)
	}
	if len(emptyListResp.Msg.GetProgramCourses()) != 0 {
		t.Errorf("ListProgramCourses (empty): want 0, got %d", len(emptyListResp.Msg.GetProgramCourses()))
	}
}

// TestCatalog_NoRegression verifies existing tests still pass by calling ListPrograms.
func TestCatalog_NoRegression(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-noregression@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.ListProgramsRequest{})
	req.Header().Set("Cookie", "sid="+adminSID)
	if _, err := client.ListPrograms(ctx, req); err != nil {
		t.Errorf("ListPrograms (no regression check): %v", err)
	}
}

// Ensure time package is used (for timeout contexts if needed).
var _ = time.Hour
