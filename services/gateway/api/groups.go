package api

type Group struct {
	ID          string  `json:"id"`
	SourceIRI   string  `json:"source_iri"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Label       *string `json:"label"`
	Logo        *string `json:"logo"`
	Page        *string `json:"page"`
	Color       *string `json:"color"`
	Position    int32   `json:"position"`
	Concept     string  `json:"concept"`
}

type Membership struct {
	UserID    string           `json:"user_id"`
	GroupID   string           `json:"group_id"`
	Role      MembershipRole   `json:"role"`
	Status    MembershipStatus `json:"status"`
	CreatedAt string           `json:"created_at,omitempty"`
	UpdatedAt string           `json:"updated_at,omitempty"`
}

type GroupMembershipSummary struct {
	User   ResourceRef      `json:"user"`
	Role   MembershipRole   `json:"role"`
	Status MembershipStatus `json:"status"`
}

type GroupMembership struct {
	User         MembershipUser   `json:"user"`
	Status       MembershipStatus `json:"status"`
	Roles        []MembershipRole `json:"roles"`
	Permissions  []string         `json:"permissions"`
	RoleRequests []RoleRequest    `json:"role_requests"`
	CreatedAt    string           `json:"created_at"`
	UpdatedAt    string           `json:"updated_at"`
}

type MembershipUser struct {
	ID         string `json:"id"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Email      string `json:"email"`
}

type RoleRequest struct {
	ID          string `json:"id"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	RequestedAt string `json:"requested_at"`
}

type CreateRoleRequest struct {
	Role string `json:"role"`
}

type MembershipDecisionRequest struct {
	Status MembershipStatus `json:"status"`
}

type CreateGroupRequest struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsSystem    *bool   `json:"is_system,omitempty"`
}

type CreateMembershipRequest struct {
	Role MembershipRole `json:"role"`
}

func (r MembershipRole) IsValid() bool {
	switch r {
	case MembershipEditorRole, MembershipReviewerRole, MembershipGroupAdminRole:
		return true
	default:
		return false
	}
}

func (s MembershipStatus) IsValid() bool {
	switch s {
	case MembershipPendingStatus, MembershipActiveStatus, MembershipRejectedStatus:
		return true
	default:
		return false
	}
}
