// Package organizations provides public canonical organization reads.
package organizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrInvalid  = errors.New("invalid organization")
	ErrNotFound = errors.New("organization not found")
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

type Store interface {
	List(context.Context) ([]Organization, error)
	Get(context.Context, uuid.UUID) (Organization, error)
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
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
