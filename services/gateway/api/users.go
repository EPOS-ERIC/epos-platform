package api

import (
	"fmt"
	"strings"
)

type User struct {
	ID             string     `json:"id"`
	AuthIdentifier string     `json:"auth_identifier"`
	Email          string     `json:"email"`
	GivenName      string     `json:"given_name"`
	FamilyName     string     `json:"family_name"`
	Status         UserStatus `json:"status"`
	CreatedAt      string     `json:"created_at,omitempty"`
	UpdatedAt      string     `json:"updated_at,omitempty"`
}

type UserStatus string

const (
	ActiveStatus   UserStatus = "ACTIVE"
	InactiveStatus UserStatus = "INACTIVE"
	BlockedStatus  UserStatus = "BLOCKED"
)

func (s UserStatus) IsValid() bool {
	switch s {
	case ActiveStatus, InactiveStatus, BlockedStatus:
		return true
	default:
		return false
	}
}

func NewUserStatusFromValue(value string) (UserStatus, error) {
	status := UserStatus(strings.ToUpper(strings.TrimSpace(value)))
	if status.IsValid() {
		return status, nil
	}

	return "", fmt.Errorf("invalid value '%s' for UserStatus", value)
}
