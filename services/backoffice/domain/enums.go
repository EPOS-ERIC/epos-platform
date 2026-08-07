package domain

type DatasetStatus string

const (
	DatasetStatusPublished DatasetStatus = "published"
	DatasetStatusDraft     DatasetStatus = "draft"
	DatasetStatusSubmitted DatasetStatus = "submitted"
	DatasetStatusArchived  DatasetStatus = "archived"
	DatasetStatusRejected  DatasetStatus = "rejected"
)

func (s DatasetStatus) Valid() bool {
	switch s {
	case DatasetStatusPublished, DatasetStatusDraft, DatasetStatusSubmitted, DatasetStatusArchived, DatasetStatusRejected:
		return true
	default:
		return false
	}
}

type ParameterKind string

const (
	ParameterKindString   ParameterKind = "string"
	ParameterKindBoolean  ParameterKind = "boolean"
	ParameterKindInteger  ParameterKind = "integer"
	ParameterKindFloat    ParameterKind = "float"
	ParameterKindDateTime ParameterKind = "datetime"
)

func (k ParameterKind) Valid() bool {
	switch k {
	case ParameterKindString, ParameterKindBoolean, ParameterKindInteger, ParameterKindFloat, ParameterKindDateTime:
		return true
	default:
		return false
	}
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBlocked  UserStatus = "blocked"
)

func (s UserStatus) Valid() bool {
	switch s {
	case UserStatusActive, UserStatusInactive, UserStatusBlocked:
		return true
	default:
		return false
	}
}

type MembershipRole string

const (
	MembershipRoleViewer   MembershipRole = "viewer"
	MembershipRoleEditor   MembershipRole = "editor"
	MembershipRoleReviewer MembershipRole = "reviewer"
	MembershipRoleAdmin    MembershipRole = "admin"
)

func (r MembershipRole) Valid() bool {
	switch r {
	case MembershipRoleViewer, MembershipRoleEditor, MembershipRoleReviewer, MembershipRoleAdmin:
		return true
	default:
		return false
	}
}

type MembershipStatus string

const (
	MembershipStatusPending  MembershipStatus = "pending"
	MembershipStatusActive   MembershipStatus = "active"
	MembershipStatusRevoked  MembershipStatus = "revoked"
	MembershipStatusRejected MembershipStatus = "rejected"
)

func (s MembershipStatus) Valid() bool {
	switch s {
	case MembershipStatusPending, MembershipStatusActive, MembershipStatusRevoked, MembershipStatusRejected:
		return true
	default:
		return false
	}
}

type PermissionEffect string

const (
	PermissionEffectAllow PermissionEffect = "allow"
	PermissionEffectDeny  PermissionEffect = "deny"
)

func (e PermissionEffect) Valid() bool {
	switch e {
	case PermissionEffectAllow, PermissionEffectDeny:
		return true
	default:
		return false
	}
}

type IdentifierType string

type ContactPointRole string

type TopLevelConcept string

const (
	TopLevelConceptData       TopLevelConcept = "data"
	TopLevelConceptFacilities TopLevelConcept = "facilities"
	TopLevelConceptSoftware   TopLevelConcept = "software"
)

func (c TopLevelConcept) Valid() bool {
	switch c {
	case TopLevelConceptData, TopLevelConceptFacilities, TopLevelConceptSoftware:
		return true
	default:
		return false
	}
}
