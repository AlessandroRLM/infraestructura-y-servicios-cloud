package authz

// PermissionSet is a set of Permission values backed by a map for O(1) membership tests.
type PermissionSet map[Permission]struct{}

// NewPermissionSet constructs a PermissionSet from the provided slice.
// Passing nil or an empty slice returns an empty (non-nil) set.
func NewPermissionSet(codes []Permission) PermissionSet {
	s := make(PermissionSet, len(codes))
	for _, p := range codes {
		s[p] = struct{}{}
	}
	return s
}

// Has reports whether the given Permission is a member of the set.
func (s PermissionSet) Has(p Permission) bool {
	_, ok := s[p]
	return ok
}
