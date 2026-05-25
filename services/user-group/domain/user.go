package domain

import "time"

type User struct {
	ID             string    `json:"id"`
	AuthIdentifier string    `json:"authIdentifier"`
	Email          string    `json:"email,omitempty"`
	GivenName      string    `json:"givenName,omitempty"`
	FamilyName     string    `json:"familyName,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
