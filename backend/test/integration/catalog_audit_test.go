package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
)

// TestCatalog_Audit_ProgramCreatedBySet verifies created_by is set from the session actor.
func TestCatalog_Audit_ProgramCreatedBySet(t *testing.T) {
	ctx := context.Background()
	adminID := seedUserWithRole(t, "catalog-audit-prog@catalog.test", "admin")
	adminSID := seedSessionInRedis(t, adminID, time.Hour)
	client := newCatalogClient(nil)

	code := "AUDIT-PROG-" + uuid.New().String()[:8]
	resp, err := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: code, Name: "Audit Program",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	id := resp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, id) })

	var createdBy, updatedBy string
	err = pgxPool.QueryRow(ctx,
		`SELECT created_by::text, updated_by::text FROM programs WHERE id = $1`,
		id,
	).Scan(&createdBy, &updatedBy)
	if err != nil {
		t.Fatalf("SELECT audit cols: %v", err)
	}
	if createdBy != adminID.String() {
		t.Errorf("programs.created_by = %q, want %q", createdBy, adminID.String())
	}
	if updatedBy != adminID.String() {
		t.Errorf("programs.updated_by = %q, want %q", updatedBy, adminID.String())
	}
}

// TestCatalog_Audit_AcademicPeriodNoCreatedBy verifies academic_periods has no created_by column.
func TestCatalog_Audit_AcademicPeriodNoCreatedBy(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-audit-ap-col@catalog.test")
	client := newCatalogClient(nil)

	resp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year: 5000, Term: 1, StartDate: "5000-03-01", EndDate: "5000-07-31",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateAcademicPeriod: %v", err)
	}
	id := resp.Msg.GetId()
	t.Cleanup(func() { cleanupAcademicPeriod(t, id) })

	// Verify column absence at the DB level by checking information_schema.
	var columnExists bool
	err = pgxPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'academic_periods' AND column_name = 'created_by'
		)
	`).Scan(&columnExists)
	if err != nil {
		t.Fatalf("check column existence: %v", err)
	}
	if columnExists {
		t.Error("academic_periods must NOT have a created_by column per §10.1")
	}

	// Also verify updated_by absence.
	err = pgxPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'academic_periods' AND column_name = 'updated_by'
		)
	`).Scan(&columnExists)
	if err != nil {
		t.Fatalf("check updated_by existence: %v", err)
	}
	if columnExists {
		t.Error("academic_periods must NOT have an updated_by column per §10.1")
	}
}

// TestCatalog_Audit_SoftDeleteProgram_UpdatedBySet verifies that soft-deleting a program sets updated_by.
func TestCatalog_Audit_SoftDeleteProgram_UpdatedBySet(t *testing.T) {
	ctx := context.Background()
	adminID := seedUserWithRole(t, "catalog-audit-del-prog@catalog.test", "admin")
	adminSID := seedSessionInRedis(t, adminID, time.Hour)
	client := newCatalogClient(nil)

	resp, err := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: "AUDIT-DEL-P-" + uuid.New().String()[:8], Name: "Audit Delete Program",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	id := resp.Msg.GetId()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, id)
	})

	if _, err := client.DeleteProgram(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramRequest{Id: id}), adminSID)); err != nil {
		t.Fatalf("DeleteProgram: %v", err)
	}

	var updatedBy string
	if err := pgxPool.QueryRow(ctx, `SELECT updated_by::text FROM programs WHERE id = $1`, id).Scan(&updatedBy); err != nil {
		t.Fatalf("SELECT programs.updated_by after soft-delete: %v", err)
	}
	if updatedBy != adminID.String() {
		t.Errorf("programs.updated_by after soft-delete = %q, want %q", updatedBy, adminID.String())
	}
}

// TestCatalog_Audit_SoftDeleteCourse_UpdatedBySet verifies that soft-deleting a course sets updated_by.
func TestCatalog_Audit_SoftDeleteCourse_UpdatedBySet(t *testing.T) {
	ctx := context.Background()
	adminID := seedUserWithRole(t, "catalog-audit-del-course@catalog.test", "admin")
	adminSID := seedSessionInRedis(t, adminID, time.Hour)
	client := newCatalogClient(nil)

	resp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code: "AUDIT-DEL-C-" + uuid.New().String()[:8], Name: "Audit Delete Course", Credits: 3,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	id := resp.Msg.GetId()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM courses WHERE id = $1`, id)
	})

	if _, err := client.DeleteCourse(ctx, withSID(connect.NewRequest(&catalogv1.DeleteCourseRequest{Id: id}), adminSID)); err != nil {
		t.Fatalf("DeleteCourse: %v", err)
	}

	var updatedBy string
	if err := pgxPool.QueryRow(ctx, `SELECT updated_by::text FROM courses WHERE id = $1`, id).Scan(&updatedBy); err != nil {
		t.Fatalf("SELECT courses.updated_by after soft-delete: %v", err)
	}
	if updatedBy != adminID.String() {
		t.Errorf("courses.updated_by after soft-delete = %q, want %q", updatedBy, adminID.String())
	}
}

// TestCatalog_Audit_SoftDelete_UpdatedAtBumped verifies that every soft-delete operation advances
// updated_at past created_at, satisfying the §10.1 rule that every update sets updated_at=now().
func TestCatalog_Audit_SoftDelete_UpdatedAtBumped(t *testing.T) {
	ctx := context.Background()
	adminID := seedUserWithRole(t, "catalog-audit-updat@catalog.test", "admin")
	adminSID := seedSessionInRedis(t, adminID, time.Hour)
	client := newCatalogClient(nil)

	t.Run("program", func(t *testing.T) {
		resp, err := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
			Code: "UDAT-P-" + uuid.New().String()[:8], Name: "UpdatedAt Program",
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateProgram: %v", err)
		}
		id := resp.Msg.GetId()
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, id)
		})

		var createdAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT created_at FROM programs WHERE id = $1`, id).Scan(&createdAt); err != nil {
			t.Fatalf("SELECT created_at: %v", err)
		}

		// Sleep ensures the delete transaction's now() is strictly later than the insert's now().
		time.Sleep(2 * time.Millisecond)

		if _, err := client.DeleteProgram(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramRequest{Id: id}), adminSID)); err != nil {
			t.Fatalf("DeleteProgram: %v", err)
		}

		var updatedAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT updated_at FROM programs WHERE id = $1`, id).Scan(&updatedAt); err != nil {
			t.Fatalf("SELECT updated_at after soft-delete: %v", err)
		}
		if updatedAt.IsZero() {
			t.Fatal("programs.updated_at is zero after soft-delete")
		}
		if !updatedAt.After(createdAt) {
			t.Errorf("programs.updated_at not bumped on soft-delete: created_at=%v updated_at=%v", createdAt, updatedAt)
		}
	})

	t.Run("course", func(t *testing.T) {
		resp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
			Code: "UDAT-C-" + uuid.New().String()[:8], Name: "UpdatedAt Course", Credits: 3,
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateCourse: %v", err)
		}
		id := resp.Msg.GetId()
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM courses WHERE id = $1`, id)
		})

		var createdAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT created_at FROM courses WHERE id = $1`, id).Scan(&createdAt); err != nil {
			t.Fatalf("SELECT created_at: %v", err)
		}

		// Sleep ensures the delete transaction's now() is strictly later than the insert's now().
		time.Sleep(2 * time.Millisecond)

		if _, err := client.DeleteCourse(ctx, withSID(connect.NewRequest(&catalogv1.DeleteCourseRequest{Id: id}), adminSID)); err != nil {
			t.Fatalf("DeleteCourse: %v", err)
		}

		var updatedAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT updated_at FROM courses WHERE id = $1`, id).Scan(&updatedAt); err != nil {
			t.Fatalf("SELECT updated_at after soft-delete: %v", err)
		}
		if updatedAt.IsZero() {
			t.Fatal("courses.updated_at is zero after soft-delete")
		}
		if !updatedAt.After(createdAt) {
			t.Errorf("courses.updated_at not bumped on soft-delete: created_at=%v updated_at=%v", createdAt, updatedAt)
		}
	})

	t.Run("academic_period", func(t *testing.T) {
		resp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
			Year: 6000, Term: 1, StartDate: "6000-03-01", EndDate: "6000-07-31",
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateAcademicPeriod: %v", err)
		}
		id := resp.Msg.GetId()
		t.Cleanup(func() { cleanupAcademicPeriod(t, id) })

		var createdAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT created_at FROM academic_periods WHERE id = $1`, id).Scan(&createdAt); err != nil {
			t.Fatalf("SELECT created_at: %v", err)
		}

		// Sleep ensures the delete transaction's now() is strictly later than the insert's now().
		time.Sleep(2 * time.Millisecond)

		if _, err := client.DeleteAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.DeleteAcademicPeriodRequest{Id: id}), adminSID)); err != nil {
			t.Fatalf("DeleteAcademicPeriod: %v", err)
		}

		var updatedAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT updated_at FROM academic_periods WHERE id = $1`, id).Scan(&updatedAt); err != nil {
			t.Fatalf("SELECT updated_at after soft-delete: %v", err)
		}
		if updatedAt.IsZero() {
			t.Fatal("academic_periods.updated_at is zero after soft-delete")
		}
		if !updatedAt.After(createdAt) {
			t.Errorf("academic_periods.updated_at not bumped on soft-delete: created_at=%v updated_at=%v", createdAt, updatedAt)
		}
	})

	t.Run("program_quota", func(t *testing.T) {
		pResp, err := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
			Code: "UDAT-PQ-P-" + uuid.New().String()[:8], Name: "P",
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateProgram: %v", err)
		}
		progID := pResp.Msg.GetId()
		t.Cleanup(func() { cleanupProgram(t, progID) })

		qResp, err := client.CreateProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramQuotaRequest{
			ProgramId: progID, Year: 7001, AdmissionQuota: 15,
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateProgramQuota: %v", err)
		}
		quotaID := qResp.Msg.GetId()
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM program_quotas WHERE id = $1`, quotaID)
		})

		var createdAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT created_at FROM program_quotas WHERE id = $1`, quotaID).Scan(&createdAt); err != nil {
			t.Fatalf("SELECT created_at: %v", err)
		}

		// Sleep ensures the delete transaction's now() is strictly later than the insert's now().
		time.Sleep(2 * time.Millisecond)

		if _, err := client.DeleteProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.DeleteProgramQuotaRequest{Id: quotaID}), adminSID)); err != nil {
			t.Fatalf("DeleteProgramQuota: %v", err)
		}

		var updatedAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT updated_at FROM program_quotas WHERE id = $1`, quotaID).Scan(&updatedAt); err != nil {
			t.Fatalf("SELECT updated_at after soft-delete: %v", err)
		}
		if updatedAt.IsZero() {
			t.Fatal("program_quotas.updated_at is zero after soft-delete")
		}
		if !updatedAt.After(createdAt) {
			t.Errorf("program_quotas.updated_at not bumped on soft-delete: created_at=%v updated_at=%v", createdAt, updatedAt)
		}
	})

	t.Run("section", func(t *testing.T) {
		pResp, err := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
			Code: "UDAT-SEC-P-" + uuid.New().String()[:8], Name: "P",
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateProgram: %v", err)
		}
		progID := pResp.Msg.GetId()
		t.Cleanup(func() { cleanupProgram(t, progID) })

		cResp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
			Code: "UDAT-SEC-C-" + uuid.New().String()[:8], Name: "C", Credits: 2,
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateCourse: %v", err)
		}
		courseID := cResp.Msg.GetId()
		t.Cleanup(func() { cleanupCourse(t, courseID) })

		apResp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
			Year: 7002, Term: 1, StartDate: "7002-03-01", EndDate: "7002-07-31",
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateAcademicPeriod: %v", err)
		}
		apID := apResp.Msg.GetId()
		t.Cleanup(func() { cleanupAcademicPeriod(t, apID) })

		secResp, err := client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
			CourseId: courseID, AcademicPeriodId: apID, SeatCapacity: 30,
		}), adminSID))
		if err != nil {
			t.Fatalf("CreateSection: %v", err)
		}
		secID := secResp.Msg.GetId()
		t.Cleanup(func() {
			_, _ = pgxPool.Exec(context.Background(), `DELETE FROM sections WHERE id = $1`, secID)
		})

		var createdAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT created_at FROM sections WHERE id = $1`, secID).Scan(&createdAt); err != nil {
			t.Fatalf("SELECT created_at: %v", err)
		}

		// Sleep ensures the delete transaction's now() is strictly later than the insert's now().
		time.Sleep(2 * time.Millisecond)

		if _, err := client.DeleteSection(ctx, withSID(connect.NewRequest(&catalogv1.DeleteSectionRequest{Id: secID}), adminSID)); err != nil {
			t.Fatalf("DeleteSection: %v", err)
		}

		var updatedAt time.Time
		if err := pgxPool.QueryRow(ctx, `SELECT updated_at FROM sections WHERE id = $1`, secID).Scan(&updatedAt); err != nil {
			t.Fatalf("SELECT updated_at after soft-delete: %v", err)
		}
		if updatedAt.IsZero() {
			t.Fatal("sections.updated_at is zero after soft-delete")
		}
		if !updatedAt.After(createdAt) {
			t.Errorf("sections.updated_at not bumped on soft-delete: created_at=%v updated_at=%v", createdAt, updatedAt)
		}
	})
}

// TestCatalog_Audit_ProgramQuotaCreatedBySet verifies created_by for program_quotas.
func TestCatalog_Audit_ProgramQuotaCreatedBySet(t *testing.T) {
	ctx := context.Background()
	adminID := seedUserWithRole(t, "catalog-audit-quota@catalog.test", "admin")
	adminSID := seedSessionInRedis(t, adminID, time.Hour)
	client := newCatalogClient(nil)

	pResp, err := client.CreateProgram(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramRequest{
		Code: "AUDIT-Q-P-" + uuid.New().String()[:8], Name: "P",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	progID := pResp.Msg.GetId()
	t.Cleanup(func() { cleanupProgram(t, progID) })

	qResp, err := client.CreateProgramQuota(ctx, withSID(connect.NewRequest(&catalogv1.CreateProgramQuotaRequest{
		ProgramId: progID, Year: 2099, AdmissionQuota: 30,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateProgramQuota: %v", err)
	}
	quotaID := qResp.Msg.GetId()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM program_quotas WHERE id = $1`, quotaID)
	})

	var createdBy string
	err = pgxPool.QueryRow(ctx, `SELECT created_by::text FROM program_quotas WHERE id = $1`, quotaID).Scan(&createdBy)
	if err != nil {
		t.Fatalf("SELECT program_quotas.created_by: %v", err)
	}
	if createdBy != adminID.String() {
		t.Errorf("program_quotas.created_by = %q, want %q", createdBy, adminID.String())
	}
}
