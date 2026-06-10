package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	gradesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1"
	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
)

// TestGradesOutcome_AllGradedPassed (AS-9): all evaluations graded, weighted final ≥ 4.0
// → SE.status = "passed" and SE.final_grade reflects the computed value.
//
// Scheme: weights 0.5/0.3/0.2, values 5.0/4.5/5.0
// Weighted sum = 5.0×0.5 + 4.5×0.3 + 5.0×0.2 = 2.5 + 1.35 + 1.0 = 4.85
// Round to 1 decimal: 4.85 → rounds to 4.9 (HALF-UP)
// 4.9 ≥ 4.0 → passed
func TestGradesOutcome_AllGradedPassed(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "outcome-passed")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"0.5", "0.3", "0.2"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "outcome-passed", fix.SectionID)
	client := newGradesClient(nil)

	values := []string{"5.0", "4.5", "5.0"}
	for i, v := range values {
		resp, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
			EvaluationId:        evals[i].GetId(),
			SectionEnrollmentId: fix.SectionEnrollmentID,
			Value:               v,
		}), teacherSID))
		if err != nil {
			t.Fatalf("RecordGrade eval[%d]: %v", i, err)
		}
		withGradeCleanup(t, resp.Msg.GetGrade().GetId())
	}

	// SE should now be passed with final_grade = 4.9.
	assertSEStatus(t, fix.SectionEnrollmentID, "passed", ptr("4.9"))
}

// TestGradesOutcome_AllGradedFailed (AS-10): all evaluations graded, weighted final < 4.0
// → SE.status = "failed".
//
// Scheme: weight 1.0, value 3.5
// Sum = 3.5 → 3.5 < 4.0 → failed
func TestGradesOutcome_AllGradedFailed(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "outcome-failed")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "outcome-failed", fix.SectionID)
	client := newGradesClient(nil)

	resp, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "3.5",
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade: %v", err)
	}
	withGradeCleanup(t, resp.Msg.GetGrade().GetId())

	assertSEStatus(t, fix.SectionEnrollmentID, "failed", ptr("3.5"))
}

// TestGradesOutcome_PartialGrading (AS-11): not all evaluations filled → SE stays in_progress.
func TestGradesOutcome_PartialGrading(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "outcome-partial")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"0.5", "0.5"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "outcome-partial", fix.SectionID)
	client := newGradesClient(nil)

	// Only grade the first evaluation.
	resp, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "7.0",
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade (partial): %v", err)
	}
	withGradeCleanup(t, resp.Msg.GetGrade().GetId())

	// SE must still be in_progress with no final_grade.
	assertSEStatus(t, fix.SectionEnrollmentID, "in_progress", nil)
}

// TestGradesOutcome_CorrectionFlipsPassedToFailed (AS-12b): correct a grade so the
// weighted final drops below 4.0 → SE flips from passed to failed.
//
// Scheme: weights 0.5/0.5
// Initial: 5.0/4.0 → sum = 2.5 + 2.0 = 4.5 → passed
// After correction of grade[1] to 1.0: 5.0/1.0 → sum = 2.5 + 0.5 = 3.0 → failed
func TestGradesOutcome_CorrectionFlipsPassedToFailed(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "outcome-flip")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"0.5", "0.5"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "outcome-flip", fix.SectionID)
	client := newGradesClient(nil)

	// Grade both evaluations: 5.0 and 4.0 → passed.
	r0, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade eval[0]: %v", err)
	}
	withGradeCleanup(t, r0.Msg.GetGrade().GetId())

	r1, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[1].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "4.0",
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade eval[1]: %v", err)
	}
	withGradeCleanup(t, r1.Msg.GetGrade().GetId())

	assertSEStatus(t, fix.SectionEnrollmentID, "passed", ptr("4.5"))

	// Correct grade[1] to 1.0 → final drops to 3.0 → failed.
	v1 := int32(1)
	_, err = client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[1].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "1.0",
		ExpectedVersion:     &v1,
	}), teacherSID))
	if err != nil {
		t.Fatalf("RecordGrade correction: %v", err)
	}

	assertSEStatus(t, fix.SectionEnrollmentID, "failed", ptr("3.0"))
}

// TestGradesOutcome_RoundHalfUpBoundary: value producing 3.95 rounds to 4.0 → passed;
// value producing 3.94 rounds to 3.9 → failed.
//
// For a single-evaluation scheme (weight=1.0):
//   - value = 3.95 → final = 3.95 → rounds to 4.0 → passed
//   - value = 3.9  → final = 3.9  → rounds to 3.9 → failed (stays < 4.0)
//
// Note: grade constraint is NUMERIC(3,1), so 3.95 can't be stored directly.
// Use a 2-eval scheme (weights 0.5/0.5) with values 4.0 and 3.9 to get 3.95.
// 4.0×0.5 + 3.9×0.5 = 2.0 + 1.95 = 3.95 → round-half-up → 4.0 → passed.
func TestGradesOutcome_RoundHalfUpBoundary_Passes(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "round-halfup-pass")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"0.5", "0.5"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "round-halfup-pass", fix.SectionID)
	client := newGradesClient(nil)

	// 4.0×0.5 + 3.9×0.5 = 3.95 → rounds to 4.0 → passed.
	for i, v := range []string{"4.0", "3.9"} {
		resp, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
			EvaluationId:        evals[i].GetId(),
			SectionEnrollmentId: fix.SectionEnrollmentID,
			Value:               v,
		}), teacherSID))
		if err != nil {
			t.Fatalf("RecordGrade eval[%d]: %v", i, err)
		}
		withGradeCleanup(t, resp.Msg.GetGrade().GetId())
	}

	assertSEStatus(t, fix.SectionEnrollmentID, "passed", ptr("4.0"))
}

// TestGradesOutcome_RoundHalfUpBoundary_Fails: sum 3.94 → rounds to 3.9 → failed.
// Use weights 0.6/0.4 and values 4.0/3.8:
// 4.0×0.6 + 3.8×0.4 = 2.4 + 1.52 = 3.92 → rounds to 3.9 → failed.
func TestGradesOutcome_RoundHalfUpBoundary_Fails(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "round-halfup-fail")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"0.6", "0.4"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "round-halfup-fail", fix.SectionID)
	client := newGradesClient(nil)

	// 4.0×0.6 + 3.8×0.4 = 2.4 + 1.52 = 3.92 → rounds to 3.9 → failed.
	for i, v := range []string{"4.0", "3.8"} {
		resp, err := client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
			EvaluationId:        evals[i].GetId(),
			SectionEnrollmentId: fix.SectionEnrollmentID,
			Value:               v,
		}), teacherSID))
		if err != nil {
			t.Fatalf("RecordGrade eval[%d]: %v", i, err)
		}
		withGradeCleanup(t, resp.Msg.GetGrade().GetId())
	}

	assertSEStatus(t, fix.SectionEnrollmentID, "failed", ptr("3.9"))
}

// TestGradesOutcome_WithdrawnSEBlocksGrade (AS-24-related): grading a withdrawn
// section_enrollment → CodeFailedPrecondition.
func TestGradesOutcome_WithdrawnSEBlocksGrade(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedGradesAdminSID(t, "withdrawn-se")
	fix := seedGradesFixture(t, adminSID)

	evals := seedEvaluationScheme(t, fix.CourseID, []string{"1.0"}, adminSID)
	_, teacherSID := gradesSeedTeacherWithSession(t, "withdrawn-se", fix.SectionID)

	// Withdraw the section enrollment.
	seClient := newSectionEnrollmentClient(nil)
	withdrawReq := connect.NewRequest(&section_enrollmentv1.WithdrawSectionRequest{Id: fix.SectionEnrollmentID})
	withdrawReq.Header().Set("Cookie", "sid="+adminSID)
	_, err := seClient.WithdrawSection(ctx, withdrawReq)
	if err != nil {
		t.Fatalf("WithdrawSection: %v", err)
	}

	client := newGradesClient(nil)
	_, err = client.RecordGrade(ctx, withSID(connect.NewRequest(&gradesv1.RecordGradeRequest{
		EvaluationId:        evals[0].GetId(),
		SectionEnrollmentId: fix.SectionEnrollmentID,
		Value:               "5.0",
	}), teacherSID))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}
