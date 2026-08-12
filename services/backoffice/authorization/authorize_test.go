package authorization

import (
	"context"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/google/uuid"
)

type fakeRepository struct {
	user         sqlc.GetAuthorizationUserRow
	admin        bool
	global       bool
	member       bool
	anyMember    bool
	groupAllowed bool
	datasetGroup uuid.UUID
	scope        sqlc.PermissionScope
	published    bool
}

func (r fakeRepository) GetAuthorizationUser(context.Context, uuid.UUID) (sqlc.GetAuthorizationUserRow, error) {
	return r.user, nil
}

func (r fakeRepository) GetPermissionScope(context.Context, string) (sqlc.PermissionScope, error) {
	return r.scope, nil
}
func (r fakeRepository) IsGlobalAdmin(context.Context, uuid.UUID) (bool, error) { return r.admin, nil }
func (r fakeRepository) HasGlobalPermission(context.Context, sqlc.HasGlobalPermissionParams) (bool, error) {
	return r.global, nil
}

func (r fakeRepository) IsActiveMember(context.Context, sqlc.IsActiveMemberParams) (bool, error) {
	return r.member, nil
}

func (r fakeRepository) HasAnyActiveMembership(context.Context, uuid.UUID) (bool, error) {
	return r.anyMember, nil
}

func (r fakeRepository) HasGroupPermission(context.Context, sqlc.HasGroupPermissionParams) (bool, error) {
	return r.groupAllowed, nil
}

func (r fakeRepository) GetDatasetGroup(context.Context, uuid.UUID) (uuid.UUID, error) {
	return r.datasetGroup, nil
}

func (r fakeRepository) GetSoftwareApplicationGroup(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r fakeRepository) GetSoftwareSourceCodeGroup(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r fakeRepository) IsDatasetPublished(context.Context, uuid.UUID) (bool, error) {
	return r.published, nil
}

func (r fakeRepository) GetDatasetVersionAuthorization(context.Context, sqlc.GetDatasetVersionAuthorizationParams) (sqlc.GetDatasetVersionAuthorizationRow, error) {
	return sqlc.GetDatasetVersionAuthorizationRow{DatasetID: uuid.New(), GroupID: r.datasetGroup, Status: func() sqlc.ResourceStatus {
		if r.published {
			return sqlc.ResourceStatusPUBLISHED
		}
		return sqlc.ResourceStatusDRAFT
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
		user:         sqlc.GetAuthorizationUserRow{Status: sqlc.UserStatusACTIVE},
		member:       true,
		groupAllowed: true,
		datasetGroup: groupID,
		scope:        sqlc.PermissionScopeGROUP,
	})

	result, err := authorizer.AuthorizeResource(context.Background(), uuid.New(), ResourceTypeDataset, uuid.New(), "resource.edit")
	if err != nil {
		t.Fatalf("authorize resource: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("allowed = false, reason = %q", result.Reason)
	}
}

func TestAuthorizeGroupDeniesInactiveUser(t *testing.T) {
	t.Parallel()
	authorizer := New(fakeRepository{user: sqlc.GetAuthorizationUserRow{Status: sqlc.UserStatusBLOCKED}, scope: sqlc.PermissionScopeGROUP})

	result, err := authorizer.AuthorizeGroup(context.Background(), uuid.New(), uuid.New(), "resource.edit")
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
		user:   sqlc.GetAuthorizationUserRow{Status: sqlc.UserStatusACTIVE},
		global: true,
		scope:  sqlc.PermissionScopeGLOBAL,
	})

	result, err := authorizer.AuthorizeGlobal(context.Background(), uuid.New(), "organization.manage")
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

	result, err := authorizer.CanReadResource(context.Background(), nil, ResourceTypeDataset, uuid.New())
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
	userID := uuid.New()

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
				user:      sqlc.GetAuthorizationUserRow{Status: sqlc.UserStatusACTIVE},
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
