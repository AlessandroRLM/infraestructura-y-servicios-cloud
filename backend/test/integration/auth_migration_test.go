package integration_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestAuthMigration_UsersTable verifies that the 000001_users migration applied correctly.
// Assertions:
//   (a) the users table exists with the required columns and the UNIQUE constraint on email (AUTH-20)
//   (b) inserting a row without specifying id yields a UUID whose version nibble is 7
//       (proves the native uuidv7() default — requires Postgres 18+)
func TestAuthMigration_UsersTable(t *testing.T) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbDSN)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	t.Run("required_columns_exist", func(t *testing.T) {
		requiredCols := []string{
			"id", "email", "password_hash",
			"created_at", "updated_at", "created_by", "updated_by", "deleted_at",
		}
		for _, col := range requiredCols {
			var exists bool
			err := pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_schema = 'public'
					  AND table_name   = 'users'
					  AND column_name  = $1
				)`, col).Scan(&exists)
			if err != nil {
				t.Fatalf("column check %q: %v", col, err)
			}
			if !exists {
				t.Errorf("column %q missing from users table", col)
			}
		}
	})

	t.Run("email_unique_constraint_exists", func(t *testing.T) {
		var count int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM information_schema.table_constraints tc
			JOIN information_schema.constraint_column_usage ccu
			  ON tc.constraint_name = ccu.constraint_name
			 AND tc.table_schema    = ccu.table_schema
			WHERE tc.table_schema   = 'public'
			  AND tc.table_name     = 'users'
			  AND ccu.column_name   = 'email'
			  AND tc.constraint_type IN ('UNIQUE', 'PRIMARY KEY')
		`).Scan(&count)
		if err != nil {
			t.Fatalf("unique constraint check: %v", err)
		}
		if count == 0 {
			t.Error("no UNIQUE constraint found on users.email")
		}
	})

	t.Run("id_default_is_uuidv7", func(t *testing.T) {
		// Insert a row without specifying id to let the DB default fire.
		// Then confirm the version nibble (bits 12-15 of octet 6) equals 7.
		var idStr string
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash)
			VALUES ('__test_uuidv7@example.com', 'nothashed')
			RETURNING id::text
		`).Scan(&idStr)
		if err != nil {
			t.Fatalf("INSERT to capture default id: %v", err)
		}
		defer func() {
			_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = '__test_uuidv7@example.com'`)
		}()

		// UUID format: xxxxxxxx-xxxx-Mxxx-Nxxx-xxxxxxxxxxxx
		// The 'M' character (index 14 in the canonical string) is the version nibble.
		if len(idStr) != 36 {
			t.Fatalf("id %q is not a canonical UUID (len=%d)", idStr, len(idStr))
		}
		versionNibble := idStr[14]
		if versionNibble != '7' {
			t.Errorf("users.id default version nibble = %q, want '7' (UUIDv7); got full id %q", versionNibble, idStr)
		}
	})
}
