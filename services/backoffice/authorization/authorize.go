// Package authorization contains access decision rules.
package authorization

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/identity"
	"github.com/google/uuid"
)

type AuthorizeRequest struct {
	AuthIdentifier          string
	Group                   string
	Service                 string
	Action                  string
	ResourceType            string
	ResourceID              string
	ResourceOwnerIdentifier string
}

type AuthorizeResult struct {
	Allowed bool
	Reason  string
}

type Authorizer struct {
	users       *identity.Users
	memberships *identity.Memberships
}

type authorizationRepository interface {
	identity.UserRepository
	identity.MembershipRepository
}

func NewAuthorizer(database authorizationRepository) *Authorizer {
	return &Authorizer{
		users:       identity.NewUsers(database),
		memberships: identity.NewMemberships(database),
	}
}

func (s *Authorizer) Decide(ctx context.Context, req AuthorizeRequest) (AuthorizeResult, error) {
	requester, err := s.users.GetByAuthIdentifier(ctx, strings.TrimSpace(req.AuthIdentifier))
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return AuthorizeResult{
				Allowed: false,
				Reason:  "requesting user not found",
			}, nil
		}
		return AuthorizeResult{}, fmt.Errorf("get requester by auth identifier: %w", err)
	}

	requesterID, err := uuid.Parse(requester.ID)
	if err != nil {
		return AuthorizeResult{}, fmt.Errorf("parse requester id: %w", err)
	}
	if requester.IsSuperAdmin {
		return AuthorizeResult{
			Allowed: true,
			Reason:  "super admin",
		}, nil
	}

	groupID, err := uuid.Parse(strings.TrimSpace(req.Group))
	if err != nil {
		return AuthorizeResult{
			Allowed: false,
			Reason:  "group is invalid",
		}, nil
	}

	isRequesterMember, err := s.memberships.IsActiveMember(ctx, requesterID, groupID)
	if err != nil {
		return AuthorizeResult{}, fmt.Errorf("check requester membership: %w", err)
	}
	if !isRequesterMember {
		return AuthorizeResult{
			Allowed: false,
			Reason:  "requesting user is not an active group member",
		}, nil
	}

	if !isDraftReadRequest(req) {
		return AuthorizeResult{
			Allowed: false,
			Reason:  "rule not applicable",
		}, nil
	}

	ownerIdentifier := strings.TrimSpace(req.ResourceOwnerIdentifier)
	if ownerIdentifier == "" {
		return AuthorizeResult{
			Allowed: false,
			Reason:  "resourceOwnerIdentifier is required for draft access",
		}, nil
	}
	if ownerIdentifier == requester.AuthIdentifier {
		return AuthorizeResult{
			Allowed: true,
			Reason:  "same owner",
		}, nil
	}

	owner, err := s.users.GetByAuthIdentifier(ctx, ownerIdentifier)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return AuthorizeResult{
				Allowed: false,
				Reason:  "draft owner not found",
			}, nil
		}
		return AuthorizeResult{}, fmt.Errorf("get owner by auth identifier: %w", err)
	}

	ownerID, err := uuid.Parse(owner.ID)
	if err != nil {
		return AuthorizeResult{}, fmt.Errorf("parse owner id: %w", err)
	}

	isOwnerMember, err := s.memberships.IsActiveMember(ctx, ownerID, groupID)
	if err != nil {
		return AuthorizeResult{}, fmt.Errorf("check owner membership: %w", err)
	}
	if !isOwnerMember {
		return AuthorizeResult{
			Allowed: false,
			Reason:  "draft owner is not an active group member",
		}, nil
	}

	return AuthorizeResult{
		Allowed: true,
		Reason:  "owners in same group can access draft",
	}, nil
}

func isDraftReadRequest(req AuthorizeRequest) bool {
	service := strings.ToLower(strings.TrimSpace(req.Service))
	action := strings.ToLower(strings.TrimSpace(req.Action))
	resourceType := strings.ToLower(strings.TrimSpace(req.ResourceType))

	if service != "metadata-lifecycle" {
		return false
	}

	if action != "read" && action != "view" && action != "get" {
		return false
	}

	return strings.Contains(resourceType, "draft") || resourceType == ""
}
