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
	backoffice *clients.Backoffice
}

func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{backoffice: backoffice}
}

func (s *Service) List(ctx context.Context) ([]domain.Group, error) {
	if s == nil || s.backoffice == nil {
		return nil, fmt.Errorf("backoffice group client is not configured")
	}

	groups, err := s.backoffice.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups from backoffice: %w", err)
	}

	return groups, nil
}

func (s *Service) Get(ctx context.Context, id string) (domain.Group, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Group{}, fmt.Errorf("invalid group id: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.Group{}, fmt.Errorf("backoffice group client is not configured")
	}

	group, err := s.backoffice.GetGroup(ctx, id)
	if err != nil {
		return domain.Group{}, fmt.Errorf("get group from backoffice: %w", err)
	}

	return group, nil
}

func (s *Service) WithdrawMembershipRequest(ctx context.Context, groupID, actorID string) error {
	groupID = strings.TrimSpace(groupID)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || actorID == "" {
		return fmt.Errorf("invalid membership request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}

	if err := s.backoffice.WithdrawMembershipRequest(ctx, groupID, actorID); err != nil {
		return fmt.Errorf("withdraw membership request from backoffice: %w", err)
	}

	return nil
}

func (s *Service) RequestMembership(ctx context.Context, groupID, actorID string, roles []string) (domain.GroupMembership, error) {
	groupID = strings.TrimSpace(groupID)
	actorID = strings.TrimSpace(actorID)
	if groupID == "" || actorID == "" {
		return domain.GroupMembership{}, fmt.Errorf("invalid membership request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.GroupMembership{}, fmt.Errorf("backoffice group client is not configured")
	}

	membership, err := s.backoffice.RequestMembership(ctx, groupID, actorID, roles)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("request membership from backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return domain.GroupMembership{}, fmt.Errorf("backoffice group client is not configured")
	}

	membership, err := s.backoffice.DecideMembership(ctx, groupID, userID, status, actorID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("decide membership in backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}

	if err := s.backoffice.RevokeMembership(ctx, groupID, userID, actorID); err != nil {
		return fmt.Errorf("revoke membership in backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return domain.RoleRequest{}, fmt.Errorf("backoffice group client is not configured")
	}

	request, err := s.backoffice.RequestRole(ctx, groupID, role, actorID)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("request role in backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return domain.GroupMembership{}, fmt.Errorf("backoffice group client is not configured")
	}

	membership, err := s.backoffice.ApplyRoleRequestDecision(ctx, groupID, requestID, status, actorID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("decide role request in backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}

	if err := s.backoffice.WithdrawRoleRequest(ctx, groupID, requestID, actorID); err != nil {
		return fmt.Errorf("withdraw role request in backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}

	if err := s.backoffice.AssignGroupRole(ctx, groupID, userID, role, actorID); err != nil {
		return fmt.Errorf("assign group role in backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}

	if err := s.backoffice.RemoveGroupRole(ctx, groupID, userID, role, actorID); err != nil {
		return fmt.Errorf("remove group role in backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}

	if err := s.backoffice.GrantGroupPermission(ctx, groupID, userID, permission, actorID); err != nil {
		return fmt.Errorf("grant group permission in backoffice: %w", err)
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
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice group client is not configured")
	}

	if err := s.backoffice.RevokeGroupPermission(ctx, groupID, userID, permission, actorID); err != nil {
		return fmt.Errorf("revoke group permission in backoffice: %w", err)
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
