package authz_test

import (
	"testing"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
)

func TestPermissionSet_Has(t *testing.T) {
	t.Parallel()

	present := authz.Permission("users.manage")
	absent := authz.Permission("catalog.manage")

	tests := []struct {
		name  string
		set   authz.PermissionSet
		query authz.Permission
		want  bool
	}{
		{
			name:  "returns true for a code in the set",
			set:   authz.NewPermissionSet([]authz.Permission{present}),
			query: present,
			want:  true,
		},
		{
			name:  "returns false for a code not in the set",
			set:   authz.NewPermissionSet([]authz.Permission{present}),
			query: absent,
			want:  false,
		},
		{
			name:  "returns false on empty set",
			set:   authz.NewPermissionSet(nil),
			query: present,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.set.Has(tt.query)
			if got != tt.want {
				t.Errorf("Has(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}
