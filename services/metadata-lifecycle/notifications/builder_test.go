package notifications

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func TestRoutesForResourcePublishedIncludesMandatoryAndOptInRoutes(t *testing.T) {
	routes, err := routesForEvent(domain.NotificationEvent{Kind: domain.NotificationEventResourcePublished})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 3 {
		t.Fatalf("route count = %d, want 3", len(routes))
	}
	if routes[0].topicKey != domain.NotificationTopicDirectResourcePublished || routes[0].scope != domain.NotificationScopeDirect {
		t.Fatalf("first route = %#v, want direct route", routes[0])
	}
}

func TestRenderEventIncludesRejectionNote(t *testing.T) {
	note := "missing license"
	_, body, err := renderNotificationEmail(domain.NotificationEvent{
		Kind: domain.NotificationEventResourceRejected,
		Resource: &domain.NotificationResource{
			Name:          "Dataset A",
			RejectionNote: &note,
		},
	}, notificationContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "Rejection note:</strong> missing license") {
		t.Fatalf("body = %q, missing rejection note", body)
	}
}

func TestRenderNotificationEmailUsesContextAndEscapesHTML(t *testing.T) {
	groupID := uuid.New()
	subject, body, err := renderNotificationEmail(domain.NotificationEvent{
		Kind: domain.NotificationEventMembershipRequested,
	}, notificationContext{
		Subject: &notificationUser{
			GivenName: "<Ada>",
			Email:     "ada@example.org",
		},
		Group: &notificationGroup{
			ID:   groupID,
			Name: "<Group>",
		},
		Links: []notificationLink{{Label: "Group", URL: "https://gateway.example.org/v1/groups/" + groupID.String()}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if subject != "New group membership request" {
		t.Fatalf("subject = %q", subject)
	}
	if !strings.Contains(body, "&lt;Ada&gt; &lt;ada@example.org&gt; requested to join &lt;Group&gt;") {
		t.Fatalf("body = %q, missing escaped context", body)
	}
	if !strings.Contains(body, "<strong>Requester:</strong> &lt;Ada&gt; &lt;ada@example.org&gt;") {
		t.Fatalf("body = %q, missing requester", body)
	}
	if strings.Contains(body, "<Ada>") || strings.Contains(body, "<Group>") {
		t.Fatalf("body = %q, contains unescaped context", body)
	}
	if !strings.Contains(body, "href=\"https://gateway.example.org/v1/groups/"+groupID.String()+"\"") {
		t.Fatalf("body = %q, missing group link", body)
	}
}

func TestRenderNotificationEmailUsesEventSpecificWording(t *testing.T) {
	actor := &notificationUser{
		ID:         "actor",
		Email:      "alice@example.org",
		GivenName:  "Alice",
		FamilyName: "Admin",
	}
	subject := &notificationUser{
		ID:         "subject",
		Email:      "bob@example.org",
		GivenName:  "Bob",
		FamilyName: "Builder",
	}
	group := &notificationGroup{Name: "Jupyter Notebooks"}
	resource := &domain.NotificationResource{Name: "Dataset A"}

	tests := []struct {
		name  string
		event domain.NotificationEvent
		want  []string
	}{
		{
			name:  "membership requested",
			event: domain.NotificationEvent{Kind: domain.NotificationEventMembershipRequested},
			want: []string{
				"Bob Builder &lt;bob@example.org&gt; requested to join Jupyter Notebooks.",
				"<strong>Requester:</strong>",
			},
		},
		{
			name:  "membership accepted",
			event: domain.NotificationEvent{Kind: domain.NotificationEventMembershipAccepted},
			want: []string{
				"Bob Builder &lt;bob@example.org&gt;'s request to join Jupyter Notebooks was approved by Alice Admin &lt;alice@example.org&gt;.",
				"<strong>Decided by:</strong>",
			},
		},
		{
			name:  "membership rejected",
			event: domain.NotificationEvent{Kind: domain.NotificationEventMembershipRejected},
			want: []string{
				"Bob Builder &lt;bob@example.org&gt;'s request to join Jupyter Notebooks was rejected by Alice Admin &lt;alice@example.org&gt;.",
				"<strong>Decided by:</strong>",
			},
		},
		{
			name:  "role requested",
			event: domain.NotificationEvent{Kind: domain.NotificationEventRoleRequested, Role: "Reviewer"},
			want: []string{
				"Bob Builder &lt;bob@example.org&gt; requested the <strong>Reviewer</strong> role in Jupyter Notebooks.",
				"<strong>Requester:</strong>",
			},
		},
		{
			name:  "role request accepted",
			event: domain.NotificationEvent{Kind: domain.NotificationEventRoleRequestAccepted, Role: "Reviewer"},
			want: []string{
				"Bob Builder &lt;bob@example.org&gt;'s request for the <strong>Reviewer</strong> role in Jupyter Notebooks was approved by Alice Admin &lt;alice@example.org&gt;.",
				"<strong>Decided by:</strong>",
			},
		},
		{
			name:  "role request rejected",
			event: domain.NotificationEvent{Kind: domain.NotificationEventRoleRequestRejected, Role: "Reviewer"},
			want: []string{
				"Bob Builder &lt;bob@example.org&gt;'s request for the <strong>Reviewer</strong> role in Jupyter Notebooks was rejected by Alice Admin &lt;alice@example.org&gt;.",
				"<strong>Decided by:</strong>",
			},
		},
		{
			name:  "role assigned",
			event: domain.NotificationEvent{Kind: domain.NotificationEventRoleAssigned, Role: "Reviewer"},
			want: []string{
				"Alice Admin &lt;alice@example.org&gt; assigned the <strong>Reviewer</strong> role in Jupyter Notebooks to Bob Builder &lt;bob@example.org&gt;.",
				"<strong>Assigned to:</strong>",
			},
		},
		{
			name:  "permission assigned",
			event: domain.NotificationEvent{Kind: domain.NotificationEventPermissionAssigned, Permission: "read"},
			want: []string{
				"Alice Admin &lt;alice@example.org&gt; granted the <strong>read</strong> permission in Jupyter Notebooks to Bob Builder &lt;bob@example.org&gt;.",
				"<strong>Granted to:</strong>",
			},
		},
		{
			name:  "resource submitted",
			event: domain.NotificationEvent{Kind: domain.NotificationEventResourceSubmitted, Resource: resource},
			want: []string{
				"Alice Admin &lt;alice@example.org&gt; submitted Dataset A for review.",
				"<strong>Performed by:</strong>",
				"<strong>Submitted by:</strong>",
			},
		},
		{
			name:  "resource published",
			event: domain.NotificationEvent{Kind: domain.NotificationEventResourcePublished, Resource: resource},
			want: []string{
				"Alice Admin &lt;alice@example.org&gt; published Dataset A.",
				"<strong>Performed by:</strong>",
				"<strong>Submitted by:</strong>",
			},
		},
		{
			name:  "resource rejected",
			event: domain.NotificationEvent{Kind: domain.NotificationEventResourceRejected, Resource: resource},
			want: []string{
				"Alice Admin &lt;alice@example.org&gt; rejected Dataset A.",
				"<strong>Performed by:</strong>",
				"<strong>Submitted by:</strong>",
			},
		},
		{
			name:  "resource unpublished",
			event: domain.NotificationEvent{Kind: domain.NotificationEventResourceUnpublished, Resource: resource},
			want: []string{
				"Alice Admin &lt;alice@example.org&gt; unpublished Dataset A.",
				"<strong>Performed by:</strong>",
				"<strong>Submitted by:</strong>",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, body, err := renderNotificationEmail(test.event, notificationContext{
				Actor:   actor,
				Subject: subject,
				Group:   group,
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.want {
				if !strings.Contains(body, want) {
					t.Errorf("body = %q, missing %q", body, want)
				}
			}
			if strings.Contains(body, "<strong>Actor:</strong>") || strings.Contains(body, "<strong>User:</strong>") {
				t.Fatalf("body = %q, contains generic participant labels", body)
			}
		})
	}
}

func TestRenderNotificationEmailDoesNotRepeatResourceSubmitter(t *testing.T) {
	actor := &notificationUser{ID: "actor", Email: "alice@example.org", GivenName: "Alice"}
	_, body, err := renderNotificationEmail(domain.NotificationEvent{
		Kind:     domain.NotificationEventResourceSubmitted,
		Resource: &domain.NotificationResource{Name: "Dataset A"},
	}, notificationContext{Actor: actor, Subject: actor})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "<strong>Submitted by:</strong>") {
		t.Fatalf("body = %q, repeated resource submitter", body)
	}
}

func TestNotificationUserLabel(t *testing.T) {
	tests := []struct {
		name string
		user *notificationUser
		want string
	}{
		{name: "nil", want: "A user"},
		{name: "name and email", user: &notificationUser{GivenName: "Ada", FamilyName: "Lovelace", Email: "ada@example.org"}, want: "Ada Lovelace <ada@example.org>"},
		{name: "email only", user: &notificationUser{Email: "ada@example.org"}, want: "ada@example.org"},
		{name: "id only", user: &notificationUser{ID: "user-1"}, want: "user-1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := notificationUserLabel(test.user); got != test.want {
				t.Fatalf("notificationUserLabel() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildAllowsMembershipEventWithoutResource(t *testing.T) {
	groupID := uuid.New()
	notifications, err := (&Service{store: builderTestStore{}}).Build(context.Background(), domain.NotificationEvent{
		Kind:    domain.NotificationEventMembershipRequested,
		GroupID: &groupID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(notifications) != 1 {
		t.Fatalf("notification count = %d, want 1", len(notifications))
	}
	if notifications[0].ResourceID != nil || notifications[0].VersionID != nil {
		t.Fatalf("resource identifiers = %v/%v, want nil", notifications[0].ResourceID, notifications[0].VersionID)
	}
}

func TestNotificationLinksUseGatewayAPI(t *testing.T) {
	groupID := uuid.New()
	resourceID := uuid.New()
	versionID := uuid.New()
	links := (&Service{publicURL: "https://gateway.example.org/environment/api"}).notificationLinks(domain.NotificationEvent{
		Kind:      domain.NotificationEventResourcePublished,
		SubjectID: "user/1",
		Resource: &domain.NotificationResource{
			Type:       domain.Dataset,
			ResourceID: resourceID,
			VersionID:  versionID,
		},
	}, notificationContext{
		Group:   &notificationGroup{ID: groupID},
		Subject: &notificationUser{ID: "user/1"},
	})

	if len(links) != 3 {
		t.Fatalf("link count = %d, want 3", len(links))
	}
	if links[0].URL != "https://gateway.example.org/environment/api/v1/groups/"+groupID.String() {
		t.Fatalf("group URL = %q", links[0].URL)
	}
	if links[0].Label != "View group" {
		t.Fatalf("group label = %q", links[0].Label)
	}
	if links[1].URL != "https://gateway.example.org/environment/api/v1/users/user%2F1" {
		t.Fatalf("user URL = %q", links[1].URL)
	}
	if links[1].Label != "View submitter" {
		t.Fatalf("user label = %q", links[1].Label)
	}
	wantResourceURL := "https://gateway.example.org/environment/api/v1/resources/dataset/" + resourceID.String() + "/versions/" + versionID.String()
	if links[2].URL != wantResourceURL {
		t.Fatalf("resource URL = %q, want %q", links[2].URL, wantResourceURL)
	}
	if links[2].Label != "View resource" {
		t.Fatalf("resource label = %q", links[2].Label)
	}
}

func TestResolveContextDeduplicatesUserLookup(t *testing.T) {
	groupID := uuid.New()
	store := &resolverTestStore{
		builderTestStore: builderTestStore{},
		user: domain.UserSummary{
			ID:        "actor",
			Email:     "actor@example.org",
			GivenName: "Ada",
		},
		userCalls: make(map[string]int),
	}
	service := &Service{store: store}

	resolved := service.resolveContext(context.Background(), domain.NotificationEvent{
		ActorID:   "actor",
		SubjectID: "actor",
		GroupID:   &groupID,
	})

	if store.userCalls["actor"] != 1 {
		t.Fatalf("user lookup count = %d, want 1", store.userCalls["actor"])
	}
	if store.groupCalls != 1 {
		t.Fatalf("group lookup count = %d, want 1", store.groupCalls)
	}
	if resolved.Actor == nil || resolved.Actor.GivenName != "Ada" || resolved.Subject != resolved.Actor {
		t.Fatalf("resolved users = %#v/%#v, want shared enriched user", resolved.Actor, resolved.Subject)
	}
	if resolved.Group == nil || resolved.Group.ID != groupID {
		t.Fatalf("resolved group = %#v", resolved.Group)
	}
}

func TestResolveContextFallsBackToIDs(t *testing.T) {
	groupID := uuid.New()
	service := &Service{
		store: &resolverTestStore{
			builderTestStore: builderTestStore{},
			userErr:          errors.New("user lookup failed"),
			groupErr:         errors.New("group lookup failed"),
			userCalls:        make(map[string]int),
		},
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	resolved := service.resolveContext(context.Background(), domain.NotificationEvent{
		ActorID:   "actor",
		SubjectID: "subject",
		GroupID:   &groupID,
	})

	if resolved.Actor == nil || resolved.Actor.ID != "actor" {
		t.Fatalf("actor fallback = %#v", resolved.Actor)
	}
	if resolved.Subject == nil || resolved.Subject.ID != "subject" {
		t.Fatalf("subject fallback = %#v", resolved.Subject)
	}
	if resolved.Group == nil || resolved.Group.ID != groupID {
		t.Fatalf("group fallback = %#v", resolved.Group)
	}
}

type builderTestStore struct{}

type resolverTestStore struct {
	builderTestStore
	user       domain.UserSummary
	userErr    error
	groupErr   error
	userCalls  map[string]int
	groupCalls int
}

func (s *resolverTestStore) GetNotificationUserContext(_ context.Context, userID string) (domain.UserSummary, error) {
	s.userCalls[userID]++
	if s.userErr != nil {
		return domain.UserSummary{}, s.userErr
	}
	return s.user, nil
}

func (s *resolverTestStore) GetNotificationGroupContext(_ context.Context, groupID uuid.UUID) (domain.Group, error) {
	s.groupCalls++
	if s.groupErr != nil {
		return domain.Group{}, s.groupErr
	}
	return domain.Group{ID: groupID, Name: "Group"}, nil
}

func (builderTestStore) ListTopics(context.Context) ([]domain.NotificationTopic, error) {
	return []domain.NotificationTopic{
		{Key: domain.NotificationTopicGroupMembershipRequested, Scope: domain.NotificationScopeGroup},
		{Key: domain.NotificationTopicGlobalMembershipRequested, Scope: domain.NotificationScopeGlobal},
	}, nil
}

func (builderTestStore) ListSubscriptions(context.Context, string) ([]domain.NotificationSubscription, error) {
	return nil, nil
}

func (builderTestStore) GetNotificationUserContext(context.Context, string) (domain.UserSummary, error) {
	return domain.UserSummary{}, nil
}

func (builderTestStore) GetNotificationGroupContext(_ context.Context, groupID uuid.UUID) (domain.Group, error) {
	return domain.Group{ID: groupID}, nil
}

func (builderTestStore) CreateGlobalSubscription(context.Context, string, domain.NotificationTopicKey, domain.NotificationChannel) error {
	return nil
}

func (builderTestStore) CreateGroupSubscription(context.Context, string, uuid.UUID, domain.NotificationTopicKey, domain.NotificationChannel) error {
	return nil
}

func (builderTestStore) DeleteGlobalSubscription(context.Context, string, domain.NotificationTopicKey, domain.NotificationChannel) error {
	return nil
}

func (builderTestStore) DeleteGroupSubscription(context.Context, string, uuid.UUID, domain.NotificationTopicKey, domain.NotificationChannel) error {
	return nil
}

func (builderTestStore) Recipient(context.Context, string) (domain.NotificationRecipient, error) {
	return builderTestRecipient(), nil
}

func (builderTestStore) GlobalRecipients(context.Context, domain.NotificationTopicKey) ([]domain.NotificationRecipient, error) {
	return []domain.NotificationRecipient{builderTestRecipient()}, nil
}

func (builderTestStore) GroupRecipients(context.Context, uuid.UUID, domain.NotificationTopicKey) ([]domain.NotificationRecipient, error) {
	return []domain.NotificationRecipient{builderTestRecipient()}, nil
}

func builderTestRecipient() domain.NotificationRecipient {
	return domain.NotificationRecipient{ID: "user", Email: "user@example.org", Status: domain.UserStatusActive}
}
