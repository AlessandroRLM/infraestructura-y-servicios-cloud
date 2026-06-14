package integration_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
)

// seedTeacherQualificationBundle creates a teacher (with teacher_profiles row) and
// inserts `count` teacher_qualifications rows for that teacher.
// Returns (teacherID, adminSID, qualIDs, cleanup).
func seedTeacherQualificationBundle(t *testing.T, suffix string, count int) (
	teacherID uuid.UUID,
	adminSID string,
	qualIDs []string,
	cleanup func(),
) {
	t.Helper()
	ctx := context.Background()

	// Seed admin for API calls.
	_, adminSID = seedUserWithSession(t, "tqpag-admin-"+suffix+"@tq.test", "admin")

	// Seed a teacher user.
	teacherID = seedUserWithRole(t, "tqpag-teacher-"+suffix+"@tq.test", "teacher")

	// Ensure teacher_profiles row (FK required by teacher_qualifications).
	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO teacher_profiles (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		teacherID,
	); err != nil {
		t.Fatalf("seedTeacherQualificationBundle: ensure teacher_profiles: %v", err)
	}

	qualIDs = make([]string, 0, count)
	for i := 0; i < count; i++ {
		var qualID uuid.UUID
		if err := pgxPool.QueryRow(ctx,
			`INSERT INTO teacher_qualifications (teacher_id, degree, year)
			 VALUES ($1, $2, $3)
			 RETURNING id`,
			teacherID,
			fmt.Sprintf("Degree-%s-%02d", suffix, i),
			2000+i,
		).Scan(&qualID); err != nil {
			t.Fatalf("seedTeacherQualificationBundle: insert qualification %d: %v", i, err)
		}
		qualIDs = append(qualIDs, qualID.String())
	}

	cleanup = func() {
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM teacher_qualifications WHERE teacher_id = $1`, teacherID)
		_, _ = pgxPool.Exec(context.Background(),
			`DELETE FROM teacher_profiles WHERE user_id = $1`, teacherID)
	}
	return teacherID, adminSID, qualIDs, cleanup
}

// --- ListTeacherQualifications pagination ---

// TestProfiles_ListTeacherQualifications_FirstPage verifies the first page returns
// page_size items and a non-empty next_page_token when more qualifications exist.
func TestProfiles_ListTeacherQualifications_FirstPage(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	teacherID, adminSID, _, cleanup := seedTeacherQualificationBundle(t, suffix, 25)
	defer cleanup()

	client := newProfilesClient(nil)
	req := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
		PageSize:  20,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	resp, err := client.ListTeacherQualifications(ctx, req)
	if err != nil {
		t.Fatalf("ListTeacherQualifications: %v", err)
	}
	if len(resp.Msg.GetQualifications()) != 20 {
		t.Errorf("got %d qualifications, want 20", len(resp.Msg.GetQualifications()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty when more pages exist")
	}
}

// TestProfiles_ListTeacherQualifications_SecondPage verifies no overlap or gap across
// two pages.
func TestProfiles_ListTeacherQualifications_SecondPage(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	teacherID, adminSID, qualIDs, cleanup := seedTeacherQualificationBundle(t, suffix, 25)
	defer cleanup()

	seededIDs := make(map[string]struct{}, len(qualIDs))
	for _, id := range qualIDs {
		seededIDs[id] = struct{}{}
	}

	client := newProfilesClient(nil)

	// Page 1.
	req1 := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
		PageSize:  20,
	})
	req1.Header().Set("Cookie", "sid="+adminSID)
	p1, err := client.ListTeacherQualifications(ctx, req1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	token := p1.Msg.GetNextPageToken()
	if token == "" {
		t.Fatal("page 1: next_page_token must be non-empty")
	}

	page1IDs := make(map[string]struct{})
	for _, q := range p1.Msg.GetQualifications() {
		page1IDs[q.GetId()] = struct{}{}
	}

	// Page 2.
	req2 := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
		PageSize:  20,
		PageToken: token,
	})
	req2.Header().Set("Cookie", "sid="+adminSID)
	p2, err := client.ListTeacherQualifications(ctx, req2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}

	// No duplicates.
	for _, q := range p2.Msg.GetQualifications() {
		if _, dup := page1IDs[q.GetId()]; dup {
			t.Errorf("duplicate qualification id %s across pages", q.GetId())
		}
	}

	// All seeded qualifications appear across both pages.
	allReturned := make(map[string]struct{})
	for _, q := range p1.Msg.GetQualifications() {
		allReturned[q.GetId()] = struct{}{}
	}
	for _, q := range p2.Msg.GetQualifications() {
		allReturned[q.GetId()] = struct{}{}
	}
	for id := range seededIDs {
		if _, ok := allReturned[id]; !ok {
			t.Errorf("seeded qualification id %s missing from paginated results", id)
		}
	}
}

// TestProfiles_ListTeacherQualifications_LastPageEmptyToken seeds a small set and walks
// to exhaustion, verifying the final page returns an empty next_page_token.
func TestProfiles_ListTeacherQualifications_LastPageEmptyToken(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	// 3 qualifications — all fit on one page of size 20.
	teacherID, adminSID, _, cleanup := seedTeacherQualificationBundle(t, "lp-"+suffix, 3)
	defer cleanup()

	client := newProfilesClient(nil)
	req := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListTeacherQualifications(ctx, req)
	if err != nil {
		t.Fatalf("ListTeacherQualifications walk: %v", err)
	}
	if resp.Msg.GetNextPageToken() != "" {
		t.Errorf("next_page_token should be empty on last page (3 quals), got %q",
			resp.Msg.GetNextPageToken())
	}
}

// TestProfiles_ListTeacherQualifications_ClampZero verifies page_size=0 is clamped to 20.
func TestProfiles_ListTeacherQualifications_ClampZero(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	teacherID, adminSID, _, cleanup := seedTeacherQualificationBundle(t, suffix, 25)
	defer cleanup()

	client := newProfilesClient(nil)
	req := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
		PageSize:  0,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListTeacherQualifications(ctx, req)
	if err != nil {
		t.Fatalf("ListTeacherQualifications: %v", err)
	}
	if len(resp.Msg.GetQualifications()) != 20 {
		t.Errorf("page_size=0 → got %d, want 20 (clamped to min)", len(resp.Msg.GetQualifications()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("next_page_token must be non-empty (more pages exist)")
	}
}

// TestProfiles_ListTeacherQualifications_ClampMax verifies page_size=999 is clamped to 200.
func TestProfiles_ListTeacherQualifications_ClampMax(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	// Seed only 5 — well under any page size limit. The point is to verify the
	// clamp is accepted without error and all rows are returned.
	teacherID, adminSID, qualIDs, cleanup := seedTeacherQualificationBundle(t, "max-"+suffix, 5)
	defer cleanup()

	client := newProfilesClient(nil)
	req := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
		PageSize:  999,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListTeacherQualifications(ctx, req)
	if err != nil {
		t.Fatalf("ListTeacherQualifications page_size=999: %v", err)
	}
	// All 5 seeded qualifications must be returned.
	if len(resp.Msg.GetQualifications()) != len(qualIDs) {
		t.Errorf("page_size=999 → got %d, want %d", len(resp.Msg.GetQualifications()), len(qualIDs))
	}
	if resp.Msg.GetNextPageToken() != "" {
		t.Error("next_page_token should be empty (all 5 quals fit within 200)")
	}
}

// TestProfiles_ListTeacherQualifications_InvalidToken verifies a malformed page_token
// returns CodeInvalidArgument.
func TestProfiles_ListTeacherQualifications_InvalidToken(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "tqpag-inv-admin@tq.test", "admin")
	client := newProfilesClient(nil)

	req := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: uuid.New().String(),
		PageToken: "not-a-uuid",
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	_, err := client.ListTeacherQualifications(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProfiles_ListTeacherQualifications_TeacherIDFilterPreserved verifies that the
// teacher_id filter is preserved alongside pagination — another teacher's qualifications
// are never returned.
func TestProfiles_ListTeacherQualifications_TeacherIDFilterPreserved(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]

	// Teacher 1 with 5 qualifications.
	teacher1ID, adminSID, qual1IDs, cleanup1 := seedTeacherQualificationBundle(t, suffix+"-t1", 5)
	defer cleanup1()

	// Teacher 2 with 3 qualifications (decoys).
	_, _, decoyIDs, cleanup2 := seedTeacherQualificationBundle(t, suffix+"-t2", 3)
	defer cleanup2()

	client := newProfilesClient(nil)
	req := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacher1ID.String(),
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListTeacherQualifications(ctx, req)
	if err != nil {
		t.Fatalf("ListTeacherQualifications: %v", err)
	}

	returned := make(map[string]struct{})
	for _, q := range resp.Msg.GetQualifications() {
		returned[q.GetId()] = struct{}{}
	}

	// All of teacher1's qualifications must appear.
	for _, id := range qual1IDs {
		if _, ok := returned[id]; !ok {
			t.Errorf("teacher1 qual %s missing from result", id)
		}
	}

	// None of teacher2's decoy qualifications may appear.
	for _, id := range decoyIDs {
		if _, ok := returned[id]; ok {
			t.Errorf("teacher2 decoy qual %s leaked into teacher1 result", id)
		}
	}
}

// TestProfiles_ListTeacherQualifications_IDDescOrder verifies results are ordered by
// id DESC.
func TestProfiles_ListTeacherQualifications_IDDescOrder(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	teacherID, adminSID, _, cleanup := seedTeacherQualificationBundle(t, suffix, 5)
	defer cleanup()

	client := newProfilesClient(nil)
	req := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
		PageSize:  20,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListTeacherQualifications(ctx, req)
	if err != nil {
		t.Fatalf("ListTeacherQualifications: %v", err)
	}
	quals := resp.Msg.GetQualifications()
	for i := 1; i < len(quals); i++ {
		if quals[i-1].GetId() <= quals[i].GetId() {
			t.Errorf("quals[%d].id=%s >= quals[%d].id=%s (want DESC order)",
				i-1, quals[i-1].GetId(), i, quals[i].GetId())
		}
	}
}

// TestProfiles_ListTeacherQualifications_SoftDeleteExcluded verifies that soft-deleted
// qualifications never appear in any page.
func TestProfiles_ListTeacherQualifications_SoftDeleteExcluded(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.New().String()[:8]
	teacherID, adminSID, qualIDs, cleanup := seedTeacherQualificationBundle(t, "sd-"+suffix, 5)
	defer cleanup()

	// Soft-delete the first qualification directly.
	deletedID := qualIDs[0]
	if _, err := pgxPool.Exec(ctx,
		`UPDATE teacher_qualifications SET deleted_at = now() WHERE id = $1`,
		deletedID,
	); err != nil {
		t.Fatalf("soft-delete qualification: %v", err)
	}

	client := newProfilesClient(nil)
	req := connect.NewRequest(&profilesv1.ListTeacherQualificationsRequest{
		TeacherId: teacherID.String(),
		PageSize:  200,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.ListTeacherQualifications(ctx, req)
	if err != nil {
		t.Fatalf("ListTeacherQualifications: %v", err)
	}

	for _, q := range resp.Msg.GetQualifications() {
		if q.GetId() == deletedID {
			t.Errorf("soft-deleted qualification %s appeared in result", deletedID)
		}
	}

	// Expect exactly 4 (5 seeded minus 1 deleted).
	if len(resp.Msg.GetQualifications()) != 4 {
		t.Errorf("got %d qualifications, want 4 (1 soft-deleted)", len(resp.Msg.GetQualifications()))
	}
}
