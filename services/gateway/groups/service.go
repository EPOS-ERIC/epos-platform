// Package groups contains Gateway group metadata use-cases.
package groups

import (
	"context"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	metadataLifecycle *clients.MetadataLifecycle
}

func NewService(metadataLifecycle *clients.MetadataLifecycle) *Service {
	return &Service{metadataLifecycle: metadataLifecycle}
}

func (s *Service) List(ctx context.Context) ([]domain.Group, error) {
	if s == nil || s.metadataLifecycle == nil {
		return nil, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	groups, err := s.metadataLifecycle.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups from metadata-lifecycle: %w", err)
	}

	return groups, nil
}

func (s *Service) Get(ctx context.Context, id string) (domain.Group, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Group{}, fmt.Errorf("invalid group id: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.Group{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	group, err := s.metadataLifecycle.GetGroup(ctx, id)
	if err != nil {
		return domain.Group{}, fmt.Errorf("get group from metadata-lifecycle: %w", err)
	}

	return group, nil
}

func (s *Service) Create(ctx context.Context, actorID string, input domain.CreateGroupInput) (domain.Group, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Concept) == "" {
		return domain.Group{}, fmt.Errorf("invalid group request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.Group{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}
	group, err := s.metadataLifecycle.CreateGroup(ctx, actorID, input)
	if err != nil {
		return domain.Group{}, fmt.Errorf("create group in metadata-lifecycle: %w", err)
	}
	return group, nil
}

func (s *Service) Replace(ctx context.Context, actorID, id string, input domain.ReplaceGroupInput) (domain.Group, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Concept) == "" {
		return domain.Group{}, fmt.Errorf("invalid group request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.Group{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}
	group, err := s.metadataLifecycle.ReplaceGroup(ctx, actorID, id, input)
	if err != nil {
		return domain.Group{}, fmt.Errorf("replace group in metadata-lifecycle: %w", err)
	}
	return group, nil
}

func (s *Service) Delete(ctx context.Context, actorID, id string) error {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid group request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}
	if err := s.metadataLifecycle.DeleteGroup(ctx, actorID, id); err != nil {
		return fmt.Errorf("delete group in metadata-lifecycle: %w", err)
	}
	return nil
}

func (s *Service) ListMemberships(ctx context.Context, groupID, actorID string) ([]domain.GroupMembership, error) {
	if strings.TrimSpace(groupID) == "" || strings.TrimSpace(actorID) == "" {
		return nil, fmt.Errorf("invalid membership request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return nil, fmt.Errorf("metadata-lifecycle group client is not configured")
	}
	memberships, err := s.metadataLifecycle.ListGroupMemberships(ctx, groupID, actorID)
	if err != nil {
		return nil, fmt.Errorf("list group memberships in metadata-lifecycle: %w", err)
	}
	return memberships, nil
}

func (s *Service) ListCategories(ctx context.Context, groupID string) ([]domain.Category, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, fmt.Errorf("invalid group id: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return nil, fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	categories, err := s.metadataLifecycle.ListCategories(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list categories in metadata-lifecycle: %w", err)
	}
	return categories, nil
}

func (s *Service) GetCategory(ctx context.Context, groupID, categoryID string) (domain.Category, error) {
	if strings.TrimSpace(groupID) == "" || strings.TrimSpace(categoryID) == "" {
		return domain.Category{}, fmt.Errorf("invalid category request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.Category{}, fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	category, err := s.metadataLifecycle.GetCategory(ctx, groupID, categoryID)
	if err != nil {
		return domain.Category{}, fmt.Errorf("get category in metadata-lifecycle: %w", err)
	}
	return category, nil
}

func (s *Service) CreateCategory(ctx context.Context, actorID, groupID string, input domain.CreateCategoryInput) (domain.Category, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(groupID) == "" || strings.TrimSpace(input.Name) == "" {
		return domain.Category{}, fmt.Errorf("invalid category request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.Category{}, fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	category, err := s.metadataLifecycle.CreateCategory(ctx, actorID, groupID, input)
	if err != nil {
		return domain.Category{}, fmt.Errorf("create category in metadata-lifecycle: %w", err)
	}
	return category, nil
}

func (s *Service) ReplaceCategory(ctx context.Context, actorID, groupID, categoryID string, input domain.ReplaceCategoryInput) (domain.Category, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(groupID) == "" || strings.TrimSpace(categoryID) == "" || strings.TrimSpace(input.Name) == "" {
		return domain.Category{}, fmt.Errorf("invalid category request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.Category{}, fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	category, err := s.metadataLifecycle.ReplaceCategory(ctx, actorID, groupID, categoryID, input)
	if err != nil {
		return domain.Category{}, fmt.Errorf("replace category in metadata-lifecycle: %w", err)
	}
	return category, nil
}

func (s *Service) DeleteCategory(ctx context.Context, actorID, groupID, categoryID string) error {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(groupID) == "" || strings.TrimSpace(categoryID) == "" {
		return fmt.Errorf("invalid category request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle category client is not configured")
	}
	if err := s.metadataLifecycle.DeleteCategory(ctx, actorID, groupID, categoryID); err != nil {
		return fmt.Errorf("delete category in metadata-lifecycle: %w", err)
	}
	return nil
}

func (s *Service) WithdrawMembershipRequest(ctx context.Context, groupID, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || actorID == "" {
		return fmt.Errorf("invalid membership request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	if err := s.metadataLifecycle.WithdrawMembershipRequest(ctx, groupID, actorID); err != nil {
		return fmt.Errorf("withdraw membership request from metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) RequestMembership(ctx context.Context, groupID, actorID string, roles []string) (domain.GroupMembership, error) {
	groupID = strings.TrimSpace(groupID)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || actorID == "" {
		return domain.GroupMembership{}, fmt.Errorf("invalid membership request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.GroupMembership{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	membership, err := s.metadataLifecycle.RequestMembership(ctx, groupID, actorID, roles)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("request membership from metadata-lifecycle: %w", err)
	}

	return membership, nil
}

func (s *Service) DecideMembership(ctx context.Context, groupID, userID, actorID, status string) (domain.GroupMembership, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	actorID = strings.TrimSpace(actorID)
	status = strings.TrimSpace(status)
	if groupID == "" || userID == "" || actorID == "" || (status != "ACTIVE" && status != "REJECTED") {
		return domain.GroupMembership{}, fmt.Errorf("invalid membership decision: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.GroupMembership{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	membership, err := s.metadataLifecycle.DecideMembership(ctx, groupID, userID, status, actorID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("decide membership in metadata-lifecycle: %w", err)
	}

	return membership, nil
}

func (s *Service) RevokeMembership(ctx context.Context, groupID, userID, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || userID == "" || actorID == "" {
		return fmt.Errorf("invalid membership revocation: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	if err := s.metadataLifecycle.RevokeMembership(ctx, groupID, userID, actorID); err != nil {
		return fmt.Errorf("revoke membership in metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) RequestRole(ctx context.Context, groupID, role, actorID string) (domain.RoleRequest, error) {
	// TODO: create a helper for these kind of checks
	groupID = strings.TrimSpace(groupID)
	role = strings.TrimSpace(role)
	actorID = strings.TrimSpace(actorID)
	// TODO: use an enum
	if groupID == "" || actorID == "" || (role != "EDITOR" && role != "REVIEWER") {
		return domain.RoleRequest{}, fmt.Errorf("invalid role request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.RoleRequest{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	request, err := s.metadataLifecycle.RequestRole(ctx, groupID, role, actorID)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("request role in metadata-lifecycle: %w", err)
	}

	return request, nil
}

func (s *Service) ApplyRoleRequestDecision(ctx context.Context, groupID, requestID, status, actorID string) (domain.GroupMembership, error) {
	groupID = strings.TrimSpace(groupID)
	requestID = strings.TrimSpace(requestID)
	status = strings.TrimSpace(status)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || requestID == "" || actorID == "" || (status != "ACCEPTED" && status != "REJECTED") {
		return domain.GroupMembership{}, fmt.Errorf("invalid role request decision: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return domain.GroupMembership{}, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	membership, err := s.metadataLifecycle.ApplyRoleRequestDecision(ctx, groupID, requestID, status, actorID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("decide role request in metadata-lifecycle: %w", err)
	}

	return membership, nil
}

func (s *Service) WithdrawRoleRequest(ctx context.Context, groupID, requestID, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	requestID = strings.TrimSpace(requestID)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || requestID == "" || actorID == "" {
		return fmt.Errorf("invalid role request withdrawal: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	if err := s.metadataLifecycle.WithdrawRoleRequest(ctx, groupID, requestID, actorID); err != nil {
		return fmt.Errorf("withdraw role request in metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) AssignGroupRole(ctx context.Context, groupID, userID, role, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || userID == "" || actorID == "" || !validGroupRole(role) {
		return fmt.Errorf("invalid group role assignment: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	if err := s.metadataLifecycle.AssignGroupRole(ctx, groupID, userID, role, actorID); err != nil {
		return fmt.Errorf("assign group role in metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) RemoveGroupRole(ctx context.Context, groupID, userID, role, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || userID == "" || actorID == "" || !validGroupRole(role) {
		return fmt.Errorf("invalid group role removal: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	if err := s.metadataLifecycle.RemoveGroupRole(ctx, groupID, userID, role, actorID); err != nil {
		return fmt.Errorf("remove group role in metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) GrantGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || userID == "" || actorID == "" || !validGroupPermission(permission) {
		return fmt.Errorf("invalid group permission grant: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	if err := s.metadataLifecycle.GrantGroupPermission(ctx, groupID, userID, permission, actorID); err != nil {
		return fmt.Errorf("grant group permission in metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) RevokeGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || userID == "" || actorID == "" || !validGroupPermission(permission) {
		return fmt.Errorf("invalid group permission revocation: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	if err := s.metadataLifecycle.RevokeGroupPermission(ctx, groupID, userID, permission, actorID); err != nil {
		return fmt.Errorf("revoke group permission in metadata-lifecycle: %w", err)
	}

	return nil
}

// TODO: use an enum instead of this
func validGroupRole(role string) bool {
	return role == "EDITOR" || role == "REVIEWER" || role == "GROUP_ADMIN"
}

func validGroupPermission(permission string) bool {
	switch permission {
	case "resource.create", "resource.edit", "resource.submit", "resource.withdraw", "resource.reopen", "resource.discard", "resource.reject", "resource.publish", "resource.unpublish", "group.membership.manage", "group.role.manage", "group.permission.manage", "category.manage":
		return true
	default:
		return false
	}
}
