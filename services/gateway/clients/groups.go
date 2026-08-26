package clients

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func (c *MetadataLifecycle) CreateGroup(ctx context.Context, actorID string, input domain.CreateGroupInput) (domain.Group, error) {
	if c == nil || c.groups == nil {
		return domain.Group{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	request := toClientCreateGroupRequest(input)

	group, response, err := c.groups.CreateGroup(ctx).XEPOSUserID(actorID).CreateGroupRequest(request).Execute()
	if err != nil {
		return domain.Group{}, wrapGroupManagementError("create metadata-lifecycle group", response, err)
	}
	return toDomainGroup(group)
}

func (c *MetadataLifecycle) ReplaceGroup(ctx context.Context, actorID, id string, input domain.ReplaceGroupInput) (domain.Group, error) {
	if c == nil || c.groups == nil {
		return domain.Group{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	request := toClientReplaceGroupRequest(input)

	group, response, err := c.groups.ReplaceGroup(ctx, id).XEPOSUserID(actorID).ReplaceGroupRequest(request).Execute()
	if err != nil {
		return domain.Group{}, wrapGroupManagementError("replace metadata-lifecycle group", response, err)
	}
	return toDomainGroup(group)
}

func (c *MetadataLifecycle) DeleteGroup(ctx context.Context, actorID, id string) error {
	if c == nil || c.groups == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}
	response, err := c.groups.DeleteGroup(ctx, id).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupManagementError("delete metadata-lifecycle group", response, err)
	}
	return nil
}

func (c *MetadataLifecycle) ListGroupMemberships(ctx context.Context, groupID, actorID string) ([]domain.GroupMembership, error) {
	if c == nil || c.access == nil {
		return nil, fmt.Errorf("metadata-lifecycle group access client is not configured")
	}
	memberships, response, err := c.access.ListGroupMemberships(ctx, groupID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapGroupAccessError("list metadata-lifecycle group memberships", response, err)
	}
	return mapSliceErr(memberships, toDomainGroupMembershipValue)
}

func (c *MetadataLifecycle) ListCategories(ctx context.Context, groupID string) ([]domain.Category, error) {
	if c == nil || c.categories == nil {
		return nil, fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	categories, response, err := c.categories.ListGroupCategories(ctx, groupID).Execute()
	if err != nil {
		return nil, wrapCategoryError("list metadata-lifecycle categories", response, err)
	}
	return mapSlice(categories, toDomainCategory), nil
}

func (c *MetadataLifecycle) GetCategory(ctx context.Context, groupID, categoryID string) (domain.Category, error) {
	if c == nil || c.categories == nil {
		return domain.Category{}, fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	category, response, err := c.categories.GetGroupCategory(ctx, groupID, categoryID).Execute()
	if err != nil {
		return domain.Category{}, wrapCategoryError("get metadata-lifecycle category", response, err)
	}
	if category == nil {
		return domain.Category{}, fmt.Errorf("decode metadata-lifecycle category: response is empty")
	}
	return toDomainCategory(*category), nil
}

func (c *MetadataLifecycle) CreateCategory(ctx context.Context, actorID, groupID string, input domain.CreateCategoryInput) (domain.Category, error) {
	if c == nil || c.categories == nil {
		return domain.Category{}, fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	category, response, err := c.categories.CreateGroupCategory(ctx, groupID).
		XEPOSUserID(actorID).
		CreateCategoryRequest(toClientCreateCategoryRequest(input)).
		Execute()
	if err != nil {
		return domain.Category{}, wrapCategoryError("create metadata-lifecycle category", response, err)
	}
	if category == nil {
		return domain.Category{}, fmt.Errorf("decode metadata-lifecycle category: response is empty")
	}
	return toDomainCategory(*category), nil
}

func (c *MetadataLifecycle) ReplaceCategory(ctx context.Context, actorID, groupID, categoryID string, input domain.ReplaceCategoryInput) (domain.Category, error) {
	if c == nil || c.categories == nil {
		return domain.Category{}, fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	request := toClientReplaceCategoryRequest(input)
	category, response, err := c.categories.ReplaceGroupCategory(ctx, groupID, categoryID).
		XEPOSUserID(actorID).
		ReplaceCategoryRequest(request).
		Execute()
	if err != nil {
		return domain.Category{}, wrapCategoryError("replace metadata-lifecycle category", response, err)
	}
	if category == nil {
		return domain.Category{}, fmt.Errorf("decode metadata-lifecycle category: response is empty")
	}
	return toDomainCategory(*category), nil
}

func (c *MetadataLifecycle) DeleteCategory(ctx context.Context, actorID, groupID, categoryID string) error {
	if c == nil || c.categories == nil {
		return fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	response, err := c.categories.DeleteGroupCategory(ctx, groupID, categoryID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapCategoryError("delete metadata-lifecycle category", response, err)
	}
	return nil
}

func wrapGroupManagementError(operation string, response *http.Response, err error) error {
	return wrapGroupAccessError(operation, response, err)
}

func wrapCategoryError(operation string, response *http.Response, err error) error {
	return wrapGroupAccessError(operation, response, err)
}
