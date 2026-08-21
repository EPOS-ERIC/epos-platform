package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("group access operation forbidden")
	ErrConflict  = errors.New("group access operation conflict")
)

type MembershipUser struct {
	ID         string
	GivenName  string
	FamilyName string
	Email      string
}

type RoleRequest struct {
	ID          uuid.UUID
	Role        string
	Status      string
	RequestedAt time.Time
}

type Membership struct {
	User         MembershipUser
	Status       string
	Roles        []string
	Permissions  []string
	RoleRequests []RoleRequest
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type MembershipRequest struct {
	Roles []string
}

type AccessStore interface {
	ListMemberships(context.Context, uuid.UUID) ([]Membership, error)
	RequestMembership(context.Context, uuid.UUID, string, []string) (Membership, error)
	WithdrawMembershipRequest(context.Context, uuid.UUID, string) error
	DecideMembership(context.Context, uuid.UUID, string, string, string) (Membership, error)
	RevokeMembership(context.Context, uuid.UUID, string) error
	RequestRole(context.Context, uuid.UUID, string, string) (RoleRequest, error)
	ApplyRoleRequestDecision(context.Context, uuid.UUID, uuid.UUID, string, string) (Membership, error)
	WithdrawRoleRequest(context.Context, uuid.UUID, uuid.UUID, string) error
	AssignRole(context.Context, uuid.UUID, string, string, string) error
	RemoveRole(context.Context, uuid.UUID, string, string) error
}

type AccessAuthorizer interface {
	AuthorizeActiveUser(context.Context, string) (authz.Result, error)
	CanListGroupMemberships(context.Context, string, uuid.UUID) (authz.Result, error)
	CanManageGroupMemberships(context.Context, string, uuid.UUID) (authz.Result, error)
	CanRevokeGroupMembership(context.Context, string, uuid.UUID, string) (authz.Result, error)
	CanRequestGroupRole(context.Context, string, uuid.UUID) (authz.Result, error)
	CanManageGroupRoles(context.Context, string, uuid.UUID) (authz.Result, error)
	CanManageSpecificGroupRole(context.Context, string, uuid.UUID, string) (authz.Result, error)
}

type AccessService struct {
	store      AccessStore
	authorizer AccessAuthorizer
}

func NewAccess(store AccessStore, authorizer AccessAuthorizer) *AccessService {
	return &AccessService{store: store, authorizer: authorizer}
}

func (s *AccessService) ListMemberships(ctx context.Context, actorID string, groupID uuid.UUID) ([]Membership, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || groupID == uuid.Nil {
		return nil, ErrInvalid
	}

	allowed, err := s.authorizer.CanListGroupMemberships(ctx, actorID, groupID)
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

func (s *AccessService) RequestMembership(ctx context.Context, actorID string, groupID uuid.UUID, request MembershipRequest) (Membership, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || groupID == uuid.Nil {
		return Membership{}, ErrInvalid
	}

	// TODO: maybe we should put all the enums in a domain package or something so that we can then validate there and it's the same everywhere?
	if err := validateRequestedRoles(request.Roles); err != nil {
		return Membership{}, err
	}

	allowed, err := s.authorizer.CanRequestGroupRole(ctx, actorID, groupID)
	if err != nil {
		return Membership{}, fmt.Errorf("authorize membership request: %w", err)
	}

	if !allowed.Allowed {
		return Membership{}, ErrForbidden
	}

	membership, err := s.store.RequestMembership(ctx, groupID, actorID, request.Roles)
	if err != nil {
		return Membership{}, fmt.Errorf("request group membership: %w", err)
	}

	return membership, nil
}

func (s *AccessService) WithdrawMembershipRequest(ctx context.Context, actorID string, groupID uuid.UUID) error {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || groupID == uuid.Nil {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize membership withdrawal: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.WithdrawMembershipRequest(ctx, groupID, actorID); err != nil {
		return fmt.Errorf("withdraw group membership request: %w", err)
	}

	return nil
}

func (s *AccessService) DecideMembership(ctx context.Context, actorID string, groupID uuid.UUID, userID, status string) (Membership, error) {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	status = strings.TrimSpace(status)
	if actorID == "" || userID == "" || groupID == uuid.Nil {
		return Membership{}, ErrInvalid
	}

	// TODO: use an enum
	if status != "ACTIVE" && status != "REJECTED" {
		return Membership{}, ErrInvalid
	}

	allowed, err := s.authorizer.CanManageGroupMemberships(ctx, actorID, groupID)
	if err != nil {
		return Membership{}, fmt.Errorf("authorize membership decision: %w", err)
	}
	if !allowed.Allowed {
		return Membership{}, ErrForbidden
	}

	membership, err := s.store.DecideMembership(ctx, groupID, userID, status, actorID)
	if err != nil {
		return Membership{}, fmt.Errorf("decide group membership: %w", err)
	}

	return membership, nil
}

func (s *AccessService) RevokeMembership(ctx context.Context, actorID string, groupID uuid.UUID, userID string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	if actorID == "" || userID == "" || groupID == uuid.Nil {
		return ErrInvalid
	}

	allowed, err := s.authorizer.CanRevokeGroupMembership(ctx, actorID, groupID, userID)
	if err != nil {
		return fmt.Errorf("authorize membership revocation: %w", err)
	}

	if !allowed.Allowed {
		return ErrForbidden
	}

	err = s.store.RevokeMembership(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("revoke group membership: %w", err)
	}

	return nil
}

func (s *AccessService) RequestRole(ctx context.Context, actorID string, groupID uuid.UUID, role string) (RoleRequest, error) {
	actorID = strings.TrimSpace(actorID)
	role = strings.TrimSpace(role)
	// TODO: use an enum with validation functions
	if actorID == "" || groupID == uuid.Nil || (role != "EDITOR" && role != "REVIEWER") {
		return RoleRequest{}, ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, actorID)
	if err != nil {
		return RoleRequest{}, fmt.Errorf("authorize role request: %w", err)
	}

	if !allowed.Allowed {
		return RoleRequest{}, ErrForbidden
	}

	request, err := s.store.RequestRole(ctx, groupID, actorID, role)
	if err != nil {
		return RoleRequest{}, fmt.Errorf("request group role: %w", err)
	}

	return request, nil
}

func (s *AccessService) ApplyRoleRequestDecision(ctx context.Context, actorID string, groupID, requestID uuid.UUID, status string) (Membership, error) {
	actorID = strings.TrimSpace(actorID)
	status = strings.TrimSpace(status)
	if actorID == "" || groupID == uuid.Nil || requestID == uuid.Nil || (status != "ACCEPTED" && status != "REJECTED") {
		return Membership{}, ErrInvalid
	}

	allowed, err := s.authorizer.CanManageGroupRoles(ctx, actorID, groupID)
	if err != nil {
		return Membership{}, fmt.Errorf("authorize role request decision: %w", err)
	}
	if !allowed.Allowed {
		return Membership{}, ErrForbidden
	}

	membership, err := s.store.ApplyRoleRequestDecision(ctx, groupID, requestID, status, actorID)
	if err != nil {
		return Membership{}, fmt.Errorf("decide group role request: %w", err)
	}

	return membership, nil
}

func (s *AccessService) WithdrawRoleRequest(ctx context.Context, actorID string, groupID, requestID uuid.UUID) error {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || groupID == uuid.Nil || requestID == uuid.Nil {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize role request withdrawal: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.WithdrawRoleRequest(ctx, groupID, requestID, actorID); err != nil {
		return fmt.Errorf("withdraw group role request: %w", err)
	}

	return nil
}

func (s *AccessService) AssignRole(ctx context.Context, actorID string, groupID uuid.UUID, userID, role string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	if actorID == "" || userID == "" || groupID == uuid.Nil || !validGroupRole(role) {
		return ErrInvalid
	}

	allowed, err := s.authorizer.CanManageSpecificGroupRole(ctx, actorID, groupID, role)
	if err != nil {
		return fmt.Errorf("authorize group role assignment: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.AssignRole(ctx, groupID, userID, role, actorID); err != nil {
		return fmt.Errorf("assign group role: %w", err)
	}

	return nil
}

func (s *AccessService) RemoveRole(ctx context.Context, actorID string, groupID uuid.UUID, userID, role string) error {
	actorID = strings.TrimSpace(actorID)
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	if actorID == "" || userID == "" || groupID == uuid.Nil || !validGroupRole(role) {
		return ErrInvalid
	}

	allowed, err := s.authorizer.CanManageSpecificGroupRole(ctx, actorID, groupID, role)
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

// TODO: use an enum instead of this
func validateRequestedRoles(roles []string) error {
	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if role != "EDITOR" && role != "REVIEWER" {
			return ErrInvalid
		}
		if _, ok := seen[role]; ok {
			return ErrInvalid
		}
		seen[role] = struct{}{}
	}
	return nil
}

// TODO: use an enum instead of this
func validGroupRole(role string) bool {
	return role == "EDITOR" || role == "REVIEWER" || role == "GROUP_ADMIN"
}
