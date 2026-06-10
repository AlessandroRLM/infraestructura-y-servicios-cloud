package reports

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	reportsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1/reportsv1connect"
)

// reportsService is the handler's consumer-side interface so unit tests can inject fakes.
type reportsService interface {
	GetSectionGradeReport(ctx context.Context, sectionID uuid.UUID) (*reportsv1.GetSectionGradeReportResponse, error)
	GetSectionOccupancyReport(ctx context.Context, periodID uuid.UUID) (*reportsv1.GetSectionOccupancyReportResponse, error)
	GetProgramSummaryReport(ctx context.Context, programID uuid.UUID, year int32) (*reportsv1.GetProgramSummaryReportResponse, error)
	GetStudentRecordReport(ctx context.Context, studentID uuid.UUID) (*reportsv1.GetStudentRecordReportResponse, error)
}

// Compile-time proof that *Service satisfies reportsService.
var _ reportsService = (*Service)(nil)

// Handler implements reportsv1connect.ReportsServiceHandler.
type Handler struct {
	svc reportsService
}

// NewHandler constructs a Connect handler wrapping the ReportsService.
func NewHandler(svc reportsService) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the ReportsService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := reportsv1connect.NewReportsServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// --- RPC implementations ---

// GetSectionGradeReport parses the section_id and delegates to the service.
func (h *Handler) GetSectionGradeReport(
	ctx context.Context,
	req *connect.Request[reportsv1.GetSectionGradeReportRequest],
) (*connect.Response[reportsv1.GetSectionGradeReportResponse], error) {
	sectionID, err := parseUUID("section_id", req.Msg.GetSectionId())
	if err != nil {
		return nil, err
	}
	resp, err := h.svc.GetSectionGradeReport(ctx, sectionID)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(resp), nil
}

// GetSectionOccupancyReport parses the academic_period_id and delegates to the service.
func (h *Handler) GetSectionOccupancyReport(
	ctx context.Context,
	req *connect.Request[reportsv1.GetSectionOccupancyReportRequest],
) (*connect.Response[reportsv1.GetSectionOccupancyReportResponse], error) {
	periodID, err := parseUUID("academic_period_id", req.Msg.GetAcademicPeriodId())
	if err != nil {
		return nil, err
	}
	resp, err := h.svc.GetSectionOccupancyReport(ctx, periodID)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(resp), nil
}

// GetProgramSummaryReport parses the program_id and delegates to the service.
func (h *Handler) GetProgramSummaryReport(
	ctx context.Context,
	req *connect.Request[reportsv1.GetProgramSummaryReportRequest],
) (*connect.Response[reportsv1.GetProgramSummaryReportResponse], error) {
	programID, err := parseUUID("program_id", req.Msg.GetProgramId())
	if err != nil {
		return nil, err
	}
	resp, err := h.svc.GetProgramSummaryReport(ctx, programID, req.Msg.GetYear())
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(resp), nil
}

// GetStudentRecordReport parses the student_id and delegates to the service.
func (h *Handler) GetStudentRecordReport(
	ctx context.Context,
	req *connect.Request[reportsv1.GetStudentRecordReportRequest],
) (*connect.Response[reportsv1.GetStudentRecordReportResponse], error) {
	studentID, err := parseUUID("student_id", req.Msg.GetStudentId())
	if err != nil {
		return nil, err
	}
	resp, err := h.svc.GetStudentRecordReport(ctx, studentID)
	if err != nil {
		return nil, MapError(ctx, err)
	}
	return connect.NewResponse(resp), nil
}

// parseUUID parses a string UUID, returning a Connect CodeInvalidArgument error on failure.
// The field parameter names the field for error context (e.g. "section_id").
func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("reports: %s is not a valid UUID: %q", field, value),
		)
	}
	return id, nil
}
