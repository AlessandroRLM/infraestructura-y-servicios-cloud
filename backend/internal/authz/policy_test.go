package authz_test

import (
	"context"
	"testing"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
)

func TestPermissionPolicy_Evaluate(t *testing.T) {
	t.Parallel()

	required := authz.PermUsersManage
	policy := authz.PermissionPolicy{}

	tests := []struct {
		name        string
		buildCtx    func() context.Context
		required    authz.Permission
		wantAllowed bool
	}{
		{
			name: "allowed when context contains the required permission",
			buildCtx: func() context.Context {
				set := authz.NewPermissionSet([]authz.Permission{required})
				return authz.WithPermissions(context.Background(), set)
			},
			required:    required,
			wantAllowed: true,
		},
		{
			name: "denied when context has no permission set",
			buildCtx: func() context.Context {
				return context.Background()
			},
			required:    required,
			wantAllowed: false,
		},
		{
			name: "denied when permission set does not contain the required code",
			buildCtx: func() context.Context {
				set := authz.NewPermissionSet([]authz.Permission{authz.PermCatalogManage})
				return authz.WithPermissions(context.Background(), set)
			},
			required:    required,
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.buildCtx()
			decision := policy.Evaluate(ctx, tt.required)
			if decision.Allowed != tt.wantAllowed {
				t.Errorf("Evaluate() Allowed = %v, want %v", decision.Allowed, tt.wantAllowed)
			}
		})
	}
}
