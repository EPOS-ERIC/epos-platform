package notifications

import (
	"context"
	"errors"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type listSubscriptionsStore struct {
	builderTestStore
	subscriptions []domain.NotificationSubscription
}

func (s listSubscriptionsStore) ListSubscriptions(context.Context, string) ([]domain.NotificationSubscription, error) {
	return s.subscriptions, nil
}

type listSubscriptionsAuthorizer struct {
	globalAllowed bool
	groups        []uuid.UUID
	err           error
}

func (a listSubscriptionsAuthorizer) CanManageGlobalNotificationSubscriptions(context.Context, string, string) (domain.Result, error) {
	return domain.Result{Allowed: a.globalAllowed}, nil
}

func (listSubscriptionsAuthorizer) CanManageGroupNotificationSubscriptions(context.Context, string, string, uuid.UUID) (domain.Result, error) {
	return domain.Result{}, nil
}

func (a listSubscriptionsAuthorizer) GroupsWithAnyPermissionForActiveMember(context.Context, string, string, []domain.GroupPermission) ([]uuid.UUID, error) {
	return a.groups, a.err
}

func TestListSubscriptionsFiltersToManagedActiveMemberGroups(t *testing.T) {
	groupID := uuid.New()
	unmanagedGroupID := uuid.New()
	subscriptions := []domain.NotificationSubscription{
		{UserID: "target", TopicKey: domain.NotificationTopicGlobalMembershipRequested},
		{UserID: "target", GroupID: &groupID, TopicKey: domain.NotificationTopicGroupMembershipRequested},
		{UserID: "target", GroupID: &unmanagedGroupID, TopicKey: domain.NotificationTopicGroupResourcePublished},
	}
	service := &Service{
		store:      listSubscriptionsStore{subscriptions: subscriptions},
		authorizer: listSubscriptionsAuthorizer{groups: []uuid.UUID{groupID}},
	}

	result, err := service.ListSubscriptions(context.Background(), "actor", "target")
	if err != nil {
		t.Fatalf("list subscriptions: %v", err)
	}
	if len(result) != 1 || result[0].GroupID == nil || *result[0].GroupID != groupID {
		t.Fatalf("subscriptions = %#v, want only group %s subscription", result, groupID)
	}
}

func TestListSubscriptionsGlobalManagerGetsAll(t *testing.T) {
	groupID := uuid.New()
	subscriptions := []domain.NotificationSubscription{
		{UserID: "target", TopicKey: domain.NotificationTopicGlobalMembershipRequested},
		{UserID: "target", GroupID: &groupID, TopicKey: domain.NotificationTopicGroupMembershipRequested},
	}
	service := &Service{
		store:      listSubscriptionsStore{subscriptions: subscriptions},
		authorizer: listSubscriptionsAuthorizer{globalAllowed: true, err: errors.New("group lookup must not run")},
	}

	result, err := service.ListSubscriptions(context.Background(), "actor", "target")
	if err != nil {
		t.Fatalf("list subscriptions: %v", err)
	}
	if len(result) != len(subscriptions) {
		t.Fatalf("subscriptions = %#v, want all %d subscriptions", result, len(subscriptions))
	}
	for i := range subscriptions {
		if result[i] != subscriptions[i] {
			t.Errorf("subscription[%d] = %#v, want %#v", i, result[i], subscriptions[i])
		}
	}
}

func TestListSubscriptionsDeniesWithoutVisibleGroup(t *testing.T) {
	service := &Service{
		store:      listSubscriptionsStore{},
		authorizer: listSubscriptionsAuthorizer{},
	}
	if _, err := service.ListSubscriptions(context.Background(), "actor", "target"); !apperror.IsCode(err, apperror.CodePermissionDenied) {
		t.Fatalf("error = %v, want forbidden", err)
	}
}

func TestListSubscriptionsReturnsEmptyWhenAuthorizedGroupsHaveNoSubscriptions(t *testing.T) {
	service := &Service{
		store:      listSubscriptionsStore{},
		authorizer: listSubscriptionsAuthorizer{groups: []uuid.UUID{uuid.New()}},
	}
	result, err := service.ListSubscriptions(context.Background(), "actor", "target")
	if err != nil {
		t.Fatalf("list subscriptions: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("subscriptions = %#v, want empty result", result)
	}
}

func TestListSubscriptionsPropagatesBulkAuthorizationError(t *testing.T) {
	wantErr := errors.New("authorization query failed")
	service := &Service{
		store:      listSubscriptionsStore{},
		authorizer: listSubscriptionsAuthorizer{err: wantErr},
	}
	if _, err := service.ListSubscriptions(context.Background(), "actor", "target"); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped authorization error", err)
	}
}
