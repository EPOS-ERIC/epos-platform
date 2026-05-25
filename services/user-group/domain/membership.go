package domain

import "time"

type Membership struct {
	UserID    string    `json:"userId"`
	GroupID   string    `json:"groupId"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
