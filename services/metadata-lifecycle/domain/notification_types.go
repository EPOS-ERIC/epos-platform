package domain

import "github.com/google/uuid"

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
)

// NotificationTopicKey identifies a subscription and recipient route.
type NotificationTopicKey string

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
	GroupID      *uuid.UUID
	ResourceType ResourceType
	ResourceID   *uuid.UUID
	VersionID    *uuid.UUID
	AccessID     *uuid.UUID
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

// NotificationEvent is the canonical event input to the route builder.
type NotificationEvent struct {
	Kind         NotificationEventKind
	ActorID      Actor
	SubjectID    string
	GroupID      *uuid.UUID
	TargetUserID string
	Role         string
	Permission   string
	AccessID     *uuid.UUID
	Resource     *NotificationResource
}
