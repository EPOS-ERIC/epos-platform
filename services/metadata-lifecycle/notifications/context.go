package notifications

import (
	"context"
	"net/url"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type notificationContext struct {
	Actor   *notificationUser
	Subject *notificationUser
	Group   *notificationGroup
	Links   []notificationLink
}

type notificationUser struct {
	ID         string
	Email      string
	GivenName  string
	FamilyName string
}

type notificationGroup struct {
	ID      uuid.UUID
	Name    string
	Concept string
}

type notificationLink struct {
	Label string
	URL   string
}

// resolveContext enriches notification IDs with user and group details.
func (s *Service) resolveContext(ctx context.Context, event domain.NotificationEvent) notificationContext {
	resolved := notificationContext{}

	loadUser := func(userID string) *notificationUser {
		if userID == "" {
			return nil
		}

		user, err := s.store.GetNotificationUserContext(ctx, userID)
		if err != nil {
			s.log.WarnContext(ctx, "load notification user context failed", "user_id", userID, "error", err)
			user = domain.UserSummary{ID: userID}
		}
		if user.ID == "" {
			user.ID = userID
		}

		result := &notificationUser{
			ID:         user.ID,
			Email:      user.Email,
			GivenName:  user.GivenName,
			FamilyName: user.FamilyName,
		}
		return result
	}

	actorID := string(event.ActorID)
	resolved.Actor = loadUser(actorID)
	subjectID := event.SubjectID
	if subjectID == "" && event.Resource != nil && event.Resource.SubmittedBy != nil {
		subjectID = *event.Resource.SubmittedBy
	}
	if subjectID != "" && subjectID == actorID {
		resolved.Subject = resolved.Actor
	} else {
		resolved.Subject = loadUser(subjectID)
	}

	if event.GroupID != nil {
		group := &notificationGroup{ID: *event.GroupID}
		value, err := s.store.GetNotificationGroupContext(ctx, *event.GroupID)
		if err != nil {
			s.log.WarnContext(ctx, "load notification group context failed", "group_id", *event.GroupID, "error", err)
		} else {
			group.Name = value.Name
			group.Concept = value.Concept
		}
		resolved.Group = group
	}

	return resolved
}

// notificationLinks builds Gateway URLs for resources referenced by a notification.
func (s *Service) notificationLinks(event domain.NotificationEvent, resolved notificationContext) []notificationLink {
	baseURL := s.publicURL
	if baseURL == "" {
		return nil
	}

	links := make([]notificationLink, 0, 3)
	if resolved.Group != nil && resolved.Group.ID != uuid.Nil {
		links = append(links, notificationLink{
			Label: "View group",
			URL:   baseURL + "/v1/groups/" + url.PathEscape(resolved.Group.ID.String()),
		})
	}
	if resolved.Subject != nil && resolved.Subject.ID != "" {
		links = append(links, notificationLink{
			Label: notificationSubjectLinkLabel(event.Kind),
			URL:   baseURL + "/v1/users/" + url.PathEscape(resolved.Subject.ID),
		})
	}
	if event.Resource != nil && event.Resource.ResourceID != uuid.Nil && event.Resource.VersionID != uuid.Nil {
		links = append(links, notificationLink{
			Label: "View resource",
			URL: baseURL + "/v1/resources/" +
				url.PathEscape(string(event.Resource.Type)) + "/" +
				event.Resource.ResourceID.String() + "/versions/" +
				event.Resource.VersionID.String(),
		})
	}

	return links
}

func notificationSubjectLinkLabel(kind domain.NotificationEventKind) string {
	switch kind {
	case domain.NotificationEventMembershipRequested,
		domain.NotificationEventMembershipAccepted,
		domain.NotificationEventMembershipRejected,
		domain.NotificationEventRoleRequested,
		domain.NotificationEventRoleRequestAccepted,
		domain.NotificationEventRoleRequestRejected:
		return "View requester"
	case domain.NotificationEventRoleAssigned, domain.NotificationEventPermissionAssigned:
		return "View recipient"
	case domain.NotificationEventResourceSubmitted,
		domain.NotificationEventResourcePublished,
		domain.NotificationEventResourceRejected,
		domain.NotificationEventResourceUnpublished:
		return "View submitter"
	default:
		return "View user"
	}
}
