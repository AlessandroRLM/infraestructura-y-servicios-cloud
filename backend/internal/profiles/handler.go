package profiles

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1/profilesv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles/profilesdb"
)

// Handler implements profilesv1connect.ProfileServiceHandler.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Connect handler wrapping the ProfileService.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the ProfileService Connect handler on mux using the provided options.
func Register(mux *http.ServeMux, h *Handler, opts ...connect.HandlerOption) {
	path, handler := profilesv1connect.NewProfileServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// UpsertUserProfile upserts a user's personal profile.
func (h *Handler) UpsertUserProfile(
	ctx context.Context,
	req *connect.Request[profilesv1.UpsertUserProfileRequest],
) (*connect.Response[profilesv1.UserProfile], error) {
	userID, err := uuid.Parse(req.Msg.GetUserId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}

	params := UpsertUserProfileParams{
		UserID:                userID,
		GivenNames:            req.Msg.GetGivenNames(),
		LastNamePaternal:      req.Msg.GetLastNamePaternal(),
		LastNameMaternal:      req.Msg.LastNameMaternal,
		NationalIDType:        req.Msg.GetNationalIdType(),
		NationalID:            req.Msg.GetNationalId(),
		BirthDate:             req.Msg.BirthDate,
		Phone:                 req.Msg.Phone,
		PersonalEmail:         req.Msg.PersonalEmail,
		AddressStreet:         req.Msg.AddressStreet,
		Commune:               req.Msg.Commune,
		Region:                req.Msg.Region,
		Country:               req.Msg.Country,
		PostalCode:            req.Msg.PostalCode,
		Sex:                   req.Msg.Sex,
		Nationality:           req.Msg.Nationality,
		PhotoURL:              req.Msg.PhotoUrl,
		EmergencyContactName:  req.Msg.EmergencyContactName,
		EmergencyContactPhone: req.Msg.EmergencyContactPhone,
	}

	row, err := h.svc.UpsertUserProfile(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(userProfileToProto(row)), nil
}

// GetUserProfile retrieves a user profile by user_id.
func (h *Handler) GetUserProfile(
	ctx context.Context,
	req *connect.Request[profilesv1.GetUserProfileRequest],
) (*connect.Response[profilesv1.UserProfile], error) {
	userID, err := uuid.Parse(req.Msg.GetUserId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}

	row, err := h.svc.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(userProfileToProto(row)), nil
}

// GetOwnProfile returns the authenticated caller's own user profile.
// No user_id is accepted from the request — self-scope is enforced via context.
func (h *Handler) GetOwnProfile(
	ctx context.Context,
	_ *connect.Request[profilesv1.GetOwnProfileRequest],
) (*connect.Response[profilesv1.UserProfile], error) {
	row, err := h.svc.GetOwnProfile(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(userProfileToProto(row)), nil
}

// UpsertStudentProfile upserts a student's academic profile.
func (h *Handler) UpsertStudentProfile(
	ctx context.Context,
	req *connect.Request[profilesv1.UpsertStudentProfileRequest],
) (*connect.Response[profilesv1.StudentProfile], error) {
	userID, err := uuid.Parse(req.Msg.GetUserId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}

	params := UpsertStudentProfileParams{
		UserID:        userID,
		AdmissionYear: req.Msg.GetAdmissionYear(),
	}

	row, err := h.svc.UpsertStudentProfile(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(studentProfileToProto(row)), nil
}

// GetStudentProfile retrieves a student profile by user_id.
func (h *Handler) GetStudentProfile(
	ctx context.Context,
	req *connect.Request[profilesv1.GetStudentProfileRequest],
) (*connect.Response[profilesv1.StudentProfile], error) {
	userID, err := uuid.Parse(req.Msg.GetUserId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}

	row, err := h.svc.GetStudentProfile(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(studentProfileToProto(row)), nil
}

// UpsertTeacherProfile upserts a teacher's departmental profile.
func (h *Handler) UpsertTeacherProfile(
	ctx context.Context,
	req *connect.Request[profilesv1.UpsertTeacherProfileRequest],
) (*connect.Response[profilesv1.TeacherProfile], error) {
	userID, err := uuid.Parse(req.Msg.GetUserId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}

	params := UpsertTeacherProfileParams{
		UserID:     userID,
		Department: req.Msg.Department,
		Title:      req.Msg.Title,
	}

	row, err := h.svc.UpsertTeacherProfile(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(teacherProfileToProto(row)), nil
}

// GetTeacherProfile retrieves a teacher profile by user_id.
func (h *Handler) GetTeacherProfile(
	ctx context.Context,
	req *connect.Request[profilesv1.GetTeacherProfileRequest],
) (*connect.Response[profilesv1.TeacherProfile], error) {
	userID, err := uuid.Parse(req.Msg.GetUserId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}

	row, err := h.svc.GetTeacherProfile(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(teacherProfileToProto(row)), nil
}

// AddTeacherQualification adds a qualification record to a teacher's profile.
func (h *Handler) AddTeacherQualification(
	ctx context.Context,
	req *connect.Request[profilesv1.AddTeacherQualificationRequest],
) (*connect.Response[profilesv1.TeacherQualification], error) {
	teacherID, err := uuid.Parse(req.Msg.GetTeacherId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid teacher_id"))
	}

	params := AddTeacherQualificationParams{
		TeacherID: teacherID,
		Degree:    req.Msg.GetDegree(),
		Year:      req.Msg.GetYear(),
	}

	row, err := h.svc.AddTeacherQualification(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(teacherQualificationToProto(row)), nil
}

// ListTeacherQualifications returns all non-deleted qualifications for a teacher.
func (h *Handler) ListTeacherQualifications(
	ctx context.Context,
	req *connect.Request[profilesv1.ListTeacherQualificationsRequest],
) (*connect.Response[profilesv1.ListTeacherQualificationsResponse], error) {
	teacherID, err := uuid.Parse(req.Msg.GetTeacherId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid teacher_id"))
	}

	rows, err := h.svc.ListTeacherQualifications(ctx, teacherID)
	if err != nil {
		return nil, mapError(err)
	}

	quals := make([]*profilesv1.TeacherQualification, 0, len(rows))
	for _, r := range rows {
		quals = append(quals, teacherQualificationToProto(r))
	}

	return connect.NewResponse(&profilesv1.ListTeacherQualificationsResponse{
		Qualifications: quals,
	}), nil
}

// mapError converts domain errors to connect.Error codes.
func mapError(err error) error {
	if errors.Is(err, ErrInvalidInput) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	// Do not forward the raw error chain to the client; internal details must not leak.
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

// userProfileToProto converts a profilesdb.UserProfile to the proto message type.
func userProfileToProto(r profilesdb.UserProfile) *profilesv1.UserProfile {
	p := &profilesv1.UserProfile{
		UserId:           uuidBytesToString(r.UserID.Bytes),
		GivenNames:       r.GivenNames,
		LastNamePaternal: r.LastNamePaternal,
		NationalIdType:   r.NationalIDType,
		NationalId:       r.NationalID,
	}
	if r.LastNameMaternal.Valid {
		p.LastNameMaternal = &r.LastNameMaternal.String
	}
	if r.BirthDate.Valid {
		s := r.BirthDate.Time.Format("2006-01-02")
		p.BirthDate = &s
	}
	if r.Phone.Valid {
		p.Phone = &r.Phone.String
	}
	if r.PersonalEmail.Valid {
		p.PersonalEmail = &r.PersonalEmail.String
	}
	if r.AddressStreet.Valid {
		p.AddressStreet = &r.AddressStreet.String
	}
	if r.Commune.Valid {
		p.Commune = &r.Commune.String
	}
	if r.Region.Valid {
		p.Region = &r.Region.String
	}
	if r.Country.Valid {
		p.Country = &r.Country.String
	}
	if r.PostalCode.Valid {
		p.PostalCode = &r.PostalCode.String
	}
	if r.Sex.Valid {
		p.Sex = &r.Sex.String
	}
	if r.Nationality.Valid {
		p.Nationality = &r.Nationality.String
	}
	if r.PhotoUrl.Valid {
		p.PhotoUrl = &r.PhotoUrl.String
	}
	if r.EmergencyContactName.Valid {
		p.EmergencyContactName = &r.EmergencyContactName.String
	}
	if r.EmergencyContactPhone.Valid {
		p.EmergencyContactPhone = &r.EmergencyContactPhone.String
	}
	return p
}

// studentProfileToProto converts a profilesdb.StudentProfile to the proto message type.
func studentProfileToProto(r profilesdb.StudentProfile) *profilesv1.StudentProfile {
	return &profilesv1.StudentProfile{
		UserId:        uuidBytesToString(r.UserID.Bytes),
		AdmissionYear: r.AdmissionYear,
	}
}

// teacherProfileToProto converts a profilesdb.TeacherProfile to the proto message type.
func teacherProfileToProto(r profilesdb.TeacherProfile) *profilesv1.TeacherProfile {
	p := &profilesv1.TeacherProfile{
		UserId: uuidBytesToString(r.UserID.Bytes),
	}
	if r.Department.Valid {
		p.Department = &r.Department.String
	}
	if r.Title.Valid {
		p.Title = &r.Title.String
	}
	return p
}

// teacherQualificationToProto converts a profilesdb.TeacherQualification to the proto message type.
func teacherQualificationToProto(r profilesdb.TeacherQualification) *profilesv1.TeacherQualification {
	return &profilesv1.TeacherQualification{
		Id:        uuidBytesToString(r.ID.Bytes),
		TeacherId: uuidBytesToString(r.TeacherID.Bytes),
		Degree:    r.Degree,
		Year:      r.Year,
	}
}

// uuidBytesToString formats a [16]byte UUID as a standard hyphenated string.
func uuidBytesToString(b [16]byte) string {
	id := uuid.UUID(b)
	return id.String()
}
