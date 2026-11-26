package notifications

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// Build resolves event routes, finds eligible recipients, and renders notifications.
//
// It skips inactive or incomplete recipients and deduplicates notifications by
// recipient, event, and channel. It returns an error when the event, topic
// configuration, or recipient lookup cannot be resolved.
func (s *Service) Build(ctx context.Context, input domain.NotificationEvent) ([]domain.Notification, error) {
	var notification domain.Notification
	var recipients []domain.NotificationRecipient

	switch event := input.(type) {
	case domain.ConversionRequested:
		n, r, err := s.conversionRequested(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.MembershipRequested:
		n, r, err := s.membershipRequested(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.MembershipAccepted:
		n, r, err := s.membershipAccepted(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.MembershipRejectedEvent:
		n, r, err := s.membershipRejected(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.RoleRequested:
		n, r, err := s.roleRequested(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.RoleRequestAcceptedEvent:
		n, r, err := s.roleRequestAccepted(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.RoleRequestRejectedEvent:
		n, r, err := s.roleRequestRejected(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.RoleAssigned:
		n, r, err := s.roleAssigned(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.PermissionAssigned:
		n, r, err := s.permissionAssigned(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.ResourceSubmitted:
		n, r, err := s.resourceSubmitted(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.ResourcePublished:
		n, r, err := s.resourcePublished(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.ResourceRejected:
		n, r, err := s.resourceRejected(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	case domain.ResourceUnpublished:
		n, r, err := s.resourceUnpublished(ctx, event)
		if err != nil {
			return nil, err
		}

		notification = n
		recipients = r
	default:
		return nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("unsupported notification event %T", input))
	}

	notifications := make([]domain.Notification, 0, len(recipients))
	seen := make(map[string]struct{}, len(recipients))
	for _, recipient := range recipients {
		if recipient.Status != domain.UserStatusActive {
			continue
		}
		if recipient.ID == "" {
			s.log.DebugContext(ctx, "skip notification recipient with missing ID")
			continue
		}
		if recipient.Email == "" {
			s.log.DebugContext(ctx, "skip notification recipient with missing email", "user_id", recipient.ID)
			continue
		}

		if _, ok := seen[recipient.ID]; ok {
			continue
		}
		seen[recipient.ID] = struct{}{}

		notification.TargetUserID = recipient.ID
		notification.TargetEmail = recipient.Email
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

//go:embed *.html
var templateFiles embed.FS

var templates = map[domain.NotificationEventKind]string{
	domain.NotificationEventConversionRequested: "conversion.html",
	domain.NotificationEventMembershipRequested: "membership_requested.html",
	domain.NotificationEventMembershipAccepted:  "membership_accepted.html",
	domain.NotificationEventMembershipRejected:  "membership_rejected.html",
	domain.NotificationEventRoleRequested:       "role_requested.html",
	domain.NotificationEventRoleRequestAccepted: "role_request_accepted.html",
	domain.NotificationEventRoleRequestRejected: "role_request_rejected.html",
	domain.NotificationEventRoleAssigned:        "role_assigned.html",
	domain.NotificationEventPermissionAssigned:  "permission_assigned.html",
	domain.NotificationEventResourceSubmitted:   "resource_submitted.html",
	domain.NotificationEventResourcePublished:   "resource_published.html",
	domain.NotificationEventResourceRejected:    "resource_rejected.html",
	domain.NotificationEventResourceUnpublished: "resource_unpublished.html",
}

func (s *Service) render(ctx context.Context, event domain.NotificationEvent) (string, error) {
	name, ok := templates[event.NotificationKind()]
	if !ok {
		return "", fmt.Errorf("unsupported notification event kind %q", event.NotificationKind())
	}

	templates, err := template.New("").Funcs(template.FuncMap{
		"pageURL":          s.pageURL,
		"resourceURL":      s.resourceURL,
		"groupRequestsURL": s.groupRequestsURL,
		"userLabel":        func(id domain.Actor) string { return s.userLabel(ctx, id) },
		"subjectLabel":     func(id string) string { return s.userLabel(ctx, domain.Actor(id)) },
		"groupLabel":       func(id uuid.UUID) string { return s.groupLabel(ctx, id) },
	}).ParseFS(templateFiles, "*.html")
	if err != nil {
		return "", fmt.Errorf("parse notification templates: %w", err)
	}

	var body strings.Builder
	if err := templates.ExecuteTemplate(&body, name, event); err != nil {
		return "", fmt.Errorf("execute notification template %q: %w", name, err)
	}

	return body.String(), nil
}

func (s *Service) groupLabel(ctx context.Context, id uuid.UUID) string {
	group, err := s.store.GetNotificationGroupContext(ctx, id)
	if err != nil {
		s.log.WarnContext(ctx, "resolve notification group failed", "group_id", id, "error", err)
		return fmt.Sprintf("Unknown group (concept: unknown, group ID: %s)", id)
	}

	name := strings.TrimSpace(group.Name)
	if name == "" {
		name = "Unknown group"
	}

	concept := strings.TrimSpace(group.Concept)
	if concept == "" {
		concept = "unknown"
	}

	return fmt.Sprintf("%s (concept: %s, group ID: %s)", name, concept, id)
}

func (s *Service) userLabel(ctx context.Context, id domain.Actor) string {
	user, err := s.store.GetNotificationUserContext(ctx, string(id))
	if err != nil {
		s.log.WarnContext(ctx, "resolve notification user failed", "user_id", id, "error", err)
		return fmt.Sprintf("Unknown name (user ID: %s)", id)
	}

	name := strings.TrimSpace(strings.TrimSpace(user.GivenName) + " " + strings.TrimSpace(user.FamilyName))
	if name == "" {
		name = "Unknown name"
	}

	if user.Email != "" {
		return fmt.Sprintf("%s (%s, user ID: %s)", name, user.Email, id)
	}

	return fmt.Sprintf("%s (user ID: %s)", name, id)
}

func (s *Service) pageURL(path string) string {
	if s.publicURL == "" {
		return ""
	}

	return s.publicURL + path
}

func (s *Service) resourceURL(resource domain.NotificationResource) string {
	section := ""
	switch resource.Type {
	case domain.Dataset:
		section = "data"
	case domain.SoftwareApplication:
		section = "software"
	case domain.SoftwareSourceCode:
		section = "facilities"
	default:
		return ""
	}

	if s.publicURL == "" || section == "" || resource.ResourceID == uuid.Nil || resource.VersionID == uuid.Nil {
		return ""
	}

	return fmt.Sprintf("%s/%s/%s/%s", s.publicURL, section, resource.ResourceID, resource.VersionID)
}

func (s *Service) groupRequestsURL(groupID uuid.UUID, concept domain.Concept) string {
	section := strings.ToLower(concept.Name)
	if s.publicURL == "" || groupID == uuid.Nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/manage-group?groupId=%s&panel=membership-requests", s.publicURL, section, groupID)
}
