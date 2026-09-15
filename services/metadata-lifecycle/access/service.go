// Package access provides group membership and permission operations.
package access

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	ListMemberships(context.Context, uuid.UUID) ([]domain.GroupMembership, error)
	RequestMembership(context.Context, uuid.UUID, string, []domain.GroupRole) (domain.GroupMembership, error)
	WithdrawMembershipRequest(context.Context, uuid.UUID, string) error
	DecideMembership(context.Context, uuid.UUID, string, domain.Decision, string) (domain.GroupMembership, error)
	RevokeMembership(context.Context, uuid.UUID, string) error
	RequestRole(context.Context, uuid.UUID, string, domain.GroupRole) (domain.RoleRequest, error)
	DecideRoleRequest(context.Context, uuid.UUID, uuid.UUID, domain.Decision, string) (domain.GroupMembership, error)
	WithdrawRoleRequest(context.Context, uuid.UUID, uuid.UUID, string) error
	AssignRole(context.Context, uuid.UUID, string, domain.GroupRole, string) error
	RemoveRole(context.Context, uuid.UUID, string, domain.GroupRole) error
	GrantPermission(context.Context, uuid.UUID, string, string, string) error
	RevokePermission(context.Context, uuid.UUID, string, string) error
}

type Authorizer interface {
	AuthorizeActiveUser(context.Context, string) (domain.Result, error)
	CanListGroupMemberships(context.Context, string, uuid.UUID) (domain.Result, error)
	CanManageGroupMemberships(context.Context, string, uuid.UUID) (domain.Result, error)
	CanRevokeGroupMembership(context.Context, string, uuid.UUID, string) (domain.Result, error)
	CanRequestGroupRole(context.Context, string, uuid.UUID) (domain.Result, error)
	CanManageGroupRoles(context.Context, string, uuid.UUID) (domain.Result, error)
	CanManageSpecificGroupRole(context.Context, string, uuid.UUID, string) (domain.Result, error)
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (domain.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
	log        *slog.Logger
}

func New(log *slog.Logger, store Store, authorizer Authorizer) *Service {
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "access-service"),
	}
}

func (s *Service) ListMemberships(ctx context.Context, actorID domain.Actor, groupID uuid.UUID) ([]domain.GroupMembership, error) {
	if groupID == uuid.Nil {
		return nil, domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.CanListGroupMemberships(ctx, string(actorID), groupID)
	if err != nil {
		return nil, fmt.Errorf("authorize group membership list: %w", err)
	}
	if !allowed.Allowed {
		return nil, domain.ErrAccessForbidden
	}

	memberships, err := s.store.ListMemberships(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group memberships: %w", err)
	}

	return memberships, nil
}

func (s *Service) RequestMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, roles []domain.GroupRole) (domain.GroupMembership, error) {
	if groupID == uuid.Nil {
		return domain.GroupMembership{}, domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("authorize membership request: %w", err)
	}
	if !allowed.Allowed {
		return domain.GroupMembership{}, domain.ErrAccessForbidden
	}

	membership, err := s.store.RequestMembership(ctx, groupID, string(actorID), roles)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("request group membership: %w", err)
	}

	return membership, nil
}

func (s *Service) WithdrawMembershipRequest(ctx context.Context, actorID domain.Actor, groupID uuid.UUID) error {
	if groupID == uuid.Nil {
		return domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize membership withdrawal: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrAccessForbidden
	}

	if err := s.store.WithdrawMembershipRequest(ctx, groupID, string(actorID)); err != nil {
		return fmt.Errorf("withdraw group membership request: %w", err)
	}

	return nil
}

func (s *Service) DecideMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, decision domain.Decision) (domain.GroupMembership, error) {
	if groupID == uuid.Nil || userID == "" {
		return domain.GroupMembership{}, domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.CanManageGroupMemberships(ctx, string(actorID), groupID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("authorize membership decision: %w", err)
	}
	if !allowed.Allowed {
		return domain.GroupMembership{}, domain.ErrAccessForbidden
	}

	membership, err := s.store.DecideMembership(ctx, groupID, userID, decision, string(actorID))
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("decide group membership: %w", err)
	}

	return membership, nil
}

func (s *Service) RevokeMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string) error {
	if groupID == uuid.Nil || userID == "" {
		return domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.CanRevokeGroupMembership(ctx, string(actorID), groupID, userID)
	if err != nil {
		return fmt.Errorf("authorize membership revocation: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrAccessForbidden
	}

	if err := s.store.RevokeMembership(ctx, groupID, userID); err != nil {
		return fmt.Errorf("revoke group membership: %w", err)
	}

	return nil
}

func (s *Service) RequestRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, role domain.GroupRole) (domain.RoleRequest, error) {
	if groupID == uuid.Nil {
		return domain.RoleRequest{}, domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.CanRequestGroupRole(ctx, string(actorID), groupID)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("authorize role request: %w", err)
	}
	if !allowed.Allowed {
		return domain.RoleRequest{}, domain.ErrAccessForbidden
	}

	request, err := s.store.RequestRole(ctx, groupID, string(actorID), role)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("request group role: %w", err)
	}

	return request, nil
}

func (s *Service) DecideRoleRequest(ctx context.Context, actorID domain.Actor, groupID, requestID uuid.UUID, decision domain.Decision) (domain.GroupMembership, error) {
	if groupID == uuid.Nil || requestID == uuid.Nil {
		return domain.GroupMembership{}, domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.CanManageGroupRoles(ctx, string(actorID), groupID)
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("authorize role request decision: %w", err)
	}
	if !allowed.Allowed {
		return domain.GroupMembership{}, domain.ErrAccessForbidden
	}

	membership, err := s.store.DecideRoleRequest(ctx, groupID, requestID, decision, string(actorID))
	if err != nil {
		return domain.GroupMembership{}, fmt.Errorf("decide group role request: %w", err)
	}

	return membership, nil
}

func (s *Service) WithdrawRoleRequest(ctx context.Context, actorID domain.Actor, groupID, requestID uuid.UUID) error {
	if groupID == uuid.Nil || requestID == uuid.Nil {
		return domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize role request withdrawal: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrAccessForbidden
	}

	if err := s.store.WithdrawRoleRequest(ctx, groupID, requestID, string(actorID)); err != nil {
		return fmt.Errorf("withdraw group role request: %w", err)
	}

	return nil
}

func (s *Service) AssignRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, role domain.GroupRole) error {
	if groupID == uuid.Nil || userID == "" {
		return domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.CanManageSpecificGroupRole(ctx, string(actorID), groupID, string(role))
	if err != nil {
		return fmt.Errorf("authorize group role assignment: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrAccessForbidden
	}

	if err := s.store.AssignRole(ctx, groupID, userID, role, string(actorID)); err != nil {
		return fmt.Errorf("assign group role: %w", err)
	}

	return nil
}

func (s *Service) RemoveRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, role domain.GroupRole) error {
	if groupID == uuid.Nil || userID == "" {
		return domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.CanManageSpecificGroupRole(ctx, string(actorID), groupID, string(role))
	if err != nil {
		return fmt.Errorf("authorize group role removal: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrAccessForbidden
	}

	if err := s.store.RemoveRole(ctx, groupID, userID, role); err != nil {
		return fmt.Errorf("remove group role: %w", err)
	}

	return nil
}

func (s *Service) GrantPermission(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID, permission string) error {
	if groupID == uuid.Nil || userID == "" || permission == "" {
		return domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.AuthorizeGroup(ctx, string(actorID), groupID, "group.permission.manage")
	if err != nil {
		return fmt.Errorf("authorize group permission grant: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrAccessForbidden
	}

	if err := s.store.GrantPermission(ctx, groupID, userID, permission, string(actorID)); err != nil {
		return fmt.Errorf("grant group permission: %w", err)
	}

	return nil
}

func (s *Service) RevokePermission(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID, permission string) error {
	if groupID == uuid.Nil || userID == "" || permission == "" {
		return domain.ErrInvalidAccess
	}

	allowed, err := s.authorizer.AuthorizeGroup(ctx, string(actorID), groupID, "group.permission.manage")
	if err != nil {
		return fmt.Errorf("authorize group permission revoke: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrAccessForbidden
	}

	if err := s.store.RevokePermission(ctx, groupID, userID, permission); err != nil {
		return fmt.Errorf("revoke group permission: %w", err)
	}

	return nil
}
