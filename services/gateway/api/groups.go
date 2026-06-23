package api

type Group struct {
	ID          string                   `json:"id"`
	Slug        string                   `json:"slug"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	IsSystem    bool                     `json:"is_system"`
	Membership  *ViewerMembershipSummary `json:"membership,omitempty"`
}

type GroupSummary struct {
	ID          string                   `json:"id"`
	Slug        string                   `json:"slug"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	IsSystem    bool                     `json:"is_system"`
	Membership  *ViewerMembershipSummary `json:"membership,omitempty"`
}

type ViewerMembershipSummary struct {
	Role   MembershipRole   `json:"role"`
	Status MembershipStatus `json:"status"`
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

type CreateGroupRequest struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsSystem    *bool   `json:"is_system,omitempty"`
}

type CreateMembershipRequest struct {
	Role MembershipRole `json:"role"`
}

type PatchMembershipRequest struct {
	Role   *MembershipRole   `json:"role,omitempty"`
	Status *MembershipStatus `json:"status,omitempty"`
}

func (r PatchMembershipRequest) IsEmpty() bool {
	return r.Role == nil && r.Status == nil
}

type PatchGroupRequest struct {
	Slug        *string `json:"slug,omitempty"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsSystem    *bool   `json:"is_system,omitempty"`
}

func (r PatchGroupRequest) IsEmpty() bool {
	return r.Slug == nil && r.Name == nil && r.Description == nil && r.IsSystem == nil
}

func (r MembershipRole) IsValid() bool {
	switch r {
	case MembershipViewerRole, MembershipEditorRole, MembershipReviewerRole, MembershipAdminRole:
		return true
	default:
		return false
	}
}

func (s MembershipStatus) IsValid() bool {
	switch s {
	case MembershipPendingStatus, MembershipActiveStatus, MembershipRevokedStatus, MembershipRejectedStatus:
		return true
	default:
		return false
	}
}
