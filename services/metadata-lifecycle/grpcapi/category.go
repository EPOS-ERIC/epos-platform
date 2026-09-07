package grpcapi

import (
	"context"
	"errors"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/categories"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, fmt.Errorf("invalid group id: %w", status.Error(codes.InvalidArgument, "invalid group id"))
	}

	values, err := c.categories.List(ctx, groupID)
	if err != nil {
		return nil, toCategoryError("list categories", err)
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
		return nil, fmt.Errorf("invalid group id: %w", status.Error(codes.InvalidArgument, "invalid group id"))
	}
	categoryID, err := uuid.Parse(request.GetCategoryId())
	if err != nil || categoryID == uuid.Nil {
		return nil, fmt.Errorf("invalid category id: %w", status.Error(codes.InvalidArgument, "invalid category id"))
	}

	value, err := c.categories.Get(ctx, groupID, categoryID)
	if err != nil {
		return nil, toCategoryError("get category", err)
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
		return nil, fmt.Errorf("invalid group id: %w", status.Error(codes.InvalidArgument, "invalid group id"))
	}
	input, err := categoryInput(request.GetName(), request.ParentCategoryId)
	if err != nil {
		return nil, fmt.Errorf("invalid category: %w", status.Error(codes.InvalidArgument, "invalid category"))
	}
	value, err := c.categories.Create(ctx, actor, groupID, input)
	if err != nil {
		return nil, toCategoryError("create category", err)
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
		return nil, fmt.Errorf("invalid category: %w", status.Error(codes.InvalidArgument, "invalid category"))
	}
	value, err := c.categories.Replace(ctx, actor, groupID, categoryID, input)
	if err != nil {
		return nil, toCategoryError("replace category", err)
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
		return nil, toCategoryError("delete category", err)
	}
	return &metadatalifecyclev1.DeleteCategoryResponse{}, nil
}

func parseCategoryIDs(group, category string) (uuid.UUID, uuid.UUID, error) {
	groupID, err := uuid.Parse(group)
	if err != nil || groupID == uuid.Nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid group id: %w", status.Error(codes.InvalidArgument, "invalid group id"))
	}
	categoryID, err := uuid.Parse(category)
	if err != nil || categoryID == uuid.Nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid category id: %w", status.Error(codes.InvalidArgument, "invalid category id"))
	}
	return groupID, categoryID, nil
}

func categoryInput(name string, parent *string) (domain.CategoryInput, error) {
	input := domain.CategoryInput{Name: name}
	if parent != nil {
		id, err := uuid.Parse(*parent)
		if err != nil || id == uuid.Nil {
			return domain.CategoryInput{}, domain.ErrInvalidCategory
		}
		input.ParentCategoryID = &id
	}
	return domain.NewCategoryInput(input)
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

func toCategoryError(operation string, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCategory):
		return fmt.Errorf("%s: %w", operation, status.Error(codes.InvalidArgument, "invalid category"))
	case errors.Is(err, domain.ErrCategoryForbidden):
		return fmt.Errorf("%s: %w", operation, status.Error(codes.PermissionDenied, "category operation is not permitted"))
	case errors.Is(err, domain.ErrCategoryNotFound):
		return fmt.Errorf("%s: %w", operation, status.Error(codes.NotFound, "category not found"))
	case errors.Is(err, domain.ErrCategoryConflict):
		return fmt.Errorf("%s: %w", operation, status.Error(codes.AlreadyExists, "category operation conflicts with existing data"))
	default:
		return fmt.Errorf("%s: %w", operation, status.Error(codes.Internal, operation))
	}
}
