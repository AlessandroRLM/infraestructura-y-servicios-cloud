package profiles_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/profiles/profilesdb"
)

// fakeQuerier implements profilesdb.Querier for testing the repository layer.
type fakeQuerier struct {
	getUserProfileErr      error
	getUserProfileResult   profilesdb.UserProfile
	getOwnProfileErr       error
	getOwnProfileResult    profilesdb.UserProfile
	getOwnProfileCalledID  pgtype.UUID
	upsertUserProfileErr   error
	upsertUserProfileArgs  profilesdb.UpsertUserProfileParams
	upsertUserProfileCalls int
}

func (f *fakeQuerier) GetUserProfile(_ context.Context, _ pgtype.UUID) (profilesdb.UserProfile, error) {
	return f.getUserProfileResult, f.getUserProfileErr
}

func (f *fakeQuerier) GetOwnProfile(_ context.Context, userID pgtype.UUID) (profilesdb.UserProfile, error) {
	f.getOwnProfileCalledID = userID
	return f.getOwnProfileResult, f.getOwnProfileErr
}

func (f *fakeQuerier) UpsertUserProfile(_ context.Context, arg profilesdb.UpsertUserProfileParams) (profilesdb.UserProfile, error) {
	f.upsertUserProfileCalls++
	f.upsertUserProfileArgs = arg
	return f.upsertUserProfileResult(), f.upsertUserProfileErr
}

func (f *fakeQuerier) upsertUserProfileResult() profilesdb.UserProfile {
	return profilesdb.UserProfile{
		UserID:           f.upsertUserProfileArgs.UserID,
		GivenNames:       f.upsertUserProfileArgs.GivenNames,
		LastNamePaternal: f.upsertUserProfileArgs.LastNamePaternal,
		NationalIDType:   f.upsertUserProfileArgs.NationalIDType,
		NationalID:       f.upsertUserProfileArgs.NationalID,
		CreatedBy:        f.upsertUserProfileArgs.CreatedBy,
		UpdatedBy:        f.upsertUserProfileArgs.UpdatedBy,
	}
}

func (f *fakeQuerier) GetStudentProfile(_ context.Context, _ pgtype.UUID) (profilesdb.StudentProfile, error) {
	return profilesdb.StudentProfile{}, nil
}

func (f *fakeQuerier) UpsertStudentProfile(_ context.Context, _ profilesdb.UpsertStudentProfileParams) (profilesdb.StudentProfile, error) {
	return profilesdb.StudentProfile{}, nil
}

func (f *fakeQuerier) GetTeacherProfile(_ context.Context, _ pgtype.UUID) (profilesdb.TeacherProfile, error) {
	return profilesdb.TeacherProfile{}, nil
}

func (f *fakeQuerier) UpsertTeacherProfile(_ context.Context, _ profilesdb.UpsertTeacherProfileParams) (profilesdb.TeacherProfile, error) {
	return profilesdb.TeacherProfile{}, nil
}

func (f *fakeQuerier) AddTeacherQualification(_ context.Context, _ profilesdb.AddTeacherQualificationParams) (profilesdb.TeacherQualification, error) {
	return profilesdb.TeacherQualification{}, nil
}

func (f *fakeQuerier) ListTeacherQualifications(_ context.Context, _ pgtype.UUID) ([]profilesdb.TeacherQualification, error) {
	return nil, nil
}

func (f *fakeQuerier) UpsertOwnProfile(_ context.Context, _ profilesdb.UpsertOwnProfileParams) (profilesdb.UserProfile, error) {
	return profilesdb.UserProfile{}, nil
}

// Compile-time check that fakeQuerier implements profilesdb.Querier.
var _ profilesdb.Querier = (*fakeQuerier)(nil)

func TestRepository_GetUserProfile_NotFound(t *testing.T) {
	t.Parallel()

	q := &fakeQuerier{getUserProfileErr: pgx.ErrNoRows}
	repo := profiles.NewPostgresRepository(q)

	_, err := repo.GetUserProfile(context.Background(), uuid.New())
	if !errors.Is(err, profiles.ErrNotFound) {
		t.Errorf("GetUserProfile pgx.ErrNoRows: got %v, want ErrNotFound", err)
	}
}

func TestRepository_GetOwnProfile_UsesCallerID(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	expectedPGID := pgtype.UUID{Bytes: callerID, Valid: true}

	q := &fakeQuerier{getOwnProfileResult: profilesdb.UserProfile{
		UserID: expectedPGID,
	}}
	repo := profiles.NewPostgresRepository(q)

	_, err := repo.GetOwnProfile(context.Background(), callerID)
	if err != nil {
		t.Fatalf("GetOwnProfile: unexpected error: %v", err)
	}

	if q.getOwnProfileCalledID != expectedPGID {
		t.Errorf("GetOwnProfile called with user_id %v, want %v", q.getOwnProfileCalledID, expectedPGID)
	}
}

func TestRepository_UpsertUserProfile_PassesAuditFields(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	actorID := uuid.New()

	q := &fakeQuerier{}
	repo := profiles.NewPostgresRepository(q)

	params := profiles.UpsertUserProfileParams{
		UserID:           targetID,
		GivenNames:       "Test",
		LastNamePaternal: "User",
		NationalIDType:   "RUT",
		NationalID:       "12345678-9",
		CreatedBy:        &actorID,
		UpdatedBy:        &actorID,
	}

	_, err := repo.UpsertUserProfile(context.Background(), params)
	if err != nil {
		t.Fatalf("UpsertUserProfile: unexpected error: %v", err)
	}

	if q.upsertUserProfileCalls != 1 {
		t.Fatalf("UpsertUserProfile called %d times, want 1", q.upsertUserProfileCalls)
	}

	gotCreatedBy := q.upsertUserProfileArgs.CreatedBy
	wantCreatedBy := pgtype.UUID{Bytes: actorID, Valid: true}
	if gotCreatedBy != wantCreatedBy {
		t.Errorf("created_by = %v, want %v", gotCreatedBy, wantCreatedBy)
	}

	gotUpdatedBy := q.upsertUserProfileArgs.UpdatedBy
	wantUpdatedBy := pgtype.UUID{Bytes: actorID, Valid: true}
	if gotUpdatedBy != wantUpdatedBy {
		t.Errorf("updated_by = %v, want %v", gotUpdatedBy, wantUpdatedBy)
	}
}
