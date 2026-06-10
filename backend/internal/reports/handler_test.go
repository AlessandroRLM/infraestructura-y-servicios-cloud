package reports

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	reportsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1"
)

// fakeService stubs the handler's service dependency to ensure the handler does not
// call the service when UUID parsing fails.
type fakeService struct {
	getSectionGradeReportCalled bool
}

func (f *fakeService) GetSectionGradeReport(_ context.Context, _ uuid.UUID) (*reportsv1.GetSectionGradeReportResponse, error) {
	f.getSectionGradeReportCalled = true
	return &reportsv1.GetSectionGradeReportResponse{}, nil
}

func (f *fakeService) GetSectionOccupancyReport(_ context.Context, _ uuid.UUID) (*reportsv1.GetSectionOccupancyReportResponse, error) {
	return &reportsv1.GetSectionOccupancyReportResponse{}, nil
}

func (f *fakeService) GetProgramSummaryReport(_ context.Context, _ uuid.UUID, _ int32) (*reportsv1.GetProgramSummaryReportResponse, error) {
	return &reportsv1.GetProgramSummaryReportResponse{}, nil
}

func (f *fakeService) GetStudentRecordReport(_ context.Context, _ uuid.UUID) (*reportsv1.GetStudentRecordReportResponse, error) {
	return &reportsv1.GetStudentRecordReportResponse{}, nil
}

// TestGetSectionGradeReport_MalformedUUID_ReturnsCodeInvalidArgument verifies that a malformed
// section_id in the request is rejected at the handler level with CodeInvalidArgument, and that
// the service method is never called.
func TestGetSectionGradeReport_MalformedUUID_ReturnsCodeInvalidArgument(t *testing.T) {
	svc := &fakeService{}
	h := NewHandler(svc)

	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
		SectionId: "not-a-valid-uuid",
	})

	_, err := h.GetSectionGradeReport(context.Background(), req)

	if err == nil {
		t.Fatal("expected an error for malformed UUID, got nil")
	}
	var connectErr *connect.Error
	if ok := connect.IsNotModifiedError(err); ok || err == nil {
		t.Fatalf("expected connect.Error, got: %v", err)
	}
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connectErr.Code())
	}
	if svc.getSectionGradeReportCalled {
		t.Error("service.GetSectionGradeReport must NOT be called when UUID is invalid")
	}
}

// TestGetSectionGradeReport_ValidUUID_DelegatesToService verifies that a valid UUID is
// parsed and forwarded to the service without error from the handler.
func TestGetSectionGradeReport_ValidUUID_DelegatesToService(t *testing.T) {
	svc := &fakeService{}
	h := NewHandler(svc)

	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
		SectionId: uuid.New().String(),
	})

	_, err := h.GetSectionGradeReport(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error for valid UUID: %v", err)
	}
	if !svc.getSectionGradeReportCalled {
		t.Error("expected service.GetSectionGradeReport to be called for valid UUID")
	}
}
