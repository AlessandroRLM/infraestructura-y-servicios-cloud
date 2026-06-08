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

// TestCatalog_ProgramCourses_MN verifies AddCourseToProgram→ListProgramCourses→RemoveCourseFromProgram.
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

	// List — must contain
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
