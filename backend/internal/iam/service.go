package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/iam/iamdb"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/pagination"
)

const (
	// pageSizeMin is the minimum effective page size after clamping.
	pageSizeMin = 20
	// pageSizeMax is the maximum effective page size after clamping.
	pageSizeMax = 200
)

// iamClamp is the shared page-size clamp for IAM list operations.
var iamClamp = pagination.Clamp{Min: pageSizeMin, Max: pageSizeMax}

// UserSummary is the domain representation of a user with roles and status.
type UserSummary struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Roles       []string
	Disabled    bool // true when users.disabled_at is non-NULL
}

// ListUsersResult holds the paginated result for ListUsers.
type ListUsersResult struct {
	Users         []UserSummary
	NextPageToken string
}

// validRoles is the set of role names accepted by AssignRole and RevokeRole.
var validRoles = map[string]struct{}{
	"admin":   {},
	"teacher": {},
	"student": {},
}

// Service implements the iam domain use cases: ListUsers, GetUser, AssignRole, RevokeRole.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the provided Repository dependency.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListUsers validates the request, clamps page_size, assembles repo params,
// detects the next page, and converts rows to domain UserSummary values.
func (s *Service) ListUsers(ctx context.Context, pageSize int32, pageToken string, query string) (ListUsersResult, error) {
	clamped := iamClamp.Apply(pageSize)

	// Parse optional page_token UUID.
	var tokenUUID *uuid.UUID
	if pageToken != "" {
		id, err := uuid.Parse(pageToken)
		if err != nil {
			return ListUsersResult{}, fmt.Errorf("%w: page_token is not a valid UUID: %q", ErrInvalidInput, pageToken)
		}
		tokenUUID = &id
	}

	// Build optional query parameter.
	var queryPtr *string
	if query != "" {
		queryPtr = &query
	}

	rows, err := s.repo.ListUsers(ctx, ListUsersParams{
		Query:     queryPtr,
		PageToken: tokenUUID,
		RowLimit:  int32(clamped + 1), // +1 to detect next page
	})
	if err != nil {
		return ListUsersResult{}, err
	}

	page := pagination.Paginate(rows, clamped)

	users := make([]UserSummary, 0, len(page.Items))
	for _, row := range page.Items {
		users = append(users, rowToUserSummary(row))
	}

	// Derive next_page_token: UUID of the last item when more pages exist.
	nextToken := pagination.TokenOf(page, func(r iamdb.ListUsersRow) uuid.UUID {
		return uuid.UUID(r.ID.Bytes)
	})

	return ListUsersResult{
		Users:         users,
		NextPageToken: nextToken,
	}, nil
}

// GetUser fetches the full UserSummary for a single user by UUID.
// Roles are loaded via a separate query (single round-trip per user).
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (UserSummary, error) {
	row, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return UserSummary{}, err
	}

	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return UserSummary{}, err
	}

	summary := UserSummary{
		ID:       uuid.UUID(row.ID.Bytes),
		Email:    row.Email,
		Disabled: row.DisabledAt.Valid,
		Roles:    roles,
	}
	summary.DisplayName = deriveDisplayName(row.GivenNames.String, row.LastNamePaternal.String, row.GivenNames.Valid && row.LastNamePaternal.Valid, row.Email)
	return summary, nil
}

// AssignRole assigns a role to a user. The operation is idempotent: if the user
// already has the role, it succeeds without creating a duplicate row. An audit
// entry is written on every call regardless of whether a new row was inserted
// (EC-05: audit documents intent, not state change).
//
// Validation order: (1) role name valid, (2) target user exists, (3) execute.
func (s *Service) AssignRole(ctx context.Context, targetUserID uuid.UUID, roleName string) (UserSummary, error) {
	if _, ok := validRoles[roleName]; !ok {
		return UserSummary{}, fmt.Errorf("%w: role %q is not valid (must be admin, teacher, or student)", ErrInvalidInput, roleName)
	}

	// Verify target user exists before inserting.
	// Spec A-3: non-existent user_id → CodeInvalidArgument (ErrInvalidInput).
	if _, err := s.repo.GetUserByID(ctx, targetUserID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return UserSummary{}, fmt.Errorf("%w: user %s does not exist", ErrInvalidInput, targetUserID)
		}
		return UserSummary{}, err
	}

	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return UserSummary{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	// Insert (idempotent). n == 0 means role already existed.
	if _, err := s.repo.AssignRole(ctx, AssignRoleParams{
		UserID:   targetUserID,
		RoleName: roleName,
		Actor:    callerID,
	}); err != nil {
		return UserSummary{}, err
	}

	// Write audit entry on EVERY call, including no-op re-assign (EC-05).
	detail, err := json.Marshal(map[string]string{"role": roleName})
	if err != nil {
		return UserSummary{}, fmt.Errorf("iam: AssignRole marshal detail: %w", err)
	}
	if err := s.repo.InsertAuditLog(ctx, AuditLogParams{
		ActorID:  callerID,
		Action:   "role.assign",
		Entity:   "users",
		EntityID: targetUserID,
		Detail:   detail,
	}); err != nil {
		return UserSummary{}, err
	}

	return s.GetUser(ctx, targetUserID)
}

// RevokeRole removes a role from a user with privilege-escalation guards.
//
// Guard order (EC-06): self-demotion check FIRST, then last-admin check, then DELETE.
//   - Self-demotion: if role == "admin" AND caller == target → ErrSelfDemotion.
//   - Last-admin: if role == "admin" AND CountAdmins() <= 1 → ErrLastAdmin.
//
// The hard DELETE and audit entry are performed atomically via RevokeRoleTx.
func (s *Service) RevokeRole(ctx context.Context, targetUserID uuid.UUID, roleName string) (UserSummary, error) {
	if _, ok := validRoles[roleName]; !ok {
		return UserSummary{}, fmt.Errorf("%w: role %q is not valid (must be admin, teacher, or student)", ErrInvalidInput, roleName)
	}

	callerID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return UserSummary{}, fmt.Errorf("%w: no authenticated user in context", ErrNotFound)
	}

	// Verify target user exists.
	if _, err := s.repo.GetUserByID(ctx, targetUserID); err != nil {
		return UserSummary{}, err
	}

	if roleName == "admin" {
		// Self-demotion guard (EC-06: checked first).
		if targetUserID == callerID {
			return UserSummary{}, fmt.Errorf("%w", ErrSelfDemotion)
		}

		// Last-admin guard.
		n, err := s.repo.CountAdmins(ctx)
		if err != nil {
			return UserSummary{}, err
		}
		if n <= 1 {
			return UserSummary{}, fmt.Errorf("%w", ErrLastAdmin)
		}
	}

	// Atomic DELETE + audit in one transaction.
	if err := s.repo.RevokeRoleTx(ctx, RevokeRoleParams{
		UserID:   targetUserID,
		RoleName: roleName,
		Actor:    callerID,
	}); err != nil {
		return UserSummary{}, err
	}

	return s.GetUser(ctx, targetUserID)
}

// rowToUserSummary converts a ListUsersRow to a UserSummary, handling the
// array_agg roles field and deriving display_name.
//
// The array_agg subquery returns interface{} from sqlc; at runtime pgx/v5
// delivers it as []interface{} where each element is a string. We perform a
// defensive type assertion and fall back to an empty slice on unexpected types.
func rowToUserSummary(row iamdb.ListUsersRow) UserSummary {
	roles := extractRoles(row.Roles)
	displayName := deriveDisplayName(
		row.GivenNames.String,
		row.LastNamePaternal.String,
		row.GivenNames.Valid && row.LastNamePaternal.Valid,
		row.Email,
	)
	return UserSummary{
		ID:          uuid.UUID(row.ID.Bytes),
		Email:       row.Email,
		DisplayName: displayName,
		Roles:       roles,
		Disabled:    row.DisabledAt.Valid,
	}
}

// extractRoles converts the array_agg interface{} value to a []string.
// pgx/v5 delivers a Postgres text[] as []interface{} where each element is a string.
// A NULL result from array_agg (user has no roles) comes as nil — return empty slice.
func extractRoles(raw interface{}) []string {
	if raw == nil {
		return []string{}
	}
	switch v := raw.(type) {
	case []interface{}:
		roles := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				roles = append(roles, s)
			}
		}
		return roles
	case []string:
		return v
	default:
		return []string{}
	}
}

// deriveDisplayName computes the user's display name.
// When the profile row is present and both name fields are non-empty, returns
// "given_names last_name_paternal". Otherwise falls back to the user's email.
func deriveDisplayName(givenNames, lastNamePaternal string, profilePresent bool, email string) string {
	if profilePresent {
		name := strings.TrimSpace(givenNames + " " + lastNamePaternal)
		if name != "" {
			return name
		}
	}
	return email
}
