package integration_test

import (
	"context"
	"sort"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	iamv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/iam/v1"
)

// ---------------------------------------------------------------------------
// Helpers specific to role-management tests.
// ---------------------------------------------------------------------------

// countUserRoles returns the number of user_roles rows for a user.
func countUserRoles(t *testing.T, userID uuid.UUID) int {
	t.Helper()
	var n int
	err := pgxPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, userID).Scan(&n)
	if err != nil {
		t.Fatalf("countUserRoles: %v", err)
	}
	return n
}

// hasRoleDB returns whether a user has a given role in the DB.
func hasRoleDB(t *testing.T, userID uuid.UUID, roleName string) bool {
	t.Helper()
	var n int
	err := pgxPool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.name = $2
	`, userID, roleName).Scan(&n)
	if err != nil {
		t.Fatalf("hasRoleDB: %v", err)
	}
	return n > 0
}

// countAuditLogs counts audit_logs rows for a target entity_id + action.
func countAuditLogs(t *testing.T, entityID uuid.UUID, action string) int {
	t.Helper()
	var n int
	err := pgxPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs WHERE entity_id = $1 AND action = $2`,
		entityID, action).Scan(&n)
	if err != nil {
		t.Fatalf("countAuditLogs: %v", err)
	}
	return n
}

// rolesFromResponse returns the sorted roles slice from a UserSummary.
func rolesFromResponse(u *iamv1.UserSummary) []string {
	got := append([]string(nil), u.GetRoles()...)
	sort.Strings(got)
	return got
}

// ---------------------------------------------------------------------------
// AssignRole — RED tests (will fail until proto + implementation land).
// ---------------------------------------------------------------------------

// TestIAMAssignRole_Denied verifies fail-closed: a teacher without users.manage is denied.
func TestIAMAssignRole_Denied(t *testing.T) {
	_, sid := seedUserWithSession(t, "iam-ar-deny-teacher@iam-roles.test", "teacher")
	targetID := seedIAMUser(t, "iam-ar-deny-target@iam-roles.test", false)
	client := newIAMClient()

	_, err := client.AssignRole(context.Background(),
		withSession(connect.NewRequest(&iamv1.AssignRoleRequest{
			UserId: targetID.String(),
			Role:   "student",
		}), sid))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestIAMAssignRole_InvalidRole verifies CodeInvalidArgument for an unknown role name.
func TestIAMAssignRole_InvalidRole(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "iam-ar-invr-admin@iam-roles.test", "admin")
	targetID := seedIAMUser(t, "iam-ar-invr-target@iam-roles.test", false)
	client := newIAMClient()

	_, err := client.AssignRole(context.Background(),
		withSession(connect.NewRequest(&iamv1.AssignRoleRequest{
			UserId: targetID.String(),
			Role:   "superadmin",
		}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestIAMAssignRole_NonExistentUser verifies CodeInvalidArgument when user_id is unknown.
func TestIAMAssignRole_NonExistentUser(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "iam-ar-nouser-admin@iam-roles.test", "admin")
	client := newIAMClient()

	_, err := client.AssignRole(context.Background(),
		withSession(connect.NewRequest(&iamv1.AssignRoleRequest{
			UserId: uuid.New().String(),
			Role:   "student",
		}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestIAMAssignRole_Success verifies successful role assignment and audit entry.
func TestIAMAssignRole_Success(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-ar-ok-admin@iam-roles.test", "admin")
	bobID := seedIAMUser(t, "bob@iam-roles.test", false)
	assignRoleDB(t, bobID, "student")
	client := newIAMClient()

	resp, err := client.AssignRole(ctx,
		withSession(connect.NewRequest(&iamv1.AssignRoleRequest{
			UserId: bobID.String(),
			Role:   "teacher",
		}), adminSID))
	if err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	// Returned UserSummary must include both roles.
	gotRoles := rolesFromResponse(resp.Msg.GetUser())
	wantRoles := []string{"student", "teacher"}
	if len(gotRoles) != len(wantRoles) {
		t.Errorf("AssignRole roles = %v, want %v", gotRoles, wantRoles)
	} else {
		for i := range wantRoles {
			if gotRoles[i] != wantRoles[i] {
				t.Errorf("AssignRole roles[%d] = %q, want %q", i, gotRoles[i], wantRoles[i])
			}
		}
	}

	// DB must have two user_roles rows.
	if n := countUserRoles(t, bobID); n != 2 {
		t.Errorf("user_roles count = %d, want 2", n)
	}

	// Audit log must have one role.assign entry.
	if n := countAuditLogs(t, bobID, "role.assign"); n != 1 {
		t.Errorf("audit_logs role.assign count = %d, want 1", n)
	}
}

// TestIAMAssignRole_Idempotent verifies that re-assigning an existing role:
//   - does not create a duplicate user_roles row, and
//   - DOES write an additional audit_logs entry (audit-on-every-call, EC-05).
func TestIAMAssignRole_Idempotent(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-ar-idem-admin@iam-roles.test", "admin")
	bobID := seedIAMUser(t, "bob-idem@iam-roles.test", false)
	assignRoleDB(t, bobID, "teacher")
	client := newIAMClient()

	// First call (role already exists — idempotent).
	_, err := client.AssignRole(ctx,
		withSession(connect.NewRequest(&iamv1.AssignRoleRequest{
			UserId: bobID.String(),
			Role:   "teacher",
		}), adminSID))
	if err != nil {
		t.Fatalf("AssignRole (idempotent): %v", err)
	}

	// Still only one user_roles row for teacher.
	if n := countUserRoles(t, bobID); n != 1 {
		t.Errorf("user_roles count after idempotent assign = %d, want 1 (no duplicate)", n)
	}

	// Audit written on every call.
	if n := countAuditLogs(t, bobID, "role.assign"); n != 1 {
		t.Errorf("audit_logs role.assign count = %d, want 1 (idempotent re-assign)", n)
	}

	// Second call.
	_, err = client.AssignRole(ctx,
		withSession(connect.NewRequest(&iamv1.AssignRoleRequest{
			UserId: bobID.String(),
			Role:   "teacher",
		}), adminSID))
	if err != nil {
		t.Fatalf("AssignRole (idempotent second call): %v", err)
	}

	// Still one DB row.
	if n := countUserRoles(t, bobID); n != 1 {
		t.Errorf("user_roles count after second idempotent assign = %d, want 1", n)
	}

	// Two audit entries total (one per call).
	if n := countAuditLogs(t, bobID, "role.assign"); n != 2 {
		t.Errorf("audit_logs role.assign count = %d, want 2 (one per call)", n)
	}
}

// ---------------------------------------------------------------------------
// RevokeRole — RED tests.
// ---------------------------------------------------------------------------

// TestIAMRevokeRole_Denied verifies fail-closed: a student without users.manage is denied.
func TestIAMRevokeRole_Denied(t *testing.T) {
	_, sid := seedUserWithSession(t, "iam-rr-deny-student@iam-roles.test", "student")
	targetID := seedIAMUser(t, "iam-rr-deny-target@iam-roles.test", false)
	assignRoleDB(t, targetID, "student")
	client := newIAMClient()

	_, err := client.RevokeRole(context.Background(),
		withSession(connect.NewRequest(&iamv1.RevokeRoleRequest{
			UserId: targetID.String(),
			Role:   "student",
		}), sid))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestIAMRevokeRole_NonAdminSuccess verifies successful revoke of a non-admin role + audit.
func TestIAMRevokeRole_NonAdminSuccess(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-rr-ok-admin@iam-roles.test", "admin")
	carolID := seedIAMUser(t, "carol@iam-roles.test", false)
	assignRoleDB(t, carolID, "teacher")
	assignRoleDB(t, carolID, "student")
	client := newIAMClient()

	resp, err := client.RevokeRole(ctx,
		withSession(connect.NewRequest(&iamv1.RevokeRoleRequest{
			UserId: carolID.String(),
			Role:   "student",
		}), adminSID))
	if err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}

	// Returned roles must only contain teacher.
	gotRoles := rolesFromResponse(resp.Msg.GetUser())
	if len(gotRoles) != 1 || gotRoles[0] != "teacher" {
		t.Errorf("RevokeRole roles = %v, want [teacher]", gotRoles)
	}

	// DB must no longer have student row.
	if hasRoleDB(t, carolID, "student") {
		t.Error("user_roles still contains student after RevokeRole")
	}

	// Audit entry written.
	if n := countAuditLogs(t, carolID, "role.revoke"); n != 1 {
		t.Errorf("audit_logs role.revoke count = %d, want 1", n)
	}
}

// TestIAMRevokeRole_SelfDemotion verifies that an admin cannot remove their own admin role.
func TestIAMRevokeRole_SelfDemotion(t *testing.T) {
	adminID, adminSID := seedUserWithSession(t, "iam-rr-self-admin@iam-roles.test", "admin")
	client := newIAMClient()

	// Ensure there's at least one other admin so last-admin guard doesn't fire first.
	otherAdminID := seedIAMUser(t, "iam-rr-self-other@iam-roles.test", false)
	assignRoleDB(t, otherAdminID, "admin")

	_, err := client.RevokeRole(context.Background(),
		withSession(connect.NewRequest(&iamv1.RevokeRoleRequest{
			UserId: adminID.String(),
			Role:   "admin",
		}), adminSID))
	// Design §9: ErrSelfDemotion → CodeFailedPrecondition.
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	// No audit entry for the blocked attempt.
	if n := countAuditLogs(t, adminID, "role.revoke"); n != 0 {
		t.Errorf("audit_logs role.revoke count = %d, want 0 (blocked self-demotion)", n)
	}
}

// TestIAMRevokeRole_LastAdmin verifies that revoking admin when only one admin exists is blocked.
func TestIAMRevokeRole_LastAdmin(t *testing.T) {
	ctx := context.Background()
	// We need a scenario with exactly one admin for the target user,
	// and a different admin caller. We'll use a helper admin for the call and
	// seed a "single admin" target.
	//
	// NOTE: We can't guarantee exactly one admin globally since other tests
	// seed admins. We approach this by using RevokeRole's target = the caller
	// themselves first (which hits self-demotion), then we need a distinct
	// scenario. Instead, we'll use a second admin as caller revoking the last
	// admin's role.
	//
	// Test: Seed ONLY one admin user (callerAdmin). There are no other admins.
	// When callerAdmin tries to revoke that same user's admin role... but that
	// would also trigger self-demotion first (EC-06). Instead seed a target who
	// is admin, and a different admin caller; make sure the target is the ONLY
	// admin. We do this by:
	//  1. Create targetAdmin with admin role.
	//  2. Create callerAdmin with admin role.
	//  3. Revoke callerAdmin's admin role via DB (now only targetAdmin is admin in our pair).
	//  4. Caller tries to revoke targetAdmin's admin. CountAdmins will include all admins
	//     globally. To make this work reliably we can't shrink global admin count.
	//
	// Simplest reliable approach: create a test where BOTH roles are set up, then
	// after DB manipulation ensure exactly 1 admin. Since we can't isolate globally,
	// we skip this test variant and test via unit test instead. The integration
	// check remains as a sanity check that calls with existing last-admin setup work.
	//
	// For the integration test we verify the error code. We rely on seeding exactly
	// ONE admin user whose admin role we try to revoke as a DIFFERENT admin. This
	// requires that CountAdmins() returns exactly 1, meaning no other tests have
	// live admin users at the same time. Since tests run sequentially (single server),
	// we'll work with the assumption that seeded admin users from OTHER tests may
	// still exist. Therefore this test IS potentially flaky unless we control the
	// global state. The reliable path is: give the target admin role, then forcibly
	// remove all other admin rows NOT the target, call RevokeRole, assert CodePermissionDenied.
	//
	// Implementation: use a unique admin target, remove all other admin rows temporarily.

	// Seed the "single admin" target and the calling admin.
	targetAdminID := seedIAMUser(t, "iam-rr-la-target@iam-rr-lastadmin.test", false)
	callerAdminID, callerAdminSID := seedUserWithSession(t, "iam-rr-la-caller@iam-rr-lastadmin.test", "admin")

	// Assign admin to target.
	assignRoleDB(t, targetAdminID, "admin")

	// Now we need CountAdmins() to return <= 1 at the moment of the check.
	// We'll revoke callerAdmin's own admin role from DB first (making targetAdminID the only admin),
	// then caller (without admin role) tries to revoke... but caller has no admin role and
	// thus no users.manage permission. We need callerAdmin to keep users.manage but only
	// target to have admin role.
	//
	// The cleanest approach for a reliable integration test: make targetAdminID
	// the ONLY admin by removing callerAdmin's admin role from DB, then re-seed
	// caller's session with a different role that STILL has users.manage. But
	// users.manage is tied to admin role. Let's keep caller as admin and target
	// as the only admin by giving caller a non-admin role that still has users.manage.
	// That's not how our schema works.
	//
	// Simplest reliable: keep both as admin (CountAdmins >= 2), then revoke
	// callerAdmin's admin via DB (CountAdmins = 1 for targetAdminID), but caller
	// loses the admin role and thus users.manage... Catch-22.
	//
	// Resolution: we must seed a third "helper admin" who makes the call, while
	// target is the only remaining admin in the pair. BUT CountAdmins is GLOBAL
	// (counts all admins in the entire DB). We can't isolate without a global lock.
	//
	// Pragmatic decision: use a separate admin caller and temporarily remove
	// all other admin rows EXCEPT target, then restore them. This is safe because
	// integration tests run sequentially (no t.Parallel() in this file).

	// Temporarily remove caller's admin role so CountAdmins reflects only targetAdminID.
	// Caller still has an active session from seedUserWithSession.
	_, err := pgxPool.Exec(ctx, `
		DELETE FROM user_roles ur USING roles r
		WHERE ur.role_id = r.id AND ur.user_id = $1 AND r.name = 'admin'
	`, callerAdminID)
	if err != nil {
		t.Fatalf("remove caller admin role: %v", err)
	}
	// Also record the current count before our manipulation for cleanup restoration.
	// We need to restore callerAdmin's admin after the test.
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pgxPool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, r.id FROM roles r WHERE r.name = 'admin'
			ON CONFLICT DO NOTHING
		`, callerAdminID)
	})

	// At this point, callerAdmin has no admin role → their session still works but
	// callerAdmin lost users.manage. We need a DIFFERENT mechanism. Let's step back.
	//
	// The only clean path: use a fresh admin (callerAdmin2) who retains admin role,
	// and ensure the global CountAdmins includes both callerAdmin2 and targetAdminID.
	// Remove all OTHER admins temporarily (the seeded ones from other tests).
	// This is complex and fragile.
	//
	// Decision: test last-admin guard via UNIT test (service_test.go) which mocks
	// CountAdmins returning 1. Keep this integration test as a best-effort check
	// that relies on test isolation (test suffix @iam-rr-lastadmin.test admins only).
	//
	// Re-implement: seed a second admin caller, both caller and target are admin.
	// Remove caller admin from DB. Caller session lost users.manage → can't call.
	// We can't test this integration without controlling CountAdmins globally.
	//
	// Final approach: seed callerAdmin AND targetAdmin, keep targetAdmin as the sole admin
	// by giving callerAdmin a non-admin role but no admin role. callerAdmin session
	// was created with "admin" role → session has users.manage loaded at session creation
	// time. BUT the RBAC loader reads roles fresh on each request from the DB.
	// So revoking callerAdmin's DB admin row BEFORE the RPC call means the RBAC
	// loader returns no admin role → no users.manage → CodePermissionDenied (wrong code).
	//
	// There's no way to test this in integration without a global-count control.
	// Skip and cover with unit test instead.
	_ = callerAdminSID
	_ = targetAdminID
	t.Skip("last-admin guard is tested via unit test (service_test.go); " +
		"integration isolation requires controlling global CountAdmins which is not feasible " +
		"without test-scoped role segregation")
}

// TestIAMRevokeRole_MultipleAdmins verifies that revoking admin succeeds when another admin exists.
func TestIAMRevokeRole_MultipleAdmins(t *testing.T) {
	ctx := context.Background()
	admin1ID, admin1SID := seedUserWithSession(t, "iam-rr-ma-admin1@iam-roles.test", "admin")
	admin2ID, _ := seedUserWithSession(t, "iam-rr-ma-admin2@iam-roles.test", "admin")
	client := newIAMClient()

	// admin1 revokes admin2's admin role. CountAdmins >= 2 (at least admin1 + admin2).
	resp, err := client.RevokeRole(ctx,
		withSession(connect.NewRequest(&iamv1.RevokeRoleRequest{
			UserId: admin2ID.String(),
			Role:   "admin",
		}), admin1SID))
	if err != nil {
		t.Fatalf("RevokeRole (multiple admins): %v", err)
	}

	// admin2 no longer has admin role in response.
	for _, r := range resp.Msg.GetUser().GetRoles() {
		if r == "admin" {
			t.Error("RevokeRole (multiple admins): admin role still present in response")
		}
	}

	// DB confirms removal.
	if hasRoleDB(t, admin2ID, "admin") {
		t.Error("user_roles still contains admin for admin2 after RevokeRole")
	}

	// Audit entry written.
	if n := countAuditLogs(t, admin2ID, "role.revoke"); n != 1 {
		t.Errorf("audit_logs role.revoke count = %d, want 1", n)
	}

	_ = admin1ID
}

// TestIAMRevokeRole_NotHaveRole verifies CodeNotFound when user doesn't have the specified role.
func TestIAMRevokeRole_NotHaveRole(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "iam-rr-nhr-admin@iam-roles.test", "admin")
	bobID := seedIAMUser(t, "bob-nhr@iam-roles.test", false)
	// bob has NO admin role
	client := newIAMClient()

	_, err := client.RevokeRole(context.Background(),
		withSession(connect.NewRequest(&iamv1.RevokeRoleRequest{
			UserId: bobID.String(),
			Role:   "admin",
		}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestIAMRevokeRole_InvalidUUID verifies CodeInvalidArgument for a malformed user_id.
func TestIAMRevokeRole_InvalidUUID(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "iam-rr-iuuid-admin@iam-roles.test", "admin")
	client := newIAMClient()

	_, err := client.RevokeRole(context.Background(),
		withSession(connect.NewRequest(&iamv1.RevokeRoleRequest{
			UserId: "not-a-uuid",
			Role:   "student",
		}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}
