// Package access provides group membership and permission operations.
package access

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

var (
	ErrInvalid   = errors.New("invalid group access request")
	ErrNotFound  = errors.New("group access resource not found")
	ErrForbidden = errors.New("group access operation forbidden")
	ErrConflict  = errors.New("group access operation conflict")
)

type Store interface {
	ListMemberships(context.Context, uuid.UUID) ([]domain.Membership, error)
	RequestMembership(context.Context, uuid.UUID, string, []domain.GroupRole) (domain.Membership, error)
	WithdrawMembershipRequest(context.Context, uuid.UUID, string) error
	DecideMembership(context.Context, uuid.UUID, string, domain.MembershipStatus, string) (domain.Membership, error)
	RevokeMembership(context.Context, uuid.UUID, string) error
	RequestRole(context.Context, uuid.UUID, string, domain.GroupRole) (domain.RoleRequest, error)
	ApplyRoleRequestDecision(context.Context, uuid.UUID, uuid.UUID, domain.RoleRequestStatus, string) (domain.Membership, error)
	WithdrawRoleRequest(context.Context, uuid.UUID, uuid.UUID, string) error
	AssignRole(context.Context, uuid.UUID, string, domain.GroupRole, string) error
	RemoveRole(context.Context, uuid.UUID, string, domain.GroupRole) error
	GrantPermission(context.Context, uuid.UUID, string, string, string) error
	RevokePermission(context.Context, uuid.UUID, string, string) error
}

type Authorizer interface {
	AuthorizeActiveUser(context.Context, string) (authz.Result, error)
	CanListGroupMemberships(context.Context, string, uuid.UUID) (authz.Result, error)
	CanManageGroupMemberships(context.Context, string, uuid.UUID) (authz.Result, error)
	CanRevokeGroupMembership(context.Context, string, uuid.UUID, string) (authz.Result, error)
	CanRequestGroupRole(context.Context, string, uuid.UUID) (authz.Result, error)
	CanManageGroupRoles(context.Context, string, uuid.UUID) (authz.Result, error)
	CanManageSpecificGroupRole(context.Context, string, uuid.UUID, string) (authz.Result, error)
	AuthorizeGroup(context.Context, string, uuid.UUID, string) (authz.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
}

func New(store Store, authorizer Authorizer) *Service {
	return &Service{store: store, authorizer: authorizer}
}

func (s *Service) ListMemberships(ctx context.Context, actorID domain.Actor, groupID uuid.UUID) ([]domain.Membership, error) {
	if groupID == uuid.Nil {
		return nil, ErrInvalid
	}

	allowed, err := s.authorizer.CanListGroupMemberships(ctx, string(actorID), groupID)
	if err != nil {
		return nil, fmt.Errorf("authorize group membership list: %w", err)
	}
	if !allowed.Allowed {
		return nil, ErrForbidden
	}

	memberships, err := s.store.ListMemberships(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group memberships: %w", err)
	}

	return memberships, nil
}

func (s *Service) RequestMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, roles []domain.GroupRole) (domain.Membership, error) {
	if groupID == uuid.Nil {
		return domain.Membership{}, ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return domain.Membership{}, fmt.Errorf("authorize membership request: %w", err)
	}
	if !allowed.Allowed {
		return domain.Membership{}, ErrForbidden
	}

	membership, err := s.store.RequestMembership(ctx, groupID, string(actorID), roles)
	if err != nil {
		return domain.Membership{}, fmt.Errorf("request group membership: %w", err)
	}

	return membership, nil
}

func (s *Service) WithdrawMembershipRequest(ctx context.Context, actorID domain.Actor, groupID uuid.UUID) error {
	if groupID == uuid.Nil {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize membership withdrawal: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.WithdrawMembershipRequest(ctx, groupID, string(actorID)); err != nil {
		return fmt.Errorf("withdraw group membership request: %w", err)
	}

	return nil
}

func (s *Service) DecideMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, status domain.MembershipStatus) (domain.Membership, error) {
	if groupID == uuid.Nil || userID == "" {
		return domain.Membership{}, ErrInvalid
	}

	allowed, err := s.authorizer.CanManageGroupMemberships(ctx, string(actorID), groupID)
	if err != nil {
		return domain.Membership{}, fmt.Errorf("authorize membership decision: %w", err)
	}
	if !allowed.Allowed {
		return domain.Membership{}, ErrForbidden
	}

	membership, err := s.store.DecideMembership(ctx, groupID, userID, status, string(actorID))
	if err != nil {
		return domain.Membership{}, fmt.Errorf("decide group membership: %w", err)
	}

	return membership, nil
}

func (s *Service) RevokeMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string) error {
	if groupID == uuid.Nil || userID == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.CanRevokeGroupMembership(ctx, string(actorID), groupID, userID)
	if err != nil {
		return fmt.Errorf("authorize membership revocation: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.RevokeMembership(ctx, groupID, userID); err != nil {
		return fmt.Errorf("revoke group membership: %w", err)
	}

	return nil
}

func (s *Service) RequestRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, role domain.GroupRole) (domain.RoleRequest, error) {
	if groupID == uuid.Nil {
		return domain.RoleRequest{}, ErrInvalid
	}

	allowed, err := s.authorizer.CanRequestGroupRole(ctx, string(actorID), groupID)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("authorize role request: %w", err)
	}
	if !allowed.Allowed {
		return domain.RoleRequest{}, ErrForbidden
	}

	request, err := s.store.RequestRole(ctx, groupID, string(actorID), role)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("request group role: %w", err)
	}

	return request, nil
}

func (s *Service) ApplyRoleRequestDecision(ctx context.Context, actorID domain.Actor, groupID, requestID uuid.UUID, status domain.RoleRequestStatus) (domain.Membership, error) {
	if groupID == uuid.Nil || requestID == uuid.Nil {
		return domain.Membership{}, ErrInvalid
	}

	allowed, err := s.authorizer.CanManageGroupRoles(ctx, string(actorID), groupID)
	if err != nil {
		return domain.Membership{}, fmt.Errorf("authorize role request decision: %w", err)
	}
	if !allowed.Allowed {
		return domain.Membership{}, ErrForbidden
	}

	membership, err := s.store.ApplyRoleRequestDecision(ctx, groupID, requestID, status, string(actorID))
	if err != nil {
		return domain.Membership{}, fmt.Errorf("decide group role request: %w", err)
	}

	return membership, nil
}

func (s *Service) WithdrawRoleRequest(ctx context.Context, actorID domain.Actor, groupID, requestID uuid.UUID) error {
	if groupID == uuid.Nil || requestID == uuid.Nil {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize role request withdrawal: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.WithdrawRoleRequest(ctx, groupID, requestID, string(actorID)); err != nil {
		return fmt.Errorf("withdraw group role request: %w", err)
	}

	return nil
}

func (s *Service) AssignRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, role domain.GroupRole) error {
	if groupID == uuid.Nil || userID == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.CanManageSpecificGroupRole(ctx, string(actorID), groupID, string(role))
	if err != nil {
		return fmt.Errorf("authorize group role assignment: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.AssignRole(ctx, groupID, userID, role, string(actorID)); err != nil {
		return fmt.Errorf("assign group role: %w", err)
	}

	return nil
}

func (s *Service) RemoveRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, role domain.GroupRole) error {
	if groupID == uuid.Nil || userID == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.CanManageSpecificGroupRole(ctx, string(actorID), groupID, string(role))
	if err != nil {
		return fmt.Errorf("authorize group role removal: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.RemoveRole(ctx, groupID, userID, role); err != nil {
		return fmt.Errorf("remove group role: %w", err)
	}

	return nil
}

func (s *Service) GrantPermission(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID, permission string) error {
	if groupID == uuid.Nil || userID == "" || permission == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGroup(ctx, string(actorID), groupID, "group.permission.manage")
	if err != nil {
		return fmt.Errorf("authorize group permission grant: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.GrantPermission(ctx, groupID, userID, permission, string(actorID)); err != nil {
		return fmt.Errorf("grant group permission: %w", err)
	}

	return nil
}

func (s *Service) RevokePermission(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID, permission string) error {
	if groupID == uuid.Nil || userID == "" || permission == "" {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGroup(ctx, string(actorID), groupID, "group.permission.manage")
	if err != nil {
		return fmt.Errorf("authorize group permission revoke: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.RevokePermission(ctx, groupID, userID, permission); err != nil {
		return fmt.Errorf("revoke group permission: %w", err)
	}

	return nil
}
