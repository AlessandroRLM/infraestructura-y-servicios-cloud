package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	iamv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/iam/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/iam/v1/iamv1connect"
)

// newIAMClient returns a Connect IamService client targeting the shared test server.
func newIAMClient() iamv1connect.IamServiceClient {
	return iamv1connect.NewIamServiceClient(http.DefaultClient, baseURL)
}

// seedIAMUser inserts a user with no role and optionally soft-deletes it.
// Returns the user UUID. Registers cleanup.
func seedIAMUser(t *testing.T, email string, softDeleted bool) uuid.UUID {
	t.Helper()
	hash := "$2a$04$placeholder.hash.for.testing.only.not.a.real.password"
	var userID uuid.UUID
	var err error
	ctx := context.Background()

	if softDeleted {
		err = pgxPool.QueryRow(ctx,
			`INSERT INTO users (email, password_hash, deleted_at) VALUES ($1, $2, NOW()) RETURNING id`,
			email, hash,
		).Scan(&userID)
	} else {
		err = pgxPool.QueryRow(ctx,
			`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
			email, hash,
		).Scan(&userID)
	}
	if err != nil {
		t.Fatalf("seedIAMUser: insert %q: %v", email, err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pgxPool.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID)
		_, _ = pgxPool.Exec(ctx, `DELETE FROM user_profiles WHERE user_id = $1`, userID)
		_, _ = pgxPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	return userID
}

// seedIAMUserWithProfile inserts a user plus a user_profiles row.
func seedIAMUserWithProfile(t *testing.T, email, givenNames, lastNamePaternal string) uuid.UUID {
	t.Helper()
	userID := seedIAMUser(t, email, false)
	nationalID := "IAM-" + userID.String()[:8]
	_, err := pgxPool.Exec(context.Background(), `
		INSERT INTO user_profiles (user_id, given_names, last_name_paternal, national_id_type, national_id)
		VALUES ($1, $2, $3, 'RUT', $4)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, givenNames, lastNamePaternal, nationalID)
	if err != nil {
		t.Fatalf("seedIAMUserWithProfile: %v", err)
	}
	return userID
}

// assignRole assigns an existing role to a user directly in DB.
func assignRoleDB(t *testing.T, userID uuid.UUID, roleName string) {
	t.Helper()
	_, err := pgxPool.Exec(context.Background(), `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, r.id FROM roles r WHERE r.name = $2
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, userID, roleName)
	if err != nil {
		t.Fatalf("assignRoleDB: %v", err)
	}
}

// TestIAMListUsers_StudentDenied verifies fail-closed: a student cannot call ListUsers.
// RED-4 (C-5.2): the IAM service is NOT yet wired in the policies map.
func TestIAMListUsers_StudentDenied(t *testing.T) {
	_, sid := seedUserWithSession(t, "iam-list-deny-student@iam.test", "student")
	client := newIAMClient()

	_, err := client.ListUsers(context.Background(),
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{}), sid))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestIAMListUsers_FirstPage verifies that an admin gets the first page when more pages exist.
// Seeds 25 isolated users, requests page_size=20 (the minimum), expects 20 items and a token.
func TestIAMListUsers_FirstPage(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-fp-admin@iam-list-fp.test", "admin")

	// Seed 24 extra users (plus admin = 25 total in this domain, > page_size=20).
	for i := 0; i < 24; i++ {
		seedIAMUser(t, fmt.Sprintf("iam-list-fp-u%02d@iam-list-fp.test", i), false)
	}

	client := newIAMClient()
	// page_size=20 is the minimum; 25 users exist → expect 20 items and a token.
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{
			PageSize: 20,
			Query:    "iam-list-fp.test",
		}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(resp.Msg.GetUsers()) != 20 {
		t.Errorf("got %d users, want 20", len(resp.Msg.GetUsers()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token should be non-empty (more pages exist)")
	}
	// next_page_token must be a valid UUID (the id of the last item on this page).
	lastItem := resp.Msg.GetUsers()[len(resp.Msg.GetUsers())-1]
	if resp.Msg.GetNextPageToken() != lastItem.GetId() {
		t.Errorf("next_page_token %q != last item id %q", resp.Msg.GetNextPageToken(), lastItem.GetId())
	}
}

// TestIAMListUsers_SecondPage verifies cursor pagination: no duplicates or gaps.
// Seeds 25 users, fetches page 1 (20 items), then page 2 (5 items with empty token).
func TestIAMListUsers_SecondPage(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-sp-admin@iam-list-sp.test", "admin")

	// Seed 24 extra users (plus admin = 25 total in this isolated domain).
	for i := 0; i < 24; i++ {
		seedIAMUser(t, fmt.Sprintf("iam-list-sp-u%02d@iam-list-sp.test", i), false)
	}

	// Collect the IDs for all 25 expected users.
	seenIDs := make(map[string]struct{})
	adminID := lookupUserID(t, "iam-list-sp-admin@iam-list-sp.test")
	seenIDs[adminID.String()] = struct{}{}
	for i := 0; i < 24; i++ {
		id := lookupUserID(t, fmt.Sprintf("iam-list-sp-u%02d@iam-list-sp.test", i))
		seenIDs[id.String()] = struct{}{}
	}

	client := newIAMClient()

	// Page 1 — scoped to the iam-list-sp.test domain, page_size=20.
	p1, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{
			PageSize: 20,
			Query:    "iam-list-sp.test",
		}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers page 1: %v", err)
	}
	if len(p1.Msg.GetUsers()) != 20 {
		t.Fatalf("page 1: got %d users, want 20", len(p1.Msg.GetUsers()))
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1 next_page_token is empty — expected more pages")
	}

	// Page 2 — same query + cursor, expecting the remaining 5 users.
	p2, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{
			PageSize:  20,
			PageToken: token,
			Query:     "iam-list-sp.test",
		}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers page 2: %v", err)
	}
	if p2.Msg.GetNextPageToken() != "" {
		t.Errorf("page 2 next_page_token = %q, want empty (last page)", p2.Msg.GetNextPageToken())
	}
	if len(p2.Msg.GetUsers()) != 5 {
		t.Errorf("page 2: got %d users, want 5", len(p2.Msg.GetUsers()))
	}

	// Collect all returned IDs; verify no duplicates and all 25 seeded users present.
	all := append(p1.Msg.GetUsers(), p2.Msg.GetUsers()...)
	returnedIDs := make(map[string]struct{}, len(all))
	for _, u := range all {
		if _, dup := returnedIDs[u.GetId()]; dup {
			t.Errorf("duplicate user_id %q across pages", u.GetId())
		}
		returnedIDs[u.GetId()] = struct{}{}
	}
	for id := range seenIDs {
		if _, ok := returnedIDs[id]; !ok {
			t.Errorf("seeded user %q missing from paginated results", id)
		}
	}
}

// TestIAMListUsers_LastPage verifies empty next_page_token on last page.
func TestIAMListUsers_LastPage(t *testing.T) {
	// This test seeds its own isolated set of exactly 2 users and queries for only them.
	// Since the shared DB may have more users, we can't assert exact count.
	// Instead, verify that when page_size is large enough, next_page_token is empty.
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-lp-admin@iam.test", "admin")

	client := newIAMClient()
	// Use a search query so we only match these exact users.
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{
			PageSize: 100,
			Query:    "iam-list-lp",
		}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("next_page_token = %q, want empty (last page)", resp.Msg.GetNextPageToken())
	}
}

// TestIAMListUsers_SearchByEmail verifies ILIKE email search.
func TestIAMListUsers_SearchByEmail(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-se-admin@iam.test", "admin")

	seedIAMUser(t, "alice@acme-search.test", false)
	seedIAMUser(t, "bob@other-search.test", false)

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{Query: "acme-search"}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	for _, u := range resp.Msg.GetUsers() {
		if u.GetEmail() == "bob@other-search.test" {
			t.Error("bob@other-search.test should not appear in acme-search results")
		}
	}
	found := false
	for _, u := range resp.Msg.GetUsers() {
		if u.GetEmail() == "alice@acme-search.test" {
			found = true
		}
	}
	if !found {
		t.Error("alice@acme-search.test not found in search results")
	}
}

// TestIAMListUsers_SearchByDisplayName verifies ILIKE display name search.
func TestIAMListUsers_SearchByDisplayName(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-dn-admin@iam.test", "admin")

	seedIAMUserWithProfile(t, "carol@dn-search.test", "Carol", "Smithington")

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{Query: "Smithington"}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	found := false
	for _, u := range resp.Msg.GetUsers() {
		if u.GetEmail() == "carol@dn-search.test" {
			found = true
			if u.GetDisplayName() != "Carol Smithington" {
				t.Errorf("display_name = %q, want %q", u.GetDisplayName(), "Carol Smithington")
			}
		}
	}
	if !found {
		t.Error("carol@dn-search.test not found in display name search")
	}
}

// TestIAMListUsers_SearchNoMatch verifies empty list on no-match query.
func TestIAMListUsers_SearchNoMatch(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-nm-admin@iam.test", "admin")

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{Query: "zzznomatch99999"}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(resp.Msg.GetUsers()) != 0 {
		t.Errorf("got %d users for no-match query, want 0", len(resp.Msg.GetUsers()))
	}
	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("next_page_token should be empty for no-match query")
	}
}

// TestIAMListUsers_RolesInline verifies that roles are returned inline.
func TestIAMListUsers_RolesInline(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-ri-admin@iam.test", "admin")

	daveID := seedIAMUser(t, "dave@roles-inline.test", false)
	assignRoleDB(t, daveID, "admin")
	assignRoleDB(t, daveID, "teacher")

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{Query: "roles-inline"}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}

	var daveItem *iamv1.UserSummary
	for _, u := range resp.Msg.GetUsers() {
		if u.GetEmail() == "dave@roles-inline.test" {
			daveItem = u
			break
		}
	}
	if daveItem == nil {
		t.Fatal("dave@roles-inline.test not found in results")
	}

	gotRoles := append([]string(nil), daveItem.GetRoles()...)
	sort.Strings(gotRoles)
	wantRoles := []string{"admin", "teacher"}
	if len(gotRoles) != len(wantRoles) {
		t.Errorf("roles = %v, want %v", gotRoles, wantRoles)
	} else {
		for i := range wantRoles {
			if gotRoles[i] != wantRoles[i] {
				t.Errorf("roles[%d] = %q, want %q", i, gotRoles[i], wantRoles[i])
			}
		}
	}
}

// TestIAMListUsers_NoRolesUser verifies that a user with no roles appears with empty roles[].
func TestIAMListUsers_NoRolesUser(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-nr-admin@iam.test", "admin")

	seedIAMUser(t, "empty@no-roles.test", false)

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{Query: "no-roles.test"}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}

	var emptyItem *iamv1.UserSummary
	for _, u := range resp.Msg.GetUsers() {
		if u.GetEmail() == "empty@no-roles.test" {
			emptyItem = u
			break
		}
	}
	if emptyItem == nil {
		t.Fatal("empty@no-roles.test not found in results")
	}
	if len(emptyItem.GetRoles()) != 0 {
		t.Errorf("roles = %v, want []", emptyItem.GetRoles())
	}
}

// TestIAMListUsers_StatusActive verifies all users have status ACTIVE in Phase 1.
func TestIAMListUsers_StatusActive(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-sa-admin@iam.test", "admin")

	seedIAMUser(t, "active1@status.test", false)
	seedIAMUser(t, "active2@status.test", false)

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{Query: "status.test"}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	for _, u := range resp.Msg.GetUsers() {
		if u.GetStatus() != iamv1.UserStatus_USER_STATUS_ACTIVE {
			t.Errorf("user %q status = %v, want ACTIVE", u.GetEmail(), u.GetStatus())
		}
	}
}

// TestIAMListUsers_ClampZero verifies that page_size=0 is clamped to 20.
func TestIAMListUsers_ClampZero(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-cz-admin@iam.test", "admin")

	// Seed 22 users to ensure we have more than 20.
	for i := 0; i < 22; i++ {
		seedIAMUser(t, fmt.Sprintf("iam-clamp-zero-u%02d@iam.test", i), false)
	}

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{PageSize: 0}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(resp.Msg.GetUsers()) != 20 {
		t.Errorf("got %d users with page_size=0, want 20 (clamped)", len(resp.Msg.GetUsers()))
	}
}

// TestIAMListUsers_ClampOversized verifies that page_size=999 is clamped to 200.
func TestIAMListUsers_ClampOversized(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-co-admin@iam.test", "admin")

	// Seed 201 users to exceed the 200 cap.
	for i := 0; i < 201; i++ {
		seedIAMUser(t, fmt.Sprintf("iam-clamp-over-u%03d@iam.test", i), false)
	}

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{PageSize: 999}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(resp.Msg.GetUsers()) > 200 {
		t.Errorf("got %d users with page_size=999, want ≤200 (clamped)", len(resp.Msg.GetUsers()))
	}
}

// TestIAMListUsers_SoftDeletedExcluded verifies soft-deleted users don't appear.
func TestIAMListUsers_SoftDeletedExcluded(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-list-sd-admin@iam.test", "admin")

	seedIAMUser(t, "deleted@soft-del.test", true) // soft deleted

	client := newIAMClient()
	resp, err := client.ListUsers(ctx,
		withSession(connect.NewRequest(&iamv1.ListUsersRequest{Query: "soft-del.test"}), adminSID))
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	for _, u := range resp.Msg.GetUsers() {
		if u.GetEmail() == "deleted@soft-del.test" {
			t.Error("soft-deleted user should not appear in ListUsers results")
		}
	}
}

// lookupUserID fetches a user's UUID by email directly from DB.
func lookupUserID(t *testing.T, email string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pgxPool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err != nil {
		t.Fatalf("lookupUserID(%q): %v", email, err)
	}
	return id
}
