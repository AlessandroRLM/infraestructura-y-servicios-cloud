package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

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

// Service implements the iam domain use cases: ListUsers and GetUser.
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
