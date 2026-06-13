package pagination_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/pagination"
)

func TestClamp_Apply(t *testing.T) {
	t.Parallel()

	c := pagination.Clamp{Min: 20, Max: 200}

	cases := []struct {
		name  string
		input int32
		want  int
	}{
		{"zero becomes Min", 0, 20},
		{"negative becomes Min", -1, 20},
		{"very negative becomes Min", -999, 20},
		{"positive below Min becomes Min", 5, 20},
		{"just below Min becomes Min", 19, 20},
		{"Min boundary", 20, 20},
		{"in-range value", 50, 50},
		{"Max boundary", 200, 200},
		{"above Max becomes Max", 201, 200},
		{"way above Max becomes Max", 999, 200},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := c.Apply(tc.input)
			if got != tc.want {
				t.Errorf("Apply(%d) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

// item is a minimal type used for generic tests.
type item struct {
	id   uuid.UUID
	name string
}

func TestPaginate_LessThanFull(t *testing.T) {
	t.Parallel()

	rows := []item{{id: uuid.New()}, {id: uuid.New()}}
	clampedSize := 5

	p := pagination.Paginate(rows, clampedSize)

	if p.HasNext {
		t.Error("HasNext = true, want false (fewer items than page size)")
	}
	if len(p.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(p.Items))
	}
}

func TestPaginate_ExactlyFull(t *testing.T) {
	t.Parallel()

	clampedSize := 3
	rows := make([]item, clampedSize) // exactly clampedSize rows, no extra
	for i := range rows {
		rows[i].id = uuid.New()
	}

	p := pagination.Paginate(rows, clampedSize)

	if p.HasNext {
		t.Error("HasNext = true, want false (no overflow sentinel row)")
	}
	if len(p.Items) != clampedSize {
		t.Errorf("len(Items) = %d, want %d", len(p.Items), clampedSize)
	}
}

func TestPaginate_HasNextAndTrimmed(t *testing.T) {
	t.Parallel()

	clampedSize := 3
	// Fetch clampedSize+1 rows (the sentinel row proving more exist).
	rows := make([]item, clampedSize+1)
	for i := range rows {
		rows[i].id = uuid.New()
	}

	p := pagination.Paginate(rows, clampedSize)

	if !p.HasNext {
		t.Error("HasNext = false, want true (overflow sentinel present)")
	}
	if len(p.Items) != clampedSize {
		t.Errorf("len(Items) = %d, want %d (must be trimmed)", len(p.Items), clampedSize)
	}
	// Sentinel row must not appear in Items.
	for _, it := range p.Items {
		if it.id == rows[clampedSize].id {
			t.Error("sentinel row leaked into Items")
		}
	}
}

func TestPaginate_OverFetch(t *testing.T) {
	t.Parallel()

	clampedSize := 3
	// Defensive: a caller fetches MORE than clampedSize+1 rows (limit mismatch).
	// The page must still be trimmed to clampedSize with HasNext=true, never
	// leak an oversized page with pagination silently disabled.
	rows := make([]item, clampedSize+3)
	for i := range rows {
		rows[i].id = uuid.New()
	}

	p := pagination.Paginate(rows, clampedSize)

	if !p.HasNext {
		t.Error("HasNext = false, want true (more rows than page size)")
	}
	if len(p.Items) != clampedSize {
		t.Errorf("len(Items) = %d, want %d (must be trimmed to page size)", len(p.Items), clampedSize)
	}
}

func TestPaginate_Empty(t *testing.T) {
	t.Parallel()

	p := pagination.Paginate([]item{}, 20)

	if p.HasNext {
		t.Error("HasNext = true, want false for empty result")
	}
	if len(p.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(p.Items))
	}
}

func TestTokenOf_HasNextReturnsLastID(t *testing.T) {
	t.Parallel()

	id1 := uuid.New()
	id2 := uuid.New()

	rows := []item{{id: id1}, {id: id2}, {id: uuid.New()}} // 3 rows, size=2
	p := pagination.Paginate(rows, 2)

	token := pagination.TokenOf(p, func(it item) uuid.UUID { return it.id })
	if token == "" {
		t.Fatal("TokenOf returned empty string, want non-empty")
	}
	if token != id2.String() {
		t.Errorf("token = %q, want %q (last item in trimmed page)", token, id2.String())
	}
}

func TestTokenOf_NoNextReturnsEmpty(t *testing.T) {
	t.Parallel()

	rows := []item{{id: uuid.New()}}
	p := pagination.Paginate(rows, 5) // fewer than clampedSize → HasNext=false

	token := pagination.TokenOf(p, func(it item) uuid.UUID { return it.id })
	if token != "" {
		t.Errorf("TokenOf returned %q, want \"\" when HasNext is false", token)
	}
}

func TestTokenOf_EmptyPageReturnsEmpty(t *testing.T) {
	t.Parallel()

	p := pagination.Paginate([]item{}, 5)

	token := pagination.TokenOf(p, func(it item) uuid.UUID { return it.id })
	if token != "" {
		t.Errorf("TokenOf returned %q, want \"\" for empty page", token)
	}
}
