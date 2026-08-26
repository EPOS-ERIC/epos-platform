package domain

type User struct {
	ID                string
	Email             string
	GivenName         string
	FamilyName        string
	Status            string
	GlobalRoles       []string
	GlobalPermissions []string
	CreatedAt         string
	UpdatedAt         string
	Memberships       []Membership
}

type UserSummary struct {
	ID                string
	Email             string
	GivenName         string
	FamilyName        string
	Status            string
	GlobalRoles       []string
	GlobalPermissions []string
	CreatedAt         string
	UpdatedAt         string
}

type Membership struct {
	Group        Group
	Roles        []string
	Status       string
	Permissions  []string
	RoleRequests []RoleRequest
	CreatedAt    string
	UpdatedAt    string
}
