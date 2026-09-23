package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) ListNotificationTopics(ctx context.Context) (*metadatalifecyclev1.ListNotificationTopicsResponse, error) {
	response, err := c.notifications.ListNotificationTopics(ctx, &metadatalifecyclev1.ListNotificationTopicsRequest{})
	if err != nil {
		return nil, c.wrapError(ctx, "list notification topics", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) ListNotificationSubscriptions(ctx context.Context, userID, actorID string) (*metadatalifecyclev1.ListNotificationSubscriptionsResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.notifications.ListNotificationSubscriptions(ctx, &metadatalifecyclev1.ListNotificationSubscriptionsRequest{UserId: userID})
	if err != nil {
		return nil, c.wrapError(ctx, "list notification subscriptions", err)
	}
	return response, nil
}

func (c *MetadataLifecycle) SubscribeNotification(ctx context.Context, userID, groupID, topicKey, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.notifications.SubscribeNotification(ctx, &metadatalifecyclev1.SubscribeNotificationRequest{
		UserId:   userID,
		GroupId:  groupID,
		TopicKey: topicKey,
		Channel:  metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL,
	})
	if err != nil {
		return c.wrapError(ctx, "subscribe notification", err)
	}
	return nil
}

func (c *MetadataLifecycle) UnsubscribeNotification(ctx context.Context, userID, groupID, topicKey, actorID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	_, err := c.notifications.UnsubscribeNotification(ctx, &metadatalifecyclev1.UnsubscribeNotificationRequest{
		UserId:   userID,
		GroupId:  groupID,
		TopicKey: topicKey,
		Channel:  metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL,
	})
	if err != nil {
		return c.wrapError(ctx, "unsubscribe notification", err)
	}
	return nil
}

func fromProtoNotification(value *metadatalifecyclev1.Notification) (domain.Notification, error) {
	if value == nil {
		return domain.Notification{}, fmt.Errorf("notification is nil")
	}

	eventKind := value.GetEventKind()
	switch eventKind {
	case metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_REQUESTED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_ACCEPTED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_MEMBERSHIP_REJECTED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_REQUESTED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_ACCEPTED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_REQUEST_REJECTED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_ROLE_ASSIGNED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_PERMISSION_ASSIGNED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_SUBMITTED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_PUBLISHED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_REJECTED,
		metadatalifecyclev1.NotificationEventKind_NOTIFICATION_EVENT_KIND_RESOURCE_UNPUBLISHED:
	default:
		return domain.Notification{}, fmt.Errorf("unsupported notification event kind: %q", eventKind)
	}

	if value.GetChannel() != metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL {
		return domain.Notification{}, fmt.Errorf("unsupported notification channel: %q", value.GetChannel())
	}
	if value.GetTargetEmail() == "" {
		return domain.Notification{}, fmt.Errorf("notification target email is empty")
	}
	if value.GetSubject() == "" {
		return domain.Notification{}, fmt.Errorf("notification subject is empty")
	}
	if value.GetBody() == "" {
		return domain.Notification{}, fmt.Errorf("notification body is empty")
	}

	return domain.Notification{
		EventKind:   eventKind.String(),
		Channel:     domain.NotificationChannelEmail,
		TargetEmail: value.GetTargetEmail(),
		Subject:     value.GetSubject(),
		Body:        value.GetBody(),
	}, nil
}
