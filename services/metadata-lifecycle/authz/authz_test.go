package authz

import (
	"context"
	"log/slog"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type fakeRepository struct {
	permissions      []domain.Permission
	user             domain.AuthorizationUser
	admin            bool
	global           bool
	globalPermission domain.GlobalPermission
	member           bool
	anyMember        bool
	groupAllowed     bool
	groupPermission  domain.GroupPermission
	datasetGroup     uuid.UUID
	published        bool
	activeGroups     []uuid.UUID
	permissionGroups []uuid.UUID
}

func (r fakeRepository) ListPermissions(context.Context) ([]domain.Permission, error) {
	return r.permissions, nil
}

func (r fakeRepository) User(context.Context, string) (domain.AuthorizationUser, error) {
	return r.user, nil
}

func (r fakeRepository) IsGlobalAdmin(context.Context, string) (bool, error) { return r.admin, nil }
func (r fakeRepository) HasGlobalPermission(_ context.Context, _ string, permission domain.GlobalPermission) (bool, error) {
	return r.global || r.globalPermission == permission, nil
}

func (r fakeRepository) IsActiveMember(context.Context, string, uuid.UUID) (bool, error) {
	return r.member, nil
}

func (r fakeRepository) HasAnyActiveMembership(context.Context, string) (bool, error) {
	return r.anyMember, nil
}

func (r fakeRepository) ListActiveUserMemberships(context.Context, string) ([]uuid.UUID, error) {
	return r.activeGroups, nil
}

func (r fakeRepository) HasGroupPermission(_ context.Context, _ string, _ uuid.UUID, permission domain.GroupPermission) (bool, error) {
	return r.groupAllowed || r.groupPermission == permission, nil
}

func (r fakeRepository) ListGroupsWithAnyPermission(context.Context, string, []domain.GroupPermission) ([]uuid.UUID, error) {
	return r.permissionGroups, nil
}

func (fakeRepository) HasGroupRole(context.Context, string, uuid.UUID, string) (bool, error) {
	return false, nil
}

func (r fakeRepository) DatasetGroup(context.Context, uuid.UUID) (uuid.UUID, error) {
	return r.datasetGroup, nil
}

func (r fakeRepository) SoftwareApplicationGroup(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r fakeRepository) SoftwareSourceCodeGroup(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r fakeRepository) IsDatasetPublished(context.Context, uuid.UUID) (bool, error) {
	return r.published, nil
}

func (r fakeRepository) DatasetVersionAuthz(_ context.Context, datasetID, _ uuid.UUID) (domain.VersionAuthz, error) {
	return domain.VersionAuthz{ResourceID: datasetID, GroupID: r.datasetGroup, Status: func() domain.Status {
		if r.published {
			return domain.Published
		}
		return domain.Draft
	}()}, nil
}

func (r fakeRepository) SoftwareApplicationVersionAuthz(_ context.Context, applicationID, _ uuid.UUID) (domain.VersionAuthz, error) {
	return domain.VersionAuthz{ResourceID: applicationID, GroupID: r.datasetGroup, Status: func() domain.Status {
		if r.published {
			return domain.Published
		}
		return domain.Draft
	}()}, nil
}

func (r fakeRepository) SoftwareSourceCodeVersionAuthz(_ context.Context, sourceCodeID, _ uuid.UUID) (domain.VersionAuthz, error) {
	return domain.VersionAuthz{ResourceID: sourceCodeID, GroupID: r.datasetGroup, Status: func() domain.Status {
		if r.published {
			return domain.Published
		}
		return domain.Draft
	}()}, nil
}

func (r fakeRepository) IsSoftwareApplicationPublished(context.Context, uuid.UUID) (bool, error) {
	return r.published, nil
}

func (r fakeRepository) IsSoftwareSourceCodePublished(context.Context, uuid.UUID) (bool, error) {
	return r.published, nil
}

func TestAuthorizeResourceUsesStoredGroup(t *testing.T) {
	t.Parallel()
	groupID := uuid.New()
	authorizer := New(slog.Default(), fakeRepository{
		user:         domain.AuthorizationUser{Status: domain.UserStatusActive},
		member:       true,
		groupAllowed: true,
		datasetGroup: groupID,
	})

	result, err := authorizer.AuthorizeResource(context.Background(), "user-1", domain.Dataset, uuid.New(), domain.GroupPermissionResourceEdit)
	if err != nil {
		t.Fatalf("authorize resource: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("allowed = false, reason = %q", result.Reason)
	}
}

func TestAuthorizeGroupDeniesBlockedUser(t *testing.T) {
	t.Parallel()
	authorizer := New(slog.Default(), fakeRepository{user: domain.AuthorizationUser{Status: domain.UserStatusBlocked}})

	result, err := authorizer.AuthorizeGroup(context.Background(), "user-1", uuid.New(), domain.GroupPermissionResourceEdit)
	if err != nil {
		t.Fatalf("authorize group: %v", err)
	}
	if result.Allowed || result.Reason != "user is not active" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAuthorizeGlobalAllowsGlobalGrant(t *testing.T) {
	t.Parallel()
	authorizer := New(slog.Default(), fakeRepository{
		user:   domain.AuthorizationUser{Status: domain.UserStatusActive},
		global: true,
	})

	result, err := authorizer.AuthorizeGlobal(context.Background(), "user-1", domain.GlobalPermissionOrganizationManage)
	if err != nil {
		t.Fatalf("authorize global: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("allowed = false, reason = %q", result.Reason)
	}
}

func TestCanManageGlobalNotificationSubscriptionsAllowsSelf(t *testing.T) {
	t.Parallel()
	authorizer := New(slog.Default(), fakeRepository{
		user: domain.AuthorizationUser{ID: "user-1", Status: domain.UserStatusActive},
	})

	result, err := authorizer.CanManageGlobalNotificationSubscriptions(context.Background(), "user-1", "user-1")
	if err != nil {
		t.Fatalf("authorize self: %v", err)
	}
	if !result.Allowed || result.Reason != "self" {
		t.Fatalf("result = %#v, want self access", result)
	}
}

func TestCanListGroupMembershipsAllowsGroupNotificationManager(t *testing.T) {
	t.Parallel()
	authorizer := New(slog.Default(), fakeRepository{
		user:            domain.AuthorizationUser{Status: domain.UserStatusActive},
		member:          true,
		groupPermission: domain.GroupPermissionNotificationManage,
	})

	result, err := authorizer.CanListGroupMemberships(context.Background(), "user-1", uuid.New())
	if err != nil {
		t.Fatalf("list group memberships authorization: %v", err)
	}
	if !result.Allowed || result.Reason != string(domain.GroupPermissionNotificationManage) {
		t.Fatalf("result = %#v", result)
	}
}

func TestCanListGroupMembershipsAllowsGlobalNotificationManager(t *testing.T) {
	t.Parallel()
	authorizer := New(slog.Default(), fakeRepository{
		user:             domain.AuthorizationUser{Status: domain.UserStatusActive},
		globalPermission: domain.GlobalPermissionNotificationManage,
	})

	result, err := authorizer.CanListGroupMemberships(context.Background(), "user-1", uuid.New())
	if err != nil {
		t.Fatalf("list group memberships authorization: %v", err)
	}
	if !result.Allowed || result.Reason != string(domain.GlobalPermissionNotificationManage) {
		t.Fatalf("result = %#v", result)
	}
}

func TestCanListUsersAllowsGlobalNotificationManager(t *testing.T) {
	t.Parallel()
	authorizer := New(slog.Default(), fakeRepository{
		user:             domain.AuthorizationUser{Status: domain.UserStatusActive},
		globalPermission: domain.GlobalPermissionNotificationManage,
	})

	result, err := authorizer.CanListUsers(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list users authorization: %v", err)
	}
	if !result.Allowed || result.Reason != string(domain.GlobalPermissionNotificationManage) {
		t.Fatalf("result = %#v", result)
	}
}

func TestAuthorizeGroupAllowsMatchingGlobalGrantWithoutMembership(t *testing.T) {
	t.Parallel()
	groupID := uuid.New()
	authorizer := New(slog.Default(), fakeRepository{
		user:   domain.AuthorizationUser{Status: domain.UserStatusActive},
		global: true,
	})

	result, err := authorizer.AuthorizeGroup(
		context.Background(),
		"user-1",
		groupID,
		domain.GroupPermissionResourceEdit,
	)
	if err != nil {
		t.Fatalf("authorize group: %v", err)
	}
	if !result.Allowed || result.Reason != string(domain.GlobalPermissionResourceEdit) {
		t.Fatalf("result = %#v", result)
	}
}

func TestGroupsWithAnyPermissionForActiveMemberIntersectsTargetMemberships(t *testing.T) {
	t.Parallel()
	permittedGroup := uuid.New()
	inactiveTargetGroup := uuid.New()
	authorizer := New(slog.Default(), fakeRepository{
		user:             domain.AuthorizationUser{Status: domain.UserStatusActive},
		activeGroups:     []uuid.UUID{permittedGroup},
		permissionGroups: []uuid.UUID{permittedGroup, inactiveTargetGroup},
	})

	groups, err := authorizer.GroupsWithAnyPermissionForActiveMember(
		context.Background(),
		"actor",
		"target",
		[]domain.GroupPermission{domain.GroupPermissionNotificationManage},
	)
	if err != nil {
		t.Fatalf("list permitted groups: %v", err)
	}
	if len(groups) != 1 || groups[0] != permittedGroup {
		t.Fatalf("groups = %v, want only active target group %s", groups, permittedGroup)
	}
}

func TestAuthorizeGroupDoesNotUseGlobalAdminBypass(t *testing.T) {
	t.Parallel()
	authorizer := New(slog.Default(), fakeRepository{
		user:  domain.AuthorizationUser{Status: domain.UserStatusActive},
		admin: true,
	})

	result, err := authorizer.AuthorizeGroup(
		context.Background(),
		"user-1",
		uuid.New(),
		domain.GroupPermissionResourceEdit,
	)
	if err != nil {
		t.Fatalf("authorize group: %v", err)
	}
	if result.Allowed || result.Reason != "user is not an active group member" {
		t.Fatalf("result = %#v", result)
	}
}

func TestCanReadPublishedResourceWithoutUser(t *testing.T) {
	t.Parallel()
	authorizer := New(slog.Default(), fakeRepository{datasetGroup: uuid.New(), published: true})

	result, err := authorizer.CanReadResource(context.Background(), nil, domain.Dataset, uuid.New())
	if err != nil {
		t.Fatalf("read published resource: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("allowed = false, reason = %q", result.Reason)
	}
}

func TestCanReadContactPointRequiresMembershipOrAdmin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	userID := "user-1"

	for _, test := range []struct {
		name    string
		admin   bool
		member  bool
		allowed bool
	}{
		{name: "no membership", allowed: false},
		{name: "active member", member: true, allowed: true},
		{name: "global admin", admin: true, allowed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := New(slog.Default(), fakeRepository{
				user:      domain.AuthorizationUser{Status: domain.UserStatusActive},
				admin:     test.admin,
				anyMember: test.member,
			}).CanReadContactPoint(ctx, userID)
			if err != nil {
				t.Fatalf("read contact point: %v", err)
			}
			if result.Allowed != test.allowed {
				t.Fatalf("allowed = %t, want %t", result.Allowed, test.allowed)
			}
		})
	}
}

func TestAuthorizeDiscoveryPreservesVersionVisibilityRules(t *testing.T) {
	t.Parallel()
	groupID := uuid.New()
	otherGroupID := uuid.New()
	candidates := []domain.DiscoveryCandidate{
		{Type: domain.Dataset, ResourceID: uuid.New(), VersionID: uuid.New(), GroupID: groupID, Status: domain.Published, Name: "published"},
		{Type: domain.SoftwareApplication, ResourceID: uuid.New(), VersionID: uuid.New(), GroupID: groupID, Status: domain.Draft, Name: "member draft"},
		{Type: domain.SoftwareSourceCode, ResourceID: uuid.New(), VersionID: uuid.New(), GroupID: otherGroupID, Status: domain.Draft, Name: "other draft"},
	}
	actor := domain.Actor("user-1")
	result, err := New(slog.Default(), fakeRepository{
		user:         domain.AuthorizationUser{ID: string(actor), Status: domain.UserStatusActive},
		activeGroups: []uuid.UUID{groupID},
	}).AuthorizeDiscovery(context.Background(), &actor, candidates)
	if err != nil {
		t.Fatalf("authorize discovery: %v", err)
	}
	if len(result.AllowedCandidates) != 2 {
		t.Fatalf("allowed candidates = %d, want 2", len(result.AllowedCandidates))
	}
	if _, ok := result.CreatedByGroupIDs[groupID]; !ok {
		t.Fatalf("created-by groups = %#v, want %s", result.CreatedByGroupIDs, groupID)
	}
	if result.AllowedCandidates[0].ResourceID != candidates[0].ResourceID || result.AllowedCandidates[1].ResourceID != candidates[1].ResourceID {
		t.Fatalf("allowed candidates = %#v", result.AllowedCandidates)
	}
}

func TestAuthorizeDiscoveryUsesGlobalAdminFromAuthorizationUser(t *testing.T) {
	t.Parallel()
	groupID := uuid.New()
	candidate := domain.DiscoveryCandidate{
		Type:       domain.Dataset,
		ResourceID: uuid.New(),
		VersionID:  uuid.New(),
		GroupID:    groupID,
		Status:     domain.Draft,
		Name:       "admin draft",
	}
	actor := domain.Actor("admin")
	result, err := New(slog.Default(), fakeRepository{
		user: domain.AuthorizationUser{
			ID:          string(actor),
			Status:      domain.UserStatusActive,
			GlobalAdmin: true,
		},
	}).AuthorizeDiscovery(context.Background(), &actor, []domain.DiscoveryCandidate{candidate})
	if err != nil {
		t.Fatalf("authorize discovery: %v", err)
	}
	if len(result.AllowedCandidates) != 1 {
		t.Fatalf("allowed candidates = %#v, want one candidate", result.AllowedCandidates)
	}
	if _, ok := result.CreatedByGroupIDs[groupID]; !ok {
		t.Fatalf("created-by groups = %#v, want %s", result.CreatedByGroupIDs, groupID)
	}
}

func TestAuthorizeDiscoveryAllowsPublishedCandidatesWithoutActor(t *testing.T) {
	t.Parallel()
	published := domain.DiscoveryCandidate{
		Type:       domain.Dataset,
		ResourceID: uuid.New(),
		VersionID:  uuid.New(),
		GroupID:    uuid.New(),
		Status:     domain.Published,
		Name:       "published",
	}
	draft := published
	draft.VersionID = uuid.New()
	draft.Status = domain.Draft
	result, err := New(slog.Default(), fakeRepository{}).AuthorizeDiscovery(context.Background(), nil, []domain.DiscoveryCandidate{published, draft})
	if err != nil {
		t.Fatalf("authorize discovery: %v", err)
	}
	if len(result.AllowedCandidates) != 1 || result.AllowedCandidates[0].ResourceID != published.ResourceID {
		t.Fatalf("allowed candidates = %#v", result.AllowedCandidates)
	}
	if len(result.CreatedByGroupIDs) != 0 {
		t.Fatalf("created-by groups = %#v, want empty", result.CreatedByGroupIDs)
	}
}
