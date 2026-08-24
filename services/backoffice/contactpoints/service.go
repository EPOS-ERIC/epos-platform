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
	ErrForbidden = errors.New("contact point operation forbidden")
	ErrConflict  = errors.New("contact point conflict")
)

type ContactPoint struct {
	ID        uuid.UUID
	SourceIRI string
	Email     string
	Role      string
}

type CreateRequest struct {
	SourceIRI string
	Email     string
	Role      string
}

type ReplaceRequest struct {
	Email string
	Role  string
}

type Store interface {
	List(context.Context) ([]ContactPoint, error)
	Get(context.Context, uuid.UUID) (ContactPoint, error)
	Create(context.Context, ContactPoint) (ContactPoint, error)
	Replace(context.Context, uuid.UUID, ReplaceRequest) (ContactPoint, error)
	Delete(context.Context, uuid.UUID) error
}

type Authorizer interface {
	CanReadContactPoint(context.Context, string) (authz.Result, error)
	AuthorizeGlobal(context.Context, string, string) (authz.Result, error)
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

func (s *Service) Create(ctx context.Context, actorID string, request CreateRequest) (ContactPoint, error) {
	actorID = strings.TrimSpace(actorID)
	request.SourceIRI = strings.TrimSpace(request.SourceIRI)
	request.Email = strings.TrimSpace(request.Email)
	request.Role = strings.TrimSpace(request.Role)
	if actorID == "" || request.Email == "" || request.Role == "" {
		return ContactPoint{}, ErrInvalid
	}

	if err := s.authorizeManage(ctx, actorID); err != nil {
		return ContactPoint{}, err
	}

	id := uuid.New()
	sourceIRI := request.SourceIRI
	if sourceIRI == "" {
		// TODO: define the "urn:epos" as a app level constant instead of hard coding it in different places
		sourceIRI = "urn:epos:contact-point:" + id.String()
	}

	contactPoint, err := s.store.Create(ctx, ContactPoint{
		ID:        id,
		SourceIRI: sourceIRI,
		Email:     request.Email,
		Role:      request.Role,
	})
	if err != nil {
		return ContactPoint{}, fmt.Errorf("create contact point: %w", err)
	}

	return contactPoint, nil
}

func (s *Service) Replace(ctx context.Context, actorID string, id uuid.UUID, request ReplaceRequest) (ContactPoint, error) {
	actorID = strings.TrimSpace(actorID)
	request.Email = strings.TrimSpace(request.Email)
	request.Role = strings.TrimSpace(request.Role)
	if actorID == "" || id == uuid.Nil || request.Email == "" || request.Role == "" {
		return ContactPoint{}, ErrInvalid
	}

	if err := s.authorizeManage(ctx, actorID); err != nil {
		return ContactPoint{}, err
	}

	contactPoint, err := s.store.Replace(ctx, id, request)
	if err != nil {
		return ContactPoint{}, fmt.Errorf("replace contact point: %w", err)
	}

	return contactPoint, nil
}

func (s *Service) Delete(ctx context.Context, actorID string, id uuid.UUID) error {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || id == uuid.Nil {
		return ErrInvalid
	}

	if err := s.authorizeManage(ctx, actorID); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete contact point: %w", err)
	}

	return nil
}

func (s *Service) authorizeManage(ctx context.Context, actorID string) error {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "contact_point.manage")
	if err != nil {
		return fmt.Errorf("authorize contact point management: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}
	return nil
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
