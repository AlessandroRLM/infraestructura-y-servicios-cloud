package integration_test

import (
	"context"
	"testing"
)

// TestMigration000017_TeachingPermissionsGranted verifies that migration 000017
// seeds the three teaching-scope permission codes and grants them to both the
// teacher and admin roles.
// S-13: teacher and admin both have all three new permissions after migration 000017.
func TestMigration000017_TeachingPermissionsGranted(t *testing.T) {
	ctx := context.Background()

	// TestMain already applied all migrations including 000017.
	// We assert against the shared database.

	newCodes := []string{
		"section.view_teaching",
		"section_enrollment.view_teaching",
		"profile.view_names",
	}

	t.Run("three_new_permission_codes_exist", func(t *testing.T) {
		for _, code := range newCodes {
			var exists bool
			err := pgxPool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM permissions WHERE code = $1
				)`, code).Scan(&exists)
			if err != nil {
				t.Fatalf("permission existence check %q: %v", code, err)
			}
			if !exists {
				t.Errorf("permission code %q does not exist after migration 000017", code)
			}
		}
	})

	t.Run("teacher_has_three_new_permissions", func(t *testing.T) {
		for _, code := range newCodes {
			var count int
			err := pgxPool.QueryRow(ctx, `
				SELECT COUNT(*) FROM role_permissions rp
				JOIN roles r ON r.id = rp.role_id
				JOIN permissions p ON p.id = rp.permission_id
				WHERE r.name = 'teacher' AND p.code = $1
			`, code).Scan(&count)
			if err != nil {
				t.Fatalf("teacher grant check %q: %v", code, err)
			}
			if count != 1 {
				t.Errorf("teacher role_permissions for %q = %d, want 1", code, count)
			}
		}
	})

	t.Run("admin_has_three_new_permissions", func(t *testing.T) {
		for _, code := range newCodes {
			var count int
			err := pgxPool.QueryRow(ctx, `
				SELECT COUNT(*) FROM role_permissions rp
				JOIN roles r ON r.id = rp.role_id
				JOIN permissions p ON p.id = rp.permission_id
				WHERE r.name = 'admin' AND p.code = $1
			`, code).Scan(&count)
			if err != nil {
				t.Fatalf("admin grant check %q: %v", code, err)
			}
			if count != 1 {
				t.Errorf("admin role_permissions for %q = %d, want 1", code, count)
			}
		}
	})

	t.Run("student_does_not_have_new_permissions", func(t *testing.T) {
		for _, code := range newCodes {
			var count int
			err := pgxPool.QueryRow(ctx, `
				SELECT COUNT(*) FROM role_permissions rp
				JOIN roles r ON r.id = rp.role_id
				JOIN permissions p ON p.id = rp.permission_id
				WHERE r.name = 'student' AND p.code = $1
			`, code).Scan(&count)
			if err != nil {
				t.Fatalf("student grant check %q: %v", code, err)
			}
			if count != 0 {
				t.Errorf("student should NOT have %q, but count = %d", code, count)
			}
		}
	})

	t.Run("section_teachers_teacher_id_idx_exists", func(t *testing.T) {
		var exists bool
		err := pgxPool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public'
				  AND tablename  = 'section_teachers'
				  AND indexname  = 'section_teachers_teacher_id_idx'
			)
		`).Scan(&exists)
		if err != nil {
			t.Fatalf("index existence check: %v", err)
		}
		if !exists {
			t.Error("index section_teachers_teacher_id_idx does not exist after migration 000017")
		}
	})
}
