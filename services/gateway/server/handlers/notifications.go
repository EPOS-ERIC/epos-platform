package handlers

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ListNotificationTopics(ctx context.Context) ([]gatewayapi.NotificationTopic, error) {
	response, err := h.metadataLifecycle.ListNotificationTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification topics: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("list notification topics: response is empty")
	}
	result := make([]gatewayapi.NotificationTopic, len(response.GetTopics()))
	for i, topic := range response.GetTopics() {
		result[i], err = fromProtoNotificationTopic(topic)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (h *ogenHandler) ListUserNotificationSubscriptions(ctx context.Context, params gatewayapi.ListUserNotificationSubscriptionsParams) ([]gatewayapi.NotificationSubscription, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	target := notificationUserID(params.UserId, actor)
	response, err := h.metadataLifecycle.ListNotificationSubscriptions(ctx, target, actor)
	if err != nil {
		return nil, fmt.Errorf("list notification subscriptions: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("list notification subscriptions: response is empty")
	}
	result := make([]gatewayapi.NotificationSubscription, len(response.GetSubscriptions()))
	for i, subscription := range response.GetSubscriptions() {
		result[i], err = fromProtoNotificationSubscription(subscription)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (h *ogenHandler) SubscribeUserNotification(ctx context.Context, params gatewayapi.SubscribeUserNotificationParams) error {
	actor, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return fmt.Errorf("subscribe notification: %w", h.metadataLifecycle.SubscribeNotification(ctx, notificationUserID(params.UserId, actor), "", params.TopicKey, actor))
}

func (h *ogenHandler) UnsubscribeUserNotification(ctx context.Context, params gatewayapi.UnsubscribeUserNotificationParams) error {
	actor, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return fmt.Errorf("unsubscribe notification: %w", h.metadataLifecycle.UnsubscribeNotification(ctx, notificationUserID(params.UserId, actor), "", params.TopicKey, actor))
}

func (h *ogenHandler) SubscribeGroupUserNotification(ctx context.Context, params gatewayapi.SubscribeGroupUserNotificationParams) error {
	actor, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return fmt.Errorf("subscribe group notification: %w", h.metadataLifecycle.SubscribeNotification(ctx, notificationUserID(params.UserId, actor), params.GroupId.String(), params.TopicKey, actor))
}

func (h *ogenHandler) UnsubscribeGroupUserNotification(ctx context.Context, params gatewayapi.UnsubscribeGroupUserNotificationParams) error {
	actor, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	return fmt.Errorf("unsubscribe group notification: %w", h.metadataLifecycle.UnsubscribeNotification(ctx, notificationUserID(params.UserId, actor), params.GroupId.String(), params.TopicKey, actor))
}

func notificationUserID(value, actor string) string {
	if value == "self" {
		return actor
	}
	return value
}

func fromProtoNotificationTopic(value *metadatalifecyclev1.NotificationTopic) (gatewayapi.NotificationTopic, error) {
	if value == nil {
		return gatewayapi.NotificationTopic{}, fmt.Errorf("decode notification topic: response contains nil topic")
	}
	scope, err := fromProtoNotificationScope(value.GetScope())
	if err != nil {
		return gatewayapi.NotificationTopic{}, err
	}
	mode, err := fromProtoNotificationMode(value.GetMode())
	if err != nil {
		return gatewayapi.NotificationTopic{}, err
	}
	return gatewayapi.NotificationTopic{
		Key:                 value.GetKey(),
		Scope:               scope,
		Mode:                mode,
		DefaultOn:           value.GetDefaultOn(),
		RequiredPermissions: value.GetRequiredPermissions(),
	}, nil
}

func fromProtoNotificationSubscription(value *metadatalifecyclev1.NotificationSubscription) (gatewayapi.NotificationSubscription, error) {
	if value == nil {
		return gatewayapi.NotificationSubscription{}, fmt.Errorf("decode notification subscription: response contains nil subscription")
	}
	channel, err := fromProtoNotificationChannel(value.GetChannel())
	if err != nil {
		return gatewayapi.NotificationSubscription{}, err
	}
	result := gatewayapi.NotificationSubscription{
		UserID:   value.GetUserId(),
		TopicKey: value.GetTopicKey(),
		Channel:  channel,
	}
	if value.GetGroupId() != "" {
		groupID, err := parseUUID("notification.group_id", value.GetGroupId())
		if err != nil {
			return gatewayapi.NotificationSubscription{}, err
		}
		result.GroupID = gatewayapi.NewOptUUID(groupID)
	}
	return result, nil
}

func fromProtoNotificationScope(value metadatalifecyclev1.NotificationScope) (gatewayapi.NotificationScope, error) {
	switch value {
	case metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_DIRECT:
		return gatewayapi.NotificationScopeDIRECT, nil
	case metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_GROUP:
		return gatewayapi.NotificationScopeGROUP, nil
	case metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_GLOBAL:
		return gatewayapi.NotificationScopeGLOBAL, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown notification scope: %v", value)
	}
}

func fromProtoNotificationMode(value metadatalifecyclev1.NotificationMode) (gatewayapi.NotificationMode, error) {
	switch value {
	case metadatalifecyclev1.NotificationMode_NOTIFICATION_MODE_MANDATORY:
		return gatewayapi.NotificationModeMANDATORY, nil
	case metadatalifecyclev1.NotificationMode_NOTIFICATION_MODE_OPT_IN:
		return gatewayapi.NotificationModeOPTIN, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown notification mode: %v", value)
	}
}

func fromProtoNotificationChannel(value metadatalifecyclev1.NotificationChannel) (gatewayapi.NotificationChannel, error) {
	if value == metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL {
		return gatewayapi.NotificationChannelEMAIL, nil
	}
	return "", fmt.Errorf("metadata-lifecycle returned unknown notification channel: %v", value)
}
