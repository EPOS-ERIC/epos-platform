package notifications

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type notificationRoute struct {
	topicKey domain.NotificationTopicKey
	scope    domain.NotificationScope
}

// Build resolves event routes, finds eligible recipients, and renders notifications.
//
// It skips inactive or incomplete recipients and deduplicates notifications by
// recipient, event, and channel. It returns an error when the event, topic
// configuration, or recipient lookup cannot be resolved.
func (s *Service) Build(ctx context.Context, event domain.NotificationEvent) ([]domain.Notification, error) {
	routes, err := routesForEvent(event)
	if err != nil {
		return nil, err
	}

	topics, err := s.store.ListTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification topics: %w", err)
	}

	topicByKey := make(map[domain.NotificationTopicKey]domain.NotificationTopic, len(topics))
	for _, topic := range topics {
		topicByKey[topic.Key] = topic
	}

	resolved := s.resolveContext(ctx, event)
	resolved.Links = s.notificationLinks(event, resolved)
	subject, body, err := renderNotificationEmail(event, resolved)
	if err != nil {
		return nil, fmt.Errorf("render notification email: %w", err)
	}
	notifications := make([]domain.Notification, 0)
	seen := make(map[string]struct{})
	for _, route := range routes {
		topic, ok := topicByKey[route.topicKey]
		if !ok || topic.Scope != route.scope {
			return nil, fmt.Errorf("notification topic %q is missing or has wrong scope", route.topicKey)
		}

		var resourceType domain.ResourceType
		var resourceID, versionID *uuid.UUID
		if event.Resource != nil {
			resourceType = event.Resource.Type
			resourceID = new(event.Resource.ResourceID)
			versionID = new(event.Resource.VersionID)
		}

		var recipients []domain.NotificationRecipient
		switch route.scope {
		case domain.NotificationScopeDirect:
			recipient, err := s.directRecipient(ctx, event)
			if err != nil {
				return nil, fmt.Errorf("get direct notification recipient: %w", err)
			}

			recipients = []domain.NotificationRecipient{recipient}
		case domain.NotificationScopeGroup:
			if event.GroupID == nil {
				return nil, fmt.Errorf("group notification event %q has no group", event.Kind)
			}

			recipients, err = s.store.GroupRecipients(ctx, *event.GroupID, route.topicKey)
			if err != nil {
				return nil, fmt.Errorf("list group notification recipients: %w", err)
			}
		case domain.NotificationScopeGlobal:
			recipients, err = s.store.GlobalRecipients(ctx, route.topicKey)
			if err != nil {
				return nil, fmt.Errorf("list global notification recipients: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown notification scope %q", route.scope)
		}

		for _, recipient := range recipients {
			if recipient.Status != domain.UserStatusActive || recipient.ID == "" || recipient.Email == "" {
				continue
			}

			key := recipient.ID + "\x00" + string(event.Kind) + "\x00" + string(domain.NotificationChannelEmail)
			if _, ok := seen[key]; ok {
				continue
			}

			seen[key] = struct{}{}
			notifications = append(notifications, domain.Notification{
				EventKind:    event.Kind,
				GroupID:      event.GroupID,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				VersionID:    versionID,
				AccessID:     event.AccessID,
				TargetUserID: recipient.ID,
				TargetEmail:  recipient.Email,
				Subject:      subject,
				Body:         body,
				Channel:      domain.NotificationChannelEmail,
			})
		}
	}

	return notifications, nil
}

func (s *Service) directRecipient(ctx context.Context, event domain.NotificationEvent) (domain.NotificationRecipient, error) {
	target := event.TargetUserID
	if target == "" && event.Resource != nil && event.Resource.SubmittedBy != nil {
		target = *event.Resource.SubmittedBy
	}
	if target == "" {
		return domain.NotificationRecipient{}, fmt.Errorf("direct notification event %q has no target", event.Kind)
	}

	recipient, err := s.store.Recipient(ctx, target)
	if err != nil {
		return domain.NotificationRecipient{}, fmt.Errorf("get notification recipient %q: %w", target, err)
	}

	return recipient, nil
}

func routesForEvent(event domain.NotificationEvent) ([]notificationRoute, error) {
	direct := func(key domain.NotificationTopicKey) []notificationRoute {
		return []notificationRoute{{topicKey: key, scope: domain.NotificationScopeDirect}}
	}

	groupGlobal := func(groupKey, globalKey domain.NotificationTopicKey) []notificationRoute {
		return []notificationRoute{
			{topicKey: groupKey, scope: domain.NotificationScopeGroup},
			{topicKey: globalKey, scope: domain.NotificationScopeGlobal},
		}
	}

	switch event.Kind {
	case domain.NotificationEventMembershipRequested:
		return groupGlobal(domain.NotificationTopicGroupMembershipRequested, domain.NotificationTopicGlobalMembershipRequested), nil
	case domain.NotificationEventMembershipAccepted:
		return direct(domain.NotificationTopicDirectMembershipAccepted), nil
	case domain.NotificationEventMembershipRejected:
		return direct(domain.NotificationTopicDirectMembershipRejected), nil
	case domain.NotificationEventRoleRequested:
		return groupGlobal(domain.NotificationTopicGroupRoleRequestRequested, domain.NotificationTopicGlobalRoleRequestRequested), nil
	case domain.NotificationEventRoleRequestAccepted:
		return direct(domain.NotificationTopicDirectRoleRequestAccepted), nil
	case domain.NotificationEventRoleRequestRejected:
		return direct(domain.NotificationTopicDirectRoleRequestRejected), nil
	case domain.NotificationEventRoleAssigned:
		return direct(domain.NotificationTopicDirectRoleAssigned), nil
	case domain.NotificationEventPermissionAssigned:
		return direct(domain.NotificationTopicDirectPermissionAssigned), nil
	case domain.NotificationEventResourceSubmitted:
		return groupGlobal(domain.NotificationTopicGroupResourceSubmitted, domain.NotificationTopicGlobalResourceSubmitted), nil
	case domain.NotificationEventResourcePublished:
		return append(direct(domain.NotificationTopicDirectResourcePublished), groupGlobal(domain.NotificationTopicGroupResourcePublished, domain.NotificationTopicGlobalResourcePublished)...), nil
	case domain.NotificationEventResourceRejected:
		return direct(domain.NotificationTopicDirectResourceRejected), nil
	case domain.NotificationEventResourceUnpublished:
		return direct(domain.NotificationTopicDirectResourceUnpublished), nil
	default:
		return nil, fmt.Errorf("unsupported notification event kind %q", event.Kind)
	}
}
