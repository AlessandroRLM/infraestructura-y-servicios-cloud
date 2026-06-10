package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	gradesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1"
)

// TestGradesScheme_CreateHappyPath (AS-1): admin creates a 3-evaluation scheme
// with weights that sum to 1.0 and verifies the rows are persisted.
func TestGradesScheme_CreateHappyPath(t *testing.T) {
	ctx := context.Background()
	adminID, adminSID := seedGradesAdminSID(t, "create-happy")
	_ = adminID
	fix := seedGradesFixture(t, adminSID)
	client := newGradesClient(nil)

	resp, err := client.CreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "0.3"},
			{Weight: "0.3"},
			{Weight: "0.4"},
		},
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateEvaluationScheme: %v", err)
	}

	evals := resp.Msg.GetEvaluations()
	if len(evals) != 3 {
		t.Fatalf("expected 3 evaluations, got %d", len(evals))
	}
	// Verify positions 1, 2, 3.
	for i, e := range evals {
		if e.GetPosition() != int32(i+1) {
			t.Errorf("evaluation[%d].position = %d, want %d", i, e.GetPosition(), i+1)
		}
		if e.GetCourseId() != fix.CourseID {
			t.Errorf("evaluation[%d].course_id = %q, want %q", i, e.GetCourseId(), fix.CourseID)
		}
	}

	// Cleanup.
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`UPDATE evaluations SET deleted_at = now() WHERE course_id = $1 AND deleted_at IS NULL`,
			fix.CourseID)
	})
}

// TestGradesScheme_CreateSingleEvaluation: a single-evaluation scheme (weight=1.0) is valid.
func TestGradesScheme_CreateSingleEvaluation(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "create-single")
	fix := seedGradesFixture(t, adminSID)
	client := newGradesClient(nil)

	resp, err := client.CreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "1.0"},
		},
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateEvaluationScheme (single): %v", err)
	}
	if len(resp.Msg.GetEvaluations()) != 1 {
		t.Fatalf("expected 1 evaluation, got %d", len(resp.Msg.GetEvaluations()))
	}

	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(),
			`UPDATE evaluations SET deleted_at = now() WHERE course_id = $1 AND deleted_at IS NULL`,
			fix.CourseID)
	})
}

// TestGradesScheme_WeightsDontSumTo1 (AS-2): weights that do not sum to 1.0 → CodeInvalidArgument.
func TestGradesScheme_WeightsDontSumTo1(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "weights-bad-sum")
	fix := seedGradesFixture(t, adminSID)
	client := newGradesClient(nil)

	_, err := client.CreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "0.4"},
			{Weight: "0.4"},
		},
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestGradesScheme_WeightOutOfRange: a weight of 0 → CodeInvalidArgument.
func TestGradesScheme_WeightOutOfRange(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "weight-zero")
	fix := seedGradesFixture(t, adminSID)
	client := newGradesClient(nil)

	_, err := client.CreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "0"},
			{Weight: "1.0"},
		},
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestGradesScheme_DuplicateScheme (AS-3): creating a second scheme for the same course
// while the first is still live → CodeAlreadyExists.
func TestGradesScheme_DuplicateScheme(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "duplicate-scheme")
	fix := seedGradesFixture(t, adminSID)
	_ = seedEvaluationScheme(t, fix.CourseID, []string{"0.5", "0.5"}, adminSID)
	client := newGradesClient(nil)

	_, err := client.CreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "1.0"},
		},
	}), adminSID))
	assertConnectCode(t, err, connect.CodeAlreadyExists)
}

// TestGradesScheme_TeacherCannotCreate (AS-21): a user with only grades.write permission
// cannot call CreateEvaluationScheme → CodePermissionDenied.
func TestGradesScheme_TeacherCannotCreate(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "teacher-cannot-create")
	fix := seedGradesFixture(t, adminSID)

	// Create a teacher user (role=teacher, which has grades.write but not grades.override).
	teacherIDStr, teacherSID := gradesSeedTeacherWithSession(t, "scheme-authz", fix.SectionID)
	_ = teacherIDStr
	client := newGradesClient(nil)

	_, err := client.CreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "1.0"},
		},
	}), teacherSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestGradesScheme_UnauthenticatedDenied (AS-22): no session → CodeUnauthenticated.
func TestGradesScheme_UnauthenticatedDenied(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "unauth-denied")
	fix := seedGradesFixture(t, adminSID)
	client := newGradesClient(nil)

	_, err := client.CreateEvaluationScheme(ctx, connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "1.0"},
		},
	}))
	// No Cookie header → unauthenticated.
	assertConnectCode(t, err, connect.CodeUnauthenticated)
}

// TestGradesScheme_StudentCannotCreate: student role → CodePermissionDenied.
func TestGradesScheme_StudentCannotCreate(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "student-cannot-create")
	fix := seedGradesFixture(t, adminSID)
	client := newGradesClient(nil)

	_, err := client.CreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.CreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "1.0"},
		},
	}), fix.StudentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

// TestGradesScheme_ListEvaluations: admin can list evaluations for a course.
func TestGradesScheme_ListEvaluations(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "list-evals")
	fix := seedGradesFixture(t, adminSID)
	_ = seedEvaluationScheme(t, fix.CourseID, []string{"0.6", "0.4"}, adminSID)
	client := newGradesClient(nil)

	resp, err := client.ListEvaluations(ctx, withSID(connect.NewRequest(&gradesv1.ListEvaluationsRequest{
		CourseId: fix.CourseID,
	}), adminSID))
	if err != nil {
		t.Fatalf("ListEvaluations: %v", err)
	}
	if len(resp.Msg.GetEvaluations()) != 2 {
		t.Errorf("ListEvaluations count = %d, want 2", len(resp.Msg.GetEvaluations()))
	}
}

// TestGradesScheme_RecreateHappyPath: admin recreates a scheme (no grades) →
// old evaluations soft-deleted, new set visible.
func TestGradesScheme_RecreateHappyPath(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "recreate-happy")
	fix := seedGradesFixture(t, adminSID)

	// Create initial scheme (2 evaluations).
	initial := seedEvaluationScheme(t, fix.CourseID, []string{"0.5", "0.5"}, adminSID)
	if len(initial) != 2 {
		t.Fatalf("initial scheme: want 2 evals, got %d", len(initial))
	}

	client := newGradesClient(nil)

	// Recreate with 3 evaluations.
	resp, err := client.RecreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.RecreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "0.3"},
			{Weight: "0.3"},
			{Weight: "0.4"},
		},
	}), adminSID))
	if err != nil {
		t.Fatalf("RecreateEvaluationScheme: %v", err)
	}

	newEvals := resp.Msg.GetEvaluations()
	if len(newEvals) != 3 {
		t.Fatalf("recreated scheme: want 3 evals, got %d", len(newEvals))
	}

	// Old evaluations must be soft-deleted.
	for _, old := range initial {
		var deletedAt *string
		err := pgxPool.QueryRow(ctx,
			`SELECT deleted_at::text FROM evaluations WHERE id = $1`, old.GetId(),
		).Scan(&deletedAt)
		if err != nil {
			t.Fatalf("check old eval %s: %v", old.GetId(), err)
		}
		if deletedAt == nil {
			t.Errorf("old evaluation %s should be soft-deleted but deleted_at is NULL", old.GetId())
		}
	}

	// ListEvaluations must return only the new 3.
	listResp, err := client.ListEvaluations(ctx, withSID(connect.NewRequest(&gradesv1.ListEvaluationsRequest{
		CourseId: fix.CourseID,
	}), adminSID))
	if err != nil {
		t.Fatalf("ListEvaluations after recreate: %v", err)
	}
	if len(listResp.Msg.GetEvaluations()) != 3 {
		t.Errorf("ListEvaluations after recreate = %d evals, want 3", len(listResp.Msg.GetEvaluations()))
	}
}

// TestGradesScheme_RecreateBlockedByGrades: recreate blocked when grades exist → CodeFailedPrecondition.
func TestGradesScheme_RecreateBlockedByGrades(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "recreate-blocked")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)

	// Add a teacher assigned to the section so RecordGrade is authorized.
	teacherIDStr, teacherSID := gradesSeedTeacherWithSession(t, "recreate-blocked", fix.SectionID)
	_ = teacherIDStr

	// Record a grade so the scheme is in use.
	_ = seedGrade(t, evals[0].GetId(), fix.SectionEnrollmentID, "5.0", teacherSID)

	client := newGradesClient(nil)

	_, err := client.RecreateEvaluationScheme(ctx, withSID(connect.NewRequest(&gradesv1.RecreateEvaluationSchemeRequest{
		CourseId: fix.CourseID,
		Evaluations: []*gradesv1.EvaluationInput{
			{Weight: "1.0"},
		},
	}), adminSID))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}
