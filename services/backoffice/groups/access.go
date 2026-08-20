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
}

type AccessAuthorizer interface {
	AuthorizeActiveUser(context.Context, string) (authz.Result, error)
	CanListGroupMemberships(context.Context, string, uuid.UUID) (authz.Result, error)
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

	allowed, err := s.authorizer.AuthorizeActiveUser(ctx, actorID)
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
