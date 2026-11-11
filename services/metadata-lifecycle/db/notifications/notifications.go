// Package notifications provides PostgreSQL persistence for notification topics and subscriptions.
package notifications

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{Queries: sqlc.New(db)}
}

func (r *Repo) ListTopics(ctx context.Context) ([]domain.NotificationTopic, error) {
	rows, err := r.ListNotificationEventTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification topics: %w", err)
	}
	result := make([]domain.NotificationTopic, 0, len(rows))
	for _, row := range rows {
		permissions, err := r.ListNotificationEventTypePermissions(ctx, row.Key)
		if err != nil {
			return nil, fmt.Errorf("list permissions for notification topic %s: %w", row.Key, err)
		}
		topic, err := toDomainTopic(row.Key, row.Label, row.Description, row.Scope, permissions)
		if err != nil {
			return nil, fmt.Errorf("parse notification topic %s: %w", row.Key, err)
		}
		result = append(result, topic)
	}
	return result, nil
}

func (r *Repo) ListSubscriptions(ctx context.Context, userID string) ([]domain.NotificationSubscription, error) {
	globalRows, err := r.ListGlobalNotificationSubscriptions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list global notification subscriptions: %w", err)
	}
	groupRows, err := r.ListGroupNotificationSubscriptions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list group notification subscriptions: %w", err)
	}

	result := make([]domain.NotificationSubscription, 0, len(globalRows)+len(groupRows))
	for _, row := range globalRows {
		topicKey, err := domain.NewNotificationTopicKey(row.EventType)
		if err != nil {
			return nil, fmt.Errorf("parse notification subscription topic %s: %w", row.EventType, err)
		}
		result = append(result, domain.NotificationSubscription{
			UserID:   row.UserID,
			TopicKey: topicKey,
			Channel:  domain.NotificationChannel(row.Channel),
		})
	}
	for _, row := range groupRows {
		topicKey, err := domain.NewNotificationTopicKey(row.EventType)
		if err != nil {
			return nil, fmt.Errorf("parse notification subscription topic %s: %w", row.EventType, err)
		}
		groupID := row.GroupID
		result = append(result, domain.NotificationSubscription{
			UserID:   row.UserID,
			GroupID:  &groupID,
			TopicKey: topicKey,
			Channel:  domain.NotificationChannel(row.Channel),
		})
	}
	return result, nil
}

func (r *Repo) GetNotificationUserContext(ctx context.Context, userID string) (domain.UserSummary, error) {
	row, err := r.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserSummary{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.UserSummary{}, fmt.Errorf("get notification user context: %w", err)
	}

	return domain.UserSummary{
		ID:         row.ID,
		Email:      row.Email,
		GivenName:  row.GivenName,
		FamilyName: row.FamilyName,
		Status:     domain.UserStatus(row.Status),
	}, nil
}

func (r *Repo) GetNotificationGroupContext(ctx context.Context, groupID uuid.UUID) (domain.NotificationGroupContext, error) {
	row, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NotificationGroupContext{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.NotificationGroupContext{}, fmt.Errorf("get notification group context: %w", err)
	}

	concept, err := r.GetConceptRecord(ctx, row.ConceptID)
	if err != nil {
		return domain.NotificationGroupContext{}, fmt.Errorf("get notification group concept: %w", err)
	}

	return domain.NotificationGroupContext{
		Name:    row.Name,
		Concept: concept.Name,
	}, nil
}

func (r *Repo) Recipient(ctx context.Context, userID string) (domain.NotificationRecipient, error) {
	row, err := r.GetNotificationUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NotificationRecipient{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.NotificationRecipient{}, fmt.Errorf("get notification recipient: %w", err)
	}
	return domain.NotificationRecipient{ID: row.ID, Email: row.Email, Status: domain.UserStatus(row.Status)}, nil
}

func (r *Repo) GlobalRecipients(ctx context.Context, topic domain.NotificationTopicKey) ([]domain.NotificationRecipient, error) {
	rows, err := r.ListGlobalNotificationRecipients(ctx, string(topic))
	if err != nil {
		return nil, fmt.Errorf("list global notification recipients: %w", err)
	}
	result := make([]domain.NotificationRecipient, len(rows))
	for i, row := range rows {
		result[i] = domain.NotificationRecipient{ID: row.ID, Email: row.Email, Status: domain.UserStatusActive}
	}
	return result, nil
}

func (r *Repo) GroupRecipients(ctx context.Context, groupID uuid.UUID, topic domain.NotificationTopicKey) ([]domain.NotificationRecipient, error) {
	rows, err := r.ListGroupNotificationRecipients(ctx, sqlc.ListGroupNotificationRecipientsParams{
		GroupID:   groupID,
		EventType: string(topic),
	})
	if err != nil {
		return nil, fmt.Errorf("list group notification recipients: %w", err)
	}
	result := make([]domain.NotificationRecipient, len(rows))
	for i, row := range rows {
		result[i] = domain.NotificationRecipient{ID: row.ID, Email: row.Email, Status: domain.UserStatusActive}
	}
	return result, nil
}

func (r *Repo) CreateGlobalSubscription(ctx context.Context, userID string, topic domain.NotificationTopicKey, channel domain.NotificationChannel) error {
	_, err := r.CreateGlobalNotificationSubscription(ctx, sqlc.CreateGlobalNotificationSubscriptionParams{
		UserID:    userID,
		EventType: string(topic),
		Channel:   sqlc.NotificationChannel(channel),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return fmt.Errorf("create global notification subscription: %w", err)
	}
	return nil
}

func (r *Repo) CreateGroupSubscription(ctx context.Context, userID string, groupID uuid.UUID, topic domain.NotificationTopicKey, channel domain.NotificationChannel) error {
	_, err := r.CreateGroupNotificationSubscription(ctx, sqlc.CreateGroupNotificationSubscriptionParams{
		UserID:    userID,
		GroupID:   groupID,
		EventType: string(topic),
		Channel:   sqlc.NotificationChannel(channel),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return fmt.Errorf("create group notification subscription: %w", err)
	}
	return nil
}

func (r *Repo) DeleteGlobalSubscription(ctx context.Context, userID string, topic domain.NotificationTopicKey, channel domain.NotificationChannel) error {
	rows, err := r.DeleteGlobalNotificationSubscription(ctx, sqlc.DeleteGlobalNotificationSubscriptionParams{
		UserID:    userID,
		EventType: string(topic),
		Channel:   sqlc.NotificationChannel(channel),
	})
	if err != nil {
		return fmt.Errorf("delete global notification subscription: %w", err)
	}
	if rows == 0 {
		return apperror.Default(apperror.CodeNotFound)
	}
	return nil
}

func (r *Repo) DeleteGroupSubscription(ctx context.Context, userID string, groupID uuid.UUID, topic domain.NotificationTopicKey, channel domain.NotificationChannel) error {
	rows, err := r.DeleteGroupNotificationSubscription(ctx, sqlc.DeleteGroupNotificationSubscriptionParams{
		UserID:    userID,
		GroupID:   groupID,
		EventType: string(topic),
		Channel:   sqlc.NotificationChannel(channel),
	})
	if err != nil {
		return fmt.Errorf("delete group notification subscription: %w", err)
	}
	if rows == 0 {
		return apperror.Default(apperror.CodeNotFound)
	}
	return nil
}

func toDomainTopic(key, label, description string, scope sqlc.NotificationScope, permissions []string) (domain.NotificationTopic, error) {
	parsedKey, err := domain.NewNotificationTopicKey(key)
	if err != nil {
		return domain.NotificationTopic{}, fmt.Errorf("validate notification topic key: %w", err)
	}
	return domain.NotificationTopic{
		Key:                 parsedKey,
		Label:               label,
		Description:         description,
		Scope:               domain.NotificationScope(scope),
		RequiredPermissions: permissions,
	}, nil
}
