// Package contactpoints provides protected canonical contact-point reads.
package contactpoints

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
	"github.com/google/uuid"
)

var (
	ErrInvalid   = errors.New("invalid contact point")
	ErrNotFound  = errors.New("contact point not found")
	ErrForbidden = errors.New("contact point read forbidden")
)

type ContactPoint struct {
	ID        uuid.UUID
	SourceIRI string
	Email     string
	Role      string
}

type Store interface {
	List(context.Context) ([]ContactPoint, error)
	Get(context.Context, uuid.UUID) (ContactPoint, error)
}

type Authorizer interface {
	CanReadContactPoint(context.Context, string) (authz.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
}

func New(store Store, authorizer Authorizer) *Service {
	return &Service{store: store, authorizer: authorizer}
}

func (s *Service) List(ctx context.Context, actorID string) ([]ContactPoint, error) {
	if err := s.authorize(ctx, actorID); err != nil {
		return nil, err
	}

	contactPoints, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list contact points: %w", err)
	}

	return contactPoints, nil
}

func (s *Service) Get(ctx context.Context, actorID string, id uuid.UUID) (ContactPoint, error) {
	if id == uuid.Nil {
		return ContactPoint{}, ErrInvalid
	}
	if err := s.authorize(ctx, actorID); err != nil {
		return ContactPoint{}, err
	}

	contactPoint, err := s.store.Get(ctx, id)
	if err != nil {
		return ContactPoint{}, fmt.Errorf("get contact point: %w", err)
	}

	return contactPoint, nil
}

func (s *Service) authorize(ctx context.Context, actorID string) error {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.CanReadContactPoint(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize contact point read: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	return nil
}
