package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NotificationGroupContext contains the labels needed when rendering notifications.
type NotificationGroupContext struct {
	Name    string
	Concept string
}

// NotificationChannel identifies a delivery channel.
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "EMAIL"
)

// NotificationScope identifies the audience route for a topic.
type NotificationScope string

const (
	NotificationScopeDirect NotificationScope = "DIRECT"
	NotificationScopeGlobal NotificationScope = "GLOBAL"
	NotificationScopeGroup  NotificationScope = "GROUP"
)

// NotificationEventKind identifies a supported notification event.
type NotificationEventKind string

const (
	NotificationEventMembershipRequested NotificationEventKind = "membership.requested"
	NotificationEventMembershipAccepted  NotificationEventKind = "membership.accepted"
	NotificationEventMembershipRejected  NotificationEventKind = "membership.rejected"
	NotificationEventRoleRequested       NotificationEventKind = "role_request.requested"
	NotificationEventRoleRequestAccepted NotificationEventKind = "role_request.accepted"
	NotificationEventRoleRequestRejected NotificationEventKind = "role_request.rejected"
	NotificationEventRoleAssigned        NotificationEventKind = "role.assigned"
	NotificationEventPermissionAssigned  NotificationEventKind = "permission.assigned"
	NotificationEventResourceSubmitted   NotificationEventKind = "resource.submitted"
	NotificationEventResourcePublished   NotificationEventKind = "resource.published"
	NotificationEventResourceRejected    NotificationEventKind = "resource.rejected"
	NotificationEventResourceUnpublished NotificationEventKind = "resource.unpublished"
	NotificationEventConversionRequested NotificationEventKind = "conversion.requested"
)

// NotificationTopicKey identifies a subscription and recipient route.
type NotificationTopicKey string

// ParseNotificationTopicKey validates and normalizes a notification topic key.
func ParseNotificationTopicKey(value string) (NotificationTopicKey, error) {
	key := NotificationTopicKey(strings.TrimSpace(value))
	switch key {
	case NotificationTopicDirectMembershipAccepted,
		NotificationTopicDirectMembershipRejected,
		NotificationTopicDirectRoleRequestAccepted,
		NotificationTopicDirectRoleRequestRejected,
		NotificationTopicDirectRoleAssigned,
		NotificationTopicDirectPermissionAssigned,
		NotificationTopicDirectResourcePublished,
		NotificationTopicDirectResourceRejected,
		NotificationTopicDirectResourceUnpublished,
		NotificationTopicGroupMembershipRequested,
		NotificationTopicGlobalMembershipRequested,
		NotificationTopicGroupRoleRequestRequested,
		NotificationTopicGlobalRoleRequestRequested,
		NotificationTopicGroupResourceSubmitted,
		NotificationTopicGroupResourcePublished,
		NotificationTopicGlobalResourceSubmitted,
		NotificationTopicGlobalResourcePublished,
		NotificationTopicGlobalConversionRequested:
		return key, nil
	default:
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
}

const (
	NotificationTopicDirectMembershipAccepted   NotificationTopicKey = "direct.membership.accepted"
	NotificationTopicDirectMembershipRejected   NotificationTopicKey = "direct.membership.rejected"
	NotificationTopicDirectRoleRequestAccepted  NotificationTopicKey = "direct.role_request.accepted"
	NotificationTopicDirectRoleRequestRejected  NotificationTopicKey = "direct.role_request.rejected"
	NotificationTopicDirectRoleAssigned         NotificationTopicKey = "direct.role.assigned"
	NotificationTopicDirectPermissionAssigned   NotificationTopicKey = "direct.permission.assigned"
	NotificationTopicDirectResourcePublished    NotificationTopicKey = "direct.resource.published"
	NotificationTopicDirectResourceRejected     NotificationTopicKey = "direct.resource.rejected"
	NotificationTopicDirectResourceUnpublished  NotificationTopicKey = "direct.resource.unpublished"
	NotificationTopicGroupMembershipRequested   NotificationTopicKey = "group.membership.requested"
	NotificationTopicGlobalMembershipRequested  NotificationTopicKey = "global.membership.requested"
	NotificationTopicGroupRoleRequestRequested  NotificationTopicKey = "group.role_request.requested"
	NotificationTopicGlobalRoleRequestRequested NotificationTopicKey = "global.role_request.requested"
	NotificationTopicGroupResourceSubmitted     NotificationTopicKey = "group.resource.submitted"
	NotificationTopicGroupResourcePublished     NotificationTopicKey = "group.resource.published"
	NotificationTopicGlobalResourceSubmitted    NotificationTopicKey = "global.resource.submitted"
	NotificationTopicGlobalResourcePublished    NotificationTopicKey = "global.resource.published"
	NotificationTopicGlobalConversionRequested  NotificationTopicKey = "global.conversion.requested"
)

// NotificationTopic is fixed catalog metadata used for routing and eligibility.
type NotificationTopic struct {
	Key                 NotificationTopicKey
	Label               string
	Description         string
	Scope               NotificationScope
	RequiredPermissions []string
}

// NotificationSubscription is a user's explicit topic subscription.
type NotificationSubscription struct {
	UserID   string
	GroupID  *uuid.UUID
	TopicKey NotificationTopicKey
	Channel  NotificationChannel
}

// NotificationRecipient contains the delivery identity used by the builder.
type NotificationRecipient struct {
	ID     string
	Email  string
	Status UserStatus
}

// Notification contains a complete email for Gateway delivery.
type Notification struct {
	EventKind    NotificationEventKind
	TargetUserID string
	TargetEmail  string
	Subject      string
	Body         string
	Channel      NotificationChannel
}

// NotificationResource identifies a resource involved in an event.
type NotificationResource struct {
	Type          ResourceType
	ResourceID    uuid.UUID
	VersionID     uuid.UUID
	Revision      int32
	Status        Status
	Name          string
	SubmittedBy   *string
	RejectionNote *string
}

// NotificationEvent is an event accepted by the notification builder.
type NotificationEvent interface {
	NotificationKind() NotificationEventKind
}

// NotificationEventCommon contains data shared by all notification events.
type NotificationEventCommon struct {
	ActorID Actor
}

type MembershipRequested struct {
	NotificationEventCommon
	SubjectID    string
	GroupID      uuid.UUID
	GroupConcept Concept
	TargetUserID string
}

func (MembershipRequested) NotificationKind() NotificationEventKind {
	return NotificationEventMembershipRequested
}

type MembershipAccepted struct {
	NotificationEventCommon
	SubjectID    string
	GroupID      uuid.UUID
	TargetUserID string
}

func (MembershipAccepted) NotificationKind() NotificationEventKind {
	return NotificationEventMembershipAccepted
}

type MembershipRejectedEvent struct {
	NotificationEventCommon
	SubjectID    string
	GroupID      uuid.UUID
	TargetUserID string
}

func (MembershipRejectedEvent) NotificationKind() NotificationEventKind {
	return NotificationEventMembershipRejected
}

type RoleRequested struct {
	NotificationEventCommon
	SubjectID    string
	GroupID      uuid.UUID
	GroupConcept Concept
	TargetUserID string
	Role         string
	Permission   string
	AccessID     *uuid.UUID
}

func (RoleRequested) NotificationKind() NotificationEventKind {
	return NotificationEventRoleRequested
}

type RoleRequestAcceptedEvent struct {
	NotificationEventCommon
	SubjectID    string
	GroupID      uuid.UUID
	TargetUserID string
	Role         string
	Permission   string
	AccessID     uuid.UUID
}

func (RoleRequestAcceptedEvent) NotificationKind() NotificationEventKind {
	return NotificationEventRoleRequestAccepted
}

type RoleRequestRejectedEvent struct {
	NotificationEventCommon
	SubjectID    string
	GroupID      uuid.UUID
	TargetUserID string
	Role         string
	Permission   string
	AccessID     uuid.UUID
}

func (RoleRequestRejectedEvent) NotificationKind() NotificationEventKind {
	return NotificationEventRoleRequestRejected
}

type RoleAssigned struct {
	NotificationEventCommon
	SubjectID    string
	GroupID      uuid.UUID
	TargetUserID string
	Role         string
	Permission   string
	AccessID     uuid.UUID
}

func (RoleAssigned) NotificationKind() NotificationEventKind {
	return NotificationEventRoleAssigned
}

type PermissionAssigned struct {
	NotificationEventCommon
	SubjectID    string
	GroupID      uuid.UUID
	TargetUserID string
	Role         string
	Permission   string
	AccessID     uuid.UUID
}

func (PermissionAssigned) NotificationKind() NotificationEventKind {
	return NotificationEventPermissionAssigned
}

type ResourceSubmitted struct {
	NotificationEventCommon
	GroupID  uuid.UUID
	Resource NotificationResource
}

func (ResourceSubmitted) NotificationKind() NotificationEventKind {
	return NotificationEventResourceSubmitted
}

type ResourcePublished struct {
	NotificationEventCommon
	GroupID  uuid.UUID
	Resource NotificationResource
}

func (ResourcePublished) NotificationKind() NotificationEventKind {
	return NotificationEventResourcePublished
}

type ResourceRejected struct {
	NotificationEventCommon
	GroupID  uuid.UUID
	Resource NotificationResource
}

func (ResourceRejected) NotificationKind() NotificationEventKind {
	return NotificationEventResourceRejected
}

type ResourceUnpublished struct {
	NotificationEventCommon
	GroupID  uuid.UUID
	Resource NotificationResource
}

func (ResourceUnpublished) NotificationKind() NotificationEventKind {
	return NotificationEventResourceUnpublished
}

type ConversionRequested struct {
	NotificationEventCommon
	Resource          NotificationResource
	DistributionID    uuid.UUID
	DistributionTitle string
	WebServiceType    WebServiceType
	Input             ConversionRequestInput
	DefaultURL        string
}

func (ConversionRequested) NotificationKind() NotificationEventKind {
	return NotificationEventConversionRequested
}
