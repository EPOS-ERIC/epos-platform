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

// GetUserByAuthIdentifier fetches local user by external auth identifier.
func (s *Service) GetUserByAuthIdentifier(ctx context.Context, authIdentifier string) (api.User, error) {
	user, resp, err := s.usergroupClient.UsersAPI.ApiV2UsersByAuthAuthIdentifierGet(ctx, authIdentifier).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return api.User{}, domain.ErrNotFound
		}

		s.log.ErrorContext(ctx, "get user by auth identifier failed", "error", err)
		return api.User{}, fmt.Errorf("get user by auth identifier: %w", err)
	}

	return mapUser(user)
}

func mapUser(user *usergroupclient.User) (api.User, error) {
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
		IsSuperAdmin:   user.GetIsSuperAdmin(),
		Memberships:    nil,
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
