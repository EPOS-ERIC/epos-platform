//nolint:dupl // Different events might have similar handling but should be different
package notifications

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func (s *Service) conversionRequested(ctx context.Context, event domain.ConversionRequested) (domain.Notification, []domain.NotificationRecipient, error) {
	recipients, err := s.store.GlobalRecipients(ctx, domain.NotificationTopicGlobalConversionRequested)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list global conversion notification recipients: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render conversion notification: %w", err))
	}

	notification := domain.Notification{
		EventKind: domain.NotificationEventConversionRequested,
		Subject:   fmt.Sprintf("Conversion requested: %s", event.Resource.Name),
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}

	return notification, recipients, nil
}

func (s *Service) membershipRequested(ctx context.Context, event domain.MembershipRequested) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("membership request notification has no valid group"))
	}

	recipients, err := s.store.GroupRecipients(ctx, event.GroupID, domain.NotificationTopicGroupMembershipRequested)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list group membership request notification recipients: %w", err))
	}

	globalRecipients, err := s.store.GlobalRecipients(ctx, domain.NotificationTopicGlobalMembershipRequested)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list global membership request notification recipients: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render membership request notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventMembershipRequested,
		Subject:   "Membership requested",
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, append(recipients, globalRecipients...), nil
}

func (s *Service) membershipAccepted(ctx context.Context, event domain.MembershipAccepted) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("membership acceptance notification has no valid group"))
	}

	if event.TargetUserID == "" {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("membership acceptance notification has no target user"))
	}

	recipient, err := s.store.Recipient(ctx, event.TargetUserID)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get membership acceptance notification recipient: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render membership acceptance notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventMembershipAccepted,
		Subject:   "Membership accepted",
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, []domain.NotificationRecipient{recipient}, nil
}

func (s *Service) membershipRejected(ctx context.Context, event domain.MembershipRejectedEvent) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("membership rejection notification has no valid group"))
	}

	if event.TargetUserID == "" {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("membership rejection notification has no target user"))
	}

	recipient, err := s.store.Recipient(ctx, event.TargetUserID)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get membership rejection notification recipient: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render membership rejection notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventMembershipRejected,
		Subject:   "Membership rejected",
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, []domain.NotificationRecipient{recipient}, nil
}

func (s *Service) roleRequested(ctx context.Context, event domain.RoleRequested) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("role request notification has no valid group"))
	}

	recipients, err := s.store.GroupRecipients(ctx, event.GroupID, domain.NotificationTopicGroupRoleRequestRequested)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list group role request notification recipients: %w", err))
	}

	globalRecipients, err := s.store.GlobalRecipients(ctx, domain.NotificationTopicGlobalRoleRequestRequested)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list global role request notification recipients: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render role request notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventRoleRequested,
		Subject:   "Role requested",
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, append(recipients, globalRecipients...), nil
}

func (s *Service) roleRequestAccepted(ctx context.Context, event domain.RoleRequestAcceptedEvent) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("role request acceptance notification has no valid group"))
	}

	if event.TargetUserID == "" {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("role request acceptance notification has no target user"))
	}

	recipient, err := s.store.Recipient(ctx, event.TargetUserID)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get role request acceptance notification recipient: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render role request acceptance notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventRoleRequestAccepted,
		Subject:   "Role request accepted",
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, []domain.NotificationRecipient{recipient}, nil
}

func (s *Service) roleRequestRejected(ctx context.Context, event domain.RoleRequestRejectedEvent) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("role request rejection notification has no valid group"))
	}

	if event.TargetUserID == "" {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("role request rejection notification has no target user"))
	}

	recipient, err := s.store.Recipient(ctx, event.TargetUserID)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get role request rejection notification recipient: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render role request rejection notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventRoleRequestRejected,
		Subject:   "Role request rejected",
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, []domain.NotificationRecipient{recipient}, nil
}

func (s *Service) roleAssigned(ctx context.Context, event domain.RoleAssigned) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.TargetUserID == "" {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("role assignment notification has no target user"))
	}

	recipient, err := s.store.Recipient(ctx, event.TargetUserID)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get role assignment notification recipient: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render role assignment notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventRoleAssigned,
		Subject:   "Role assigned",
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, []domain.NotificationRecipient{recipient}, nil
}

func (s *Service) permissionAssigned(ctx context.Context, event domain.PermissionAssigned) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.TargetUserID == "" {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("permission assignment notification has no target user"))
	}

	recipient, err := s.store.Recipient(ctx, event.TargetUserID)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get permission assignment notification recipient: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render permission assignment notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventPermissionAssigned,
		Subject:   "Permission assigned",
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, []domain.NotificationRecipient{recipient}, nil
}

func (s *Service) resourceSubmitted(ctx context.Context, event domain.ResourceSubmitted) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("resource submission notification has no valid group"))
	}

	recipients, err := s.store.GroupRecipients(ctx, event.GroupID, domain.NotificationTopicGroupResourceSubmitted)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list group resource submission notification recipients: %w", err))
	}

	globalRecipients, err := s.store.GlobalRecipients(ctx, domain.NotificationTopicGlobalResourceSubmitted)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list global resource submission notification recipients: %w", err))
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render resource submission notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventResourceSubmitted,
		Subject:   fmt.Sprintf("Resource submitted: %s", event.Resource.Name),
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, append(recipients, globalRecipients...), nil
}

func (s *Service) resourcePublished(ctx context.Context, event domain.ResourcePublished) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("resource publication notification has no valid group"))
	}

	var recipients []domain.NotificationRecipient
	if event.Resource.SubmittedBy != nil && *event.Resource.SubmittedBy != "" {
		recipient, err := s.store.Recipient(ctx, *event.Resource.SubmittedBy)
		if err != nil {
			return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get resource publication notification recipient: %w", err))
		}
		recipients = append(recipients, recipient)
	}

	groupRecipients, err := s.store.GroupRecipients(ctx, event.GroupID, domain.NotificationTopicGroupResourcePublished)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list group resource publication notification recipients: %w", err))
	}
	recipients = append(recipients, groupRecipients...)

	globalRecipients, err := s.store.GlobalRecipients(ctx, domain.NotificationTopicGlobalResourcePublished)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("list global resource publication notification recipients: %w", err))
	}
	recipients = append(recipients, globalRecipients...)

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render resource publication notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventResourcePublished,
		Subject:   fmt.Sprintf("Resource published: %s", event.Resource.Name),
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, recipients, nil
}

func (s *Service) resourceRejected(ctx context.Context, event domain.ResourceRejected) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("resource rejection notification has no valid group"))
	}

	var recipients []domain.NotificationRecipient
	if event.Resource.SubmittedBy != nil && *event.Resource.SubmittedBy != "" {
		recipient, err := s.store.Recipient(ctx, *event.Resource.SubmittedBy)
		if err != nil {
			return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get resource rejection notification recipient: %w", err))
		}
		recipients = append(recipients, recipient)
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render resource rejection notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventResourceRejected,
		Subject:   fmt.Sprintf("Resource rejected: %s", event.Resource.Name),
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, recipients, nil
}

func (s *Service) resourceUnpublished(ctx context.Context, event domain.ResourceUnpublished) (domain.Notification, []domain.NotificationRecipient, error) {
	if event.GroupID == uuid.Nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("resource unpublication notification has no valid group"))
	}

	var recipients []domain.NotificationRecipient
	if event.Resource.SubmittedBy != nil && *event.Resource.SubmittedBy != "" {
		recipient, err := s.store.Recipient(ctx, *event.Resource.SubmittedBy)
		if err != nil {
			return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("get resource unpublication notification recipient: %w", err))
		}
		recipients = append(recipients, recipient)
	}

	body, err := s.render(ctx, event)
	if err != nil {
		return domain.Notification{}, nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("render resource unpublication notification: %w", err))
	}

	return domain.Notification{
		EventKind: domain.NotificationEventResourceUnpublished,
		Subject:   fmt.Sprintf("Resource unpublished: %s", event.Resource.Name),
		Body:      body,
		Channel:   domain.NotificationChannelEmail,
	}, recipients, nil
}
