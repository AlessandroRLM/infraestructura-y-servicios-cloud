package integration_test

import (
	"context"
	"testing"
)

// TestRBACMigration_TablesAndSeed verifies that the RBAC DDL and seed migrations
// apply correctly and produce the expected schema and data.
func TestRBACMigration_TablesAndSeed(t *testing.T) {
	ctx := context.Background()

	// The shared test server's TestMain already applied all migrations, including
	// 000003 and 000004. We run assertions against that shared database.

	t.Run("four_rbac_tables_exist", func(t *testing.T) {
		tables := []string{"roles", "permissions", "role_permissions", "user_roles"}
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

	t.Run("join_tables_have_only_created_at", func(t *testing.T) {
		joinTables := []string{"role_permissions", "user_roles"}
		disallowed := []string{"updated_at", "deleted_at", "version"}
		for _, tbl := range joinTables {
			for _, col := range disallowed {
				var exists bool
				err := pgxPool.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM information_schema.columns
						WHERE table_schema = 'public'
						  AND table_name   = $1
						  AND column_name  = $2
					)`, tbl, col).Scan(&exists)
				if err != nil {
					t.Fatalf("column check %q.%q: %v", tbl, col, err)
				}
				if exists {
					t.Errorf("join table %q must not have column %q", tbl, col)
				}
			}
			// created_at must exist
			var hasCreatedAt bool
			err := pgxPool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_schema = 'public'
					  AND table_name   = $1
					  AND column_name  = 'created_at'
				)`, tbl).Scan(&hasCreatedAt)
			if err != nil {
				t.Fatalf("created_at check for %q: %v", tbl, err)
			}
			if !hasCreatedAt {
				t.Errorf("join table %q is missing created_at", tbl)
			}
		}
	})

	t.Run("roles_count_is_3", func(t *testing.T) {
		var count int
		if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM roles`).Scan(&count); err != nil {
			t.Fatalf("count roles: %v", err)
		}
		if count != 3 {
			t.Errorf("roles count = %d, want 3", count)
		}
	})

	t.Run("permissions_count_is_13", func(t *testing.T) {
		var count int
		if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM permissions`).Scan(&count); err != nil {
			t.Fatalf("count permissions: %v", err)
		}
		if count != 13 {
			t.Errorf("permissions count = %d, want 13", count)
		}
	})

	t.Run("admin_has_13_role_permissions", func(t *testing.T) {
		var count int
		err := pgxPool.QueryRow(ctx, `
			SELECT COUNT(*) FROM role_permissions rp
			JOIN roles r ON r.id = rp.role_id
			WHERE r.name = 'admin'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("admin role_permissions count: %v", err)
		}
		if count != 13 {
			t.Errorf("admin role_permissions count = %d, want 13", count)
		}
	})

	t.Run("teacher_has_4_role_permissions", func(t *testing.T) {
		var count int
		err := pgxPool.QueryRow(ctx, `
			SELECT COUNT(*) FROM role_permissions rp
			JOIN roles r ON r.id = rp.role_id
			WHERE r.name = 'teacher'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("teacher role_permissions count: %v", err)
		}
		if count != 4 {
			t.Errorf("teacher role_permissions count = %d, want 4", count)
		}
	})

	t.Run("teacher_has_correct_permissions", func(t *testing.T) {
		rows, err := pgxPool.Query(ctx, `
			SELECT p.code FROM permissions p
			JOIN role_permissions rp ON rp.permission_id = p.id
			JOIN roles r ON r.id = rp.role_id
			WHERE r.name = 'teacher'
			ORDER BY p.code
		`)
		if err != nil {
			t.Fatalf("teacher permissions query: %v", err)
		}
		defer rows.Close()
		want := map[string]struct{}{
			"grades.write":     {},
			"grades.read":      {},
			"reports.read":     {},
			"profile.view_own": {},
		}
		got := map[string]struct{}{}
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err != nil {
				t.Fatalf("scan: %v", err)
			}
			got[code] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows error: %v", err)
		}
		// No unexpected permissions (over-grant check).
		for code := range got {
			if _, ok := want[code]; !ok {
				t.Errorf("teacher has unexpected permission %q", code)
			}
		}
		// No missing permissions (under-grant check).
		for code := range want {
			if _, ok := got[code]; !ok {
				t.Errorf("teacher is missing expected permission %q", code)
			}
		}
	})

	t.Run("student_has_5_role_permissions", func(t *testing.T) {
		var count int
		err := pgxPool.QueryRow(ctx, `
			SELECT COUNT(*) FROM role_permissions rp
			JOIN roles r ON r.id = rp.role_id
			WHERE r.name = 'student'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("student role_permissions count: %v", err)
		}
		if count != 5 {
			t.Errorf("student role_permissions count = %d, want 5", count)
		}
	})

	t.Run("student_has_correct_permissions", func(t *testing.T) {
		rows, err := pgxPool.Query(ctx, `
			SELECT p.code FROM permissions p
			JOIN role_permissions rp ON rp.permission_id = p.id
			JOIN roles r ON r.id = rp.role_id
			WHERE r.name = 'student'
			ORDER BY p.code
		`)
		if err != nil {
			t.Fatalf("student permissions query: %v", err)
		}
		defer rows.Close()
		want := map[string]struct{}{
			"enrollment.view_own":         {},
			"grades.view_own":             {},
			"profile.view_own":            {},
			"section_enrollment.view_own": {},
			"sections.enroll":             {},
		}
		got := map[string]struct{}{}
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err != nil {
				t.Fatalf("scan: %v", err)
			}
			got[code] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows error: %v", err)
		}
		// No unexpected permissions (over-grant check).
		for code := range got {
			if _, ok := want[code]; !ok {
				t.Errorf("student has unexpected permission %q", code)
			}
		}
		// No missing permissions (under-grant check).
		for code := range want {
			if _, ok := got[code]; !ok {
				t.Errorf("student is missing expected permission %q", code)
			}
		}
	})

	t.Run("bootstrap_admin_has_admin_role", func(t *testing.T) {
		var count int
		err := pgxPool.QueryRow(ctx, `
			SELECT COUNT(*) FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = 'a0000000-0000-0000-0000-000000000001'
			  AND r.name = 'admin'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("bootstrap admin user_roles query: %v", err)
		}
		if count != 1 {
			t.Errorf("bootstrap admin user_roles rows for admin role = %d, want 1", count)
		}
	})

	t.Run("seed_is_idempotent", func(t *testing.T) {
		// Re-run the seed SQL inline using ON CONFLICT DO NOTHING — must not error
		// and counts must remain unchanged.
		_, err := pgxPool.Exec(ctx, `
			INSERT INTO roles (name) VALUES ('admin'), ('teacher'), ('student')
			ON CONFLICT (name) DO NOTHING
		`)
		if err != nil {
			t.Fatalf("idempotent roles re-insert: %v", err)
		}

		var roleCount int
		if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM roles`).Scan(&roleCount); err != nil {
			t.Fatalf("role count after re-seed: %v", err)
		}
		if roleCount != 3 {
			t.Errorf("role count after re-seed = %d, want 3", roleCount)
		}

		var permCount int
		if err := pgxPool.QueryRow(ctx, `SELECT COUNT(*) FROM permissions`).Scan(&permCount); err != nil {
			t.Fatalf("permission count after re-seed: %v", err)
		}
		if permCount != 13 {
			t.Errorf("permission count after re-seed = %d, want 13", permCount)
		}
	})
}
