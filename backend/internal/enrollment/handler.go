package enrollment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/enrollment/v1/enrollmentv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/enrollment/enrollmentdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/connectutil"
)

// Handler implements enrollmentv1connect.EnrollmentServiceHandler.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Connect handler wrapping the EnrollmentService.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the EnrollmentService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := enrollmentv1connect.NewEnrollmentServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// MapError converts domain errors to connect.Error codes.
// Exported so that the handler_test package can validate the mapping.
// Unrecognized errors map to CodeInternal with a generic message — the original error
// is never forwarded so that internal details cannot leak to callers.
func MapError(err error) error {
	if errors.Is(err, ErrInvalidInput) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, ErrAlreadyExists) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	if errors.Is(err, ErrQuotaFull) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if errors.Is(err, ErrQuotaNotFound) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if errors.Is(err, ErrInvalidTransition) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	// Unmapped errors: log the original, emit a generic message. Single-handling rule:
	// map or log, not both — this is the only place that logs for unmapped failures.
	slog.Error("enrollment: unhandled internal error", "err", err)
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

// --- Management RPCs ---

// CreateEnrollment creates a new pending enrollment for the given student and program.
func (h *Handler) CreateEnrollment(
	ctx context.Context,
	req *connect.Request[enrollmentv1.CreateEnrollmentRequest],
) (*connect.Response[enrollmentv1.Enrollment], error) {
	row, err := h.svc.CreateEnrollment(ctx,
		req.Msg.GetStudentId(),
		req.Msg.GetProgramId(),
		req.Msg.GetYear(),
	)
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(enrollmentToProto(row)), nil
}

// MarkEnrollmentPaid transitions a pending enrollment to paid and sets paid_at.
func (h *Handler) MarkEnrollmentPaid(
	ctx context.Context,
	req *connect.Request[enrollmentv1.MarkEnrollmentPaidRequest],
) (*connect.Response[enrollmentv1.Enrollment], error) {
	row, err := h.svc.MarkEnrollmentPaid(ctx, req.Msg.GetId())
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(enrollmentToProto(row)), nil
}

// CancelEnrollment transitions a pending or paid enrollment to cancelled.
func (h *Handler) CancelEnrollment(
	ctx context.Context,
	req *connect.Request[enrollmentv1.CancelEnrollmentRequest],
) (*connect.Response[enrollmentv1.CancelEnrollmentResponse], error) {
	if err := h.svc.CancelEnrollment(ctx, req.Msg.GetId()); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&enrollmentv1.CancelEnrollmentResponse{}), nil
}

// GetEnrollment retrieves a live enrollment by id.
func (h *Handler) GetEnrollment(
	ctx context.Context,
	req *connect.Request[enrollmentv1.GetEnrollmentRequest],
) (*connect.Response[enrollmentv1.Enrollment], error) {
	row, err := h.svc.GetEnrollment(ctx, req.Msg.GetId())
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(enrollmentToProto(row)), nil
}

// ListEnrollments returns live enrollments, optionally filtered.
func (h *Handler) ListEnrollments(
	ctx context.Context,
	req *connect.Request[enrollmentv1.ListEnrollmentsRequest],
) (*connect.Response[enrollmentv1.ListEnrollmentsResponse], error) {
	f := ListEnrollmentsFilter{}

	if s := req.Msg.GetStudentId(); s != "" {
		id, err := connectutil.ParseUUID(s)
		if err != nil {
			return nil, err
		}
		f.StudentID = &id
	}
	if s := req.Msg.GetProgramId(); s != "" {
		id, err := connectutil.ParseUUID(s)
		if err != nil {
			return nil, err
		}
		f.ProgramID = &id
	}
	if y := req.Msg.GetYear(); y != 0 {
		yr := y
		f.Year = &yr
	}
	if s := req.Msg.GetStatus(); s != "" {
		f.Status = &s
	}

	rows, err := h.svc.ListEnrollments(ctx, f)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*enrollmentv1.Enrollment, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, enrollmentToProto(r))
	}
	return connect.NewResponse(&enrollmentv1.ListEnrollmentsResponse{Enrollments: protos}), nil
}

// --- View-own RPCs ---

// ListOwnEnrollments returns enrollments for the authenticated student.
// The student identity is injected from context; the request carries no student_id field.
func (h *Handler) ListOwnEnrollments(
	ctx context.Context,
	_ *connect.Request[enrollmentv1.ListOwnEnrollmentsRequest],
) (*connect.Response[enrollmentv1.ListEnrollmentsResponse], error) {
	rows, err := h.svc.ListOwnEnrollments(ctx)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*enrollmentv1.Enrollment, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, enrollmentToProto(r))
	}
	return connect.NewResponse(&enrollmentv1.ListEnrollmentsResponse{Enrollments: protos}), nil
}

// GetOwnEnrollment retrieves an enrollment by id only if the caller is the enrollment's student.
// Ownership mismatch and genuine absence both return CodeNotFound — existence is never disclosed.
func (h *Handler) GetOwnEnrollment(
	ctx context.Context,
	req *connect.Request[enrollmentv1.GetOwnEnrollmentRequest],
) (*connect.Response[enrollmentv1.Enrollment], error) {
	row, err := h.svc.GetOwnEnrollment(ctx, req.Msg.GetId())
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(enrollmentToProto(row)), nil
}

// --- Proto converter ---

// enrollmentToProto converts a database enrollment row to its proto representation.
// Optional fields (paid_at, deleted_at, created_by, updated_by) are omitted when null.
func enrollmentToProto(r enrollmentdb.Enrollment) *enrollmentv1.Enrollment {
	e := &enrollmentv1.Enrollment{
		Id:        uuidToString(r.ID),
		StudentId: uuidToString(r.StudentID),
		ProgramId: uuidToString(r.ProgramID),
		Year:      r.Year,
		Status:    r.Status,
		CreatedAt: r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: r.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.PaidAt.Valid {
		s := r.PaidAt.Time.Format("2006-01-02T15:04:05Z07:00")
		e.PaidAt = &s
	}
	if r.DeletedAt.Valid {
		s := r.DeletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		e.DeletedAt = &s
	}
	if r.CreatedBy.Valid {
		s := uuidToString(r.CreatedBy)
		e.CreatedBy = &s
	}
	if r.UpdatedBy.Valid {
		s := uuidToString(r.UpdatedBy)
		e.UpdatedBy = &s
	}
	return e
}

// uuidToString converts a pgtype.UUID to a standard hyphenated string.
func uuidToString(id pgtype.UUID) string {
	return uuid.UUID(id.Bytes).String()
}
