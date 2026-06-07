package rbac_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/rbac"
)

// fakeQuerier is a test double for rbacdb.Querier.
type fakeQuerier struct {
	codes []string
}

func (f *fakeQuerier) GetPermissionsForUser(_ context.Context, _ pgtype.UUID) ([]string, error) {
	return f.codes, nil
}

func TestPostgresRoleLoader_Load(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name        string
		codes       []string
		checkHas    authz.Permission
		wantHas     bool
		wantLen     int
	}{
		{
			name:     "returns PermissionSet with correct membership for known codes",
			codes:    []string{"users.manage", "grades.read"},
			checkHas: authz.PermUsersManage,
			wantHas:  true,
			wantLen:  2,
		},
		{
			name:     "Has returns false for absent code",
			codes:    []string{"grades.read"},
			checkHas: authz.PermUsersManage,
			wantHas:  false,
			wantLen:  1,
		},
		{
			name:     "returns empty PermissionSet when querier returns empty slice",
			codes:    []string{},
			checkHas: authz.PermUsersManage,
			wantHas:  false,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := &fakeQuerier{codes: tt.codes}
			loader := rbac.NewPostgresRoleLoader(q)

			set, err := loader.Load(context.Background(), userID)
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if len(set) != tt.wantLen {
				t.Errorf("Load() set length = %d, want %d", len(set), tt.wantLen)
			}
			if got := set.Has(tt.checkHas); got != tt.wantHas {
				t.Errorf("set.Has(%q) = %v, want %v", tt.checkHas, got, tt.wantHas)
			}
		})
	}
}
