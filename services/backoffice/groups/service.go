// Package groups provides group metadata operations.
package groups

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrInvalid  = errors.New("invalid group")
	ErrNotFound = errors.New("group not found")
)

// Group is public group metadata.
type Group struct {
	ID          uuid.UUID
	SourceIRI   string
	Name        string
	Description *string
	Label       *string
	Logo        *string
	Page        *string
	Color       *string
	Position    int32
	Concept     string
}

type Store interface {
	List(ctx context.Context) ([]Group, error)
	Get(ctx context.Context, id uuid.UUID) (Group, error)
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context) ([]Group, error) {
	groups, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	return groups, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Group, error) {
	if id == uuid.Nil {
		return Group{}, ErrInvalid
	}

	group, err := s.store.Get(ctx, id)
	if err != nil {
		return Group{}, fmt.Errorf("get group: %w", err)
	}

	return group, nil
}
