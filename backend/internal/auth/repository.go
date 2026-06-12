package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/authdb"
)

// User is the domain model returned by Repository.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
}

// ErrUserNotFound is returned when no matching user exists.
var ErrUserNotFound = fmt.Errorf("auth: user not found")

// Repository provides access to user data.
type Repository interface {
	GetUserByEmail(ctx context.Context, email string) (User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
}

type postgresRepository struct {
	q authdb.Querier
}

// Compile-time proof that *postgresRepository satisfies the Repository interface.
var _ Repository = (*postgresRepository)(nil)

// NewPostgresRepository constructs a Repository backed by the given sqlc Querier.
func NewPostgresRepository(q authdb.Querier) Repository {
	return &postgresRepository{q: q}
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("auth: GetUserByEmail: %w", err)
	}
	id, err := pgUUIDToUUID(row.ID)
	if err != nil {
		return User{}, err
	}
	return User{
		ID:           id,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
	}, nil
}

func (r *postgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	row, err := r.q.GetUserByID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("auth: GetUserByID: %w", err)
	}
	uid, err := pgUUIDToUUID(row.ID)
	if err != nil {
		return User{}, err
	}
	return User{ID: uid, Email: row.Email}, nil
}

func (r *postgresRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	err := r.q.UpdatePasswordHash(ctx, authdb.UpdatePasswordHashParams{
		ID:           pgID,
		PasswordHash: hash,
	})
	if err != nil {
		return fmt.Errorf("auth: UpdatePasswordHash: %w", err)
	}
	return nil
}

func pgUUIDToUUID(pgID pgtype.UUID) (uuid.UUID, error) {
	if !pgID.Valid {
		return uuid.UUID{}, fmt.Errorf("auth: null UUID from database")
	}
	return pgID.Bytes, nil
}
