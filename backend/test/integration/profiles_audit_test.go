package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
)

// TestProfilesAudit_CreatedByPopulatedOnInsert verifies that created_by is set to the acting admin's
// user_id when a user profile is first created.
func TestProfilesAudit_CreatedByPopulatedOnInsert(t *testing.T) {
	ctx := context.Background()
	targetID := seedUserWithRole(t, "audit-target-insert@profiles.test", "student")
	cleanupUserProfile(t, targetID)
	adminID, adminSID := seedUserWithSession(t, "audit-admin-insert@profiles.test", "admin")

	client := newProfilesClient(nil)

	nationalID := "AUDIT-INSERT-" + targetID.String()[:8]
	req := &profilesv1.UpsertUserProfileRequest{
		UserId:           targetID.String(),
		GivenNames:       "Audit",
		LastNamePaternal: "Insert",
		NationalIdType:   "RUT",
		NationalId:       nationalID,
	}
	_, err := client.UpsertUserProfile(ctx, withSession(connect.NewRequest(req), adminSID))
	if err != nil {
		t.Fatalf("UpsertUserProfile (insert): %v", err)
	}

	var createdByID uuid.UUID
	err = pgxPool.QueryRow(ctx,
		`SELECT created_by FROM user_profiles WHERE user_id = $1`, targetID,
	).Scan(&createdByID)
	if err != nil {
		t.Fatalf("SELECT created_by: %v", err)
	}
	if createdByID != adminID {
		t.Errorf("created_by = %v, want admin user_id %v", createdByID, adminID)
	}
}

// TestProfilesAudit_UpdatedByPopulatedOnUpdate verifies that updated_by is set on the update path
// and that created_by is preserved from the first insert.
// Two distinct admins are used: admin1 performs the initial insert, admin2 performs the update.
// This detects regressions where the conflict/update path overwrites created_by.
func TestProfilesAudit_UpdatedByPopulatedOnUpdate(t *testing.T) {
	ctx := context.Background()
	targetID := seedUserWithRole(t, "audit-target-update@profiles.test", "student")
	cleanupUserProfile(t, targetID)
	admin1ID, admin1SID := seedUserWithSession(t, "audit-admin1-update@profiles.test", "admin")
	admin2ID, admin2SID := seedUserWithSession(t, "audit-admin2-update@profiles.test", "admin")

	client := newProfilesClient(nil)

	nationalID := "AUDIT-UPDATE-" + targetID.String()[:8]

	// admin1 performs the initial insert.
	insertReq := &profilesv1.UpsertUserProfileRequest{
		UserId:           targetID.String(),
		GivenNames:       "InsertPass",
		LastNamePaternal: "Update",
		NationalIdType:   "RUT",
		NationalId:       nationalID,
	}
	if _, err := client.UpsertUserProfile(ctx, withSession(connect.NewRequest(insertReq), admin1SID)); err != nil {
		t.Fatalf("UpsertUserProfile (insert, admin1): %v", err)
	}

	// admin2 performs the update on the same row.
	updateReq := &profilesv1.UpsertUserProfileRequest{
		UserId:           targetID.String(),
		GivenNames:       "UpdatePass",
		LastNamePaternal: "Update",
		NationalIdType:   "RUT",
		NationalId:       nationalID,
	}
	if _, err := client.UpsertUserProfile(ctx, withSession(connect.NewRequest(updateReq), admin2SID)); err != nil {
		t.Fatalf("UpsertUserProfile (update, admin2): %v", err)
	}

	var createdByID, updatedByID uuid.UUID
	err := pgxPool.QueryRow(ctx,
		`SELECT created_by, updated_by FROM user_profiles WHERE user_id = $1`, targetID,
	).Scan(&createdByID, &updatedByID)
	if err != nil {
		t.Fatalf("SELECT created_by, updated_by: %v", err)
	}
	// created_by must still be admin1 — the conflict path must not overwrite it.
	if createdByID != admin1ID {
		t.Errorf("created_by after update = %v, want admin1 %v (should be preserved from insert)", createdByID, admin1ID)
	}
	// updated_by must be admin2 — the actor who performed the update.
	if updatedByID != admin2ID {
		t.Errorf("updated_by = %v, want admin2 %v", updatedByID, admin2ID)
	}
}

// TestProfilesAudit_StudentAndTeacherProfilesSetAuditColumns verifies the same audit pattern
// for student_profiles, teacher_profiles, and teacher_qualifications.
func TestProfilesAudit_StudentAndTeacherProfilesSetAuditColumns(t *testing.T) {
	ctx := context.Background()
	targetID := seedUserWithRole(t, "audit-multi@profiles.test", "teacher")
	cleanupTeacherProfile(t, targetID)
	adminID, adminSID := seedUserWithSession(t, "audit-admin-multi@profiles.test", "admin")

	client := newProfilesClient(nil)

	t.Run("student_profiles_created_by", func(t *testing.T) {
		studentTargetID := seedUserWithRole(t, "audit-student-multi@profiles.test", "student")
		cleanupStudentProfile(t, studentTargetID)

		req := &profilesv1.UpsertStudentProfileRequest{
			UserId:        studentTargetID.String(),
			AdmissionYear: 2024,
		}
		_, err := client.UpsertStudentProfile(ctx, withSession(connect.NewRequest(req), adminSID))
		if err != nil {
			t.Fatalf("UpsertStudentProfile: %v", err)
		}

		var createdByID uuid.UUID
		if err := pgxPool.QueryRow(ctx,
			`SELECT created_by FROM student_profiles WHERE user_id = $1`, studentTargetID,
		).Scan(&createdByID); err != nil {
			t.Fatalf("SELECT created_by from student_profiles: %v", err)
		}
		if createdByID != adminID {
			t.Errorf("student_profiles created_by = %v, want %v", createdByID, adminID)
		}
	})

	t.Run("teacher_profiles_created_by", func(t *testing.T) {
		_, err := client.UpsertTeacherProfile(ctx, withSession(connect.NewRequest(&profilesv1.UpsertTeacherProfileRequest{
			UserId: targetID.String(),
		}), adminSID))
		if err != nil {
			t.Fatalf("UpsertTeacherProfile: %v", err)
		}

		var createdByID uuid.UUID
		if err := pgxPool.QueryRow(ctx,
			`SELECT created_by FROM teacher_profiles WHERE user_id = $1`, targetID,
		).Scan(&createdByID); err != nil {
			t.Fatalf("SELECT created_by from teacher_profiles: %v", err)
		}
		if createdByID != adminID {
			t.Errorf("teacher_profiles created_by = %v, want %v", createdByID, adminID)
		}
	})

	t.Run("teacher_qualifications_created_by", func(t *testing.T) {
		req := &profilesv1.AddTeacherQualificationRequest{
			TeacherId: targetID.String(),
			Degree:    "MSc",
			Year:      2018,
		}
		_, err := client.AddTeacherQualification(ctx, withSession(connect.NewRequest(req), adminSID))
		if err != nil {
			t.Fatalf("AddTeacherQualification: %v", err)
		}

		var createdByID uuid.UUID
		if err := pgxPool.QueryRow(ctx,
			`SELECT created_by FROM teacher_qualifications WHERE teacher_id = $1 ORDER BY created_at LIMIT 1`, targetID,
		).Scan(&createdByID); err != nil {
			t.Fatalf("SELECT created_by from teacher_qualifications: %v", err)
		}
		if createdByID != adminID {
			t.Errorf("teacher_qualifications created_by = %v, want %v", createdByID, adminID)
		}
	})
}
