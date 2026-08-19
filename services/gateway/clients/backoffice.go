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
	backofficeclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

type Backoffice struct {
	identity  backofficeclient.InternalIdentityAPI
	resources backofficeclient.ResourcesAPI
}

type ActorIdentity struct {
	AuthIdentifier string `json:"auth_identifier"`
	Email          string `json:"email"`
	GivenName      string `json:"given_name"`
	FamilyName     string `json:"family_name"`
}

type User struct {
	ID                string       `json:"id"`
	AuthIdentifier    string       `json:"auth_identifier"`
	Email             string       `json:"email"`
	GivenName         string       `json:"given_name"`
	FamilyName        string       `json:"family_name"`
	Status            string       `json:"status"`
	GlobalRoles       []string     `json:"global_roles"`
	GlobalPermissions []string     `json:"global_permissions"`
	CreatedAt         string       `json:"created_at"`
	UpdatedAt         string       `json:"updated_at"`
	Memberships       []Membership `json:"memberships"`
}

type Membership struct {
	Group       Group    `json:"group"`
	Roles       []string `json:"roles"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

	clientConfig := backofficeclient.NewConfiguration()
	clientConfig.Servers[0].URL = normalizedBaseURL
	clientConfig.HTTPClient = upstreamClient.StandardClient()

	apiClient := backofficeclient.NewAPIClient(clientConfig)
	return &Backoffice{
		identity:  apiClient.InternalIdentityAPI,
		resources: apiClient.ResourcesAPI,
	}, nil
}

// Resources exposes typed Backoffice resource lifecycle operations.
func (c *Backoffice) Resources() backofficeclient.ResourcesAPI {
	if c == nil {
		return nil
	}
	return c.resources
}

func (c *Backoffice) GetUserByAuthIdentifier(ctx context.Context, authIdentifier string) (User, error) {
	if c == nil || c.identity == nil {
		return User{}, fmt.Errorf("backoffice identity client is not configured")
	}
	user, response, err := c.identity.GetUserByAuthIdentifier(ctx, authIdentifier).Execute()
	if err != nil {
		return User{}, getUserError("get backoffice user", response, err)
	}
	return userFromGenerated(user)
}

func (c *Backoffice) GetUserByID(ctx context.Context, userID, actorID string) (User, error) {
	if c == nil || c.identity == nil {
		return User{}, fmt.Errorf("backoffice identity client is not configured")
	}
	user, response, err := c.identity.GetUserByID(ctx, userID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return User{}, getUserError("get backoffice user", response, err)
	}
	return userFromGenerated(user)
}

func (c *Backoffice) EnsureUser(ctx context.Context, identity ActorIdentity) (User, error) {
	user, err := c.GetUserByAuthIdentifier(ctx, identity.AuthIdentifier)
	if errors.Is(err, domain.ErrNotFound) {
		user, err = c.CreateUser(ctx, identity)
		if errors.Is(err, domain.ErrConflict) {
			user, err = c.GetUserByAuthIdentifier(ctx, identity.AuthIdentifier)
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
	user, response, err := c.identity.CreateUser(ctx).CreateUserRequest(backofficeclient.CreateUserRequest{
		AuthIdentifier: identity.AuthIdentifier,
		Email:          identity.Email,
		GivenName:      identity.GivenName,
		FamilyName:     identity.FamilyName,
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

func (c *Backoffice) DeleteUser(ctx context.Context, userID string) error {
	if c == nil || c.identity == nil {
		return fmt.Errorf("backoffice identity client is not configured")
	}
	response, err := c.identity.DeleteUser(ctx, userID).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return domain.ErrNotFound
		}
		return generatedError("delete backoffice user", err)
	}
	return nil
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

func userFromGenerated(value *backofficeclient.User) (User, error) {
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
		AuthIdentifier:    value.AuthIdentifier,
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
