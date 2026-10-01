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
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
func NewService(log *slog.Logger, baseURL string) (*Service, error) {
	normalizedBaseURL, err := urlutil.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize auth base URL: %w", err)
	}

	return &Service{
		log:     log.With("component", "auth-service"),
		baseURL: normalizedBaseURL,
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

	result, err := s.Validate(ctx, token)
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

	if !result.Active {
		s.log.WarnContext(ctx, "auth token rejected")
		return Identity{}, apperror.Default(apperror.CodeUnauthenticated)
	}

	if strings.TrimSpace(result.Identity.AuthIdentifier) == "" {
		s.log.WarnContext(ctx, "auth token missing identity")
		return Identity{}, apperror.Default(apperror.CodeUnauthenticated)
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
		if resp.StatusCode == http.StatusUnauthorized {
			return ValidationResult{}, apperror.Default(apperror.CodeUnauthenticated)
		}
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
