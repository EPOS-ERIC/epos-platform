package grpcapi

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toProtoNotification(value domain.Notification) (*metadatalifecyclev1.Notification, error) {
	eventKind, err := toProtoNotificationEventKind(value.EventKind)
	if err != nil {
		return nil, err
	}

	channel, err := toProtoNotificationChannel(value.Channel)
	if err != nil {
		return nil, err
	}

	result := &metadatalifecyclev1.Notification{
		EventKind:    eventKind,
		TargetUserId: value.TargetUserID,
		TargetEmail:  value.TargetEmail,
		Subject:      value.Subject,
		Body:         value.Body,
		Channel:      channel,
	}
	if value.GroupID != nil {
		result.GroupId = new(value.GroupID.String())
	}
	if value.ResourceID != nil {
		result.ResourceId = new(value.ResourceID.String())
	}
	if value.VersionID != nil {
		result.VersionId = new(value.VersionID.String())
	}
	if value.AccessID != nil {
		result.AccessId = new(value.AccessID.String())
	}
	if value.ResourceType != "" {
		result.ResourceType = toGRPCResourceType(value.ResourceType)
		if result.ResourceType == metadatalifecyclev1.ResourceType_RESOURCE_TYPE_UNSPECIFIED {
			return nil, fmt.Errorf("unknown resource type: %q", value.ResourceType)
		}
	}
	return result, nil
}

//nolint:dupl // Notification enum conversion intentionally uses explicit one-to-one mappings.
func toProtoNotificationEventKind(value domain.NotificationEventKind) (metadatalifecyclev1.NotificationEventKind, error) {
	switch value {
	case domain.NotificationEventMembershipRequested:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_REQUESTED, nil
	case domain.NotificationEventMembershipAccepted:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED, nil
	case domain.NotificationEventMembershipRejected:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_REJECTED, nil
	case domain.NotificationEventRoleRequested:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_REQUESTED, nil
	case domain.NotificationEventRoleRequestAccepted:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_ACCEPTED, nil
	case domain.NotificationEventRoleRequestRejected:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_REJECTED, nil
	case domain.NotificationEventRoleAssigned:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_ASSIGNED, nil
	case domain.NotificationEventPermissionAssigned:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_PERMISSION_ASSIGNED, nil
	case domain.NotificationEventResourceSubmitted:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_SUBMITTED, nil
	case domain.NotificationEventResourcePublished:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_PUBLISHED, nil
	case domain.NotificationEventResourceRejected:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_REJECTED, nil
	case domain.NotificationEventResourceUnpublished:
		return metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_UNPUBLISHED, nil
	default:
		return 0, fmt.Errorf("unknown notification event kind: %q", value)
	}
}

func toProtoNotificationChannel(value domain.NotificationChannel) (metadatalifecyclev1.NotificationChannel, error) {
	switch value {
	case domain.NotificationChannelEmail:
		return metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL, nil
	default:
		return 0, fmt.Errorf("unknown notification channel: %q", value)
	}
}
