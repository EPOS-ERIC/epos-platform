package handlers

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ListNotificationTopics(ctx context.Context) ([]gatewayapi.NotificationTopic, error) {
	response, err := h.metadataLifecycleClient.ListNotificationTopics(ctx)
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
			return nil, fmt.Errorf("map notification topic at index %d: %w", i, err)
		}
	}
	return result, nil
}

func (h *ogenHandler) ListUserNotificationSubscriptions(ctx context.Context, params gatewayapi.ListUserNotificationSubscriptionsParams) ([]gatewayapi.NotificationSubscription, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	target := notificationUserID(params.UserId, actor)
	response, err := h.metadataLifecycleClient.ListNotificationSubscriptions(ctx, target, actor)
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
			return nil, fmt.Errorf("map notification subscription at index %d: %w", i, err)
		}
	}
	return result, nil
}

func (h *ogenHandler) SubscribeUserNotification(ctx context.Context, params gatewayapi.SubscribeUserNotificationParams) error {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	topicKey, err := toProtoGlobalNotificationTopicKey(params.TopicKey)
	if err != nil {
		return fmt.Errorf("map global notification topic: %w", err)
	}
	if err := h.metadataLifecycleClient.SubscribeNotification(ctx, notificationUserID(params.UserId, actor), "", topicKey, actor); err != nil {
		return fmt.Errorf("subscribe notification: %w", err)
	}
	return nil
}

func (h *ogenHandler) UnsubscribeUserNotification(ctx context.Context, params gatewayapi.UnsubscribeUserNotificationParams) error {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	topicKey, err := toProtoGlobalNotificationTopicKey(params.TopicKey)
	if err != nil {
		return fmt.Errorf("map global notification topic: %w", err)
	}
	if err := h.metadataLifecycleClient.UnsubscribeNotification(ctx, notificationUserID(params.UserId, actor), "", topicKey, actor); err != nil {
		return fmt.Errorf("unsubscribe notification: %w", err)
	}
	return nil
}

func (h *ogenHandler) SubscribeGroupUserNotification(ctx context.Context, params gatewayapi.SubscribeGroupUserNotificationParams) error {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	topicKey, err := toProtoGroupNotificationTopicKey(params.TopicKey)
	if err != nil {
		return fmt.Errorf("map group notification topic: %w", err)
	}
	if err := h.metadataLifecycleClient.SubscribeNotification(ctx, notificationUserID(params.UserId, actor), params.GroupId.String(), topicKey, actor); err != nil {
		return fmt.Errorf("subscribe group notification: %w", err)
	}
	return nil
}

func (h *ogenHandler) UnsubscribeGroupUserNotification(ctx context.Context, params gatewayapi.UnsubscribeGroupUserNotificationParams) error {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	topicKey, err := toProtoGroupNotificationTopicKey(params.TopicKey)
	if err != nil {
		return fmt.Errorf("map group notification topic: %w", err)
	}
	if err := h.metadataLifecycleClient.UnsubscribeNotification(ctx, notificationUserID(params.UserId, actor), params.GroupId.String(), topicKey, actor); err != nil {
		return fmt.Errorf("unsubscribe group notification: %w", err)
	}
	return nil
}

func notificationUserID(value, actor string) string {
	if value == "self" {
		return actor
	}
	return value
}

func toProtoGlobalNotificationTopicKey(value gatewayapi.GlobalNotificationTopicKey) (metadatalifecyclev1.NotificationTopicKey, error) {
	switch value {
	case gatewayapi.GlobalNotificationTopicKeyGlobalMembershipRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_MEMBERSHIP_REQUESTED, nil
	case gatewayapi.GlobalNotificationTopicKeyGlobalRoleRequestRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_ROLE_REQUEST_REQUESTED, nil
	case gatewayapi.GlobalNotificationTopicKeyGlobalResourceSubmitted:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_SUBMITTED, nil
	case gatewayapi.GlobalNotificationTopicKeyGlobalResourcePublished:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_PUBLISHED, nil
	case gatewayapi.GlobalNotificationTopicKeyGlobalConversionRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_CONVERSION_REQUESTED, nil
	default:
		return 0, fmt.Errorf("unknown global notification topic key: %q", value)
	}
}

func toProtoGroupNotificationTopicKey(value gatewayapi.GroupNotificationTopicKey) (metadatalifecyclev1.NotificationTopicKey, error) {
	switch value {
	case gatewayapi.GroupNotificationTopicKeyGroupMembershipRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_MEMBERSHIP_REQUESTED, nil
	case gatewayapi.GroupNotificationTopicKeyGroupRoleRequestRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_ROLE_REQUEST_REQUESTED, nil
	case gatewayapi.GroupNotificationTopicKeyGroupResourceSubmitted:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_SUBMITTED, nil
	case gatewayapi.GroupNotificationTopicKeyGroupResourcePublished:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_PUBLISHED, nil
	default:
		return 0, fmt.Errorf("unknown group notification topic key: %q", value)
	}
}

func fromProtoNotificationTopicKey(value metadatalifecyclev1.NotificationTopicKey) (gatewayapi.NotificationTopicKey, error) {
	switch value {
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_MEMBERSHIP_REQUESTED:
		return gatewayapi.NotificationTopicKeyGroupMembershipRequested, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_MEMBERSHIP_REQUESTED:
		return gatewayapi.NotificationTopicKeyGlobalMembershipRequested, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_ROLE_REQUEST_REQUESTED:
		return gatewayapi.NotificationTopicKeyGroupRoleRequestRequested, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_ROLE_REQUEST_REQUESTED:
		return gatewayapi.NotificationTopicKeyGlobalRoleRequestRequested, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_SUBMITTED:
		return gatewayapi.NotificationTopicKeyGroupResourceSubmitted, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_PUBLISHED:
		return gatewayapi.NotificationTopicKeyGroupResourcePublished, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_SUBMITTED:
		return gatewayapi.NotificationTopicKeyGlobalResourceSubmitted, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_PUBLISHED:
		return gatewayapi.NotificationTopicKeyGlobalResourcePublished, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_CONVERSION_REQUESTED:
		return gatewayapi.NotificationTopicKeyGlobalConversionRequested, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown notification topic key %s", value.String())
	}
}

func fromProtoNotificationTopic(value *metadatalifecyclev1.NotificationTopic) (gatewayapi.NotificationTopic, error) {
	if value == nil {
		return gatewayapi.NotificationTopic{}, fmt.Errorf("decode notification topic: response contains nil topic")
	}
	scope, err := fromProtoNotificationScope(value.GetScope())
	if err != nil {
		return gatewayapi.NotificationTopic{}, err
	}
	key, err := fromProtoNotificationTopicKey(value.GetKey())
	if err != nil {
		return gatewayapi.NotificationTopic{}, fmt.Errorf("decode notification topic key: %w", err)
	}
	return gatewayapi.NotificationTopic{
		Key:                 key,
		Label:               value.GetLabel(),
		Description:         value.GetDescription(),
		Scope:               scope,
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
	topicKey, err := fromProtoNotificationTopicKey(value.GetTopicKey())
	if err != nil {
		return gatewayapi.NotificationSubscription{}, fmt.Errorf("decode notification subscription topic key: %w", err)
	}
	result := gatewayapi.NotificationSubscription{
		UserID:   value.GetUserId(),
		TopicKey: topicKey,
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
	case metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_GROUP:
		return gatewayapi.NotificationScopeGROUP, nil
	case metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_GLOBAL:
		return gatewayapi.NotificationScopeGLOBAL, nil
	case metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_DIRECT:
	default:
	}
	return "", fmt.Errorf("unsupported notification scope: %v", value)
}

func fromProtoNotificationChannel(value metadatalifecyclev1.NotificationChannel) (gatewayapi.NotificationChannel, error) {
	if value == metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL {
		return gatewayapi.NotificationChannelEMAIL, nil
	}
	return "", fmt.Errorf("metadata-lifecycle returned unknown notification channel: %v", value)
}
