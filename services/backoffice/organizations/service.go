// Package organizations provides public canonical organization reads.
package organizations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
	"github.com/google/uuid"
)

var (
	ErrInvalid   = errors.New("invalid organization")
	ErrNotFound  = errors.New("organization not found")
	ErrForbidden = errors.New("organization operation forbidden")
	ErrConflict  = errors.New("organization conflict")
)

type Organization struct {
	ID                    uuid.UUID
	SourceIRI             string
	Name                  string
	Street                *string
	Locality              *string
	PostalCode            *string
	Country               *string
	LogoURL               *string
	PageURL               *string
	Email                 *string
	LEICode               *string
	Identifiers           []Identifier
	ParentOrganizationIDs []uuid.UUID
	ContactPointIDs       []uuid.UUID
}

type Identifier struct {
	ID        uuid.UUID
	SourceIRI string
	Type      string
	Value     string
}

type IdentifierRequest struct {
	SourceIRI string
	Type      string
	Value     string
}

type CreateRequest struct {
	SourceIRI       string
	Name            string
	Street          *string
	Locality        *string
	PostalCode      *string
	Country         *string
	LogoURL         *string
	PageURL         *string
	Email           *string
	LEICode         *string
	Identifiers     []IdentifierRequest
	Parents         []uuid.UUID
	ContactPointIDs []uuid.UUID
}

type ReplaceRequest struct {
	Name            string
	Street          *string
	Locality        *string
	PostalCode      *string
	Country         *string
	LogoURL         *string
	PageURL         *string
	Email           *string
	LEICode         *string
	Identifiers     []IdentifierRequest
	Parents         []uuid.UUID
	ContactPointIDs []uuid.UUID
}

type Store interface {
	List(context.Context) ([]Organization, error)
	Get(context.Context, uuid.UUID) (Organization, error)
	Create(context.Context, Organization, []Identifier, []uuid.UUID, []uuid.UUID) (Organization, error)
	Replace(context.Context, uuid.UUID, ReplaceRequest, []Identifier, []uuid.UUID, []uuid.UUID) (Organization, error)
}

type Authorizer interface {
	AuthorizeGlobal(context.Context, string, string) (authz.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
}

func New(store Store, authorizer Authorizer) *Service {
	return &Service{store: store, authorizer: authorizer}
}

func (s *Service) Create(ctx context.Context, actorID string, request CreateRequest) (Organization, error) {
	actorID = strings.TrimSpace(actorID)
	request.SourceIRI = strings.TrimSpace(request.SourceIRI)
	request.Name = strings.TrimSpace(request.Name)
	if actorID == "" || request.Name == "" {
		return Organization{}, ErrInvalid
	}
	if err := validateUUIDs(request.Parents); err != nil {
		return Organization{}, err
	}
	if err := validateUUIDs(request.ContactPointIDs); err != nil {
		return Organization{}, err
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "organization.manage")
	if err != nil {
		return Organization{}, fmt.Errorf("authorize organization creation: %w", err)
	}
	if !allowed.Allowed {
		return Organization{}, ErrForbidden
	}

	organizationID := uuid.New()
	if request.SourceIRI == "" {
		request.SourceIRI = "urn:epos:organization:" + organizationID.String()
	}
	identifiers := make([]Identifier, 0, len(request.Identifiers))
	for _, item := range request.Identifiers {
		item.SourceIRI = strings.TrimSpace(item.SourceIRI)
		item.Type = strings.TrimSpace(item.Type)
		item.Value = strings.TrimSpace(item.Value)
		if item.Type == "" || item.Value == "" {
			return Organization{}, ErrInvalid
		}

		id := uuid.New()
		if item.SourceIRI == "" {
			item.SourceIRI = "urn:epos:identifier:" + id.String()
		}

		identifiers = append(identifiers, Identifier{
			ID:        id,
			SourceIRI: item.SourceIRI,
			Type:      item.Type,
			Value:     item.Value,
		})
	}

	organization, err := s.store.Create(ctx, Organization{
		ID:                    organizationID,
		SourceIRI:             request.SourceIRI,
		Name:                  request.Name,
		Street:                request.Street,
		Locality:              request.Locality,
		PostalCode:            request.PostalCode,
		Country:               request.Country,
		LogoURL:               request.LogoURL,
		PageURL:               request.PageURL,
		Email:                 request.Email,
		LEICode:               request.LEICode,
		Identifiers:           nil,
		ParentOrganizationIDs: nil,
		ContactPointIDs:       nil,
	}, identifiers, request.Parents, request.ContactPointIDs)
	if err != nil {
		return Organization{}, fmt.Errorf("create organization: %w", err)
	}

	return organization, nil
}

func (s *Service) Replace(ctx context.Context, actorID string, id uuid.UUID, request ReplaceRequest) (Organization, error) {
	actorID = strings.TrimSpace(actorID)
	request.Name = strings.TrimSpace(request.Name)
	if actorID == "" || id == uuid.Nil || request.Name == "" {
		return Organization{}, ErrInvalid
	}
	if err := validateUUIDs(request.Parents); err != nil {
		return Organization{}, err
	}
	if err := validateUUIDs(request.ContactPointIDs); err != nil {
		return Organization{}, err
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "organization.manage")
	if err != nil {
		return Organization{}, fmt.Errorf("authorize organization replacement: %w", err)
	}
	if !allowed.Allowed {
		return Organization{}, ErrForbidden
	}

	identifiers, err := buildIdentifiers(request.Identifiers)
	if err != nil {
		return Organization{}, err
	}
	organization, err := s.store.Replace(ctx, id, request, identifiers, request.Parents, request.ContactPointIDs)
	if err != nil {
		return Organization{}, fmt.Errorf("replace organization: %w", err)
	}
	return organization, nil
}

func buildIdentifiers(requests []IdentifierRequest) ([]Identifier, error) {
	identifiers := make([]Identifier, 0, len(requests))
	for _, item := range requests {
		item.SourceIRI = strings.TrimSpace(item.SourceIRI)
		item.Type = strings.TrimSpace(item.Type)
		item.Value = strings.TrimSpace(item.Value)
		if item.Type == "" || item.Value == "" {
			return nil, ErrInvalid
		}
		id := uuid.New()
		if item.SourceIRI == "" {
			item.SourceIRI = "urn:epos:identifier:" + id.String()
		}
		identifiers = append(identifiers, Identifier{ID: id, SourceIRI: item.SourceIRI, Type: item.Type, Value: item.Value})
	}
	return identifiers, nil
}

func validateUUIDs(values []uuid.UUID) error {
	for _, value := range values {
		if value == uuid.Nil {
			return ErrInvalid
		}
	}
	return nil
}

func (s *Service) List(ctx context.Context) ([]Organization, error) {
	organizations, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return organizations, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Organization, error) {
	if id == uuid.Nil {
		return Organization{}, ErrInvalid
	}

	organization, err := s.store.Get(ctx, id)
	if err != nil {
		return Organization{}, fmt.Errorf("get organization: %w", err)
	}

	return organization, nil
}
