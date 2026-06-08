package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
)

// TestCatalog_DeleteProgram_BlockedByProgramCourses verifies CodeFailedPrecondition.
func TestCatalog_DeleteProgram_BlockedByProgramCourses(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-del-prog-pc@catalog.test")
	client := newCatalogClient(nil)

	// Create program and course, then associate.
	pResp, _ := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: "DEL-PC-P-" + uuid.New().String()[:8], Name: "P",
	}), adminSID))
	progID := pResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID) })

	cResp, _ := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code: "DEL-PC-C-" + uuid.New().String()[:8], Name: "C", Credits: 3,
	}), adminSID))
	courseID := cResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID) })

	// Associate
	if _, err := client.AddCourseToProgram(ctx, withSID(connect.NewRequest(&catalogv1.AddCourseToProgramRequest{
		ProgramId: progID, CourseId: courseID,
	}), adminSID)); err != nil {
		t.Fatalf("AddCourseToProgram: %v", err)
	}

	// Attempt to delete — blocked
	_, err := client.DeleteProgram(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramRequest{Id: progID}), adminSID))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	// Program row must still be live.
	var deletedAt *string
	if err := pgxPool.QueryRow(ctx, `SELECT deleted_at::text FROM programs WHERE id = $1`, progID).Scan(&deletedAt); err != nil {
		t.Fatalf("SELECT deleted_at from programs: %v", err)
	}
	if deletedAt != nil {
		t.Errorf("DeleteProgram (blocked): program row was soft-deleted, expected live")
	}

	// Clean up association so the cleanup can succeed.
	if _, err := client.RemoveCourseFromProgram(ctx, withSID(connect.NewRequest(&catalogv1.RemoveCourseFromProgramRequest{
		ProgramId: progID, CourseId: courseID,
	}), adminSID)); err != nil {
		t.Logf("RemoveCourseFromProgram (cleanup): %v", err)
	}
}

// TestCatalog_DeleteProgram_BlockedByLiveQuota then UnblockedAfterQuotaDeleted.
func TestCatalog_DeleteProgram_BlockedByLiveQuotaThenAllowed(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-del-prog-quota@catalog.test")
	client := newCatalogClient(nil)

	pResp, _ := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: "DEL-Q-P-" + uuid.New().String()[:8], Name: "P",
	}), adminSID))
	progID := pResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID) })

	// Create live quota
	qResp, _ := client.CreateProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramQuotaRequest{
		ProgramId: progID, Year: 2030, AdmissionQuota: 40,
	}), adminSID))
	quotaID := qResp.Msg.GetId()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM program_quotas WHERE id = $1`, quotaID)
	})

	// Delete program — blocked
	_, err := client.DeleteProgram(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramRequest{Id: progID}), adminSID))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	// Soft-delete the quota
	if _, err := client.DeleteProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramQuotaRequest{Id: quotaID}), adminSID)); err != nil {
		t.Fatalf("DeleteProgramQuota: %v", err)
	}

	// Delete program — now succeeds
	if _, err := client.DeleteProgram(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramRequest{Id: progID}), adminSID)); err != nil {
		t.Errorf("DeleteProgram (after quota deleted): %v", err)
	}
}

// TestCatalog_DeleteCourse_BlockedByProgramCourses verifies that deleting a course
// that still has a live program association returns CodeFailedPrecondition, and that
// deleting succeeds once the association is removed.
func TestCatalog_DeleteCourse_BlockedByProgramCourses(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-del-course-pc@catalog.test")
	client := newCatalogClient(nil)

	pResp, err := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: "DEL-CPC-P-" + uuid.New().String()[:8], Name: "P",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	progID := pResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID) })

	cResp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code: "DEL-CPC-C-" + uuid.New().String()[:8], Name: "C", Credits: 3,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	courseID := cResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID) })

	// Associate course to program.
	if _, err := client.AddCourseToProgram(ctx, withSID(connect.NewRequest(&catalogv1.AddCourseToProgramRequest{
		ProgramId: progID, CourseId: courseID,
	}), adminSID)); err != nil {
		t.Fatalf("AddCourseToProgram: %v", err)
	}

	// Delete course while associated — must be blocked.
	_, err = client.DeleteCourse(ctx, withSID(connect.NewRequest(&catalogv1.DeleteCourseRequest{Id: courseID}), adminSID))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	// Remove association.
	if _, err := client.RemoveCourseFromProgram(ctx, withSID(connect.NewRequest(&catalogv1.RemoveCourseFromProgramRequest{
		ProgramId: progID, CourseId: courseID,
	}), adminSID)); err != nil {
		t.Fatalf("RemoveCourseFromProgram: %v", err)
	}

	// Delete course after association removed — must succeed.
	if _, err := client.DeleteCourse(ctx, withSID(connect.NewRequest(&catalogv1.DeleteCourseRequest{Id: courseID}), adminSID)); err != nil {
		t.Errorf("DeleteCourse (after association removed): %v", err)
	}
}

// TestCatalog_DeleteProgram_AbsentID_NotFound verifies that deleting a non-existent
// or already-deleted program returns CodeNotFound.
func TestCatalog_DeleteProgram_AbsentID_NotFound(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-del-absent-prog@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.DeleteProgram(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramRequest{
		Id: uuid.New().String(),
	}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestCatalog_DeleteCourse_AbsentID_NotFound verifies CodeNotFound for absent course.
func TestCatalog_DeleteCourse_AbsentID_NotFound(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-del-absent-course@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.DeleteCourse(ctx, withSID(connect.NewRequest(&catalogv1.DeleteCourseRequest{
		Id: uuid.New().String(),
	}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestCatalog_DeleteAcademicPeriod_AbsentID_NotFound verifies CodeNotFound for absent period.
func TestCatalog_DeleteAcademicPeriod_AbsentID_NotFound(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-del-absent-ap@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.DeleteAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.DeleteAcademicPeriodRequest{
		Id: uuid.New().String(),
	}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestCatalog_DeleteProgramQuota_AbsentID_NotFound verifies CodeNotFound for absent quota.
func TestCatalog_DeleteProgramQuota_AbsentID_NotFound(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-del-absent-quota@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.DeleteProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramQuotaRequest{
		Id: uuid.New().String(),
	}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestCatalog_RemoveCourseFromProgram_AbsentAssociation_NotFound verifies CodeNotFound
// when the association does not exist.
func TestCatalog_RemoveCourseFromProgram_AbsentAssociation_NotFound(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-remove-absent-assoc@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.RemoveCourseFromProgram(ctx, withSID(connect.NewRequest(&catalogv1.RemoveCourseFromProgramRequest{
		ProgramId: uuid.New().String(),
		CourseId:  uuid.New().String(),
	}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestCatalog_DeleteProgramQuota_NoDependents always succeeds.
func TestCatalog_DeleteProgramQuota_NoDependents(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-del-quota-ok@catalog.test")
	client := newCatalogClient(nil)

	pResp, _ := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: "DEL-QND-P-" + uuid.New().String()[:8], Name: "P",
	}), adminSID))
	progID := pResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID) })

	qResp, _ := client.CreateProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramQuotaRequest{
		ProgramId: progID, Year: 2031, AdmissionQuota: 20,
	}), adminSID))
	quotaID := qResp.Msg.GetId()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM program_quotas WHERE id = $1`, quotaID)
	})

	// Delete quota — OK
	if _, err := client.DeleteProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramQuotaRequest{Id: quotaID}), adminSID)); err != nil {
		t.Errorf("DeleteProgramQuota: %v", err)
	}

	// Get quota — NotFound
	_, err := client.GetProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.GetProgramQuotaRequest{Id: quotaID}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}
