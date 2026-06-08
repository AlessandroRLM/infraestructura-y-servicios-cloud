package catalog

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1/catalogv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/catalog/catalogdb"
)

// Handler implements catalogv1connect.CatalogServiceHandler.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Connect handler wrapping the CatalogService.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the CatalogService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := catalogv1connect.NewCatalogServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// MapError converts domain errors to connect.Error codes.
// Exported so that the handler_test package can validate the mapping.
// Unrecognized errors map to CodeInternal with a generic message — the raw error
// is never forwarded so that internal details (table names, constraint text) cannot
// leak to callers.
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
	if errors.Is(err, ErrHasDependents) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

// parseUUID parses a string UUID and returns CodeInvalidArgument on failure.
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid UUID"))
	}
	return id, nil
}

// --- Programs ---

func (h *Handler) CreateProgram(
	ctx context.Context,
	req *connect.Request[catalogv1.CreateProgramRequest],
) (*connect.Response[catalogv1.Program], error) {
	row, err := h.svc.CreateProgram(ctx, CreateProgramParams{
		Code: req.Msg.GetCode(),
		Name: req.Msg.GetName(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(programToProto(row)), nil
}

func (h *Handler) UpdateProgram(
	ctx context.Context,
	req *connect.Request[catalogv1.UpdateProgramRequest],
) (*connect.Response[catalogv1.Program], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateProgram(ctx, id, UpdateProgramParams{
		Code: req.Msg.GetCode(),
		Name: req.Msg.GetName(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(programToProto(row)), nil
}

func (h *Handler) GetProgram(
	ctx context.Context,
	req *connect.Request[catalogv1.GetProgramRequest],
) (*connect.Response[catalogv1.Program], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.GetProgram(ctx, id)
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(programToProto(row)), nil
}

func (h *Handler) ListPrograms(
	ctx context.Context,
	_ *connect.Request[catalogv1.ListProgramsRequest],
) (*connect.Response[catalogv1.ListProgramsResponse], error) {
	rows, err := h.svc.ListPrograms(ctx)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*catalogv1.Program, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, programToProto(r))
	}
	return connect.NewResponse(&catalogv1.ListProgramsResponse{Programs: protos}), nil
}

func (h *Handler) DeleteProgram(
	ctx context.Context,
	req *connect.Request[catalogv1.DeleteProgramRequest],
) (*connect.Response[catalogv1.DeleteProgramResponse], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteProgram(ctx, id); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&catalogv1.DeleteProgramResponse{}), nil
}

// --- Courses ---

func (h *Handler) CreateCourse(
	ctx context.Context,
	req *connect.Request[catalogv1.CreateCourseRequest],
) (*connect.Response[catalogv1.Course], error) {
	row, err := h.svc.CreateCourse(ctx, CreateCourseParams{
		Code:    req.Msg.GetCode(),
		Name:    req.Msg.GetName(),
		Credits: req.Msg.GetCredits(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(courseToProto(row)), nil
}

func (h *Handler) UpdateCourse(
	ctx context.Context,
	req *connect.Request[catalogv1.UpdateCourseRequest],
) (*connect.Response[catalogv1.Course], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateCourse(ctx, id, UpdateCourseParams{
		Code:    req.Msg.GetCode(),
		Name:    req.Msg.GetName(),
		Credits: req.Msg.GetCredits(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(courseToProto(row)), nil
}

func (h *Handler) GetCourse(
	ctx context.Context,
	req *connect.Request[catalogv1.GetCourseRequest],
) (*connect.Response[catalogv1.Course], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.GetCourse(ctx, id)
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(courseToProto(row)), nil
}

func (h *Handler) ListCourses(
	ctx context.Context,
	_ *connect.Request[catalogv1.ListCoursesRequest],
) (*connect.Response[catalogv1.ListCoursesResponse], error) {
	rows, err := h.svc.ListCourses(ctx)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*catalogv1.Course, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, courseToProto(r))
	}
	return connect.NewResponse(&catalogv1.ListCoursesResponse{Courses: protos}), nil
}

func (h *Handler) DeleteCourse(
	ctx context.Context,
	req *connect.Request[catalogv1.DeleteCourseRequest],
) (*connect.Response[catalogv1.DeleteCourseResponse], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteCourse(ctx, id); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&catalogv1.DeleteCourseResponse{}), nil
}

// --- Academic periods ---

func (h *Handler) CreateAcademicPeriod(
	ctx context.Context,
	req *connect.Request[catalogv1.CreateAcademicPeriodRequest],
) (*connect.Response[catalogv1.AcademicPeriod], error) {
	row, err := h.svc.CreateAcademicPeriod(ctx, CreateAcademicPeriodServiceParams{
		Year:      req.Msg.GetYear(),
		Term:      req.Msg.GetTerm(),
		StartDate: req.Msg.GetStartDate(),
		EndDate:   req.Msg.GetEndDate(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(academicPeriodToProto(row)), nil
}

func (h *Handler) UpdateAcademicPeriod(
	ctx context.Context,
	req *connect.Request[catalogv1.UpdateAcademicPeriodRequest],
) (*connect.Response[catalogv1.AcademicPeriod], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateAcademicPeriod(ctx, id, UpdateAcademicPeriodServiceParams{
		Year:      req.Msg.GetYear(),
		Term:      req.Msg.GetTerm(),
		StartDate: req.Msg.GetStartDate(),
		EndDate:   req.Msg.GetEndDate(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(academicPeriodToProto(row)), nil
}

func (h *Handler) GetAcademicPeriod(
	ctx context.Context,
	req *connect.Request[catalogv1.GetAcademicPeriodRequest],
) (*connect.Response[catalogv1.AcademicPeriod], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.GetAcademicPeriod(ctx, id)
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(academicPeriodToProto(row)), nil
}

func (h *Handler) ListAcademicPeriods(
	ctx context.Context,
	_ *connect.Request[catalogv1.ListAcademicPeriodsRequest],
) (*connect.Response[catalogv1.ListAcademicPeriodsResponse], error) {
	rows, err := h.svc.ListAcademicPeriods(ctx)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*catalogv1.AcademicPeriod, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, academicPeriodToProto(r))
	}
	return connect.NewResponse(&catalogv1.ListAcademicPeriodsResponse{AcademicPeriods: protos}), nil
}

func (h *Handler) DeleteAcademicPeriod(
	ctx context.Context,
	req *connect.Request[catalogv1.DeleteAcademicPeriodRequest],
) (*connect.Response[catalogv1.DeleteAcademicPeriodResponse], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteAcademicPeriod(ctx, id); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&catalogv1.DeleteAcademicPeriodResponse{}), nil
}

// --- Program quotas ---

func (h *Handler) CreateProgramQuota(
	ctx context.Context,
	req *connect.Request[catalogv1.CreateProgramQuotaRequest],
) (*connect.Response[catalogv1.ProgramQuota], error) {
	row, err := h.svc.CreateProgramQuota(ctx, CreateProgramQuotaServiceParams{
		ProgramID: req.Msg.GetProgramId(),
		Year:      req.Msg.GetYear(),
		Capacity:  req.Msg.GetAdmissionQuota(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(programQuotaToProto(row)), nil
}

func (h *Handler) UpdateProgramQuota(
	ctx context.Context,
	req *connect.Request[catalogv1.UpdateProgramQuotaRequest],
) (*connect.Response[catalogv1.ProgramQuota], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateProgramQuota(ctx, id, UpdateProgramQuotaServiceParams{
		Year:     req.Msg.GetYear(),
		Capacity: req.Msg.GetAdmissionQuota(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(programQuotaToProto(row)), nil
}

func (h *Handler) GetProgramQuota(
	ctx context.Context,
	req *connect.Request[catalogv1.GetProgramQuotaRequest],
) (*connect.Response[catalogv1.ProgramQuota], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.GetProgramQuota(ctx, id)
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(programQuotaToProto(row)), nil
}

func (h *Handler) ListProgramQuotas(
	ctx context.Context,
	req *connect.Request[catalogv1.ListProgramQuotasRequest],
) (*connect.Response[catalogv1.ListProgramQuotasResponse], error) {
	programID, err := parseUUID(req.Msg.GetProgramId())
	if err != nil {
		return nil, err
	}
	rows, err := h.svc.ListProgramQuotas(ctx, programID)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*catalogv1.ProgramQuota, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, programQuotaToProto(r))
	}
	return connect.NewResponse(&catalogv1.ListProgramQuotasResponse{ProgramQuotas: protos}), nil
}

func (h *Handler) DeleteProgramQuota(
	ctx context.Context,
	req *connect.Request[catalogv1.DeleteProgramQuotaRequest],
) (*connect.Response[catalogv1.DeleteProgramQuotaResponse], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteProgramQuota(ctx, id); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&catalogv1.DeleteProgramQuotaResponse{}), nil
}

// --- Program-course M:N ---

func (h *Handler) AddCourseToProgram(
	ctx context.Context,
	req *connect.Request[catalogv1.AddCourseToProgramRequest],
) (*connect.Response[catalogv1.ProgramCourse], error) {
	programID, err := parseUUID(req.Msg.GetProgramId())
	if err != nil {
		return nil, err
	}
	courseID, err := parseUUID(req.Msg.GetCourseId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.AddCourseToProgram(ctx, programID, courseID)
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(programCourseToProto(row)), nil
}

func (h *Handler) RemoveCourseFromProgram(
	ctx context.Context,
	req *connect.Request[catalogv1.RemoveCourseFromProgramRequest],
) (*connect.Response[catalogv1.RemoveCourseFromProgramResponse], error) {
	programID, err := parseUUID(req.Msg.GetProgramId())
	if err != nil {
		return nil, err
	}
	courseID, err := parseUUID(req.Msg.GetCourseId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.RemoveCourseFromProgram(ctx, programID, courseID); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&catalogv1.RemoveCourseFromProgramResponse{}), nil
}

func (h *Handler) ListProgramCourses(
	ctx context.Context,
	req *connect.Request[catalogv1.ListProgramCoursesRequest],
) (*connect.Response[catalogv1.ListProgramCoursesResponse], error) {
	programID, err := parseUUID(req.Msg.GetProgramId())
	if err != nil {
		return nil, err
	}
	rows, err := h.svc.ListProgramCourses(ctx, programID)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*catalogv1.ProgramCourse, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, programCourseToProto(r))
	}
	return connect.NewResponse(&catalogv1.ListProgramCoursesResponse{ProgramCourses: protos}), nil
}

// --- Proto converters ---

func programToProto(r catalogdb.Program) *catalogv1.Program {
	p := &catalogv1.Program{
		Id:        uuidToString(r.ID),
		Code:      r.Code,
		Name:      r.Name,
		CreatedAt: r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: r.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.DeletedAt.Valid {
		s := r.DeletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		p.DeletedAt = &s
	}
	if r.CreatedBy.Valid {
		s := uuidToString(r.CreatedBy)
		p.CreatedBy = &s
	}
	if r.UpdatedBy.Valid {
		s := uuidToString(r.UpdatedBy)
		p.UpdatedBy = &s
	}
	return p
}

func courseToProto(r catalogdb.Course) *catalogv1.Course {
	c := &catalogv1.Course{
		Id:        uuidToString(r.ID),
		Code:      r.Code,
		Name:      r.Name,
		Credits:   r.Credits,
		CreatedAt: r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: r.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.DeletedAt.Valid {
		s := r.DeletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		c.DeletedAt = &s
	}
	if r.CreatedBy.Valid {
		s := uuidToString(r.CreatedBy)
		c.CreatedBy = &s
	}
	if r.UpdatedBy.Valid {
		s := uuidToString(r.UpdatedBy)
		c.UpdatedBy = &s
	}
	return c
}

func academicPeriodToProto(r catalogdb.AcademicPeriod) *catalogv1.AcademicPeriod {
	p := &catalogv1.AcademicPeriod{
		Id:        uuidToString(r.ID),
		Year:      r.Year,
		Term:      r.Term,
		StartDate: FormatDate(r.StartDate),
		EndDate:   FormatDate(r.EndDate),
		CreatedAt: r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: r.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.DeletedAt.Valid {
		s := r.DeletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		p.DeletedAt = &s
	}
	return p
}

func programQuotaToProto(r catalogdb.ProgramQuota) *catalogv1.ProgramQuota {
	p := &catalogv1.ProgramQuota{
		Id:             uuidToString(r.ID),
		ProgramId:      uuidToString(r.ProgramID),
		Year:           r.Year,
		AdmissionQuota: r.Capacity,
		CreatedAt:      r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      r.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.DeletedAt.Valid {
		s := r.DeletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		p.DeletedAt = &s
	}
	if r.CreatedBy.Valid {
		s := uuidToString(r.CreatedBy)
		p.CreatedBy = &s
	}
	if r.UpdatedBy.Valid {
		s := uuidToString(r.UpdatedBy)
		p.UpdatedBy = &s
	}
	return p
}

func programCourseToProto(r catalogdb.ProgramCourse) *catalogv1.ProgramCourse {
	return &catalogv1.ProgramCourse{
		ProgramId: uuidToString(r.ProgramID),
		CourseId:  uuidToString(r.CourseID),
		CreatedAt: r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// --- Sections ---

func (h *Handler) CreateSection(
	ctx context.Context,
	req *connect.Request[catalogv1.CreateSectionRequest],
) (*connect.Response[catalogv1.Section], error) {
	row, err := h.svc.CreateSection(ctx, CreateSectionServiceParams{
		CourseID:         req.Msg.GetCourseId(),
		AcademicPeriodID: req.Msg.GetAcademicPeriodId(),
		SeatCapacity:     req.Msg.GetSeatCapacity(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(sectionToProto(row)), nil
}

func (h *Handler) UpdateSection(
	ctx context.Context,
	req *connect.Request[catalogv1.UpdateSectionRequest],
) (*connect.Response[catalogv1.Section], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.UpdateSection(ctx, id, UpdateSectionServiceParams{
		SeatCapacity: req.Msg.GetSeatCapacity(),
	})
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(sectionToProto(row)), nil
}

func (h *Handler) GetSection(
	ctx context.Context,
	req *connect.Request[catalogv1.GetSectionRequest],
) (*connect.Response[catalogv1.Section], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.GetSection(ctx, id)
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(sectionToProto(row)), nil
}

func (h *Handler) ListSections(
	ctx context.Context,
	_ *connect.Request[catalogv1.ListSectionsRequest],
) (*connect.Response[catalogv1.ListSectionsResponse], error) {
	rows, err := h.svc.ListSections(ctx)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*catalogv1.Section, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, sectionToProto(r))
	}
	return connect.NewResponse(&catalogv1.ListSectionsResponse{Sections: protos}), nil
}

func (h *Handler) DeleteSection(
	ctx context.Context,
	req *connect.Request[catalogv1.DeleteSectionRequest],
) (*connect.Response[catalogv1.DeleteSectionResponse], error) {
	id, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteSection(ctx, id); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&catalogv1.DeleteSectionResponse{}), nil
}

// --- Section-teacher M:N ---

func (h *Handler) AssignTeacherToSection(
	ctx context.Context,
	req *connect.Request[catalogv1.AssignTeacherToSectionRequest],
) (*connect.Response[catalogv1.SectionTeacher], error) {
	sectionID, err := parseUUID(req.Msg.GetSectionId())
	if err != nil {
		return nil, err
	}
	teacherID, err := parseUUID(req.Msg.GetTeacherId())
	if err != nil {
		return nil, err
	}
	row, err := h.svc.AssignTeacherToSection(ctx, sectionID, teacherID)
	if err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(sectionTeacherToProto(row)), nil
}

func (h *Handler) RemoveTeacherFromSection(
	ctx context.Context,
	req *connect.Request[catalogv1.RemoveTeacherFromSectionRequest],
) (*connect.Response[catalogv1.RemoveTeacherFromSectionResponse], error) {
	sectionID, err := parseUUID(req.Msg.GetSectionId())
	if err != nil {
		return nil, err
	}
	teacherID, err := parseUUID(req.Msg.GetTeacherId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.RemoveTeacherFromSection(ctx, sectionID, teacherID); err != nil {
		return nil, MapError(err)
	}
	return connect.NewResponse(&catalogv1.RemoveTeacherFromSectionResponse{}), nil
}

func (h *Handler) ListSectionTeachers(
	ctx context.Context,
	req *connect.Request[catalogv1.ListSectionTeachersRequest],
) (*connect.Response[catalogv1.ListSectionTeachersResponse], error) {
	sectionID, err := parseUUID(req.Msg.GetSectionId())
	if err != nil {
		return nil, err
	}
	rows, err := h.svc.ListSectionTeachers(ctx, sectionID)
	if err != nil {
		return nil, MapError(err)
	}
	protos := make([]*catalogv1.SectionTeacher, 0, len(rows))
	for _, r := range rows {
		protos = append(protos, sectionTeacherToProto(r))
	}
	return connect.NewResponse(&catalogv1.ListSectionTeachersResponse{SectionTeachers: protos}), nil
}

// --- Proto converters (sections) ---

func sectionToProto(r catalogdb.Section) *catalogv1.Section {
	s := &catalogv1.Section{
		Id:               uuidToString(r.ID),
		CourseId:         uuidToString(r.CourseID),
		AcademicPeriodId: uuidToString(r.AcademicPeriodID),
		SeatCapacity:     r.Capacity,
		CreatedAt:        r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        r.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.DeletedAt.Valid {
		ts := r.DeletedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		s.DeletedAt = &ts
	}
	if r.CreatedBy.Valid {
		id := uuidToString(r.CreatedBy)
		s.CreatedBy = &id
	}
	if r.UpdatedBy.Valid {
		id := uuidToString(r.UpdatedBy)
		s.UpdatedBy = &id
	}
	return s
}

func sectionTeacherToProto(r catalogdb.SectionTeacher) *catalogv1.SectionTeacher {
	return &catalogv1.SectionTeacher{
		SectionId: uuidToString(r.SectionID),
		TeacherId: uuidToString(r.TeacherID),
		CreatedAt: r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// uuidToString converts a pgtype.UUID to a standard hyphenated string.
func uuidToString(id pgtype.UUID) string {
	return uuid.UUID(id.Bytes).String()
}
