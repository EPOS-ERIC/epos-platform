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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

func annotateSpan(ctx context.Context, attrs ...attribute.KeyValue) {
	if len(attrs) == 0 {
		return
	}

	trace.SpanFromContext(ctx).SetAttributes(attrs...)
}

// EnsureUser returns existing local user or creates one from auth identity.
func (s *Service) EnsureUser(ctx context.Context, identity Identity) (api.User, error) {
	annotateSpan(ctx,
		attribute.String("usergroup.operation", "ensure_user"),
	)

	if identity.AuthIdentifier == "" {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.User{}, fmt.Errorf("ensure user: %w", domain.ErrBadRequest)
	}

	user, err := s.GetUserByAuthIdentifier(ctx, identity.AuthIdentifier)
	if err == nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "existing"))
		return user, nil
	}

	if !errors.Is(err, domain.ErrNotFound) {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "ensure user lookup failed", "error", err)
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
			annotateSpan(ctx, attribute.String("usergroup.result", "conflict_retry"))
			return s.GetUserByAuthIdentifier(ctx, identity.AuthIdentifier)
		}

		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "create user failed", "error", err)
		return api.User{}, fmt.Errorf("create user: %w", err)
	}

	user, mapErr := mapUser(created)
	if mapErr != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map created user failed", "error", mapErr)
		return api.User{}, mapErr
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "created"))
	return user, nil
}

// ListUsers fetches all local users.
func (s *Service) ListUsers(ctx context.Context, actor api.User) ([]api.User, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "list_users"))

	if !actor.IsSuperAdmin {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return nil, fmt.Errorf("list users: %w", domain.ErrForbidden)
	}

	users, _, err := s.usergroupClient.UsersAPI.ApiV2UsersGet(ctx).Execute()
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "list users failed", "error", err)
		return nil, fmt.Errorf("list users from usergroup API: %w", err)
	}

	mapped := make([]api.User, 0, len(users))
	for i := range users {
		user, err := mapUser(&users[i])
		if err != nil {
			annotateSpan(ctx, attribute.String("usergroup.result", "error"))
			s.log.ErrorContext(ctx, "list users map failed", "error", err)
			return nil, fmt.Errorf("list users: map user %d: %w", i, err)
		}
		mapped = append(mapped, user)
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}

// GetUser fetches local user by internal id.
func (s *Service) GetUser(ctx context.Context, actor api.User, id string) (api.User, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "get_user"))

	if !actor.IsSuperAdmin && actor.ID != id {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return api.User{}, fmt.Errorf("get user: %w", domain.ErrForbidden)
	}

	user, _, err := s.usergroupClient.UsersAPI.ApiV2UsersUserIdGet(ctx, id).Execute()
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "get user failed", "error", err)
		return api.User{}, fmt.Errorf("get user: %w", err)
	}

	mapped, err := mapUser(user)
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map user failed", "error", err)
		return api.User{}, err
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}

// PatchUser updates mutable local user fields.
func (s *Service) PatchUser(ctx context.Context, actor api.User, id string, patch api.PatchUserRequest) (api.User, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "patch_user"))

	if !actor.IsSuperAdmin {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return api.User{}, fmt.Errorf("patch user: %w", domain.ErrForbidden)
	}

	if patch.Status != nil && !patch.Status.IsValid() {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
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
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return api.User{}, domain.ErrNotFound
		}

		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return api.User{}, domain.ErrBadRequest
		}

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			annotateSpan(ctx, attribute.String("usergroup.result", "timeout"))
			s.log.ErrorContext(ctx, "patch user timeout", "error", err)
			return api.User{}, fmt.Errorf("patch user timeout: %w: %w", domain.ErrUpstreamTimeout, err)
		}

		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "patch user failed", "error", err)
		return api.User{}, fmt.Errorf("patch user: %w", err)
	}

	mapped, err := mapUser(user)
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map patched user failed", "error", err)
		return api.User{}, err
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}
