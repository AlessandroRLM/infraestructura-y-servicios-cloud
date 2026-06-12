package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth/session"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/config"
)

// LoginResult holds the session ID minted on successful login.
type LoginResult struct {
	SessionID string
}

// RequestPasswordResetResult carries the optional dev-only reset token.
type RequestPasswordResetResult struct {
	// DevToken is only set when cfg.AppEnv != "production".
	DevToken string
}

// Service orchestrates authentication business logic.
type Service struct {
	repo      Repository
	store     session.Store
	loader    RoleLoader
	cfg       config.Config
	dummyHash []byte // precomputed at configured cost for constant-time user-not-found guard
}

// NewService constructs an AuthService with the provided dependencies.
// It precomputes a bcrypt dummy hash at the configured cost so that the
// user-not-found path in Login does the same amount of work as a real
// password comparison, preventing timing-based user enumeration.
func NewService(repo Repository, store session.Store, loader RoleLoader, cfg config.Config) *Service {
	dummy, err := bcrypt.GenerateFromPassword([]byte("invalid-password-placeholder"), cfg.BcryptCost)
	if err != nil {
		// cfg.BcryptCost is validated by config.Load; this should never happen.
		panic(fmt.Sprintf("auth: NewService: failed to precompute dummy hash: %v", err))
	}
	return &Service{
		repo:      repo,
		store:     store,
		loader:    loader,
		cfg:       cfg,
		dummyHash: dummy,
	}
}

// Login validates credentials, mints a session, and stores it in Redis.
// Returns ErrInvalidCredentials when credentials are wrong or the user is deleted.
func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	if email == "" || password == "" {
		return LoginResult{}, ErrInvalidInput
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Constant-time guard: compare against a precomputed dummy hash at the
			// configured bcrypt cost so timing matches a real password comparison,
			// preventing user enumeration via response latency.
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("auth: login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	sid, err := NewSessionID()
	if err != nil {
		return LoginResult{}, fmt.Errorf("auth: login: mint session id: %w", err)
	}

	sess := session.Session{
		UserID:   user.ID,
		IssuedAt: time.Now().UTC(),
	}
	if err := s.store.Create(ctx, sid, sess, s.cfg.SessionTTL); err != nil {
		return LoginResult{}, fmt.Errorf("auth: login: store session: %w", err)
	}

	return LoginResult{SessionID: sid}, nil
}

// Logout deletes the session from Redis.
func (s *Service) Logout(ctx context.Context, sid string) error {
	if err := s.store.Delete(ctx, sid); err != nil {
		return fmt.Errorf("auth: logout: %w", err)
	}
	return nil
}

// RequestPasswordReset mints a reset token for the user.
// If the email is unknown, it silently succeeds (no user enumeration).
// When cfg.AppEnv != "production", the token is returned in the result.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (RequestPasswordResetResult, error) {
	if email == "" {
		return RequestPasswordResetResult{}, ErrInvalidInput
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Silent success — do not reveal whether the email exists.
			return RequestPasswordResetResult{}, nil
		}
		return RequestPasswordResetResult{}, fmt.Errorf("auth: request-reset: %w", err)
	}

	token, err := NewSessionID()
	if err != nil {
		return RequestPasswordResetResult{}, fmt.Errorf("auth: request-reset: mint token: %w", err)
	}

	if err := s.store.SetReset(ctx, token, user.ID, s.cfg.ResetTokenTTL); err != nil {
		return RequestPasswordResetResult{}, fmt.Errorf("auth: request-reset: store token: %w", err)
	}

	var result RequestPasswordResetResult
	if s.cfg.AppEnv != "production" {
		result.DevToken = token
	}
	return result, nil
}

// ConfirmPasswordReset consumes the reset token and updates the user's password.
func (s *Service) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	if token == "" || newPassword == "" {
		return ErrInvalidInput
	}

	userID, err := s.store.GetDelReset(ctx, token)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return ErrInvalidToken
		}
		return fmt.Errorf("auth: confirm-reset: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("auth: confirm-reset: hash: %w", err)
	}

	if err := s.repo.UpdatePasswordHash(ctx, userID, string(hash)); err != nil {
		return fmt.Errorf("auth: confirm-reset: update: %w", err)
	}
	return nil
}

// SessionResult holds the resolved identity and authority for an authenticated user.
type SessionResult struct {
	UserID      string
	Email       string
	Roles       []string
	Permissions []string
}

// GetSession resolves the session data for the authenticated caller.
// It reads user_id and permissions from ctx (set by the session interceptor),
// fetches email from the database, and fetches role names via the loader.
// Returns ErrUserNotFound when the account was deleted after the session was issued.
func (s *Service) GetSession(ctx context.Context) (SessionResult, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return SessionResult{}, fmt.Errorf("auth: GetSession: user ID missing from context")
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return SessionResult{}, ErrUserNotFound
		}
		return SessionResult{}, fmt.Errorf("auth: GetSession: %w", err)
	}

	roles, err := s.loader.LoadRoles(ctx, userID)
	if err != nil {
		return SessionResult{}, fmt.Errorf("auth: GetSession: %w", err)
	}

	var permStrings []string
	if perms, ok := authz.PermissionsFromContext(ctx); ok {
		permStrings = make([]string, 0, len(perms))
		for p := range perms {
			permStrings = append(permStrings, string(p))
		}
	} else {
		permStrings = make([]string, 0)
	}
	slices.Sort(permStrings)
	slices.Sort(roles)

	return SessionResult{
		UserID:      userID.String(),
		Email:       user.Email,
		Roles:       roles,
		Permissions: permStrings,
	}, nil
}

// ErrInvalidCredentials is returned when email/password do not match or the user is deleted.
var ErrInvalidCredentials = fmt.Errorf("auth: invalid credentials")

// ErrInvalidInput is returned when required fields are blank.
var ErrInvalidInput = fmt.Errorf("auth: invalid input")

// ErrInvalidToken is returned when a password-reset token is absent, expired, or already consumed.
// It is intentionally distinct from ErrInvalidCredentials so it maps to CodeInvalidArgument
// rather than CodeUnauthenticated.
var ErrInvalidToken = fmt.Errorf("auth: invalid or expired token")
