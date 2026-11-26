// Package notifications manages notification topics and user subscriptions.
package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	ListTopics(context.Context) ([]domain.NotificationTopic, error)
	ListSubscriptions(context.Context, string) ([]domain.NotificationSubscription, error)
	GetNotificationUserContext(context.Context, string) (domain.UserSummary, error)
	GetNotificationGroupContext(context.Context, uuid.UUID) (domain.NotificationGroupContext, error)
	CreateGlobalSubscription(context.Context, string, domain.NotificationTopicKey, domain.NotificationChannel) error
	CreateGroupSubscription(context.Context, string, uuid.UUID, domain.NotificationTopicKey, domain.NotificationChannel) error
	DeleteGlobalSubscription(context.Context, string, domain.NotificationTopicKey, domain.NotificationChannel) error
	DeleteGroupSubscription(context.Context, string, uuid.UUID, domain.NotificationTopicKey, domain.NotificationChannel) error
	Recipient(context.Context, string) (domain.NotificationRecipient, error)
	GlobalRecipients(context.Context, domain.NotificationTopicKey) ([]domain.NotificationRecipient, error)
	GroupRecipients(context.Context, uuid.UUID, domain.NotificationTopicKey) ([]domain.NotificationRecipient, error)
}

type Authorizer interface {
	CanManageGlobalNotificationSubscriptions(context.Context, string, string) (domain.Result, error)
	CanManageGroupNotificationSubscriptions(context.Context, string, string, uuid.UUID) (domain.Result, error)
	CanSubscribeToNotificationTopic(context.Context, string, *uuid.UUID, []string) (domain.Result, error)
	GroupsWithAnyPermissionForActiveMember(context.Context, string, string, []domain.GroupPermission) ([]uuid.UUID, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
	publicURL  string
	log        *slog.Logger
}

func New(log *slog.Logger, store Store, authorizer Authorizer, publicURL string) *Service {
	if log == nil {
		panic("notifications: nil logger")
	}
	if store == nil {
		panic("notifications: nil store")
	}
	if authorizer == nil {
		panic("notifications: nil authorizer")
	}
	return &Service{
		store:      store,
		authorizer: authorizer,
		publicURL:  strings.TrimRight(publicURL, "/"),
		log:        log.With("component", "notification-service"),
	}
}

func (s *Service) ListTopics(ctx context.Context) ([]domain.NotificationTopic, error) {
	topics, err := s.store.ListTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification topics: %w", err)
	}

	result := make([]domain.NotificationTopic, 0, len(topics))
	for _, topic := range topics {
		if topic.Scope != domain.NotificationScopeGroup && topic.Scope != domain.NotificationScopeGlobal {
			continue
		}

		result = append(result, topic)
	}

	return result, nil
}

func (s *Service) ListSubscriptions(ctx context.Context, actor, userID string) ([]domain.NotificationSubscription, error) {
	allowed, err := s.authorizer.CanManageGlobalNotificationSubscriptions(ctx, actor, userID)
	if err != nil {
		return nil, fmt.Errorf("authorize notification subscription list: %w", err)
	}

	visibleGroupIDs := map[uuid.UUID]struct{}{}
	if !allowed.Allowed {
		groupIDs, err := s.authorizer.GroupsWithAnyPermissionForActiveMember(ctx, actor, userID, []domain.GroupPermission{domain.GroupPermissionNotificationManage})
		if err != nil {
			return nil, fmt.Errorf("authorize group notification subscription list: %w", err)
		}
		if len(groupIDs) == 0 {
			return nil, apperror.Default(apperror.CodePermissionDenied)
		}

		for _, groupID := range groupIDs {
			visibleGroupIDs[groupID] = struct{}{}
		}
	}

	subscriptions, err := s.store.ListSubscriptions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list notification subscriptions: %w", err)
	}

	// if the user can manage global notifications or is self then return all subsciptions
	if allowed.Allowed {
		return subscriptions, nil
	}

	visible := make([]domain.NotificationSubscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		if subscription.GroupID == nil {
			continue
		}

		if _, ok := visibleGroupIDs[*subscription.GroupID]; ok {
			visible = append(visible, subscription)
		}
	}

	return visible, nil
}

func (s *Service) Subscribe(ctx context.Context, actor, userID string, groupID *uuid.UUID, topicKey domain.NotificationTopicKey, channel domain.NotificationChannel) (domain.NotificationSubscription, error) {
	if err := s.authorizeSubscription(ctx, actor, userID, groupID, topicKey, channel); err != nil {
		return domain.NotificationSubscription{}, err
	}
	if groupID == nil {
		if err := s.store.CreateGlobalSubscription(ctx, userID, topicKey, channel); err != nil {
			return domain.NotificationSubscription{}, fmt.Errorf("create global notification subscription: %w", err)
		}
		return domain.NotificationSubscription{UserID: userID, TopicKey: topicKey, Channel: channel}, nil
	}

	if err := s.store.CreateGroupSubscription(ctx, userID, *groupID, topicKey, channel); err != nil {
		return domain.NotificationSubscription{}, fmt.Errorf("create group notification subscription: %w", err)
	}
	return domain.NotificationSubscription{UserID: userID, GroupID: groupID, TopicKey: topicKey, Channel: channel}, nil
}

func (s *Service) Unsubscribe(ctx context.Context, actor, userID string, groupID *uuid.UUID, topicKey domain.NotificationTopicKey, channel domain.NotificationChannel) error {
	if err := s.authorizeSubscription(ctx, actor, userID, groupID, topicKey, channel); err != nil {
		return err
	}

	if groupID == nil {
		if err := s.store.DeleteGlobalSubscription(ctx, userID, topicKey, channel); err != nil {
			return fmt.Errorf("delete global notification subscription: %w", err)
		}
		return nil
	}

	if err := s.store.DeleteGroupSubscription(ctx, userID, *groupID, topicKey, channel); err != nil {
		return fmt.Errorf("delete group notification subscription: %w", err)
	}
	return nil
}

func (s *Service) authorizeSubscription(ctx context.Context, actor, userID string, groupID *uuid.UUID, topicKey domain.NotificationTopicKey, channel domain.NotificationChannel) error {
	topic, err := s.topic(ctx, topicKey)
	if err != nil {
		return err
	}
	if channel != domain.NotificationChannelEmail {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	if groupID == nil {
		if topic.Scope != domain.NotificationScopeGlobal {
			return apperror.Default(apperror.CodeValidationFailed)
		}
		allowed, err := s.authorizer.CanManageGlobalNotificationSubscriptions(ctx, actor, userID)
		if err != nil {
			return fmt.Errorf("authorize global notification subscription: %w", err)
		}
		if !allowed.Allowed {
			return apperror.Default(apperror.CodePermissionDenied)
		}
		return s.authorizeTopicEligibility(ctx, userID, groupID, topic)
	}

	if topic.Scope != domain.NotificationScopeGroup || *groupID == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}
	allowed, err := s.authorizer.CanManageGroupNotificationSubscriptions(ctx, actor, userID, *groupID)
	if err != nil {
		return fmt.Errorf("authorize group notification subscription: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}
	return s.authorizeTopicEligibility(ctx, userID, groupID, topic)
}

func (s *Service) authorizeTopicEligibility(ctx context.Context, userID string, groupID *uuid.UUID, topic domain.NotificationTopic) error {
	allowed, err := s.authorizer.CanSubscribeToNotificationTopic(ctx, userID, groupID, topic.RequiredPermissions)
	if err != nil {
		return fmt.Errorf("authorize notification topic eligibility: %w", err)
	}

	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	return nil
}

func (s *Service) topic(ctx context.Context, key domain.NotificationTopicKey) (domain.NotificationTopic, error) {
	if key == "" {
		return domain.NotificationTopic{}, apperror.Default(apperror.CodeValidationFailed)
	}
	topics, err := s.store.ListTopics(ctx)
	if err != nil {
		return domain.NotificationTopic{}, fmt.Errorf("list notification topics: %w", err)
	}
	for _, topic := range topics {
		if topic.Key == key {
			return topic, nil
		}
	}
	return domain.NotificationTopic{}, apperror.Default(apperror.CodeNotFound)
}
