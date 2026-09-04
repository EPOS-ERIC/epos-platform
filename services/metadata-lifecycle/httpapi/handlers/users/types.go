package users

import "github.com/google/uuid"

type internalUserRequest struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

type userResponse struct {
	ID                string                   `json:"id"`
	Email             string                   `json:"email"`
	GivenName         string                   `json:"given_name"`
	FamilyName        string                   `json:"family_name"`
	Status            string                   `json:"status"`
	GlobalRoles       []string                 `json:"global_roles"`
	GlobalPermissions []string                 `json:"global_permissions"`
	CreatedAt         string                   `json:"created_at"`
	UpdatedAt         string                   `json:"updated_at"`
	Memberships       []userMembershipResponse `json:"memberships"`
}

type userMembershipResponse struct {
	Group        groupResponse         `json:"group"`
	Roles        []string              `json:"roles"`
	Status       string                `json:"status"`
	Permissions  []string              `json:"permissions"`
	RoleRequests []roleRequestResponse `json:"role_requests"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
}

type roleRequestResponse struct {
	ID          uuid.UUID `json:"id"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	RequestedAt string    `json:"requested_at"`
}

type groupResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
