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

type User struct {
	ID                string
	Email             string
	GivenName         string
	FamilyName        string
	Status            string
	GlobalRoles       []string
	GlobalPermissions []string
	CreatedAt         string
	UpdatedAt         string
	Memberships       []Membership
}

type Membership struct {
	Group       Group
	Roles       []string
	Status      string
	Permissions []string
	CreatedAt   string
	UpdatedAt   string
}

type Group struct {
	ID   string
	Name string
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

func (c *Backoffice) GetUserByID(ctx context.Context, userID, actorID string) (User, error) {
	if c == nil || c.users == nil {
		return User{}, fmt.Errorf("backoffice identity client is not configured")
	}

	user, response, err := c.users.GetUser(ctx, userID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return User{}, getUserError("get backoffice user", response, err)
	}

	return userFromGenerated(user)
}

func (c *Backoffice) ListUsers(ctx context.Context, actorID string) ([]User, error) {
	if c == nil || c.users == nil {
		return nil, fmt.Errorf("backoffice identity client is not configured")
	}

	users, response, err := c.users.ListUsers(ctx).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, getUserError("list backoffice users", response, err)
	}

	result := make([]User, 0, len(users))
	for _, user := range users {
		result = append(result, User{
			ID:                user.Id,
			Email:             user.Email,
			GivenName:         user.GivenName,
			FamilyName:        user.FamilyName,
			Status:            user.Status,
			GlobalRoles:       append([]string{}, user.GlobalRoles...),
			GlobalPermissions: append([]string{}, user.GlobalPermissions...),
			CreatedAt:         "",
			UpdatedAt:         "",
			Memberships:       []Membership{},
		})
	}

	return result, nil
}

func (c *Backoffice) EnsureUser(ctx context.Context, identity ActorIdentity) (User, error) {
	user, err := c.GetUserByID(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
	if errors.Is(err, domain.ErrNotFound) {
		user, err = c.CreateUser(ctx, identity)
		if errors.Is(err, domain.ErrConflict) {
			user, err = c.GetUserByID(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
		}
	}
	if err != nil {
		return User{}, fmt.Errorf("ensure backoffice user: %w", err)
	}
	return user, nil
}

func (c *Backoffice) CreateUser(ctx context.Context, identity ActorIdentity) (User, error) {
	if c == nil || c.identity == nil {
		return User{}, fmt.Errorf("backoffice identity client is not configured")
	}
	user, response, err := c.identity.ProvisionUser(ctx).CreateUserRequest(client.CreateUserRequest{
		Id:         identity.AuthIdentifier,
		Email:      identity.Email,
		GivenName:  identity.GivenName,
		FamilyName: identity.FamilyName,
	}).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusConflict {
			return User{}, domain.ErrConflict
		}
		return User{}, generatedError("create backoffice user", err)
	}
	resolved, err := userFromGenerated(user)
	if err != nil {
		return User{}, err
	}
	if resolved.ID == "" {
		return User{}, fmt.Errorf("decode backoffice user: missing id")
	}
	return resolved, nil
}

func getUserError(operation string, response *http.Response, err error) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusNotFound:
			return domain.ErrNotFound
		case http.StatusForbidden:
			return domain.ErrForbidden
		}
	}
	return generatedError(operation, err)
}

func generatedError(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func userFromGenerated(value *client.User) (User, error) {
	if value == nil {
		return User{}, fmt.Errorf("decode backoffice user: response is empty")
	}
	memberships := make([]Membership, 0, len(value.Memberships))
	for _, membership := range value.Memberships {
		memberships = append(memberships, Membership{
			Group: Group{
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
	return User{
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
