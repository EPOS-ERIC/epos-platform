// Package clients provides outbound service clients for the gateway.
package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

type Backoffice struct {
	identity  client.InternalIdentityAPI
	users     client.UsersAPI
	resources client.ResourcesAPI
	imports   client.ImportsAPI
}

type ActorIdentity struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

func NewBackoffice(baseURL string, upstreamCfg config.Upstream) (*Backoffice, error) {
	normalizedBaseURL, err := urlutil.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize backoffice base URL: %w", err)
	}

	upstreamClient, err := upstream.NewClient(upstream.Config{
		Service:     "backoffice",
		Timeout:     upstreamCfg.Timeout,
		RealTimeout: upstreamCfg.RealTimeout,
		CacheTTL:    0,
		MaxEntries:  upstreamCfg.MaxEntries,
		Transport:   nil,
	})
	if err != nil {
		return nil, fmt.Errorf("create backoffice client: %w", err)
	}

	clientConfig := client.NewConfiguration()
	clientConfig.Servers[0].URL = normalizedBaseURL
	clientConfig.HTTPClient = upstreamClient.StandardClient()

	apiClient := client.NewAPIClient(clientConfig)
	return &Backoffice{
		identity:  apiClient.InternalIdentityAPI,
		users:     apiClient.UsersAPI,
		resources: apiClient.ResourcesAPI,
		imports:   apiClient.ImportsAPI,
	}, nil
}

// Imports exposes typed Backoffice metadata import operations.
func (c *Backoffice) Imports() client.ImportsAPI {
	if c == nil {
		return nil
	}
	return c.imports
}

// Resources exposes typed Backoffice resource lifecycle operations.
func (c *Backoffice) Resources() client.ResourcesAPI {
	if c == nil {
		return nil
	}
	return c.resources
}

func (c *Backoffice) GetUserByID(ctx context.Context, userID, actorID string) (domain.User, error) {
	if c == nil || c.users == nil {
		return domain.User{}, fmt.Errorf("backoffice identity client is not configured")
	}

	user, response, err := c.users.GetUser(ctx, userID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return domain.User{}, wrapUserError("get backoffice user", response, err)
	}

	return toDomainUser(user)
}

func (c *Backoffice) ListUsers(ctx context.Context, actorID string) ([]domain.UserSummary, error) {
	if c == nil || c.users == nil {
		return nil, fmt.Errorf("backoffice identity client is not configured")
	}

	users, response, err := c.users.ListUsers(ctx).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapUserError("list backoffice users", response, err)
	}

	result := make([]domain.UserSummary, 0, len(users))
	for _, user := range users {
		result = append(result, domain.UserSummary{
			ID:                user.Id,
			Email:             user.Email,
			GivenName:         user.GivenName,
			FamilyName:        user.FamilyName,
			Status:            user.Status,
			GlobalRoles:       append([]string{}, user.GlobalRoles...),
			GlobalPermissions: append([]string{}, user.GlobalPermissions...),
			CreatedAt:         "",
			UpdatedAt:         "",
		})
	}

	return result, nil
}

func (c *Backoffice) EnsureUser(ctx context.Context, identity ActorIdentity) (domain.User, error) {
	user, err := c.GetUserByID(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
	if errors.Is(err, domain.ErrNotFound) {
		user, err = c.CreateUser(ctx, identity)
		if errors.Is(err, domain.ErrConflict) {
			user, err = c.GetUserByID(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
		}
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("ensure backoffice user: %w", err)
	}

	return user, nil
}

func (c *Backoffice) CreateUser(ctx context.Context, identity ActorIdentity) (domain.User, error) {
	if c == nil || c.identity == nil {
		return domain.User{}, fmt.Errorf("backoffice identity client is not configured")
	}

	user, response, err := c.identity.ProvisionUser(ctx).CreateUserRequest(client.CreateUserRequest{
		Id:         identity.AuthIdentifier,
		Email:      identity.Email,
		GivenName:  identity.GivenName,
		FamilyName: identity.FamilyName,
	}).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusConflict {
			return domain.User{}, domain.ErrConflict
		}

		return domain.User{}, wrapConnectionError("create backoffice user", err)
	}

	resolved, err := toDomainUser(user)
	if err != nil {
		return domain.User{}, err
	}

	return resolved, nil
}

func (c *Backoffice) ImportTurtle(ctx context.Context, actorID string, body []byte) (*domain.ImportResult, error) {
	if c == nil || c.imports == nil {
		return nil, fmt.Errorf("backoffice import client is not configured")
	}

	result, response, err := c.imports.ImportTurtleMetadata(ctx).XEPOSUserID(actorID).Body(string(body)).Execute()
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusBadRequest:
				return nil, fmt.Errorf("backoffice import rejected request: %w", domain.ErrBadRequest)
			case http.StatusForbidden:
				return nil, fmt.Errorf("backoffice import forbidden: %w", domain.ErrForbidden)
			case http.StatusConflict:
				return nil, fmt.Errorf("backoffice import conflict: %w", domain.ErrConflict)
			}
			return nil, fmt.Errorf("backoffice import failed with status %d: %w", response.StatusCode, err)
		}
		return nil, wrapConnectionError("backoffice import request", err)
	}

	if result == nil {
		return nil, fmt.Errorf("decode backoffice import result: response is empty")
	}

	responseResult := &domain.ImportResult{
		Groups:               result.Groups,
		Categories:           result.Categories,
		Organizations:        result.Organizations,
		ContactPoints:        result.ContactPoints,
		Datasets:             result.Datasets,
		SoftwareApplications: result.SoftwareApplications,
		SoftwareSourceCodes:  result.SoftwareSourceCodes,
		Warnings:             make([]domain.ImportDiagnostic, 0, len(result.Warnings)),
	}
	for _, warning := range result.Warnings {
		responseResult.Warnings = append(responseResult.Warnings, domain.ImportDiagnostic{
			Severity:  warning.Severity,
			Subject:   warning.Subject,
			Predicate: warning.Predicate,
			Message:   warning.Message,
		})
	}

	return responseResult, nil
}

func wrapUserError(operation string, response *http.Response, err error) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusNotFound:
			return domain.ErrNotFound
		case http.StatusForbidden:
			return domain.ErrForbidden
		}
	}
	return wrapConnectionError(operation, err)
}

func wrapConnectionError(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func toDomainUser(value *client.User) (domain.User, error) {
	if value == nil {
		return domain.User{}, fmt.Errorf("decode backoffice user: response is empty")
	}

	memberships := make([]domain.Membership, 0, len(value.Memberships))
	for _, membership := range value.Memberships {
		memberships = append(memberships, domain.Membership{
			Group: domain.Group{
				ID:   membership.Group.Id,
				Name: membership.Group.Name,
			},
			Roles:       append([]string{}, membership.Roles...),
			Status:      membership.Status,
			Permissions: append([]string{}, membership.Permissions...),
			CreatedAt:   membership.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:   membership.UpdatedAt.Format(time.RFC3339Nano),
		})
	}

	return domain.User{
		ID:                value.Id,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            value.Status,
		GlobalRoles:       append([]string{}, value.GlobalRoles...),
		GlobalPermissions: append([]string{}, value.GlobalPermissions...),
		CreatedAt:         value.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         value.UpdatedAt.Format(time.RFC3339Nano),
		Memberships:       memberships,
	}, nil
}
