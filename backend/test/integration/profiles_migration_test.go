package integration_test

import (
	"context"
	"testing"
)

// TestProfilesMigration_TablesAndSeed verifies that the profiles DDL and seed migrations
// apply correctly and produce the expected schema and data.
func TestProfilesMigration_TablesAndSeed(t *testing.T) {
	ctx := context.Background()

	// TestMain already applied all migrations including 000005 and 000006.

	t.Run("four_profile_tables_exist", func(t *testing.T) {
		tables := []string{
			"user_profiles",
			"student_profiles",
			"teacher_profiles",
			"teacher_qualifications",
		}
		for _, tbl := range tables {
			var exists bool
			err := pgxPool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.tables
					WHERE table_schema = 'public' AND table_name = $1
				)`, tbl).Scan(&exists)
			if err != nil {
				t.Fatalf("table check %q: %v", tbl, err)
			}
			if !exists {
				t.Errorf("table %q does not exist", tbl)
			}
		}
	})

	t.Run("user_profiles_national_id_has_unique_constraint", func(t *testing.T) {
		var count int
		err := pgxPool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
			  ON kcu.constraint_name = tc.constraint_name
			  AND kcu.table_schema = tc.table_schema
			WHERE tc.table_schema = 'public'
			  AND tc.table_name   = 'user_profiles'
			  AND tc.constraint_type = 'UNIQUE'
			  AND kcu.column_name = 'national_id'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("unique constraint check: %v", err)
		}
		if count == 0 {
			t.Error("user_profiles.national_id does not have a UNIQUE constraint")
		}
	})

	t.Run("teacher_qualifications_teacher_id_is_fk_to_teacher_profiles", func(t *testing.T) {
		var count int
		err := pgxPool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM information_schema.referential_constraints rc
			JOIN information_schema.key_column_usage kcu
			  ON kcu.constraint_name = rc.constraint_name
			  AND kcu.table_schema = rc.constraint_schema
			WHERE rc.constraint_schema = 'public'
			  AND kcu.table_name = 'teacher_qualifications'
			  AND kcu.column_name = 'teacher_id'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("FK check: %v", err)
		}
		if count == 0 {
			t.Error("teacher_qualifications.teacher_id does not have an FK to teacher_profiles(user_id)")
		}
	})

	t.Run("all_profile_tables_have_audit_columns", func(t *testing.T) {
		tables := []string{
			"user_profiles",
			"student_profiles",
			"teacher_profiles",
			"teacher_qualifications",
		}
		auditCols := []string{"created_at", "updated_at", "deleted_at", "created_by", "updated_by"}
		for _, tbl := range tables {
			for _, col := range auditCols {
				var exists bool
				err := pgxPool.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM information_schema.columns
						WHERE table_schema = 'public'
						  AND table_name   = $1
						  AND column_name  = $2
					)`, tbl, col).Scan(&exists)
				if err != nil {
					t.Fatalf("audit column check %q.%q: %v", tbl, col, err)
				}
				if !exists {
					t.Errorf("table %q is missing audit column %q", tbl, col)
				}
			}
		}
	})

	t.Run("permissions_count_is_14_after_seed", func(t *testing.T) {
		var count int
		if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM permissions`).Scan(&count); err != nil {
			t.Fatalf("count permissions: %v", err)
		}
		if count != 14 {
			t.Errorf("permissions count = %d, want 14", count)
		}
	})

	t.Run("profile_view_own_assigned_to_all_three_roles", func(t *testing.T) {
		rows, err := pgxPool.Query(ctx, `
			SELECT r.name FROM roles r
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE p.code = 'profile.view_own'
			ORDER BY r.name
		`)
		if err != nil {
			t.Fatalf("query role assignments: %v", err)
		}
		defer rows.Close()
		got := map[string]struct{}{}
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatalf("scan role name: %v", err)
			}
			got[name] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows error: %v", err)
		}
		want := []string{"admin", "teacher", "student"}
		for _, role := range want {
			if _, ok := got[role]; !ok {
				t.Errorf("profile.view_own not assigned to role %q", role)
			}
		}
	})

	t.Run("profile_seed_is_idempotent", func(t *testing.T) {
		_, err := pgxPool.Exec(ctx, `
			INSERT INTO permissions (code, description)
			VALUES ('profile.view_own', 'View own personal profile')
			ON CONFLICT (code) DO NOTHING
		`)
		if err != nil {
			t.Fatalf("idempotent permission re-insert: %v", err)
		}

		var count int
		if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM permissions`).Scan(&count); err != nil {
			t.Fatalf("count permissions after re-seed: %v", err)
		}
		if count != 14 {
			t.Errorf("permissions count after re-seed = %d, want 14", count)
		}
	})
}
