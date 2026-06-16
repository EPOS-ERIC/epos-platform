// Package usergroup contains resource detail domain use-cases.
package usergroup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/usergroupclient"
)

type Identity struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

type Service struct {
	log             *slog.Logger
	usergroupClient *usergroupclient.APIClient
}

// NewService builds usergroup service for local identity lookup and creation.
func NewService(usergroupClient *usergroupclient.APIClient) *Service {
	return &Service{
		log:             logging.Get("usergroup-service"),
		usergroupClient: usergroupClient,
	}
}

// EnsureUser returns existing local user or creates one from auth identity.
func (s *Service) EnsureUser(ctx context.Context, identity Identity) (api.User, error) {
	if identity.AuthIdentifier == "" {
		return api.User{}, fmt.Errorf("ensure user: %w", domain.ErrBadRequest)
	}

	user, err := s.GetUserByAuthIdentifier(ctx, identity.AuthIdentifier)
	if err == nil {
		return user, nil
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return api.User{}, err
	}

	createRequest := usergroupclient.NewCreateUserRequest(identity.AuthIdentifier)
	status := usergroupclient.UserStatusActive
	createRequest.SetStatus(status)
	if identity.Email != "" {
		createRequest.SetEmail(identity.Email)
	}
	if identity.GivenName != "" {
		createRequest.SetGivenName(identity.GivenName)
	}
	if identity.FamilyName != "" {
		createRequest.SetFamilyName(identity.FamilyName)
	}

	created, resp, err := s.usergroupClient.UsersAPI.ApiV2UsersPost(ctx).CreateUserRequest(*createRequest).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusConflict {
			return s.GetUserByAuthIdentifier(ctx, identity.AuthIdentifier)
		}

		return api.User{}, fmt.Errorf("create user: %w", err)
	}

	return mapUser(created)
}

// ListUsers fetches all local users.
func (s *Service) ListUsers(ctx context.Context, actor api.User) ([]api.User, error) {
	if !actor.IsSuperAdmin {
		return nil, fmt.Errorf("list users: %w", domain.ErrForbidden)
	}

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

// GetUser fetches local user by internal id.
func (s *Service) GetUser(ctx context.Context, actor api.User, id string) (api.User, error) {
	if !actor.IsSuperAdmin && actor.ID != id {
		return api.User{}, fmt.Errorf("get user: %w", domain.ErrForbidden)
	}

	user, _, err := s.usergroupClient.UsersAPI.ApiV2UsersUserIdGet(ctx, id).Execute()
	if err != nil {
		return api.User{}, fmt.Errorf("get user: %w", err)
	}

	return mapUser(user)
}

// PatchUser updates mutable local user fields.
func (s *Service) PatchUser(ctx context.Context, actor api.User, id string, patch api.PatchUserRequest) (api.User, error) {
	if !actor.IsSuperAdmin {
		return api.User{}, fmt.Errorf("patch user: %w", domain.ErrForbidden)
	}

	if patch.Status != nil && !patch.Status.IsValid() {
		return api.User{}, fmt.Errorf("patch user: %w", domain.ErrBadRequest)
	}

	request := usergroupclient.NewPatchUserRequest()
	if patch.Email != nil {
		request.SetEmail(*patch.Email)
	}
	if patch.GivenName != nil {
		request.SetGivenName(*patch.GivenName)
	}
	if patch.FamilyName != nil {
		request.SetFamilyName(*patch.FamilyName)
	}
	if patch.IsSuperAdmin != nil {
		request.SetIsSuperAdmin(*patch.IsSuperAdmin)
	}
	if patch.Status != nil {
		request.SetStatus(usergroupclient.UserStatus(*patch.Status))
	}

	user, resp, err := s.usergroupClient.UsersAPI.ApiV2UsersUserIdPatch(ctx, id).PatchUserRequest(*request).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return api.User{}, domain.ErrNotFound
		}

		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			return api.User{}, domain.ErrBadRequest
		}

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return api.User{}, fmt.Errorf("patch user timeout: %w: %w", domain.ErrUpstreamTimeout, err)
		}

		return api.User{}, fmt.Errorf("patch user: %w", err)
	}

	return mapUser(user)
}
