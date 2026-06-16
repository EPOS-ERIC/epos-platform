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

	if created == nil {
		return api.User{}, domain.ErrNotFound
	}

	return mapUser(created)
}
