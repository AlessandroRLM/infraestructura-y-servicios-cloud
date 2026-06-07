// Package session manages session and password-reset tokens in Redis.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Session is the value stored in Redis for each active session.
type Session struct {
	UserID   uuid.UUID `json:"user_id"`
	IssuedAt time.Time `json:"issued_at"`
}

// Store manages opaque session tokens in Redis.
type Store interface {
	// Create persists a new session with the given TTL.
	Create(ctx context.Context, sid string, s Session, ttl time.Duration) error
	// Touch atomically retrieves the session and renews its TTL (GETEX).
	// Returns a zero-value Session and ErrNotFound when the key is absent.
	Touch(ctx context.Context, sid string, ttl time.Duration) (Session, error)
	// Delete removes the session key.
	Delete(ctx context.Context, sid string) error

	// SetReset stores a single-use password-reset token mapped to a user ID.
	SetReset(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error
	// GetDelReset atomically retrieves and deletes the reset token (GETDEL).
	// Returns ErrNotFound when the token is absent or expired.
	GetDelReset(ctx context.Context, token string) (uuid.UUID, error)
}

// ErrNotFound is returned by Touch and GetDelReset when the key does not exist.
var ErrNotFound = fmt.Errorf("session: not found")

type redisStore struct {
	client *redis.Client
}

// NewRedisStore constructs a Store backed by the provided go-redis client.
func NewRedisStore(client *redis.Client) Store {
	return &redisStore{client: client}
}

func sessionKey(sid string) string { return "session:" + sid }
func resetKey(token string) string { return "reset:" + token }

// Create serialises the Session to JSON and stores it with the given TTL.
func (s *redisStore) Create(ctx context.Context, sid string, sess Session, ttl time.Duration) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}
	return s.client.Set(ctx, sessionKey(sid), data, ttl).Err()
}

// Touch atomically gets the session value and resets its TTL (GETEX EX <ttl>).
// This is a single round-trip sliding-window renewal.
func (s *redisStore) Touch(ctx context.Context, sid string, ttl time.Duration) (Session, error) {
	data, err := s.client.GetEx(ctx, sessionKey(sid), ttl).Bytes()
	if err != nil {
		if err == redis.Nil {
			return Session{}, ErrNotFound
		}
		return Session{}, fmt.Errorf("session: GETEX: %w", err)
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return Session{}, fmt.Errorf("session: unmarshal: %w", err)
	}
	return sess, nil
}

// Delete removes the session key.
func (s *redisStore) Delete(ctx context.Context, sid string) error {
	return s.client.Del(ctx, sessionKey(sid)).Err()
}

// SetReset stores a password-reset token pointing to the user's UUID (plain string).
func (s *redisStore) SetReset(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	return s.client.Set(ctx, resetKey(token), userID.String(), ttl).Err()
}

// GetDelReset atomically reads and deletes the reset token (GETDEL).
// Returns ErrNotFound when the key is absent.
func (s *redisStore) GetDelReset(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := s.client.GetDel(ctx, resetKey(token)).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.UUID{}, ErrNotFound
		}
		return uuid.UUID{}, fmt.Errorf("session: GETDEL: %w", err)
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("session: parse uuid: %w", err)
	}
	return id, nil
}
