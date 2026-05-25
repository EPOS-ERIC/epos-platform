package repository

import (
	"errors"
	"sync"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/domain"
)

var (
	ErrFriendshipNotFound = errors.New("friendship not found")
	ErrFriendshipInvalid  = errors.New("invalid friendship")
)

type FriendshipRepository interface {
	Create(userID, friendID string) (domain.Friendship, error)
	Delete(userID, friendID string) error
	AreFriends(userID, friendID string) (bool, error)
}

type friendshipKey struct {
	userID   string
	friendID string
}

type FriendshipMemoryRepository struct {
	mu          sync.RWMutex
	friendships map[friendshipKey]domain.Friendship
}

func NewFriendshipMemoryRepository() *FriendshipMemoryRepository {
	return &FriendshipMemoryRepository{friendships: map[friendshipKey]domain.Friendship{}}
}

func (r *FriendshipMemoryRepository) Create(userID, friendID string) (domain.Friendship, error) {
	if userID == friendID {
		return domain.Friendship{}, ErrFriendshipInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	r.friendships[friendshipKey{userID: userID, friendID: friendID}] = domain.Friendship{UserID: userID, FriendID: friendID, CreatedAt: now}
	r.friendships[friendshipKey{userID: friendID, friendID: userID}] = domain.Friendship{UserID: friendID, FriendID: userID, CreatedAt: now}

	return domain.Friendship{UserID: userID, FriendID: friendID, CreatedAt: now}, nil
}

func (r *FriendshipMemoryRepository) Delete(userID, friendID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := friendshipKey{userID: userID, friendID: friendID}
	if _, ok := r.friendships[key]; !ok {
		return ErrFriendshipNotFound
	}

	delete(r.friendships, key)
	delete(r.friendships, friendshipKey{userID: friendID, friendID: userID})
	return nil
}

func (r *FriendshipMemoryRepository) AreFriends(userID, friendID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.friendships[friendshipKey{userID: userID, friendID: friendID}]
	return ok, nil
}
