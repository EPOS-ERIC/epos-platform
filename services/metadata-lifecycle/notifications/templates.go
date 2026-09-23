package notifications

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type notificationTemplateData struct {
	Event   domain.NotificationEvent
	Context notificationContext
}

//go:embed notification.html
var notificationTemplateFiles embed.FS

var notificationHTMLTemplate = template.Must(template.New("notification.html").Funcs(template.FuncMap{
	"userLabel":    notificationUserLabel,
	"sameUser":     notificationUsersEqual,
	"groupName":    notificationGroupName,
	"resourceName": notificationResourceName,
	"subject":      notificationSubject,
}).ParseFS(notificationTemplateFiles, "notification.html"))

func renderNotificationEmail(event domain.NotificationEvent, context notificationContext) (string, string, error) {
	var body bytes.Buffer
	if err := notificationHTMLTemplate.Execute(&body, notificationTemplateData{Event: event, Context: context}); err != nil {
		return "", "", fmt.Errorf("execute notification template: %w", err)
	}
	return notificationSubject(event), body.String(), nil
}

func notificationUserName(user *notificationUser) string {
	if user == nil {
		return "A user"
	}
	name := strings.TrimSpace(user.GivenName + " " + user.FamilyName)
	if name != "" {
		return name
	}
	if user.Email != "" {
		return user.Email
	}
	if user.ID != "" {
		return user.ID
	}
	return "A user"
}

func notificationUserLabel(user *notificationUser) string {
	name := notificationUserName(user)
	if user == nil || user.Email == "" || user.Email == name {
		return name
	}
	return name + " <" + user.Email + ">"
}

func notificationUsersEqual(left, right *notificationUser) bool {
	if left == right {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	if left.ID == "" {
		return false
	}
	return left.ID == right.ID
}

func notificationGroupName(group *notificationGroup) string {
	if group == nil {
		return "the group"
	}
	if group.Name != "" {
		return group.Name
	}
	if group.ID != uuid.Nil {
		return group.ID.String()
	}
	return "the group"
}

func notificationResourceName(resource *domain.NotificationResource) string {
	if resource == nil {
		return "the resource"
	}
	if resource.Name != "" {
		return resource.Name
	}
	if resource.Type != "" {
		return string(resource.Type)
	}
	return "the resource"
}

func notificationSubject(event domain.NotificationEvent) string {
	switch event.Kind {
	case domain.NotificationEventMembershipRequested:
		return "New group membership request"
	case domain.NotificationEventMembershipAccepted:
		return "Group membership request accepted"
	case domain.NotificationEventMembershipRejected:
		return "Group membership request rejected"
	case domain.NotificationEventRoleRequested:
		return "New group role request"
	case domain.NotificationEventRoleRequestAccepted:
		return "Group role request accepted"
	case domain.NotificationEventRoleRequestRejected:
		return "Group role request rejected"
	case domain.NotificationEventRoleAssigned:
		return "Group role assigned"
	case domain.NotificationEventPermissionAssigned:
		return "Group permission assigned"
	case domain.NotificationEventResourceSubmitted:
		return "Resource submitted for review"
	case domain.NotificationEventResourcePublished:
		return "Resource published"
	case domain.NotificationEventResourceRejected:
		return "Resource rejected"
	case domain.NotificationEventResourceUnpublished:
		return "Resource unpublished"
	default:
		return "Metadata lifecycle notification"
	}
}
