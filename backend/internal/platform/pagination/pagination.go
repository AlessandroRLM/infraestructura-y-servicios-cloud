// Package pagination provides reusable, domain-agnostic helpers for cursor-based
// (keyset) pagination. SQL queries stay per-domain; only arithmetic and token
// extraction are shared here.
package pagination

import "github.com/google/uuid"

// Clamp defines minimum and maximum page-size bounds.
// Use Apply to clamp a caller-supplied page_size to a safe range.
type Clamp struct {
	Min int
	Max int
}

// Apply returns a page size within [Min, Max].
// Values below Min (including ≤ 0) become Min; values above Max become Max;
// values in range are returned unchanged.
func (c Clamp) Apply(pageSize int32) int {
	n := int(pageSize)
	if n < c.Min {
		return c.Min
	}
	if n > c.Max {
		return c.Max
	}
	return n
}

// Page is the result of Paginate: a slice of items trimmed to the requested size
// and a flag indicating whether more items exist beyond this page.
type Page[T any] struct {
	Items   []T
	HasNext bool
}

// Paginate takes a slice of rows fetched with limit = clampedSize+1 and returns a
// Page. When more than clampedSize rows are present, HasNext is true and Items is
// trimmed to clampedSize (defensive against over-fetch, not just the exact +1
// sentinel). Otherwise all rows are returned and HasNext is false.
func Paginate[T any](rows []T, clampedSize int) Page[T] {
	if len(rows) > clampedSize {
		return Page[T]{
			Items:   rows[:clampedSize],
			HasNext: true,
		}
	}
	return Page[T]{
		Items:   rows,
		HasNext: false,
	}
}

// TokenOf returns the UUID of the last item in the page as a string, for use as
// the next_page_token in a response. Returns "" when the page is empty or HasNext
// is false (no more items exist).
func TokenOf[T any](p Page[T], id func(T) uuid.UUID) string {
	if !p.HasNext || len(p.Items) == 0 {
		return ""
	}
	return id(p.Items[len(p.Items)-1]).String()
}
