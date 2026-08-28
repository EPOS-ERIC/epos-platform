package access

import (
	"time"

	"github.com/google/uuid"
)

type membershipRequest struct {
	Roles []string `json:"roles"`
}

type membershipDecisionRequest struct {
	Status string `json:"status"`
}

type roleRequest struct {
	Role string `json:"role"`
}

type roleRequestDecisionRequest struct {
	Status string `json:"status"`
}

type membershipResponse struct {
	User         membershipUserResponse `json:"user"`
	Status       string                 `json:"status"`
	Roles        []string               `json:"roles"`
	Permissions  []string               `json:"permissions"`
	RoleRequests []roleRequestResponse  `json:"role_requests"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type membershipUserResponse struct {
	ID         string `json:"id"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Email      string `json:"email"`
}

type roleRequestResponse struct {
	ID          uuid.UUID `json:"id"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requested_at"`
}
