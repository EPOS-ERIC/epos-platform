package repository

import (
	"database/sql"
	"errors"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/domain"
)

type FriendshipPostgresRepository struct {
	db *sql.DB
}

func NewFriendshipPostgresRepository(db *sql.DB) *FriendshipPostgresRepository {
	return &FriendshipPostgresRepository{db: db}
}

func (r *FriendshipPostgresRepository) Create(userID, friendID string) (domain.Friendship, error) {
	if userID == friendID {
		return domain.Friendship{}, ErrFriendshipInvalid
	}

	_, err := r.db.Exec(`
		INSERT INTO friendships (user_id, friend_id)
		VALUES ($1, $2), ($2, $1)
		ON CONFLICT DO NOTHING
	`, userID, friendID)
	if err != nil {
		return domain.Friendship{}, err
	}

	var f domain.Friendship
	err = r.db.QueryRow(`SELECT user_id::text, friend_id::text, created_at FROM friendships WHERE user_id = $1 AND friend_id = $2`, userID, friendID).
		Scan(&f.UserID, &f.FriendID, &f.CreatedAt)
	if err != nil {
		return domain.Friendship{}, err
	}

	return f, nil
}

func (r *FriendshipPostgresRepository) Delete(userID, friendID string) error {
	res, err := r.db.Exec(`DELETE FROM friendships WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`, userID, friendID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrFriendshipNotFound
	}

	return nil
}

func (r *FriendshipPostgresRepository) AreFriends(userID, friendID string) (bool, error) {
	var one int
	err := r.db.QueryRow(`SELECT 1 FROM friendships WHERE user_id = $1 AND friend_id = $2`, userID, friendID).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
