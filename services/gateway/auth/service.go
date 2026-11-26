// Package auth validates external access tokens and extracts identity claims.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Identity struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

type Service struct {
	log         *slog.Logger
	userInfoURL string
	client      *http.Client
}

// NewService builds auth service for token validation against external provider.
func NewService(log *slog.Logger, userInfoURL string) (*Service, error) {
	userInfoURL = strings.TrimSpace(userInfoURL)
	parsedURL, err := url.Parse(userInfoURL)
	if err != nil || parsedURL.Hostname() == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, fmt.Errorf("auth userinfo URL must be an absolute HTTP(S) URL without credentials, query or fragment")
	}

	return &Service{
		log:         log.With("component", "auth-service"),
		userInfoURL: userInfoURL,
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
			Timeout:   10 * time.Second,
		},
	}, nil
}

// Authenticate validates token and returns resolved identity claims.
func (s *Service) Authenticate(ctx context.Context, token string) (Identity, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Identity{}, apperror.Default(apperror.CodeUnauthenticated)
	}

	identity, err := s.Validate(ctx, token)
	if err != nil {
		s.log.ErrorContext(ctx, "auth token validation failed", "error", err)
		if apperror.IsCode(err, apperror.CodeUnauthenticated) {
			return Identity{}, err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return Identity{}, apperror.Wrap(apperror.CodeUpstreamTimeout, "", fmt.Errorf("validate access token: %w", err))
		}
		return Identity{}, apperror.Wrap(apperror.CodeUpstreamUnavailable, "", fmt.Errorf("validate access token: %w", err))
	}

	if strings.TrimSpace(identity.AuthIdentifier) == "" {
		s.log.WarnContext(ctx, "auth token missing identity")
		return Identity{}, apperror.Default(apperror.CodeUnauthenticated)
	}

	return identity, nil
}

// Validate checks token state at external auth provider.
func (s *Service) Validate(ctx context.Context, token string) (Identity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.userInfoURL, nil)
	if err != nil {
		return Identity{}, fmt.Errorf("build auth validation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return Identity{}, fmt.Errorf("call auth validation endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return Identity{}, apperror.Default(apperror.CodeUnauthenticated)
		}
		return Identity{}, fmt.Errorf("auth validation request failed: %s", resp.Status)
	}

	var payload struct {
		AuthID     string `json:"eduPersonUniqueId"`
		Email      string `json:"email"`
		GivenName  string `json:"firstname"`
		FamilyName string `json:"lastName"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Identity{}, fmt.Errorf("decode auth validation response: %w", err)
	}

	return Identity{
		AuthIdentifier: payload.AuthID,
		Email:          payload.Email,
		GivenName:      payload.GivenName,
		FamilyName:     payload.FamilyName,
	}, nil
}
