// Package groups provides group metadata operations.
package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
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

type Category struct {
	ID               uuid.UUID
	GroupID          uuid.UUID
	ParentCategoryID *uuid.UUID
	SourceIRI        string
	Name             string
}

type CreateCategoryRequest struct {
	SourceIRI        string
	Name             string
	ParentCategoryID *uuid.UUID
}

type ReplaceCategoryRequest struct {
	Name             string
	ParentCategoryID *uuid.UUID
}

type CreateRequest struct {
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

type ReplaceRequest struct {
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
	Create(ctx context.Context, group Group) (Group, error)
	Replace(ctx context.Context, id uuid.UUID, request ReplaceRequest) (Group, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListCategories(ctx context.Context, groupID uuid.UUID) ([]Category, error)
	GetCategory(ctx context.Context, groupID, categoryID uuid.UUID) (Category, error)
	CreateCategory(ctx context.Context, category Category) (Category, error)
	ReplaceCategory(ctx context.Context, groupID, categoryID uuid.UUID, request ReplaceCategoryRequest) (Category, error)
}

type Authorizer interface {
	AuthorizeAdmin(context.Context, string) (authz.Result, error)
	AuthorizeGlobal(context.Context, string, string) (authz.Result, error)
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (authz.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
}

func New(store Store, authorizer Authorizer) *Service {
	return &Service{store: store, authorizer: authorizer}
}

func (s *Service) Create(ctx context.Context, actorID string, request CreateRequest) (Group, error) {
	actorID = strings.TrimSpace(actorID)
	request.Name = strings.TrimSpace(request.Name)
	request.SourceIRI = strings.TrimSpace(request.SourceIRI)
	request.Concept = strings.TrimSpace(request.Concept)
	if actorID == "" || request.Name == "" || request.Concept == "" {
		return Group{}, ErrInvalid
	}
	if request.Concept != "DATA" && request.Concept != "FACILITIES" && request.Concept != "SOFTWARE" {
		return Group{}, ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeAdmin(ctx, actorID)
	if err != nil {
		return Group{}, fmt.Errorf("authorize group creation: %w", err)
	}
	if !allowed.Allowed {
		return Group{}, ErrForbidden
	}

	id := uuid.New()
	sourceIRI := request.SourceIRI
	if sourceIRI == "" {
		// TODO: define once and for all how to handle source iri for new resources
		sourceIRI = "urn:epos:group:" + id.String()
	}

	group, err := s.store.Create(ctx, Group{
		ID:          id,
		SourceIRI:   sourceIRI,
		Name:        request.Name,
		Description: request.Description,
		Label:       request.Label,
		Logo:        request.Logo,
		Page:        request.Page,
		Color:       request.Color,
		Position:    request.Position,
		Concept:     request.Concept,
	})
	if err != nil {
		return Group{}, fmt.Errorf("create group: %w", err)
	}

	return group, nil
}

func (s *Service) Replace(ctx context.Context, actorID string, id uuid.UUID, request ReplaceRequest) (Group, error) {
	actorID = strings.TrimSpace(actorID)
	request.Name = strings.TrimSpace(request.Name)
	request.Concept = strings.TrimSpace(request.Concept)
	if actorID == "" || id == uuid.Nil || request.Name == "" || request.Concept == "" {
		return Group{}, ErrInvalid
	}
	if request.Concept != "DATA" && request.Concept != "FACILITIES" && request.Concept != "SOFTWARE" {
		return Group{}, ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "category_scheme.manage")
	if err != nil {
		return Group{}, fmt.Errorf("authorize group replacement: %w", err)
	}
	if !allowed.Allowed {
		return Group{}, ErrForbidden
	}

	group, err := s.store.Replace(ctx, id, request)
	if err != nil {
		return Group{}, fmt.Errorf("replace group: %w", err)
	}

	return group, nil
}

func (s *Service) Delete(ctx context.Context, actorID string, id uuid.UUID) error {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || id == uuid.Nil {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeAdmin(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize group deletion: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	return nil
}

func (s *Service) ListCategories(ctx context.Context, groupID uuid.UUID) ([]Category, error) {
	if groupID == uuid.Nil {
		return nil, ErrInvalid
	}

	categories, err := s.store.ListCategories(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	return categories, nil
}

func (s *Service) GetCategory(ctx context.Context, groupID, categoryID uuid.UUID) (Category, error) {
	if groupID == uuid.Nil || categoryID == uuid.Nil {
		return Category{}, ErrInvalid
	}

	category, err := s.store.GetCategory(ctx, groupID, categoryID)
	if err != nil {
		return Category{}, fmt.Errorf("get category: %w", err)
	}

	return category, nil
}

func (s *Service) CreateCategory(ctx context.Context, actorID string, groupID uuid.UUID, request CreateCategoryRequest) (Category, error) {
	actorID = strings.TrimSpace(actorID)
	request.SourceIRI = strings.TrimSpace(request.SourceIRI)
	request.Name = strings.TrimSpace(request.Name)
	if actorID == "" || groupID == uuid.Nil || request.Name == "" {
		return Category{}, ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGroup(ctx, actorID, groupID, "category.manage")
	if err != nil {
		return Category{}, fmt.Errorf("authorize category creation: %w", err)
	}
	if !allowed.Allowed {
		return Category{}, ErrForbidden
	}

	id := uuid.New()
	sourceIRI := request.SourceIRI
	if sourceIRI == "" {
		sourceIRI = "urn:epos:category:" + id.String()
	}

	category, err := s.store.CreateCategory(ctx, Category{
		ID:               id,
		GroupID:          groupID,
		ParentCategoryID: request.ParentCategoryID,
		SourceIRI:        sourceIRI,
		Name:             request.Name,
	})
	if err != nil {
		return Category{}, fmt.Errorf("create category: %w", err)
	}

	return category, nil
}

func (s *Service) ReplaceCategory(ctx context.Context, actorID string, groupID, categoryID uuid.UUID, request ReplaceCategoryRequest) (Category, error) {
	actorID = strings.TrimSpace(actorID)
	request.Name = strings.TrimSpace(request.Name)
	if actorID == "" || groupID == uuid.Nil || categoryID == uuid.Nil || request.Name == "" {
		return Category{}, ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGroup(ctx, actorID, groupID, "category.manage")
	if err != nil {
		return Category{}, fmt.Errorf("authorize category replacement: %w", err)
	}
	if !allowed.Allowed {
		return Category{}, ErrForbidden
	}

	category, err := s.store.ReplaceCategory(ctx, groupID, categoryID, request)
	if err != nil {
		return Category{}, fmt.Errorf("replace category: %w", err)
	}

	return category, nil
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
