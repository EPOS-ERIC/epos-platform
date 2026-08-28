package authz

import (
	"context"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type fakeRepository struct {
	user         domain.AuthorizationUser
	admin        bool
	global       bool
	member       bool
	anyMember    bool
	groupAllowed bool
	datasetGroup uuid.UUID
	scope        domain.PermissionScope
	published    bool
}

func (r fakeRepository) User(context.Context, string) (domain.AuthorizationUser, error) {
	return r.user, nil
}

func (r fakeRepository) PermissionScope(context.Context, string) (domain.PermissionScope, error) {
	return r.scope, nil
}
func (r fakeRepository) IsGlobalAdmin(context.Context, string) (bool, error) { return r.admin, nil }
func (r fakeRepository) HasGlobalPermission(context.Context, string, string) (bool, error) {
	return r.global, nil
}

func (r fakeRepository) IsActiveMember(context.Context, string, uuid.UUID) (bool, error) {
	return r.member, nil
}

func (r fakeRepository) HasAnyActiveMembership(context.Context, string) (bool, error) {
	return r.anyMember, nil
}

func (r fakeRepository) HasGroupPermission(context.Context, string, uuid.UUID, string) (bool, error) {
	return r.groupAllowed, nil
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
	authorizer := New(fakeRepository{
		user:         domain.AuthorizationUser{Status: domain.UserStatusActive},
		member:       true,
		groupAllowed: true,
		datasetGroup: groupID,
		scope:        domain.PermissionScopeGroup,
	})

	result, err := authorizer.AuthorizeResource(context.Background(), "user-1", domain.Dataset, uuid.New(), "resource.edit")
	if err != nil {
		t.Fatalf("authorize resource: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("allowed = false, reason = %q", result.Reason)
	}
}

func TestAuthorizeGroupDeniesInactiveUser(t *testing.T) {
	t.Parallel()
	authorizer := New(fakeRepository{user: domain.AuthorizationUser{Status: domain.UserStatusBlocked}, scope: domain.PermissionScopeGroup})

	result, err := authorizer.AuthorizeGroup(context.Background(), "user-1", uuid.New(), "resource.edit")
	if err != nil {
		t.Fatalf("authorize group: %v", err)
	}
	if result.Allowed || result.Reason != "user is not active" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAuthorizeGlobalAllowsGlobalGrant(t *testing.T) {
	t.Parallel()
	authorizer := New(fakeRepository{
		user:   domain.AuthorizationUser{Status: domain.UserStatusActive},
		global: true,
		scope:  domain.PermissionScopeGlobal,
	})

	result, err := authorizer.AuthorizeGlobal(context.Background(), "user-1", "organization.manage")
	if err != nil {
		t.Fatalf("authorize global: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("allowed = false, reason = %q", result.Reason)
	}
}

func TestCanReadPublishedResourceWithoutUser(t *testing.T) {
	t.Parallel()
	authorizer := New(fakeRepository{datasetGroup: uuid.New(), published: true})

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
			result, err := New(fakeRepository{
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
