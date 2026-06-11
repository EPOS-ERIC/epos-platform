package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/db"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/db/sqlc"
	"github.com/google/uuid"
)

var friendshipsLog = logging.Get("friendships-service")

type Friendship struct {
	UserID    string    `json:"userId"`
	FriendID  string    `json:"friendId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Friendships struct {
	db *db.PostgresDB
}

func NewFriendships(database *db.PostgresDB) *Friendships {
	return &Friendships{
		db: database,
	}
}

func (s *Friendships) Create(ctx context.Context, userID, friendID uuid.UUID) (Friendship, error) {
	if userID == friendID {
		return Friendship{}, ErrInvalid
	}

	if _, err := s.db.GetFriendship(ctx, sqlc.GetFriendshipParams{UserID: userID, FriendID: friendID}); err == nil {
		return Friendship{}, ErrConflict
	}

	if _, err := s.db.GetUserByID(ctx, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Friendship{}, ErrNotFound
		}
		friendshipsLog.ErrorContext(ctx, "get user by id failed", "user_id", userID, "error", err)
		return Friendship{}, fmt.Errorf("get user by id: %w", err)
	}

	if _, err := s.db.GetUserByID(ctx, friendID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Friendship{}, ErrNotFound
		}
		friendshipsLog.ErrorContext(ctx, "get friend by id failed", "friend_user_id", friendID, "error", err)
		return Friendship{}, fmt.Errorf("get friend by id: %w", err)
	}

	if err := s.db.CreateFriendship(ctx, sqlc.CreateFriendshipParams{
		UserID:     userID,
		FriendID:   friendID,
		UserID_2:   friendID,
		FriendID_2: userID,
	}); err != nil {
		friendshipsLog.ErrorContext(ctx, "create friendship failed", "user_id", userID, "friend_user_id", friendID, "error", err)
		return Friendship{}, fmt.Errorf("create friendship: %w", err)
	}

	row, err := s.db.GetFriendship(ctx, sqlc.GetFriendshipParams{UserID: userID, FriendID: friendID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Friendship{}, ErrNotFound
		}
		friendshipsLog.ErrorContext(ctx, "get friendship failed", "user_id", userID, "friend_user_id", friendID, "error", err)
		return Friendship{}, fmt.Errorf("get friendship: %w", err)
	}

	return Friendship{
		UserID:    row.UserID.String(),
		FriendID:  row.FriendID.String(),
		CreatedAt: row.CreatedAt,
	}, nil
}

func (s *Friendships) Delete(ctx context.Context, userID, friendID uuid.UUID) error {
	if _, err := s.db.GetFriendship(ctx, sqlc.GetFriendshipParams{UserID: userID, FriendID: friendID}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		friendshipsLog.ErrorContext(ctx, "get friendship failed", "user_id", userID, "friend_user_id", friendID, "error", err)
		return fmt.Errorf("get friendship: %w", err)
	}

	if err := s.db.DeleteFriendship(ctx, sqlc.DeleteFriendshipParams{
		UserID:     userID,
		FriendID:   friendID,
		UserID_2:   friendID,
		FriendID_2: userID,
	}); err != nil {
		friendshipsLog.ErrorContext(ctx, "delete friendship failed", "user_id", userID, "friend_user_id", friendID, "error", err)
		return fmt.Errorf("delete friendship: %w", err)
	}

	return nil
}

func (s *Friendships) AreFriends(ctx context.Context, userID, friendID uuid.UUID) (bool, error) {
	result, err := s.db.AreFriends(ctx, sqlc.AreFriendsParams{UserID: userID, FriendID: friendID})
	if err != nil {
		friendshipsLog.ErrorContext(ctx, "check friendship failed", "user_id", userID, "friend_user_id", friendID, "error", err)
		return false, fmt.Errorf("check friendship: %w", err)
	}

	return result, nil
}
