package usergroup

import (
	"context"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func (s *Service) GetUser(ctx context.Context, id string) (api.User, error) {
	user, _, err := s.usergroupClient.DefaultAPI.ApiV2UsersUserIdGet(ctx, id).Execute()
	if err != nil {
		return api.User{}, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		return api.User{}, fmt.Errorf("get user: %w", domain.ErrNotFound)
	}

	status, err := api.NewUserStatusFromValue(user.GetStatus())
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
