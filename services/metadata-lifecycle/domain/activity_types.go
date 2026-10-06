package domain

import (
	"time"

	"github.com/google/uuid"
)

type ActivityAction string

const (
	ActivityActionResourceCreated        ActivityAction = "RESOURCE_CREATED"
	ActivityActionResourceVersionCreated ActivityAction = "RESOURCE_VERSION_CREATED"
	ActivityActionResourceSubmitted      ActivityAction = "RESOURCE_SUBMITTED"
	ActivityActionResourceRejected       ActivityAction = "RESOURCE_REJECTED"
	ActivityActionResourcePublished      ActivityAction = "RESOURCE_PUBLISHED"
	ActivityActionResourceArchived       ActivityAction = "RESOURCE_ARCHIVED"
	ActivityActionRoleRequested          ActivityAction = "ROLE_REQUESTED"
	ActivityActionRoleRequestAccepted    ActivityAction = "ROLE_REQUEST_ACCEPTED"
	ActivityActionRoleRequestRejected    ActivityAction = "ROLE_REQUEST_REJECTED"
	ActivityActionRoleAssigned           ActivityAction = "ROLE_ASSIGNED"
	ActivityActionPermissionGranted      ActivityAction = "PERMISSION_GRANTED"
	ActivityActionUserRegistered         ActivityAction = "USER_REGISTERED"
	ActivityActionUserBlocked            ActivityAction = "USER_BLOCKED"
	ActivityActionGroupCreated           ActivityAction = "GROUP_CREATED"
	ActivityActionCategoryCreated        ActivityAction = "CATEGORY_CREATED"
	ActivityActionMembershipCreated      ActivityAction = "MEMBERSHIP_CREATED"
)

func (a ActivityAction) Valid() bool {
	switch a {
	case ActivityActionResourceCreated,
		ActivityActionResourceVersionCreated,
		ActivityActionResourceSubmitted,
		ActivityActionResourceRejected,
		ActivityActionResourcePublished,
		ActivityActionResourceArchived,
		ActivityActionRoleRequested,
		ActivityActionRoleRequestAccepted,
		ActivityActionRoleRequestRejected,
		ActivityActionRoleAssigned,
		ActivityActionPermissionGranted,
		ActivityActionUserRegistered,
		ActivityActionUserBlocked,
		ActivityActionGroupCreated,
		ActivityActionCategoryCreated,
		ActivityActionMembershipCreated:
		return true
	default:
		return false
	}
}

type ActivityFilter struct {
	GroupID  *uuid.UUID
	DateFrom *time.Time
	DateTo   *time.Time
}

type ActivityEvent struct {
	Action       ActivityAction
	OccurredAt   time.Time
	ActorID      *string
	GroupID      *uuid.UUID
	ResourceType *ResourceType
	ResourceID   *uuid.UUID
	VersionID    *uuid.UUID
	TargetUserID *string
	Role         *string
	Permission   *string
	Note         *string
	Reason       *string
}
