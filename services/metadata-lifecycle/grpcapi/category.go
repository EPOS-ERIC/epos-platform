package grpcapi

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/categories"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type CategoryServer struct {
	metadatalifecyclev1.UnimplementedCategoryServiceServer
	categories *categories.Service
}

func NewCategory(categories *categories.Service) *CategoryServer {
	return &CategoryServer{categories: categories}
}

func (c *CategoryServer) ListCategories(ctx context.Context, request *metadatalifecyclev1.ListCategoriesRequest) (*metadatalifecyclev1.ListCategoriesResponse, error) {
	groupID, err := uuid.Parse(request.GetGroupId())
	if err != nil || groupID == uuid.Nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid group id")
	}

	values, err := c.categories.List(ctx, groupID)
	if err != nil {
		return nil, err
	}

	result := make([]*metadatalifecyclev1.CanonicalCategory, len(values))
	for i, value := range values {
		result[i] = toProtoCategory(value)
	}

	return &metadatalifecyclev1.ListCategoriesResponse{Categories: result}, nil
}

func (c *CategoryServer) GetCategory(ctx context.Context, request *metadatalifecyclev1.GetCategoryRequest) (*metadatalifecyclev1.GetCategoryResponse, error) {
	groupID, err := uuid.Parse(request.GetGroupId())
	if err != nil || groupID == uuid.Nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid group id")
	}
	categoryID, err := uuid.Parse(request.GetCategoryId())
	if err != nil || categoryID == uuid.Nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid category id")
	}

	value, err := c.categories.Get(ctx, groupID, categoryID)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.GetCategoryResponse{Category: toProtoCategory(value)}, nil
}

func (c *CategoryServer) CreateCategory(ctx context.Context, request *metadatalifecyclev1.CreateCategoryRequest) (*metadatalifecyclev1.CreateCategoryResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := uuid.Parse(request.GetGroupId())
	if err != nil || groupID == uuid.Nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid group id")
	}
	input, err := categoryInput(request.GetName(), request.ParentCategoryId)
	if err != nil {
		return nil, err
	}
	value, err := c.categories.Create(ctx, actor, groupID, input)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.CreateCategoryResponse{Category: toProtoCategory(value)}, nil
}

func (c *CategoryServer) ReplaceCategory(ctx context.Context, request *metadatalifecyclev1.ReplaceCategoryRequest) (*metadatalifecyclev1.ReplaceCategoryResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, categoryID, err := parseCategoryIDs(request.GetGroupId(), request.GetCategoryId())
	if err != nil {
		return nil, err
	}
	input, err := categoryInput(request.GetName(), request.ParentCategoryId)
	if err != nil {
		return nil, err
	}
	value, err := c.categories.Replace(ctx, actor, groupID, categoryID, input)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.ReplaceCategoryResponse{Category: toProtoCategory(value)}, nil
}

func (c *CategoryServer) DeleteCategory(ctx context.Context, request *metadatalifecyclev1.DeleteCategoryRequest) (*metadatalifecyclev1.DeleteCategoryResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, categoryID, err := parseCategoryIDs(request.GetGroupId(), request.GetCategoryId())
	if err != nil {
		return nil, err
	}
	if err := c.categories.Delete(ctx, actor, groupID, categoryID); err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DeleteCategoryResponse{}, nil
}

func parseCategoryIDs(group, category string) (uuid.UUID, uuid.UUID, error) {
	groupID, err := uuid.Parse(group)
	if err != nil || groupID == uuid.Nil {
		return uuid.Nil, uuid.Nil, apperror.New(apperror.CodeValidationFailed, "invalid group id")
	}
	categoryID, err := uuid.Parse(category)
	if err != nil || categoryID == uuid.Nil {
		return uuid.Nil, uuid.Nil, apperror.New(apperror.CodeValidationFailed, "invalid category id")
	}
	return groupID, categoryID, nil
}

func categoryInput(name string, parent *string) (domain.CategoryInput, error) {
	input := domain.CategoryInput{Name: name}
	if parent != nil {
		id, err := uuid.Parse(*parent)
		if err != nil || id == uuid.Nil {
			return domain.CategoryInput{}, apperror.Default(apperror.CodeValidationFailed)
		}

		input.ParentCategoryID = &id
	}
	if err := input.Normalize(); err != nil {
		return domain.CategoryInput{}, fmt.Errorf("validate category input: %w", err)
	}

	return input, nil
}

func toProtoCategory(value domain.Category) *metadatalifecyclev1.CanonicalCategory {
	result := &metadatalifecyclev1.CanonicalCategory{
		Id:      value.ID.String(),
		GroupId: value.GroupID.String(),
		Name:    value.Name,
	}
	if value.ParentCategoryID != nil {
		parentID := value.ParentCategoryID.String()
		result.ParentCategoryId = &parentID
	}
	return result
}
