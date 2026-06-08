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
	pgxPool.QueryRow(ctx, `SELECT deleted_at::text FROM programs WHERE id = $1`, progID).Scan(&deletedAt) //nolint:errcheck
	if deletedAt != nil {
		t.Errorf("DeleteProgram (blocked): program row was soft-deleted, expected live")
	}

	// Clean up association so the cleanup can succeed.
	client.RemoveCourseFromProgram(ctx, withSID(connect.NewRequest(&catalogv1.RemoveCourseFromProgramRequest{ //nolint:errcheck
		ProgramId: progID, CourseId: courseID,
	}), adminSID))
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
