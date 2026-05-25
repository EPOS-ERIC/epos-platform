package domain

import "time"

type Friendship struct {
	UserID    string    `json:"userId"`
	FriendID  string    `json:"friendId"`
	CreatedAt time.Time `json:"createdAt"`
}
