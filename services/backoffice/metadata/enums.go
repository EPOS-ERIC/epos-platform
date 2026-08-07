package metadata

type DatasetStatus string

const (
	DatasetStatusPublished DatasetStatus = "PUBLISHED"
	DatasetStatusDraft     DatasetStatus = "DRAFT"
	DatasetStatusSubmitted DatasetStatus = "SUBMITTED"
	DatasetStatusArchived  DatasetStatus = "ARCHIVED"
	DatasetStatusRejected  DatasetStatus = "REJECTED"
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
	ParameterKindString   ParameterKind = "STRING"
	ParameterKindBoolean  ParameterKind = "BOOLEAN"
	ParameterKindInteger  ParameterKind = "INTEGER"
	ParameterKindFloat    ParameterKind = "FLOAT"
	ParameterKindDateTime ParameterKind = "DATETIME"
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
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
	UserStatusBlocked  UserStatus = "BLOCKED"
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
	MembershipRoleViewer   MembershipRole = "VIEWER"
	MembershipRoleEditor   MembershipRole = "EDITOR"
	MembershipRoleReviewer MembershipRole = "REVIEWER"
	MembershipRoleAdmin    MembershipRole = "ADMIN"
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
	MembershipStatusPending  MembershipStatus = "PENDING"
	MembershipStatusActive   MembershipStatus = "ACTIVE"
	MembershipStatusRevoked  MembershipStatus = "REVOKED"
	MembershipStatusRejected MembershipStatus = "REJECTED"
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
	PermissionEffectAllow PermissionEffect = "ALLOW"
	PermissionEffectDeny  PermissionEffect = "DENY"
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
	TopLevelConceptData       TopLevelConcept = "DATA"
	TopLevelConceptFacilities TopLevelConcept = "FACILITIES"
	TopLevelConceptSoftware   TopLevelConcept = "SOFTWARE"
)

func (c TopLevelConcept) Valid() bool {
	switch c {
	case TopLevelConceptData, TopLevelConceptFacilities, TopLevelConceptSoftware:
		return true
	default:
		return false
	}
}
