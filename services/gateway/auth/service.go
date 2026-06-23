// Package auth validates external access tokens and extracts identity claims.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
)

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrProviderUnavailable = errors.New("auth provider unavailable")
)

type Identity struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

type ValidationResult struct {
	Active   bool
	Identity Identity
}

type Service struct {
	log     *slog.Logger
	baseURL string
	client  *http.Client
}

// NewService builds auth service for token validation against external provider.
func NewService(baseURL string, upstreamCfg config.Upstream) (*Service, error) {
	normalizedBaseURL, err := urlutil.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize auth base URL: %w", err)
	}

	// TODO: see if we need an upstream client here too or if a default http client is fine
	return &Service{
		log:     logging.Get("auth-service"),
		baseURL: normalizedBaseURL,
		client: &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       upstreamCfg.Timeout,
		},
	}, nil
}

// Authenticate validates token and returns resolved identity claims.
func (s *Service) Authenticate(ctx context.Context, token string) (Identity, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Identity{}, ErrUnauthorized
	}

	result, err := s.Validate(ctx, token)
	if err != nil {
		s.log.ErrorContext(ctx, "auth token validation failed", "error", err)
		return Identity{}, fmt.Errorf("%w: validate access token: %v", ErrProviderUnavailable, err)
	}

	if !result.Active {
		s.log.WarnContext(ctx, "auth token rejected")
		return Identity{}, ErrUnauthorized
	}

	if strings.TrimSpace(result.Identity.AuthIdentifier) == "" {
		s.log.WarnContext(ctx, "auth token missing identity")
		return Identity{}, ErrUnauthorized
	}

	return result.Identity, nil
}

// Validate checks token state at external auth provider.
func (s *Service) Validate(ctx context.Context, token string) (ValidationResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/oauth/validate", nil)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("build auth validation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("call auth validation endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ValidationResult{}, fmt.Errorf("auth validation request failed: %s", resp.Status)
	}

	var payload struct {
		Active     bool   `json:"active"`
		AuthID     string `json:"eduPersonUniqueId"`
		Email      string `json:"email"`
		GivenName  string `json:"firstname"`
		FamilyName string `json:"lastName"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ValidationResult{}, fmt.Errorf("decode auth validation response: %w", err)
	}

	return ValidationResult{
		Active: payload.Active,
		Identity: Identity{
			AuthIdentifier: payload.AuthID,
			Email:          payload.Email,
			GivenName:      payload.GivenName,
			FamilyName:     payload.FamilyName,
		},
	}, nil
}
