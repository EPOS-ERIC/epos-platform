package clients

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
)

func (c *Backoffice) CreateGroup(ctx context.Context, actorID string, input domain.CreateGroupInput) (domain.Group, error) {
	if c == nil || c.groups == nil {
		return domain.Group{}, fmt.Errorf("backoffice group client is not configured")
	}

	request := client.CreateGroupRequest{
		SourceIri:   input.SourceIRI,
		Name:        input.Name,
		Description: client.NullableString{},
		Label:       client.NullableString{},
		Logo:        client.NullableString{},
		Page:        client.NullableString{},
		Color:       client.NullableString{},
		Position:    input.Position,
		Concept:     input.Concept,
	}
	setNullable(&request.Description, input.Description)
	setNullable(&request.Label, input.Label)
	setNullable(&request.Logo, input.Logo)
	setNullable(&request.Page, input.Page)
	setNullable(&request.Color, input.Color)

	group, response, err := c.groups.CreateGroup(ctx).XEPOSUserID(actorID).CreateGroupRequest(request).Execute()
	if err != nil {
		return domain.Group{}, wrapGroupManagementError("create backoffice group", response, err)
	}
	return toDomainGroup(group)
}

func (c *Backoffice) ReplaceGroup(ctx context.Context, actorID, id string, input domain.ReplaceGroupInput) (domain.Group, error) {
	if c == nil || c.groups == nil {
		return domain.Group{}, fmt.Errorf("backoffice group client is not configured")
	}

	request := client.ReplaceGroupRequest{
		Name:        input.Name,
		Description: client.NullableString{},
		Label:       client.NullableString{},
		Logo:        client.NullableString{},
		Page:        client.NullableString{},
		Color:       client.NullableString{},
		Position:    input.Position,
		Concept:     input.Concept,
	}
	setNullable(&request.Description, input.Description)
	setNullable(&request.Label, input.Label)
	setNullable(&request.Logo, input.Logo)
	setNullable(&request.Page, input.Page)
	setNullable(&request.Color, input.Color)

	group, response, err := c.groups.ReplaceGroup(ctx, id).XEPOSUserID(actorID).ReplaceGroupRequest(request).Execute()
	if err != nil {
		return domain.Group{}, wrapGroupManagementError("replace backoffice group", response, err)
	}
	return toDomainGroup(group)
}

func (c *Backoffice) DeleteGroup(ctx context.Context, actorID, id string) error {
	if c == nil || c.groups == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}
	response, err := c.groups.DeleteGroup(ctx, id).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupManagementError("delete backoffice group", response, err)
	}
	return nil
}

func (c *Backoffice) ListGroupMemberships(ctx context.Context, groupID, actorID string) ([]domain.GroupMembership, error) {
	if c == nil || c.access == nil {
		return nil, fmt.Errorf("backoffice group access client is not configured")
	}
	memberships, response, err := c.access.ListGroupMemberships(ctx, groupID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapGroupAccessError("list backoffice group memberships", response, err)
	}
	result := make([]domain.GroupMembership, 0, len(memberships))
	for _, membership := range memberships {
		mapped, err := toDomainGroupMembership(&membership)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (c *Backoffice) ListCategories(ctx context.Context, groupID string) ([]domain.Category, error) {
	if c == nil || c.categories == nil {
		return nil, fmt.Errorf("backoffice category client is not configured")
	}
	categories, response, err := c.categories.ListGroupCategories(ctx, groupID).Execute()
	if err != nil {
		return nil, wrapCategoryError("list backoffice categories", response, err)
	}
	result := make([]domain.Category, 0, len(categories))
	for _, category := range categories {
		result = append(result, toDomainCategory(category))
	}
	return result, nil
}

func (c *Backoffice) GetCategory(ctx context.Context, groupID, categoryID string) (domain.Category, error) {
	if c == nil || c.categories == nil {
		return domain.Category{}, fmt.Errorf("backoffice category client is not configured")
	}
	category, response, err := c.categories.GetGroupCategory(ctx, groupID, categoryID).Execute()
	if err != nil {
		return domain.Category{}, wrapCategoryError("get backoffice category", response, err)
	}
	if category == nil {
		return domain.Category{}, fmt.Errorf("decode backoffice category: response is empty")
	}
	return toDomainCategory(*category), nil
}

func (c *Backoffice) CreateCategory(ctx context.Context, actorID, groupID string, input domain.CreateCategoryInput) (domain.Category, error) {
	if c == nil || c.categories == nil {
		return domain.Category{}, fmt.Errorf("backoffice category client is not configured")
	}
	category, response, err := c.categories.CreateGroupCategory(ctx, groupID).
		XEPOSUserID(actorID).
		CreateCategoryRequest(client.CreateCategoryRequest{SourceIri: input.SourceIRI, Name: input.Name, ParentCategoryId: input.ParentCategoryID}).
		Execute()
	if err != nil {
		return domain.Category{}, wrapCategoryError("create backoffice category", response, err)
	}
	if category == nil {
		return domain.Category{}, fmt.Errorf("decode backoffice category: response is empty")
	}
	return toDomainCategory(*category), nil
}

func (c *Backoffice) ReplaceCategory(ctx context.Context, actorID, groupID, categoryID string, input domain.ReplaceCategoryInput) (domain.Category, error) {
	if c == nil || c.categories == nil {
		return domain.Category{}, fmt.Errorf("backoffice category client is not configured")
	}
	request := client.ReplaceCategoryRequest{Name: input.Name, ParentCategoryId: client.NullableString{}}
	setNullable(&request.ParentCategoryId, input.ParentCategoryID)
	category, response, err := c.categories.ReplaceGroupCategory(ctx, groupID, categoryID).
		XEPOSUserID(actorID).
		ReplaceCategoryRequest(request).
		Execute()
	if err != nil {
		return domain.Category{}, wrapCategoryError("replace backoffice category", response, err)
	}
	if category == nil {
		return domain.Category{}, fmt.Errorf("decode backoffice category: response is empty")
	}
	return toDomainCategory(*category), nil
}

func (c *Backoffice) DeleteCategory(ctx context.Context, actorID, groupID, categoryID string) error {
	if c == nil || c.categories == nil {
		return fmt.Errorf("backoffice category client is not configured")
	}
	response, err := c.categories.DeleteGroupCategory(ctx, groupID, categoryID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapCategoryError("delete backoffice category", response, err)
	}
	return nil
}

func toDomainCategory(category client.CanonicalCategory) domain.Category {
	return domain.Category{ID: category.Id, GroupID: category.GroupId, ParentCategoryID: category.ParentCategoryId.Get(), SourceIRI: category.SourceIri, Name: category.Name}
}

func wrapGroupManagementError(operation string, response *http.Response, err error) error {
	return wrapGroupAccessError(operation, response, err)
}

func wrapCategoryError(operation string, response *http.Response, err error) error {
	return wrapGroupAccessError(operation, response, err)
}
