package repository

import (
	"errors"
	"sync"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/domain"
)

var (
	ErrGroupNotFound   = errors.New("group not found")
	ErrGroupSlugExists = errors.New("group slug already exists")
)

type GroupRepository interface {
	List() []domain.Group
	GetByID(id string) (domain.Group, error)
	Create(slug, name, description string, isSystem bool) (domain.Group, error)
	Replace(id, slug, name, description string, isSystem bool) (domain.Group, error)
	Patch(id string, slug, name, description *string, isSystem *bool) (domain.Group, error)
	Delete(id string) error
}

type GroupMemoryRepository struct {
	mu     sync.RWMutex
	groups map[string]domain.Group
}

func NewGroupMemoryRepository() *GroupMemoryRepository {
	return &GroupMemoryRepository{groups: map[string]domain.Group{}}
}

func (r *GroupMemoryRepository) List() []domain.Group {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Group, 0, len(r.groups))
	for _, g := range r.groups {
		result = append(result, g)
	}

	return result
}

func (r *GroupMemoryRepository) GetByID(id string) (domain.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	g, ok := r.groups[id]
	if !ok {
		return domain.Group{}, ErrGroupNotFound
	}

	return g, nil
}

func (r *GroupMemoryRepository) Create(slug, name, description string, isSystem bool) (domain.Group, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.slugInUse(slug, "") {
		return domain.Group{}, ErrGroupSlugExists
	}

	now := time.Now().UTC()
	g := domain.Group{
		ID:          newUUIDLike(),
		Slug:        slug,
		Name:        name,
		Description: description,
		IsSystem:    isSystem,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.groups[g.ID] = g

	return g, nil
}

func (r *GroupMemoryRepository) Replace(id, slug, name, description string, isSystem bool) (domain.Group, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	g, ok := r.groups[id]
	if !ok {
		return domain.Group{}, ErrGroupNotFound
	}
	if r.slugInUse(slug, id) {
		return domain.Group{}, ErrGroupSlugExists
	}

	g.Slug = slug
	g.Name = name
	g.Description = description
	g.IsSystem = isSystem
	g.UpdatedAt = time.Now().UTC()
	r.groups[id] = g

	return g, nil
}

func (r *GroupMemoryRepository) Patch(id string, slug, name, description *string, isSystem *bool) (domain.Group, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	g, ok := r.groups[id]
	if !ok {
		return domain.Group{}, ErrGroupNotFound
	}

	if slug != nil {
		if r.slugInUse(*slug, id) {
			return domain.Group{}, ErrGroupSlugExists
		}
		g.Slug = *slug
	}
	if name != nil {
		g.Name = *name
	}
	if description != nil {
		g.Description = *description
	}
	if isSystem != nil {
		g.IsSystem = *isSystem
	}

	g.UpdatedAt = time.Now().UTC()
	r.groups[id] = g

	return g, nil
}

func (r *GroupMemoryRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.groups[id]; !ok {
		return ErrGroupNotFound
	}

	delete(r.groups, id)
	return nil
}

func (r *GroupMemoryRepository) slugInUse(slug, exceptID string) bool {
	for _, g := range r.groups {
		if g.Slug == slug && g.ID != exceptID {
			return true
		}
	}

	return false
}
