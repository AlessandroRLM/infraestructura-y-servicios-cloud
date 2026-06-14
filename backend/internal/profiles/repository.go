// Package profiles implements the profiles vertical slice: personal data,
// student/teacher sub-profiles, and teacher qualifications.
package profiles

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles/profilesdb"
)

// ErrNotFound is returned by repository methods when the requested row does not exist.
var ErrNotFound = fmt.Errorf("profiles: not found")

// ErrInvalidInput is returned by service validation when required fields are missing or invalid.
var ErrInvalidInput = fmt.Errorf("profiles: invalid input")

// Repository provides data access for all profile tables.
type Repository interface {
	UpsertUserProfile(ctx context.Context, p UpsertUserProfileParams) (profilesdb.UserProfile, error)
	GetUserProfile(ctx context.Context, userID uuid.UUID) (profilesdb.UserProfile, error)
	GetOwnProfile(ctx context.Context, callerID uuid.UUID) (profilesdb.UserProfile, error)
	UpsertOwnProfile(ctx context.Context, p UpsertOwnProfileParams) (profilesdb.UserProfile, error)
	UpsertStudentProfile(ctx context.Context, p UpsertStudentProfileParams) (profilesdb.StudentProfile, error)
	GetStudentProfile(ctx context.Context, userID uuid.UUID) (profilesdb.StudentProfile, error)
	UpsertTeacherProfile(ctx context.Context, p UpsertTeacherProfileParams) (profilesdb.TeacherProfile, error)
	GetTeacherProfile(ctx context.Context, userID uuid.UUID) (profilesdb.TeacherProfile, error)
	AddTeacherQualification(ctx context.Context, p AddTeacherQualificationParams) (profilesdb.TeacherQualification, error)
	ListTeacherQualifications(ctx context.Context, teacherID uuid.UUID) ([]profilesdb.TeacherQualification, error)
}

// UpsertOwnProfileParams carries the 11 self-editable fields for PATCH updates.
// Each *string field encodes three states:
//   - nil: field absent — COALESCE preserves the existing column value.
//   - &"": field present-empty — sets column to empty string (clear).
//   - &"value": field present-non-empty — sets column to the given value.
type UpsertOwnProfileParams struct {
	UserID                uuid.UUID
	BirthDate             *string
	Phone                 *string
	PersonalEmail         *string
	AddressStreet         *string
	Commune               *string
	Region                *string
	Country               *string
	PostalCode            *string
	PhotoURL              *string
	EmergencyContactName  *string
	EmergencyContactPhone *string
	UpdatedBy             *uuid.UUID
}

// UpsertUserProfileParams carries all fields for inserting or updating a user_profiles row.
type UpsertUserProfileParams struct {
	UserID                uuid.UUID
	GivenNames            string
	LastNamePaternal      string
	LastNameMaternal      *string
	NationalIDType        string
	NationalID            string
	BirthDate             *string
	Phone                 *string
	PersonalEmail         *string
	AddressStreet         *string
	Commune               *string
	Region                *string
	Country               *string
	PostalCode            *string
	Sex                   *string
	Nationality           *string
	PhotoURL              *string
	EmergencyContactName  *string
	EmergencyContactPhone *string
	CreatedBy             *uuid.UUID
	UpdatedBy             *uuid.UUID
}

// UpsertStudentProfileParams carries fields for inserting or updating a student_profiles row.
type UpsertStudentProfileParams struct {
	UserID        uuid.UUID
	AdmissionYear int32
	CreatedBy     *uuid.UUID
	UpdatedBy     *uuid.UUID
}

// UpsertTeacherProfileParams carries fields for inserting or updating a teacher_profiles row.
type UpsertTeacherProfileParams struct {
	UserID     uuid.UUID
	Department *string
	Title      *string
	CreatedBy  *uuid.UUID
	UpdatedBy  *uuid.UUID
}

// AddTeacherQualificationParams carries fields for a new teacher_qualifications row.
type AddTeacherQualificationParams struct {
	TeacherID uuid.UUID
	Degree    string
	Year      int32
	CreatedBy *uuid.UUID
	UpdatedBy *uuid.UUID
}

type postgresRepository struct {
	q profilesdb.Querier
}

// Compile-time proof that *postgresRepository satisfies the Repository interface.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by the given sqlc Querier.
func NewPostgresRepository(q profilesdb.Querier) Repository {
	return &postgresRepository{q: q}
}

func (r *postgresRepository) UpsertUserProfile(ctx context.Context, p UpsertUserProfileParams) (profilesdb.UserProfile, error) {
	row, err := r.q.UpsertUserProfile(ctx, profilesdb.UpsertUserProfileParams{
		UserID:                pgtype.UUID{Bytes: p.UserID, Valid: true},
		GivenNames:            p.GivenNames,
		LastNamePaternal:      p.LastNamePaternal,
		LastNameMaternal:      optionalText(p.LastNameMaternal),
		NationalIDType:        p.NationalIDType,
		NationalID:            p.NationalID,
		BirthDate:             optionalDate(p.BirthDate),
		Phone:                 optionalText(p.Phone),
		PersonalEmail:         optionalText(p.PersonalEmail),
		AddressStreet:         optionalText(p.AddressStreet),
		Commune:               optionalText(p.Commune),
		Region:                optionalText(p.Region),
		Country:               optionalText(p.Country),
		PostalCode:            optionalText(p.PostalCode),
		Sex:                   optionalText(p.Sex),
		Nationality:           optionalText(p.Nationality),
		PhotoUrl:              optionalText(p.PhotoURL),
		EmergencyContactName:  optionalText(p.EmergencyContactName),
		EmergencyContactPhone: optionalText(p.EmergencyContactPhone),
		CreatedBy:             optionalUUID(p.CreatedBy),
		UpdatedBy:             optionalUUID(p.UpdatedBy),
	})
	if err != nil {
		return profilesdb.UserProfile{}, fmt.Errorf("profiles: UpsertUserProfile: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) GetUserProfile(ctx context.Context, userID uuid.UUID) (profilesdb.UserProfile, error) {
	row, err := r.q.GetUserProfile(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profilesdb.UserProfile{}, ErrNotFound
		}
		return profilesdb.UserProfile{}, fmt.Errorf("profiles: GetUserProfile: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) GetOwnProfile(ctx context.Context, callerID uuid.UUID) (profilesdb.UserProfile, error) {
	row, err := r.q.GetOwnProfile(ctx, pgtype.UUID{Bytes: callerID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profilesdb.UserProfile{}, ErrNotFound
		}
		return profilesdb.UserProfile{}, fmt.Errorf("profiles: GetOwnProfile: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) UpsertOwnProfile(ctx context.Context, p UpsertOwnProfileParams) (profilesdb.UserProfile, error) {
	row, err := r.q.UpsertOwnProfile(ctx, profilesdb.UpsertOwnProfileParams{
		UserID:                pgtype.UUID{Bytes: p.UserID, Valid: true},
		BirthDate:             patchDate(p.BirthDate),
		Phone:                 optionalText(p.Phone),
		PersonalEmail:         optionalText(p.PersonalEmail),
		AddressStreet:         optionalText(p.AddressStreet),
		Commune:               optionalText(p.Commune),
		Region:                optionalText(p.Region),
		Country:               optionalText(p.Country),
		PostalCode:            optionalText(p.PostalCode),
		PhotoUrl:              optionalText(p.PhotoURL),
		EmergencyContactName:  optionalText(p.EmergencyContactName),
		EmergencyContactPhone: optionalText(p.EmergencyContactPhone),
		UpdatedBy:             optionalUUID(p.UpdatedBy),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profilesdb.UserProfile{}, ErrNotFound
		}
		return profilesdb.UserProfile{}, fmt.Errorf("profiles: UpsertOwnProfile: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) UpsertStudentProfile(ctx context.Context, p UpsertStudentProfileParams) (profilesdb.StudentProfile, error) {
	row, err := r.q.UpsertStudentProfile(ctx, profilesdb.UpsertStudentProfileParams{
		UserID:        pgtype.UUID{Bytes: p.UserID, Valid: true},
		AdmissionYear: p.AdmissionYear,
		CreatedBy:     optionalUUID(p.CreatedBy),
		UpdatedBy:     optionalUUID(p.UpdatedBy),
	})
	if err != nil {
		return profilesdb.StudentProfile{}, fmt.Errorf("profiles: UpsertStudentProfile: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) GetStudentProfile(ctx context.Context, userID uuid.UUID) (profilesdb.StudentProfile, error) {
	row, err := r.q.GetStudentProfile(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profilesdb.StudentProfile{}, ErrNotFound
		}
		return profilesdb.StudentProfile{}, fmt.Errorf("profiles: GetStudentProfile: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) UpsertTeacherProfile(ctx context.Context, p UpsertTeacherProfileParams) (profilesdb.TeacherProfile, error) {
	row, err := r.q.UpsertTeacherProfile(ctx, profilesdb.UpsertTeacherProfileParams{
		UserID:     pgtype.UUID{Bytes: p.UserID, Valid: true},
		Department: optionalText(p.Department),
		Title:      optionalText(p.Title),
		CreatedBy:  optionalUUID(p.CreatedBy),
		UpdatedBy:  optionalUUID(p.UpdatedBy),
	})
	if err != nil {
		return profilesdb.TeacherProfile{}, fmt.Errorf("profiles: UpsertTeacherProfile: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) GetTeacherProfile(ctx context.Context, userID uuid.UUID) (profilesdb.TeacherProfile, error) {
	row, err := r.q.GetTeacherProfile(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profilesdb.TeacherProfile{}, ErrNotFound
		}
		return profilesdb.TeacherProfile{}, fmt.Errorf("profiles: GetTeacherProfile: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) AddTeacherQualification(ctx context.Context, p AddTeacherQualificationParams) (profilesdb.TeacherQualification, error) {
	row, err := r.q.AddTeacherQualification(ctx, profilesdb.AddTeacherQualificationParams{
		TeacherID: pgtype.UUID{Bytes: p.TeacherID, Valid: true},
		Degree:    p.Degree,
		Year:      p.Year,
		CreatedBy: optionalUUID(p.CreatedBy),
		UpdatedBy: optionalUUID(p.UpdatedBy),
	})
	if err != nil {
		return profilesdb.TeacherQualification{}, fmt.Errorf("profiles: AddTeacherQualification: %w", err)
	}
	return row, nil
}

func (r *postgresRepository) ListTeacherQualifications(ctx context.Context, teacherID uuid.UUID) ([]profilesdb.TeacherQualification, error) {
	rows, err := r.q.ListTeacherQualifications(ctx, pgtype.UUID{Bytes: teacherID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("profiles: ListTeacherQualifications: %w", err)
	}
	return rows, nil
}

// optionalText converts a *string to pgtype.Text.
func optionalText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// optionalDate converts a *string (ISO 8601 date) to pgtype.Date.
// Returns an invalid (null) pgtype.Date when s is nil.
func optionalDate(s *string) pgtype.Date {
	if s == nil {
		return pgtype.Date{}
	}
	var d pgtype.Date
	if err := d.Scan(*s); err != nil {
		return pgtype.Date{}
	}
	return d
}

// optionalUUID converts a *uuid.UUID to pgtype.UUID.
func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// patchDate converts a *string (ISO 8601 date) to pgtype.Date for PATCH semantics.
// nil → pgtype.Date{} (null → COALESCE preserves existing value).
// &"" → pgtype.Date{} (treated as skip; birth_date cannot be cleared via empty string).
// &"YYYY-MM-DD" → parsed date (sets the column).
func patchDate(s *string) pgtype.Date {
	if s == nil || *s == "" {
		return pgtype.Date{}
	}
	return optionalDate(s)
}
