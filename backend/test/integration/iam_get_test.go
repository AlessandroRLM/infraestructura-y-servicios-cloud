package integration_test

import (
	"context"
	"sort"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	iamv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/iam/v1"
)

// TestIAMGetUser_Found verifies that an admin can retrieve a user with roles and status.
func TestIAMGetUser_Found(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "iam-get-found-admin@iam.test", "admin")

	targetID := seedIAMUserWithProfile(t, "alice@get-found.test", "Alice", "Walker")
	assignRoleDB(t, targetID, "teacher")
	assignRoleDB(t, targetID, "student")

	client := newIAMClient()
	resp, err := client.GetUser(ctx,
		withSession(connect.NewRequest(&iamv1.GetUserRequest{UserId: targetID.String()}), adminSID))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	u := resp.Msg.GetUser()
	if u.GetId() != targetID.String() {
		t.Errorf("id = %q, want %q", u.GetId(), targetID.String())
	}
	if u.GetEmail() != "alice@get-found.test" {
		t.Errorf("email = %q, want %q", u.GetEmail(), "alice@get-found.test")
	}
	if u.GetDisplayName() != "Alice Walker" {
		t.Errorf("display_name = %q, want %q", u.GetDisplayName(), "Alice Walker")
	}
	if u.GetStatus() != iamv1.UserStatus_USER_STATUS_ACTIVE {
		t.Errorf("status = %v, want ACTIVE", u.GetStatus())
	}

	gotRoles := append([]string(nil), u.GetRoles()...)
	sort.Strings(gotRoles)
	wantRoles := []string{"student", "teacher"}
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

// TestIAMGetUser_NotFound verifies CodeNotFound for a non-existent UUID.
func TestIAMGetUser_NotFound(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "iam-get-nf-admin@iam.test", "admin")
	client := newIAMClient()

	nonExistent := uuid.New().String()
	_, err := client.GetUser(context.Background(),
		withSession(connect.NewRequest(&iamv1.GetUserRequest{UserId: nonExistent}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestIAMGetUser_SoftDeleted verifies CodeNotFound for a soft-deleted user.
func TestIAMGetUser_SoftDeleted(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "iam-get-sd-admin@iam.test", "admin")
	deletedID := seedIAMUser(t, "gone@soft-del-get.test", true) // soft deleted

	client := newIAMClient()
	_, err := client.GetUser(context.Background(),
		withSession(connect.NewRequest(&iamv1.GetUserRequest{UserId: deletedID.String()}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// TestIAMGetUser_BadUUID verifies CodeInvalidArgument for a non-UUID user_id.
func TestIAMGetUser_BadUUID(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "iam-get-bad-uuid-admin@iam.test", "admin")
	client := newIAMClient()

	_, err := client.GetUser(context.Background(),
		withSession(connect.NewRequest(&iamv1.GetUserRequest{UserId: "not-a-uuid"}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestIAMGetUser_NoProfileRow verifies that display_name falls back to email.
func TestIAMGetUser_NoProfileRow(t *testing.T) {
	_, adminSID := seedUserWithSession(t, "iam-get-np-admin@iam.test", "admin")
	noprofileID := seedIAMUser(t, "noprofile@get-fallback.test", false) // no profile

	client := newIAMClient()
	resp, err := client.GetUser(context.Background(),
		withSession(connect.NewRequest(&iamv1.GetUserRequest{UserId: noprofileID.String()}), adminSID))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if resp.Msg.GetUser().GetDisplayName() != "noprofile@get-fallback.test" {
		t.Errorf("display_name = %q, want email fallback %q",
			resp.Msg.GetUser().GetDisplayName(), "noprofile@get-fallback.test")
	}
}

// TestIAMGetUser_StudentDenied verifies CodePermissionDenied for a student caller.
func TestIAMGetUser_StudentDenied(t *testing.T) {
	_, studentSID := seedUserWithSession(t, "iam-get-deny-student@iam.test", "student")
	targetID := seedIAMUser(t, "target@get-deny.test", false)

	client := newIAMClient()
	_, err := client.GetUser(context.Background(),
		withSession(connect.NewRequest(&iamv1.GetUserRequest{UserId: targetID.String()}), studentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}
