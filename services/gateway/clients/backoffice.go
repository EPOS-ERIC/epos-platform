// Package clients provides outbound service clients for the gateway.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

type Backoffice struct {
	baseURL string
	http    *http.Client
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

type BackofficeResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
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

	return &Backoffice{
		baseURL: normalizedBaseURL,
		http:    upstreamClient.StandardClient(),
	}, nil
}

func (c *Backoffice) GetUserByAuthIdentifier(ctx context.Context, authIdentifier string) (User, error) {
	path, err := url.JoinPath("/api/v1/users/by-auth", url.PathEscape(authIdentifier))
	if err != nil {
		return User{}, fmt.Errorf("build backoffice user path: %w", err)
	}
	response, err := c.Request(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return User{}, fmt.Errorf("get backoffice user: %w", err)
	}
	if response.StatusCode == http.StatusNotFound {
		return User{}, domain.ErrNotFound
	}
	if response.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("get backoffice user: unexpected status %d", response.StatusCode)
	}
	var user User
	if err := json.Unmarshal(response.Body, &user); err != nil {
		return User{}, fmt.Errorf("decode backoffice user: %w", err)
	}
	return user, nil
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
	body, err := json.Marshal(identity)
	if err != nil {
		return User{}, fmt.Errorf("encode actor identity: %w", err)
	}

	response, err := c.Request(ctx, http.MethodPost, "/api/v1/users", body, "")
	if err != nil {
		return User{}, fmt.Errorf("create backoffice user: %w", err)
	}
	if response.StatusCode == http.StatusConflict {
		return User{}, domain.ErrConflict
	}
	if response.StatusCode != http.StatusCreated {
		return User{}, fmt.Errorf("create backoffice user: unexpected status %d", response.StatusCode)
	}

	var user User
	if err := json.Unmarshal(response.Body, &user); err != nil {
		return User{}, fmt.Errorf("decode backoffice user: %w", err)
	}
	if user.ID == "" {
		return User{}, fmt.Errorf("decode backoffice user: missing id")
	}
	return user, nil
}

func (c *Backoffice) DeleteUser(ctx context.Context, userID string) error {
	path, err := url.JoinPath("/api/v1/users", url.PathEscape(userID))
	if err != nil {
		return fmt.Errorf("build backoffice delete path: %w", err)
	}
	response, err := c.Request(ctx, http.MethodDelete, path, nil, "")
	if err != nil {
		return fmt.Errorf("delete backoffice user: %w", err)
	}
	if response.StatusCode == http.StatusNotFound {
		return domain.ErrNotFound
	}
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete backoffice user: unexpected status %d", response.StatusCode)
	}
	return nil
}

func (c *Backoffice) Request(ctx context.Context, method, path string, body []byte, actorID string) (BackofficeResponse, error) {
	if c == nil || c.http == nil {
		return BackofficeResponse{}, fmt.Errorf("backoffice client is not configured")
	}

	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return BackofficeResponse{}, fmt.Errorf("parse backoffice base URL: %w", err)
	}
	endpoint.Path, err = url.JoinPath(endpoint.Path, path)
	if err != nil {
		return BackofficeResponse{}, fmt.Errorf("join backoffice request path: %w", err)
	}
	endpoint.RawPath = ""

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return BackofficeResponse{}, fmt.Errorf("create backoffice request: %w", err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if actorID != "" {
		req.Header.Set("X-EPOS-User-ID", actorID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return BackofficeResponse{}, fmt.Errorf("backoffice request: %w: %w", domain.ErrUpstreamTimeout, err)
		}
		return BackofficeResponse{}, fmt.Errorf("backoffice request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return BackofficeResponse{}, fmt.Errorf("read backoffice response: %w", err)
	}
	return BackofficeResponse{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        responseBody,
	}, nil
}
