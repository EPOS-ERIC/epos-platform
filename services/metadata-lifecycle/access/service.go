// Package access provides group membership and permission operations.
package access

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	ListMemberships(context.Context, uuid.UUID) ([]domain.GroupMembership, error)
	RequestMembership(context.Context, uuid.UUID, string, []domain.GroupRole) (domain.GroupMembership, domain.TopLevelConcept, error)
	WithdrawMembershipRequest(context.Context, uuid.UUID, string) error
	DecideMembership(context.Context, uuid.UUID, string, domain.Decision, string) (domain.GroupMembership, error)
	RevokeMembership(context.Context, uuid.UUID, string) error
	RequestRole(context.Context, uuid.UUID, string, domain.GroupRole) (domain.RoleRequest, domain.TopLevelConcept, error)
	DecideRoleRequest(context.Context, uuid.UUID, uuid.UUID, domain.Decision, string) (domain.RoleRequestDecision, error)
	WithdrawRoleRequest(context.Context, uuid.UUID, uuid.UUID, string) error
	AssignRole(context.Context, uuid.UUID, string, domain.GroupRole, string) (bool, error)
	RemoveRole(context.Context, uuid.UUID, string, domain.GroupRole) error
	GrantPermission(context.Context, uuid.UUID, string, domain.GroupPermission, string) (bool, error)
	RevokePermission(context.Context, uuid.UUID, string, domain.GroupPermission) error
}

type Authorizer interface {
	AuthorizeActiveUser(context.Context, string) (domain.Result, error)
	CanListGroupMemberships(context.Context, string, uuid.UUID) (domain.Result, error)
	CanManageGroupMemberships(context.Context, string, uuid.UUID) (domain.Result, error)
	CanRevokeGroupMembership(context.Context, string, uuid.UUID, string) (domain.Result, error)
	CanRequestGroupRole(context.Context, string, uuid.UUID) (domain.Result, error)
	CanManageGroupRoles(context.Context, string, uuid.UUID) (domain.Result, error)
	CanManageSpecificGroupRole(context.Context, string, uuid.UUID, string) (domain.Result, error)
	AuthorizeGroup(context.Context, string, uuid.UUID, domain.GroupPermission) (domain.Result, error)
}

type Notifier interface {
	Build(context.Context, domain.NotificationEvent) ([]domain.Notification, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
	log        *slog.Logger
	notifier   Notifier
}

func New(log *slog.Logger, store Store, authorizer Authorizer, notifier Notifier) *Service {
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "access-service"),
		notifier:   notifier,
	}
}

func (s *Service) ListMemberships(ctx context.Context, actorID domain.Actor, groupID uuid.UUID) ([]domain.GroupMembership, error) {
	if groupID == uuid.Nil {
		return nil, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.CanListGroupMemberships(ctx, string(actorID), groupID)
	if err != nil {
		return nil, fmt.Errorf("authorize group membership list: %w", err)
	}
	if !allowed.Allowed {
		return nil, apperror.Default(apperror.CodePermissionDenied)
	}

	memberships, err := s.store.ListMemberships(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group memberships: %w", err)
	}

	return memberships, nil
}

func (s *Service) RequestMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, roles []domain.GroupRole) (domain.GroupMembership, []domain.Notification, error) {
	if groupID == uuid.Nil {
		return domain.GroupMembership{}, nil, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return domain.GroupMembership{}, nil, fmt.Errorf("authorize membership request: %w", err)
	}
	if !allowed.Allowed {
		return domain.GroupMembership{}, nil, apperror.Default(apperror.CodePermissionDenied)
	}

	membership, concept, err := s.store.RequestMembership(ctx, groupID, string(actorID), roles)
	if err != nil {
		return domain.GroupMembership{}, nil, fmt.Errorf("request group membership: %w", err)
	}

	notification, err := s.notifier.Build(ctx, domain.MembershipRequested{
		ActorID:      actorID,
		SubjectID:    membership.User.ID,
		GroupID:      groupID,
		GroupConcept: concept,
		TargetUserID: string(actorID),
	})
	if err != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventMembershipRequested, "error", err)
		notification = nil
	}
	return membership, notification, nil
}

func (s *Service) WithdrawMembershipRequest(ctx context.Context, actorID domain.Actor, groupID uuid.UUID) error {
	if groupID == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize membership withdrawal: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	if err := s.store.WithdrawMembershipRequest(ctx, groupID, string(actorID)); err != nil {
		return fmt.Errorf("withdraw group membership request: %w", err)
	}

	return nil
}

func (s *Service) DecideMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, decision domain.Decision) (domain.GroupMembership, []domain.Notification, error) {
	if groupID == uuid.Nil || userID == "" {
		return domain.GroupMembership{}, nil, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.CanManageGroupMemberships(ctx, string(actorID), groupID)
	if err != nil {
		return domain.GroupMembership{}, nil, fmt.Errorf("authorize membership decision: %w", err)
	}
	if !allowed.Allowed {
		return domain.GroupMembership{}, nil, apperror.Default(apperror.CodePermissionDenied)
	}

	membership, err := s.store.DecideMembership(ctx, groupID, userID, decision, string(actorID))
	if err != nil {
		return domain.GroupMembership{}, nil, fmt.Errorf("decide group membership: %w", err)
	}

	var event domain.NotificationEvent = domain.MembershipRejectedEvent{
		ActorID:      actorID,
		SubjectID:    userID,
		GroupID:      groupID,
		TargetUserID: userID,
	}
	if decision == domain.DecisionAccept {
		event = domain.MembershipAccepted{
			ActorID:      actorID,
			SubjectID:    userID,
			GroupID:      groupID,
			TargetUserID: userID,
		}
	}
	notification, err := s.notifier.Build(ctx, event)
	if err != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", event.NotificationKind(), "error", err)
		notification = nil
	}
	return membership, notification, nil
}

func (s *Service) RevokeMembership(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string) error {
	if groupID == uuid.Nil || userID == "" {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.CanRevokeGroupMembership(ctx, string(actorID), groupID, userID)
	if err != nil {
		return fmt.Errorf("authorize membership revocation: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	if err := s.store.RevokeMembership(ctx, groupID, userID); err != nil {
		return fmt.Errorf("revoke group membership: %w", err)
	}

	return nil
}

func (s *Service) RequestRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, role domain.GroupRole) (domain.RoleRequest, []domain.Notification, error) {
	if groupID == uuid.Nil {
		return domain.RoleRequest{}, nil, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.CanRequestGroupRole(ctx, string(actorID), groupID)
	if err != nil {
		return domain.RoleRequest{}, nil, fmt.Errorf("authorize role request: %w", err)
	}
	if !allowed.Allowed {
		return domain.RoleRequest{}, nil, apperror.Default(apperror.CodePermissionDenied)
	}

	request, concept, err := s.store.RequestRole(ctx, groupID, string(actorID), role)
	if err != nil {
		return domain.RoleRequest{}, nil, fmt.Errorf("request group role: %w", err)
	}

	notification, err := s.notifier.Build(ctx, domain.RoleRequested{
		ActorID:      actorID,
		SubjectID:    string(actorID),
		GroupID:      groupID,
		GroupConcept: concept,
		TargetUserID: string(actorID),
		Role:         string(role),
	})
	if err != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventRoleRequested, "error", err)
		notification = nil
	}
	return request, notification, nil
}

func (s *Service) DecideRoleRequest(ctx context.Context, actorID domain.Actor, groupID, requestID uuid.UUID, decision domain.Decision) (domain.GroupMembership, []domain.Notification, error) {
	if groupID == uuid.Nil || requestID == uuid.Nil {
		return domain.GroupMembership{}, nil, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.CanManageGroupRoles(ctx, string(actorID), groupID)
	if err != nil {
		return domain.GroupMembership{}, nil, fmt.Errorf("authorize role request decision: %w", err)
	}
	if !allowed.Allowed {
		return domain.GroupMembership{}, nil, apperror.Default(apperror.CodePermissionDenied)
	}

	result, err := s.store.DecideRoleRequest(ctx, groupID, requestID, decision, string(actorID))
	if err != nil {
		return domain.GroupMembership{}, nil, fmt.Errorf("decide group role request: %w", err)
	}

	var event domain.NotificationEvent = domain.RoleRequestRejectedEvent{
		ActorID:      actorID,
		SubjectID:    result.Membership.User.ID,
		GroupID:      groupID,
		TargetUserID: result.Membership.User.ID,
		Role:         string(result.Request.Role),
		AccessID:     result.Request.ID,
	}
	if decision == domain.DecisionAccept {
		event = domain.RoleRequestAcceptedEvent{
			ActorID:      actorID,
			SubjectID:    result.Membership.User.ID,
			GroupID:      groupID,
			TargetUserID: result.Membership.User.ID,
			Role:         string(result.Request.Role),
			AccessID:     result.Request.ID,
		}
	}
	notification, err := s.notifier.Build(ctx, event)
	if err != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", event.NotificationKind(), "error", err)
		notification = nil
	}
	return result.Membership, notification, nil
}

func (s *Service) WithdrawRoleRequest(ctx context.Context, actorID domain.Actor, groupID, requestID uuid.UUID) error {
	if groupID == uuid.Nil || requestID == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, string(actorID))
	if err != nil {
		return fmt.Errorf("authorize role request withdrawal: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	if err := s.store.WithdrawRoleRequest(ctx, groupID, requestID, string(actorID)); err != nil {
		return fmt.Errorf("withdraw group role request: %w", err)
	}

	return nil
}

func (s *Service) AssignRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, role domain.GroupRole) ([]domain.Notification, error) {
	if groupID == uuid.Nil || userID == "" {
		return nil, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.CanManageSpecificGroupRole(ctx, string(actorID), groupID, string(role))
	if err != nil {
		return nil, fmt.Errorf("authorize group role assignment: %w", err)
	}
	if !allowed.Allowed {
		return nil, apperror.Default(apperror.CodePermissionDenied)
	}

	changed, err := s.store.AssignRole(ctx, groupID, userID, role, string(actorID))
	if err != nil {
		return nil, fmt.Errorf("assign group role: %w", err)
	}
	if !changed {
		return nil, nil
	}

	notification, err := s.notifier.Build(ctx, domain.RoleAssigned{
		ActorID:      actorID,
		SubjectID:    userID,
		GroupID:      groupID,
		TargetUserID: userID,
		Role:         string(role),
	})
	if err != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventRoleAssigned, "error", err)
		notification = nil
	}
	return notification, nil
}

func (s *Service) RemoveRole(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, role domain.GroupRole) error {
	if groupID == uuid.Nil || userID == "" {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.CanManageSpecificGroupRole(ctx, string(actorID), groupID, string(role))
	if err != nil {
		return fmt.Errorf("authorize group role removal: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	err = s.store.RemoveRole(ctx, groupID, userID, role)
	if err != nil {
		return fmt.Errorf("remove group role: %w", err)
	}

	return nil
}

func (s *Service) GrantPermission(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, permission domain.GroupPermission) ([]domain.Notification, error) {
	if groupID == uuid.Nil || userID == "" || permission == "" {
		return nil, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeGroup(ctx, string(actorID), groupID, domain.GroupPermissionPermissionManage)
	if err != nil {
		return nil, fmt.Errorf("authorize group permission grant: %w", err)
	}
	if !allowed.Allowed {
		return nil, apperror.Default(apperror.CodePermissionDenied)
	}

	changed, err := s.store.GrantPermission(ctx, groupID, userID, permission, string(actorID))
	if err != nil {
		return nil, fmt.Errorf("grant group permission: %w", err)
	}
	if !changed {
		return nil, nil
	}

	notification, err := s.notifier.Build(ctx, domain.PermissionAssigned{
		ActorID:      actorID,
		SubjectID:    userID,
		GroupID:      groupID,
		TargetUserID: userID,
		Permission:   string(permission),
	})
	if err != nil {
		s.log.ErrorContext(ctx, "build notification notification failed", "event", domain.NotificationEventPermissionAssigned, "error", err)
		notification = nil
	}
	return notification, nil
}

func (s *Service) RevokePermission(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, userID string, permission domain.GroupPermission) error {
	if groupID == uuid.Nil || userID == "" || permission == "" {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeGroup(ctx, string(actorID), groupID, domain.GroupPermissionPermissionManage)
	if err != nil {
		return fmt.Errorf("authorize group permission revoke: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	err = s.store.RevokePermission(ctx, groupID, userID, permission)
	if err != nil {
		return fmt.Errorf("revoke group permission: %w", err)
	}

	return nil
}
