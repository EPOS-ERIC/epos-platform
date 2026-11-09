package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	notificationservice "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/notifications"
	"github.com/google/uuid"
)

type NotificationServer struct {
	metadatalifecyclev1.UnimplementedNotificationServiceServer
	notifications *notificationservice.Service
}

func NewNotification(service *notificationservice.Service) *NotificationServer {
	return &NotificationServer{notifications: service}
}

func (s *NotificationServer) ListNotificationTopics(ctx context.Context, _ *metadatalifecyclev1.ListNotificationTopicsRequest) (*metadatalifecyclev1.ListNotificationTopicsResponse, error) {
	topics, err := s.notifications.ListTopics(ctx)
	if err != nil {
		return nil, err
	}
	values, err := mapSliceErr(topics, toProtoNotificationTopic)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.ListNotificationTopicsResponse{Topics: values}, nil
}

func (s *NotificationServer) ListNotificationSubscriptions(ctx context.Context, request *metadatalifecyclev1.ListNotificationSubscriptionsRequest) (*metadatalifecyclev1.ListNotificationSubscriptionsResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if request.GetUserId() == "" {
		return nil, apperror.New(apperror.CodeValidationFailed, "user id is required")
	}
	subscriptions, err := s.notifications.ListSubscriptions(ctx, actor, request.GetUserId())
	if err != nil {
		return nil, err
	}
	values, err := mapSliceErr(subscriptions, toProtoNotificationSubscription)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.ListNotificationSubscriptionsResponse{Subscriptions: values}, nil
}

func (s *NotificationServer) SubscribeNotification(ctx context.Context, request *metadatalifecyclev1.SubscribeNotificationRequest) (*metadatalifecyclev1.SubscribeNotificationResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := optionalNotificationGroupID(request.GetGroupId())
	if err != nil {
		return nil, err
	}
	if request.GetUserId() == "" {
		return nil, apperror.New(apperror.CodeValidationFailed, "user id is required")
	}
	topicKey, err := toDomainNotificationTopicKey(request.GetTopicKey())
	if err != nil {
		return nil, err
	}
	subscription, err := s.notifications.Subscribe(ctx, actor, request.GetUserId(), groupID, topicKey, toDomainNotificationChannel(request.GetChannel()))
	if err != nil {
		return nil, err
	}
	value, err := toProtoNotificationSubscription(subscription)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.SubscribeNotificationResponse{Subscription: value}, nil
}

func (s *NotificationServer) UnsubscribeNotification(ctx context.Context, request *metadatalifecyclev1.UnsubscribeNotificationRequest) (*metadatalifecyclev1.UnsubscribeNotificationResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := optionalNotificationGroupID(request.GetGroupId())
	if err != nil {
		return nil, err
	}
	if request.GetUserId() == "" {
		return nil, apperror.New(apperror.CodeValidationFailed, "user id is required")
	}
	topicKey, err := toDomainNotificationTopicKey(request.GetTopicKey())
	if err != nil {
		return nil, err
	}
	if err := s.notifications.Unsubscribe(ctx, actor, request.GetUserId(), groupID, topicKey, toDomainNotificationChannel(request.GetChannel())); err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.UnsubscribeNotificationResponse{}, nil
}

func toProtoNotificationTopic(value domain.NotificationTopic) (*metadatalifecyclev1.NotificationTopic, error) {
	scope, err := toProtoNotificationScope(value.Scope)
	if err != nil {
		return nil, err
	}
	key, err := toProtoNotificationTopicKey(value.Key)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.NotificationTopic{
		Key:                 key,
		Label:               value.Label,
		Description:         value.Description,
		Scope:               scope,
		RequiredPermissions: value.RequiredPermissions,
	}, nil
}

func toProtoNotificationSubscription(value domain.NotificationSubscription) (*metadatalifecyclev1.NotificationSubscription, error) {
	channel, err := toProtoNotificationChannel(value.Channel)
	if err != nil {
		return nil, err
	}
	topicKey, err := toProtoNotificationTopicKey(value.TopicKey)
	if err != nil {
		return nil, err
	}
	result := &metadatalifecyclev1.NotificationSubscription{
		UserId:   value.UserID,
		TopicKey: topicKey,
		Channel:  channel,
	}
	if value.GroupID != nil {
		result.GroupId = value.GroupID.String()
	}
	return result, nil
}

func toProtoNotificationScope(value domain.NotificationScope) (metadatalifecyclev1.NotificationScope, error) {
	switch value {
	case domain.NotificationScopeDirect:
		return metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_DIRECT, nil
	case domain.NotificationScopeGroup:
		return metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_GROUP, nil
	case domain.NotificationScopeGlobal:
		return metadatalifecyclev1.NotificationScope_NOTIFICATION_SCOPE_GLOBAL, nil
	default:
		return 0, apperror.Default(apperror.CodeInternal)
	}
}

func toDomainNotificationChannel(value metadatalifecyclev1.NotificationChannel) domain.NotificationChannel {
	if value == metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL {
		return domain.NotificationChannelEmail
	}
	return ""
}

func toDomainNotificationTopicKey(value metadatalifecyclev1.NotificationTopicKey) (domain.NotificationTopicKey, error) {
	switch value {
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_MEMBERSHIP_REQUESTED:
		return domain.NotificationTopicGroupMembershipRequested, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_MEMBERSHIP_REQUESTED:
		return domain.NotificationTopicGlobalMembershipRequested, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_ROLE_REQUEST_REQUESTED:
		return domain.NotificationTopicGroupRoleRequestRequested, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_ROLE_REQUEST_REQUESTED:
		return domain.NotificationTopicGlobalRoleRequestRequested, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_SUBMITTED:
		return domain.NotificationTopicGroupResourceSubmitted, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_PUBLISHED:
		return domain.NotificationTopicGroupResourcePublished, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_SUBMITTED:
		return domain.NotificationTopicGlobalResourceSubmitted, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_PUBLISHED:
		return domain.NotificationTopicGlobalResourcePublished, nil
	case metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_CONVERSION_REQUESTED:
		return domain.NotificationTopicGlobalConversionRequested, nil
	default:
		return "", apperror.New(apperror.CodeValidationFailed, "invalid notification topic key")
	}
}

func toProtoNotificationTopicKey(value domain.NotificationTopicKey) (metadatalifecyclev1.NotificationTopicKey, error) {
	switch value {
	case domain.NotificationTopicGroupMembershipRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_MEMBERSHIP_REQUESTED, nil
	case domain.NotificationTopicGlobalMembershipRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_MEMBERSHIP_REQUESTED, nil
	case domain.NotificationTopicGroupRoleRequestRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_ROLE_REQUEST_REQUESTED, nil
	case domain.NotificationTopicGlobalRoleRequestRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_ROLE_REQUEST_REQUESTED, nil
	case domain.NotificationTopicGroupResourceSubmitted:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_SUBMITTED, nil
	case domain.NotificationTopicGroupResourcePublished:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GROUP_RESOURCE_PUBLISHED, nil
	case domain.NotificationTopicGlobalResourceSubmitted:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_SUBMITTED, nil
	case domain.NotificationTopicGlobalResourcePublished:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_RESOURCE_PUBLISHED, nil
	case domain.NotificationTopicGlobalConversionRequested:
		return metadatalifecyclev1.NotificationTopicKey_NOTIFICATION_TOPIC_KEY_GLOBAL_CONVERSION_REQUESTED, nil
	case domain.NotificationTopicDirectMembershipAccepted,
		domain.NotificationTopicDirectMembershipRejected,
		domain.NotificationTopicDirectRoleRequestAccepted,
		domain.NotificationTopicDirectRoleRequestRejected,
		domain.NotificationTopicDirectRoleAssigned,
		domain.NotificationTopicDirectPermissionAssigned,
		domain.NotificationTopicDirectResourcePublished,
		domain.NotificationTopicDirectResourceRejected,
		domain.NotificationTopicDirectResourceUnpublished:
		return 0, apperror.Default(apperror.CodeInternal)
	default:
		return 0, apperror.Default(apperror.CodeInternal)
	}
}

func optionalNotificationGroupID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid group id")
	}
	return &id, nil
}
