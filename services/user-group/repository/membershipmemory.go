package repository

import (
	"errors"
	"sync"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/domain"
)

var (
	ErrMembershipNotFound      = errors.New("membership not found")
	ErrMembershipInvalidRole   = errors.New("invalid membership role")
	ErrMembershipInvalidStatus = errors.New("invalid membership status")

	allowedMembershipRoles = map[string]struct{}{
		"VIEWER": {}, "EDITOR": {}, "REVIEWER": {}, "ADMIN": {},
	}
	allowedMembershipStatuses = map[string]struct{}{
		"PENDING": {}, "ACTIVE": {}, "REVOKED": {}, "REJECTED": {},
	}
)

type MembershipRepository interface {
	ListByGroupID(groupID string) []domain.Membership
	Upsert(userID, groupID, role, status string) (domain.Membership, error)
	Delete(userID, groupID string) error
	IsActiveMember(userID, groupID string) (bool, error)
}

type membershipKey struct {
	userID  string
	groupID string
}

type MembershipMemoryRepository struct {
	mu          sync.RWMutex
	memberships map[membershipKey]domain.Membership
}

func NewMembershipMemoryRepository() *MembershipMemoryRepository {
	return &MembershipMemoryRepository{memberships: map[membershipKey]domain.Membership{}}
}

func (r *MembershipMemoryRepository) ListByGroupID(groupID string) []domain.Membership {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := []domain.Membership{}
	for _, m := range r.memberships {
		if m.GroupID == groupID {
			result = append(result, m)
		}
	}

	return result
}

func (r *MembershipMemoryRepository) Upsert(userID, groupID, role, status string) (domain.Membership, error) {
	if !isAllowedMembershipRole(role) {
		return domain.Membership{}, ErrMembershipInvalidRole
	}
	if !isAllowedMembershipStatus(status) {
		return domain.Membership{}, ErrMembershipInvalidStatus
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := membershipKey{userID: userID, groupID: groupID}
	now := time.Now().UTC()
	m, ok := r.memberships[key]
	if !ok {
		m = domain.Membership{UserID: userID, GroupID: groupID, CreatedAt: now}
	}
	m.Role = role
	m.Status = status
	m.UpdatedAt = now
	r.memberships[key] = m

	return m, nil
}

func (r *MembershipMemoryRepository) Delete(userID, groupID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := membershipKey{userID: userID, groupID: groupID}
	if _, ok := r.memberships[key]; !ok {
		return ErrMembershipNotFound
	}
	delete(r.memberships, key)

	return nil
}

func (r *MembershipMemoryRepository) IsActiveMember(userID, groupID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.memberships[membershipKey{userID: userID, groupID: groupID}]
	if !ok {
		return false, nil
	}

	return m.Status == "ACTIVE", nil
}

func isAllowedMembershipRole(role string) bool {
	_, ok := allowedMembershipRoles[role]
	return ok
}

func isAllowedMembershipStatus(status string) bool {
	_, ok := allowedMembershipStatuses[status]
	return ok
}
