package authz_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
)

// ── Decision constructors ──────────────────────────────────────────────────

func TestDecision_Allow(t *testing.T) {
	t.Parallel()

	d := authz.Allow()
	if !d.Allowed {
		t.Error("Allow() should return Allowed = true")
	}
	if d.Reason != "" {
		t.Errorf("Allow() Reason should be empty, got %q", d.Reason)
	}
	if d.Err != nil {
		t.Errorf("Allow() Err should be nil, got %v", d.Err)
	}
}

func TestDecision_Deny(t *testing.T) {
	t.Parallel()

	d := authz.Deny("missing permission: users.manage")
	if d.Allowed {
		t.Error("Deny() should return Allowed = false")
	}
	if d.Reason != "missing permission: users.manage" {
		t.Errorf("Deny() Reason = %q, want %q", d.Reason, "missing permission: users.manage")
	}
	if d.Err != nil {
		t.Errorf("Deny() Err should be nil, got %v", d.Err)
	}
}

func TestDecision_DenyErr(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("db error")
	d := authz.DenyErr("internal failure", sentinel)
	if d.Allowed {
		t.Error("DenyErr() should return Allowed = false")
	}
	if d.Reason != "internal failure" {
		t.Errorf("DenyErr() Reason = %q, want %q", d.Reason, "internal failure")
	}
	if !errors.Is(d.Err, sentinel) {
		t.Errorf("DenyErr() Err should wrap sentinel, got %v", d.Err)
	}
}

func TestDecision_ZeroValue_IsDenied(t *testing.T) {
	t.Parallel()

	var d authz.Decision
	if d.Allowed {
		t.Error("zero-value Decision should be denied (fail-closed)")
	}
}

// ── AllOf ──────────────────────────────────────────────────────────────────

func TestAllOf_Empty_Denies(t *testing.T) {
	t.Parallel()

	pf := authz.AllOf() // no policies — fail-closed
	req := authz.AccessRequest{}
	d := pf(context.Background(), req)
	if d.Allowed {
		t.Error("AllOf() with no policies should deny (fail-closed)")
	}
}

func TestAllOf_AllAllow_Allows(t *testing.T) {
	t.Parallel()

	always := func(_ context.Context, _ authz.AccessRequest) authz.Decision {
		return authz.Allow()
	}
	pf := authz.AllOf(always, always, always)
	d := pf(context.Background(), authz.AccessRequest{})
	if !d.Allowed {
		t.Error("AllOf with all-allow policies should allow")
	}
}

func TestAllOf_FirstDenyWins(t *testing.T) {
	t.Parallel()

	allow := func(_ context.Context, _ authz.AccessRequest) authz.Decision {
		return authz.Allow()
	}
	denyFirst := func(_ context.Context, _ authz.AccessRequest) authz.Decision {
		return authz.Deny("first denied")
	}
	denySecond := func(_ context.Context, _ authz.AccessRequest) authz.Decision {
		return authz.Deny("second denied — should not appear")
	}

	pf := authz.AllOf(allow, denyFirst, denySecond)
	d := pf(context.Background(), authz.AccessRequest{})
	if d.Allowed {
		t.Error("AllOf should deny when any policy denies")
	}
	if d.Reason != "first denied" {
		t.Errorf("AllOf Reason = %q, want %q (first denier should win)", d.Reason, "first denied")
	}
}

func TestAllOf_PropagatesDenyErr(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("policy error")
	errPolicy := func(_ context.Context, _ authz.AccessRequest) authz.Decision {
		return authz.DenyErr("failed", sentinel)
	}
	pf := authz.AllOf(errPolicy)
	d := pf(context.Background(), authz.AccessRequest{})
	if d.Allowed {
		t.Error("AllOf should propagate deny with Err")
	}
	if !errors.Is(d.Err, sentinel) {
		t.Errorf("AllOf should propagate Err, got %v", d.Err)
	}
}

// ── RequirePermission ──────────────────────────────────────────────────────

func TestRequirePermission_Allow(t *testing.T) {
	t.Parallel()

	set := authz.NewPermissionSet([]authz.Permission{authz.PermUsersManage})
	ctx := authz.WithPermissions(context.Background(), set)
	req := authz.AccessRequest{Permissions: set}

	pf := authz.RequirePermission(authz.PermUsersManage)
	d := pf(ctx, req)
	if !d.Allowed {
		t.Errorf("RequirePermission should allow when permission is present; reason: %s", d.Reason)
	}
}

func TestRequirePermission_Deny_MissingPermission(t *testing.T) {
	t.Parallel()

	set := authz.NewPermissionSet([]authz.Permission{authz.PermCatalogManage})
	ctx := authz.WithPermissions(context.Background(), set)
	req := authz.AccessRequest{Permissions: set}

	pf := authz.RequirePermission(authz.PermUsersManage)
	d := pf(ctx, req)
	if d.Allowed {
		t.Error("RequirePermission should deny when permission is absent")
	}
	if d.Reason == "" {
		t.Error("RequirePermission deny should carry a reason")
	}
}

func TestRequirePermission_Deny_EmptySet(t *testing.T) {
	t.Parallel()

	set := authz.NewPermissionSet(nil)
	req := authz.AccessRequest{Permissions: set}

	pf := authz.RequirePermission(authz.PermUsersManage)
	d := pf(context.Background(), req)
	if d.Allowed {
		t.Error("RequirePermission should deny on empty PermissionSet")
	}
}

func TestRequirePermission_IgnoresReqRequired(t *testing.T) {
	t.Parallel()

	// req.Required is set to a different permission than the closed-over one.
	// RequirePermission must use its own captured permission, not req.Required.
	set := authz.NewPermissionSet([]authz.Permission{authz.PermUsersManage})
	req := authz.AccessRequest{
		Permissions: set,
		Required:    authz.PermCatalogManage, // different from captured PermUsersManage
	}

	pf := authz.RequirePermission(authz.PermUsersManage) // captured: PermUsersManage
	d := pf(context.Background(), req)
	if !d.Allowed {
		t.Errorf("RequirePermission must use its captured permission, not req.Required; reason: %s", d.Reason)
	}
}

// ── RequireSelf ────────────────────────────────────────────────────────────

func TestRequireSelf_Allow_WhenSubjectEqualsOwner(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	req := authz.AccessRequest{
		SubjectID:       id,
		ResourceOwnerID: id,
		HasResource:     true,
	}

	pf := authz.RequireSelf()
	d := pf(context.Background(), req)
	if !d.Allowed {
		t.Errorf("RequireSelf should allow when subject == owner and HasResource; reason: %s", d.Reason)
	}
}

func TestRequireSelf_Deny_WhenSubjectDiffersFromOwner(t *testing.T) {
	t.Parallel()

	req := authz.AccessRequest{
		SubjectID:       uuid.New(),
		ResourceOwnerID: uuid.New(),
		HasResource:     true,
	}

	pf := authz.RequireSelf()
	d := pf(context.Background(), req)
	if d.Allowed {
		t.Error("RequireSelf should deny when subject != owner")
	}
	if d.Reason == "" {
		t.Error("RequireSelf deny should carry a reason")
	}
}

func TestRequireSelf_Deny_WhenHasResourceFalse(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	req := authz.AccessRequest{
		SubjectID:       id,
		ResourceOwnerID: id,
		HasResource:     false, // resource not set, even though IDs match
	}

	pf := authz.RequireSelf()
	d := pf(context.Background(), req)
	if d.Allowed {
		t.Error("RequireSelf should deny when HasResource is false")
	}
}
