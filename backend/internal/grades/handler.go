package grades

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	gradesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/grades/v1/gradesv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/grades/gradesdb"
)

// Handler implements gradesv1connect.GradesServiceHandler.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Connect handler wrapping the GradesService.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the GradesService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := gradesv1connect.NewGradesServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// --- RPC implementations ---

func (h *Handler) CreateEvaluationScheme(
	ctx context.Context,
	req *connect.Request[gradesv1.CreateEvaluationSchemeRequest],
) (*connect.Response[gradesv1.CreateEvaluationSchemeResponse], error) {
	weights := make([]string, len(req.Msg.Evaluations))
	for i, e := range req.Msg.Evaluations {
		weights[i] = e.Weight
	}
	evals, err := h.svc.CreateEvaluationScheme(ctx, req.Msg.CourseId, weights)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(&gradesv1.CreateEvaluationSchemeResponse{
		Evaluations: evaluationsToProto(evals),
	}), nil
}

func (h *Handler) RecreateEvaluationScheme(
	ctx context.Context,
	req *connect.Request[gradesv1.RecreateEvaluationSchemeRequest],
) (*connect.Response[gradesv1.RecreateEvaluationSchemeResponse], error) {
	weights := make([]string, len(req.Msg.Evaluations))
	for i, e := range req.Msg.Evaluations {
		weights[i] = e.Weight
	}
	evals, err := h.svc.RecreateEvaluationScheme(ctx, req.Msg.CourseId, weights)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(&gradesv1.RecreateEvaluationSchemeResponse{
		Evaluations: evaluationsToProto(evals),
	}), nil
}

func (h *Handler) ListEvaluations(
	ctx context.Context,
	req *connect.Request[gradesv1.ListEvaluationsRequest],
) (*connect.Response[gradesv1.ListEvaluationsResponse], error) {
	evals, err := h.svc.ListEvaluations(ctx, req.Msg.CourseId)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(&gradesv1.ListEvaluationsResponse{
		Evaluations: evaluationsToProto(evals),
	}), nil
}

func (h *Handler) RecordGrade(
	ctx context.Context,
	req *connect.Request[gradesv1.RecordGradeRequest],
) (*connect.Response[gradesv1.RecordGradeResponse], error) {
	grade, _, err := h.svc.RecordGrade(ctx,
		req.Msg.EvaluationId,
		req.Msg.SectionEnrollmentId,
		req.Msg.Value,
		req.Msg.ExpectedVersion,
	)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(&gradesv1.RecordGradeResponse{
		Grade: gradeToProto(grade),
	}), nil
}

func (h *Handler) OverrideGrade(
	ctx context.Context,
	req *connect.Request[gradesv1.OverrideGradeRequest],
) (*connect.Response[gradesv1.OverrideGradeResponse], error) {
	grade, _, err := h.svc.OverrideGrade(ctx,
		req.Msg.EvaluationId,
		req.Msg.SectionEnrollmentId,
		req.Msg.Value,
		req.Msg.ExpectedVersion,
	)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(&gradesv1.OverrideGradeResponse{
		Grade: gradeToProto(grade),
	}), nil
}

func (h *Handler) ListGradesForSection(
	ctx context.Context,
	req *connect.Request[gradesv1.ListGradesForSectionRequest],
) (*connect.Response[gradesv1.ListGradesForSectionResponse], error) {
	grades, err := h.svc.ListGradesForSection(ctx, req.Msg.SectionId)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	result := make([]*gradesv1.Grade, len(grades))
	for i, g := range grades {
		result[i] = gradeToProto(g)
	}
	return connect.NewResponse(&gradesv1.ListGradesForSectionResponse{
		Grades: result,
	}), nil
}

func (h *Handler) GetGrade(
	ctx context.Context,
	req *connect.Request[gradesv1.GetGradeRequest],
) (*connect.Response[gradesv1.GetGradeResponse], error) {
	grade, err := h.svc.GetGrade(ctx, req.Msg.Id)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(&gradesv1.GetGradeResponse{
		Grade: gradeToProto(grade),
	}), nil
}

func (h *Handler) ListOwnGrades(
	ctx context.Context,
	_ *connect.Request[gradesv1.ListOwnGradesRequest],
) (*connect.Response[gradesv1.ListOwnGradesResponse], error) {
	grades, err := h.svc.ListOwnGrades(ctx)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	result := make([]*gradesv1.OwnGrade, len(grades))
	for i, g := range grades {
		result[i] = ownGradeToProto(g)
	}
	return connect.NewResponse(&gradesv1.ListOwnGradesResponse{
		Grades: result,
	}), nil
}

// --- Proto converters ---

func evaluationsToProto(evals []gradesdb.Evaluation) []*gradesv1.Evaluation {
	result := make([]*gradesv1.Evaluation, len(evals))
	for i, e := range evals {
		result[i] = evaluationToProto(e)
	}
	return result
}

func evaluationToProto(e gradesdb.Evaluation) *gradesv1.Evaluation {
	return &gradesv1.Evaluation{
		Id:        uuidToString(e.ID),
		CourseId:  uuidToString(e.CourseID),
		Weight:    numericToString(e.Weight),
		Position:  e.Position,
		CreatedAt: e.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: e.UpdatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// gradeToProto converts a DB grade to the teacher/admin wire format (includes graded_by).
func gradeToProto(g gradesdb.Grade) *gradesv1.Grade {
	return &gradesv1.Grade{
		Id:                  uuidToString(g.ID),
		EvaluationId:        uuidToString(g.EvaluationID),
		SectionEnrollmentId: uuidToString(g.SectionEnrollmentID),
		GradedBy:            uuidToString(g.GradedBy),
		Value:               numericToString(g.Value),
		Version:             g.Version,
		CreatedAt:           g.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:           g.UpdatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// ownGradeToProto converts a DB grade to the student-facing wire format (NO graded_by).
func ownGradeToProto(g gradesdb.Grade) *gradesv1.OwnGrade {
	return &gradesv1.OwnGrade{
		Id:                  uuidToString(g.ID),
		EvaluationId:        uuidToString(g.EvaluationID),
		SectionEnrollmentId: uuidToString(g.SectionEnrollmentID),
		Value:               numericToString(g.Value),
		Version:             g.Version,
		CreatedAt:           g.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:           g.UpdatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// uuidToString converts a pgtype.UUID to its string representation.
func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		id.Bytes[0:4], id.Bytes[4:6], id.Bytes[6:8], id.Bytes[8:10], id.Bytes[10:16])
}
