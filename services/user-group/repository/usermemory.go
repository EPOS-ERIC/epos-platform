package repository

import (
	"errors"
	"sync"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/domain"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserAuthIDExists     = errors.New("user auth identifier already exists")
	ErrUserInvalidStatus    = errors.New("invalid user status")
	allowedUserStatusValues = map[string]struct{}{"ACTIVE": {}, "INACTIVE": {}, "BLOCKED": {}}
)

type UserRepository interface {
	List() []domain.User
	GetByID(id string) (domain.User, error)
	GetByAuthIdentifier(authIdentifier string) (domain.User, error)
	Create(authIdentifier, email, givenName, familyName, status string) (domain.User, error)
	Replace(id, authIdentifier, email, givenName, familyName, status string) (domain.User, error)
	Patch(id string, authIdentifier, email, givenName, familyName, status *string) (domain.User, error)
	Delete(id string) error
}

type UserMemoryRepository struct {
	mu    sync.RWMutex
	users map[string]domain.User
}

func NewUserMemoryRepository() *UserMemoryRepository {
	return &UserMemoryRepository{users: map[string]domain.User{}}
}

func (r *UserMemoryRepository) List() []domain.User {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.User, 0, len(r.users))
	for _, u := range r.users {
		result = append(result, u)
	}

	return result
}

func (r *UserMemoryRepository) GetByID(id string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.users[id]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}

	return u, nil
}

func (r *UserMemoryRepository) GetByAuthIdentifier(authIdentifier string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.users {
		if u.AuthIdentifier == authIdentifier {
			return u, nil
		}
	}

	return domain.User{}, ErrUserNotFound
}

func (r *UserMemoryRepository) Create(authIdentifier, email, givenName, familyName, status string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.authIdentifierInUse(authIdentifier, "") {
		return domain.User{}, ErrUserAuthIDExists
	}
	if !isAllowedUserStatus(status) {
		return domain.User{}, ErrUserInvalidStatus
	}

	now := time.Now().UTC()
	u := domain.User{
		ID:             newUUIDLike(),
		AuthIdentifier: authIdentifier,
		Email:          email,
		GivenName:      givenName,
		FamilyName:     familyName,
		Status:         status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	r.users[u.ID] = u

	return u, nil
}

func (r *UserMemoryRepository) Replace(id, authIdentifier, email, givenName, familyName, status string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.users[id]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}
	if r.authIdentifierInUse(authIdentifier, id) {
		return domain.User{}, ErrUserAuthIDExists
	}
	if !isAllowedUserStatus(status) {
		return domain.User{}, ErrUserInvalidStatus
	}

	u.AuthIdentifier = authIdentifier
	u.Email = email
	u.GivenName = givenName
	u.FamilyName = familyName
	u.Status = status
	u.UpdatedAt = time.Now().UTC()
	r.users[id] = u

	return u, nil
}

func (r *UserMemoryRepository) Patch(id string, authIdentifier, email, givenName, familyName, status *string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.users[id]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}

	if authIdentifier != nil {
		if r.authIdentifierInUse(*authIdentifier, id) {
			return domain.User{}, ErrUserAuthIDExists
		}
		u.AuthIdentifier = *authIdentifier
	}
	if email != nil {
		u.Email = *email
	}
	if givenName != nil {
		u.GivenName = *givenName
	}
	if familyName != nil {
		u.FamilyName = *familyName
	}
	if status != nil {
		if !isAllowedUserStatus(*status) {
			return domain.User{}, ErrUserInvalidStatus
		}
		u.Status = *status
	}

	u.UpdatedAt = time.Now().UTC()
	r.users[id] = u

	return u, nil
}

func (r *UserMemoryRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[id]; !ok {
		return ErrUserNotFound
	}

	delete(r.users, id)
	return nil
}

func (r *UserMemoryRepository) authIdentifierInUse(authIdentifier, exceptID string) bool {
	for _, u := range r.users {
		if u.AuthIdentifier == authIdentifier && u.ID != exceptID {
			return true
		}
	}

	return false
}

func isAllowedUserStatus(status string) bool {
	_, ok := allowedUserStatusValues[status]
	return ok
}
