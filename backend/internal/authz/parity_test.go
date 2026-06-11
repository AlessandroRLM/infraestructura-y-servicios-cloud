package authz_test

import (
	"testing"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
)

// TestAllPermissions_ParityWithSeedData asserts that the typed constants in the authz
// package match exactly the permission codes inserted by the seed migration.
// This test has no database dependency — it compares constant string values directly,
// ensuring that a mis-spelled or missing constant is caught at compile/test time.
func TestAllPermissions_ParityWithSeedData(t *testing.T) {
	t.Parallel()

	// These are the exact codes inserted by the seed migrations.
	// The list must match AllPermissions in both count and content.
	expectedCodes := map[authz.Permission]struct{}{
		"users.manage":                {},
		"catalog.manage":              {},
		"enrollment.manage":           {},
		"sections.enroll":             {},
		"enrollment.view_own":         {},
		"grades.write":                {},
		"grades.read":                 {},
		"grades.view_own":             {},
		"reports.read":                {},
		"audit.read":                  {},
		"grades.override":             {},
		"profile.view_own":            {},
		"section_enrollment.view_own": {},
	}

	if len(authz.AllPermissions) != 13 {
		t.Errorf("AllPermissions length = %d, want 13", len(authz.AllPermissions))
	}

	seen := make(map[authz.Permission]struct{}, len(authz.AllPermissions))
	for _, p := range authz.AllPermissions {
		if _, ok := expectedCodes[p]; !ok {
			t.Errorf("AllPermissions contains unexpected code %q", p)
		}
		seen[p] = struct{}{}
	}

	for code := range expectedCodes {
		if _, ok := seen[code]; !ok {
			t.Errorf("AllPermissions is missing expected code %q", code)
		}
	}
}
