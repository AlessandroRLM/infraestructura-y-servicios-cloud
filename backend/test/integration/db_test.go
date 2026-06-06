package integration_test

import (
	"testing"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
	migrations "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/migrations"
)

// IT-6: Migrations apply cleanly to a real Postgres container.
// TestMain already ran Migrate once; calling it again exercises ErrNoChange (idempotent).
func TestMigrationsApply(t *testing.T) {
	// First call already happened in TestMain. This verifies idempotency (ErrNoChange → nil).
	if err := db.Migrate(dbDSN, migrations.FS); err != nil {
		t.Fatalf("db.Migrate() unexpected error on second run: %v", err)
	}
}
