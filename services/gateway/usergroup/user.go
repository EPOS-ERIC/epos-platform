package usergroup

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/usergroupclient"
)

// GetUser fetches local user by internal id.
func (s *Service) GetUser(ctx context.Context, id string) (api.User, error) {
	user, _, err := s.usergroupClient.UsersAPI.ApiV2UsersUserIdGet(ctx, id).Execute()
	if err != nil {
		return api.User{}, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		return api.User{}, fmt.Errorf("get user: %w", domain.ErrNotFound)
	}

	status, err := api.NewUserStatusFromValue(string(user.GetStatus()))
	if err != nil {
		return api.User{}, fmt.Errorf("parse user status: %w", err)
	}

	return api.User{
		ID:             user.GetId(),
		AuthIdentifier: user.GetAuthIdentifier(),
		Email:          user.GetEmail(),
		GivenName:      user.GetGivenName(),
		FamilyName:     user.GetFamilyName(),
		Status:         status,
		CreatedAt:      formatTime(user.GetCreatedAtOk()),
		UpdatedAt:      formatTime(user.GetUpdatedAtOk()),
	}, nil
}

// GetUserByAuthIdentifier fetches local user by external auth identifier.
func (s *Service) GetUserByAuthIdentifier(ctx context.Context, authIdentifier string) (api.User, error) {
	user, resp, err := s.usergroupClient.UsersAPI.ApiV2UsersByAuthAuthIdentifierGet(ctx, authIdentifier).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return api.User{}, domain.ErrNotFound
		}

		return api.User{}, fmt.Errorf("get user by auth identifier: %w", err)
	}

	if user == nil {
		return api.User{}, domain.ErrNotFound
	}

	return mapUser(user)
}

// ListUsers fetches all local users.
func (s *Service) ListUsers(ctx context.Context) ([]api.User, error) {
	users, _, err := s.usergroupClient.UsersAPI.ApiV2UsersGet(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("list users from usergroup API: %w", err)
	}

	mapped := make([]api.User, 0, len(users))
	for i := range users {
		user, err := mapUser(&users[i])
		if err != nil {
			return nil, fmt.Errorf("list users: map user %d: %w", i, err)
		}
		mapped = append(mapped, user)
	}

	return mapped, nil
}

func mapUser(user *usergroupclient.User) (api.User, error) {
	status, err := api.NewUserStatusFromValue(string(user.GetStatus()))
	if err != nil {
		return api.User{}, fmt.Errorf("parse user status: %w", err)
	}

	return api.User{
		ID:             user.GetId(),
		AuthIdentifier: user.GetAuthIdentifier(),
		Email:          user.GetEmail(),
		GivenName:      user.GetGivenName(),
		FamilyName:     user.GetFamilyName(),
		Status:         status,
		CreatedAt:      formatTime(user.GetCreatedAtOk()),
		UpdatedAt:      formatTime(user.GetUpdatedAtOk()),
	}, nil
}

func formatTime(value *time.Time, ok bool) string {
	if !ok || value == nil {
		return ""
	}

	return value.Format(time.RFC3339)
}
