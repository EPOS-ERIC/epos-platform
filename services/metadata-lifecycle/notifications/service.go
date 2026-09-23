// Package notifications manages notification topics and user subscriptions.
package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	ListTopics(context.Context) ([]domain.NotificationTopic, error)
	ListSubscriptions(context.Context, string) ([]domain.NotificationSubscription, error)
	GetNotificationUserContext(context.Context, string) (domain.UserSummary, error)
	GetNotificationGroupContext(context.Context, uuid.UUID) (domain.Group, error)
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
}

type Service struct {
	store      Store
	authorizer Authorizer
	publicURL  string
	log        *slog.Logger
}

func New(log *slog.Logger, store Store, authorizer Authorizer, publicURL string) *Service {
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
	return topics, nil
}

func (s *Service) ListSubscriptions(ctx context.Context, actor domain.Actor, userID string) ([]domain.NotificationSubscription, error) {
	allowed, err := s.authorizer.CanManageGlobalNotificationSubscriptions(ctx, string(actor), userID)
	if err != nil {
		return nil, fmt.Errorf("authorize notification subscription list: %w", err)
	}
	if !allowed.Allowed {
		return nil, domain.ErrNotificationForbidden
	}
	subscriptions, err := s.store.ListSubscriptions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list notification subscriptions: %w", err)
	}
	return subscriptions, nil
}

func (s *Service) Subscribe(ctx context.Context, actor domain.Actor, userID string, groupID *uuid.UUID, topicKey domain.NotificationTopicKey, channel domain.NotificationChannel) (domain.NotificationSubscription, error) {
	topic, err := s.authorizeSubscription(ctx, actor, userID, groupID, topicKey, channel)
	if err != nil {
		return domain.NotificationSubscription{}, err
	}
	if groupID == nil {
		if err := s.store.CreateGlobalSubscription(ctx, userID, topicKey, channel); err != nil {
			return domain.NotificationSubscription{}, fmt.Errorf("create global notification subscription: %w", err)
		}
		return domain.NotificationSubscription{UserID: userID, TopicKey: topic.Key, Channel: channel}, nil
	}

	if err := s.store.CreateGroupSubscription(ctx, userID, *groupID, topicKey, channel); err != nil {
		return domain.NotificationSubscription{}, fmt.Errorf("create group notification subscription: %w", err)
	}
	return domain.NotificationSubscription{UserID: userID, GroupID: groupID, TopicKey: topic.Key, Channel: channel}, nil
}

func (s *Service) Unsubscribe(ctx context.Context, actor domain.Actor, userID string, groupID *uuid.UUID, topicKey domain.NotificationTopicKey, channel domain.NotificationChannel) error {
	_, err := s.authorizeSubscription(ctx, actor, userID, groupID, topicKey, channel)
	if err != nil {
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

func (s *Service) authorizeSubscription(ctx context.Context, actor domain.Actor, userID string, groupID *uuid.UUID, topicKey domain.NotificationTopicKey, channel domain.NotificationChannel) (domain.NotificationTopic, error) {
	topic, err := s.topic(ctx, topicKey)
	if err != nil {
		return domain.NotificationTopic{}, err
	}
	if channel != domain.NotificationChannelEmail || topic.Mode != domain.NotificationModeOptIn || topic.Scope == domain.NotificationScopeDirect {
		return domain.NotificationTopic{}, domain.ErrNotificationInvalid
	}

	if groupID == nil {
		if topic.Scope != domain.NotificationScopeGlobal {
			return domain.NotificationTopic{}, domain.ErrNotificationInvalid
		}
		allowed, err := s.authorizer.CanManageGlobalNotificationSubscriptions(ctx, string(actor), userID)
		if err != nil {
			return domain.NotificationTopic{}, fmt.Errorf("authorize global notification subscription: %w", err)
		}
		if !allowed.Allowed {
			return domain.NotificationTopic{}, domain.ErrNotificationForbidden
		}
		return topic, nil
	}

	if topic.Scope != domain.NotificationScopeGroup || *groupID == uuid.Nil {
		return domain.NotificationTopic{}, domain.ErrNotificationInvalid
	}
	allowed, err := s.authorizer.CanManageGroupNotificationSubscriptions(ctx, string(actor), userID, *groupID)
	if err != nil {
		return domain.NotificationTopic{}, fmt.Errorf("authorize group notification subscription: %w", err)
	}
	if !allowed.Allowed {
		return domain.NotificationTopic{}, domain.ErrNotificationForbidden
	}
	return topic, nil
}

func (s *Service) topic(ctx context.Context, key domain.NotificationTopicKey) (domain.NotificationTopic, error) {
	if key == "" {
		return domain.NotificationTopic{}, domain.ErrNotificationInvalid
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
	return domain.NotificationTopic{}, domain.ErrNotificationNotFound
}
