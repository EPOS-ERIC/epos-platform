package grpcapi

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	notificationservice "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/notifications"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, wrapHandlerError(err)
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
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	subscriptions, err := s.notifications.ListSubscriptions(ctx, actor, request.GetUserId())
	if err != nil {
		return nil, wrapHandlerError(err)
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
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	topicKey, err := toDomainNotificationTopicKey(request.GetTopicKey())
	if err != nil {
		return nil, err
	}
	subscription, err := s.notifications.Subscribe(ctx, actor, request.GetUserId(), groupID, topicKey, toDomainNotificationChannel(request.GetChannel()))
	if err != nil {
		return nil, wrapHandlerError(err)
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
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	topicKey, err := toDomainNotificationTopicKey(request.GetTopicKey())
	if err != nil {
		return nil, err
	}
	if err := s.notifications.Unsubscribe(ctx, actor, request.GetUserId(), groupID, topicKey, toDomainNotificationChannel(request.GetChannel())); err != nil {
		return nil, wrapHandlerError(err)
	}
	return &metadatalifecyclev1.UnsubscribeNotificationResponse{}, nil
}

func toProtoNotificationTopic(value domain.NotificationTopic) (*metadatalifecyclev1.NotificationTopic, error) {
	scope, err := toProtoNotificationScope(value.Scope)
	if err != nil {
		return nil, err
	}
	mode, err := toProtoNotificationMode(value.Mode)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.NotificationTopic{
		Key:                 string(value.Key),
		Scope:               scope,
		Mode:                mode,
		DefaultOn:           value.DefaultOn,
		RequiredPermissions: value.RequiredPermissions,
	}, nil
}

func toProtoNotificationSubscription(value domain.NotificationSubscription) (*metadatalifecyclev1.NotificationSubscription, error) {
	channel, err := toProtoNotificationChannel(value.Channel)
	if err != nil {
		return nil, err
	}
	result := &metadatalifecyclev1.NotificationSubscription{
		UserId:   value.UserID,
		TopicKey: string(value.TopicKey),
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
		return 0, status.Errorf(codes.Internal, "unknown notification scope: %q", value)
	}
}

func toProtoNotificationMode(value domain.NotificationMode) (metadatalifecyclev1.NotificationMode, error) {
	switch value {
	case domain.NotificationModeMandatory:
		return metadatalifecyclev1.NotificationMode_NOTIFICATION_MODE_MANDATORY, nil
	case domain.NotificationModeOptIn:
		return metadatalifecyclev1.NotificationMode_NOTIFICATION_MODE_OPT_IN, nil
	default:
		return 0, status.Errorf(codes.Internal, "unknown notification mode: %q", value)
	}
}

func toDomainNotificationChannel(value metadatalifecyclev1.NotificationChannel) domain.NotificationChannel {
	if value == metadatalifecyclev1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL {
		return domain.NotificationChannelEmail
	}
	return ""
}

func toDomainNotificationTopicKey(value string) (domain.NotificationTopicKey, error) {
	key, err := domain.NewNotificationTopicKey(value)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, "invalid notification topic key")
	}
	return key, nil
}

func optionalNotificationGroupID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group id")
	}
	return &id, nil
}
