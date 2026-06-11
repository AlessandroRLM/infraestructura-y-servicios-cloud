package sectionenrollment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1/section_enrollmentv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/connectutil"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/pgconv"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/sectionenrollment/sectionenrollmentdb"
)

// Handler implements section_enrollmentv1connect.SectionEnrollmentServiceHandler.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Connect handler wrapping the SectionEnrollmentService.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the SectionEnrollmentService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := section_enrollmentv1connect.NewSectionEnrollmentServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// MapError converts domain errors to connect.Error codes.
// Exported so that the handler_test package can validate the mapping.
// Unrecognized errors map to CodeInternal with a generic message — the original error
// is never forwarded so that internal details cannot leak to callers.
// Single-handling rule: handle an error once (map or log, not both).
func MapError(err error) error {
	switch {
	// Load-resilience codes — must remain distinct (see error taxonomy in spec).
	case errors.Is(err, ErrAdmissionSaturated):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, ErrLockTimeout):
		return connect.NewError(connect.CodeUnavailable, err)

	// Business precondition failures — all map to FailedPrecondition.
	case errors.Is(err, ErrSectionFull):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrWindowClosed):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrNotPaid):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrCourseNotInProgram):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrEnrollmentYearMismatch):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrInvalidTransition):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrWithdrawnNotRevivable):
		return connect.NewError(connect.CodeFailedPrecondition, err)

	// Standard codes.
	case errors.Is(err, ErrAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Unmapped errors: log the original; emit a generic message.
	// This is the only place that logs for unmapped internal failures.
	slog.Error("section_enrollment: unhandled internal error", "err", err)
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

// --- Student self-service RPCs ---

// EnrollOwnSection creates a section inscription for the authenticated student.
// The request must carry both section_id and program_id; program_id identifies which
// paid enrollment to link, removing ambiguity for students in multiple programs.
func (h *Handler) EnrollOwnSection(
	ctx context.Context,
	req *connect.Request[section_enrollmentv1.EnrollOwnSectionRequest],
) (*connect.Response[section_enrollmentv1.SectionEnrollment], error) {
	if req.Msg.GetProgramId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("program_id is required"))
	}
	if _, err := connectutil.ParseUUID(req.Msg.GetProgramId()); err != nil {
		return nil, err
	}
	row, err := h.svc.EnrollOwnSection(ctx, req.Msg.GetSectionId(), req.Msg.GetProgramId())
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(sectionEnrollmentToProto(row)), nil
}

// ListOwnSectionEnrollments returns all live inscriptions for the authenticated student.
func (h *Handler) ListOwnSectionEnrollments(
	ctx context.Context,
	_ *connect.Request[section_enrollmentv1.ListOwnSectionEnrollmentsRequest],
) (*connect.Response[section_enrollmentv1.ListSectionEnrollmentsResponse], error) {
	rows, err := h.svc.ListOwnSectionEnrollments(ctx)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*section_enrollmentv1.SectionEnrollment, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, sectionEnrollmentToProto(r))
	}
	return connect.NewResponse(&section_enrollmentv1.ListSectionEnrollmentsResponse{
		SectionEnrollments: protos,
	}), nil
}

// GetOwnSectionEnrollment retrieves an inscription by id only if the caller owns it.
// Ownership mismatch and genuine absence both return CodeNotFound — existence is never disclosed.
func (h *Handler) GetOwnSectionEnrollment(
	ctx context.Context,
	req *connect.Request[section_enrollmentv1.GetOwnSectionEnrollmentRequest],
) (*connect.Response[section_enrollmentv1.SectionEnrollment], error) {
	row, err := h.svc.GetOwnSectionEnrollment(ctx, req.Msg.GetId())
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(sectionEnrollmentToProto(row)), nil
}

// --- Admin-managed RPCs ---

// EnrollSection creates or revives a section inscription for any student.
func (h *Handler) EnrollSection(
	ctx context.Context,
	req *connect.Request[section_enrollmentv1.EnrollSectionRequest],
) (*connect.Response[section_enrollmentv1.SectionEnrollment], error) {
	row, err := h.svc.EnrollSection(ctx, req.Msg.GetEnrollmentId(), req.Msg.GetSectionId())
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(sectionEnrollmentToProto(row)), nil
}

// WithdrawSection transitions an in_progress inscription to withdrawn (admin-only).
func (h *Handler) WithdrawSection(
	ctx context.Context,
	req *connect.Request[section_enrollmentv1.WithdrawSectionRequest],
) (*connect.Response[section_enrollmentv1.WithdrawSectionResponse], error) {
	if _, err := h.svc.WithdrawSection(ctx, req.Msg.GetId()); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&section_enrollmentv1.WithdrawSectionResponse{}), nil
}

// GetSectionEnrollment retrieves a live inscription by id.
func (h *Handler) GetSectionEnrollment(
	ctx context.Context,
	req *connect.Request[section_enrollmentv1.GetSectionEnrollmentRequest],
) (*connect.Response[section_enrollmentv1.SectionEnrollment], error) {
	row, err := h.svc.GetSectionEnrollment(ctx, req.Msg.GetId())
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(sectionEnrollmentToProto(row)), nil
}

// ListSectionEnrollments returns live inscriptions, optionally filtered.
func (h *Handler) ListSectionEnrollments(
	ctx context.Context,
	req *connect.Request[section_enrollmentv1.ListSectionEnrollmentsRequest],
) (*connect.Response[section_enrollmentv1.ListSectionEnrollmentsResponse], error) {
	f := ListSectionEnrollmentsFilter{}

	if s := req.Msg.GetSectionId(); s != "" {
		id, err := connectutil.ParseUUID(s)
		if err != nil {
			return nil, err
		}
		f.SectionID = &id
	}
	if s := req.Msg.GetEnrollmentId(); s != "" {
		id, err := connectutil.ParseUUID(s)
		if err != nil {
			return nil, err
		}
		f.EnrollmentID = &id
	}
	if s := req.Msg.GetStatus(); s != "" {
		f.Status = &s
	}

	rows, err := h.svc.ListSectionEnrollments(ctx, f)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*section_enrollmentv1.SectionEnrollment, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, sectionEnrollmentToProto(r))
	}
	return connect.NewResponse(&section_enrollmentv1.ListSectionEnrollmentsResponse{
		SectionEnrollments: protos,
	}), nil
}

// --- Proto converter ---

// sectionEnrollmentToProto converts a database row to its proto representation.
func sectionEnrollmentToProto(r sectionenrollmentdb.SectionEnrollment) *section_enrollmentv1.SectionEnrollment {
	se := &section_enrollmentv1.SectionEnrollment{
		Id:           uuidToString(r.ID),
		EnrollmentId: uuidToString(r.EnrollmentID),
		SectionId:    uuidToString(r.SectionID),
		Status:       r.Status,
		RegisteredAt: r.RegisteredAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:    r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    r.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		FinalGrade:   pgconv.NumericToString(r.FinalGrade),
	}
	if r.DeletedAt.Valid {
		s := r.DeletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		se.DeletedAt = &s
	}
	return se
}

// uuidToString converts a pgtype.UUID to a standard hyphenated string.
func uuidToString(id pgtype.UUID) string {
	return uuid.UUID(id.Bytes).String()
}

