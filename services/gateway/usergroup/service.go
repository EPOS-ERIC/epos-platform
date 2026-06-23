// Package usergroup contains resource detail domain use-cases.
package usergroup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/usergroupclient"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Identity struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

type Service struct {
	log             *slog.Logger
	usergroupClient *usergroupclient.APIClient
}

// NewService builds usergroup service for local identity lookup and creation.
func NewService(usergroupClient *usergroupclient.APIClient) *Service {
	return &Service{
		log:             logging.Get("usergroup-service"),
		usergroupClient: usergroupClient,
	}
}

func annotateSpan(ctx context.Context, attrs ...attribute.KeyValue) {
	if len(attrs) == 0 {
		return
	}

	trace.SpanFromContext(ctx).SetAttributes(attrs...)
}

func (s *Service) upstreamError(ctx context.Context, err error, op string) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		annotateSpan(ctx, attribute.String("usergroup.result", "timeout"))
		s.log.ErrorContext(ctx, op+" timeout", "error", err)
		return fmt.Errorf("%s timeout: %w: %w", op, domain.ErrUpstreamTimeout, err)
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "error"))
	s.log.ErrorContext(ctx, op+" failed", "error", err)
	return fmt.Errorf("%s: %w", op, err)
}

// EnsureUser returns existing local user or creates one from auth identity.
func (s *Service) EnsureUser(ctx context.Context, identity Identity) (api.User, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "ensure_user"))

	if identity.AuthIdentifier == "" {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.User{}, fmt.Errorf("ensure user: %w", domain.ErrBadRequest)
	}

	user, err := s.GetUserByAuthIdentifier(ctx, identity.AuthIdentifier)
	if err == nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "existing"))
		return user, nil
	}

	if !errors.Is(err, domain.ErrNotFound) {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "ensure user lookup failed", "error", err)
		return api.User{}, err
	}

	createRequest := usergroupclient.NewCreateUserRequest(identity.AuthIdentifier)
	status := usergroupclient.UserStatusActive
	createRequest.SetStatus(status)
	if identity.Email != "" {
		createRequest.SetEmail(identity.Email)
	}
	if identity.GivenName != "" {
		createRequest.SetGivenName(identity.GivenName)
	}
	if identity.FamilyName != "" {
		createRequest.SetFamilyName(identity.FamilyName)
	}

	created, resp, err := s.usergroupClient.UsersAPI.ApiV2UsersPost(ctx).CreateUserRequest(*createRequest).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusConflict {
			annotateSpan(ctx, attribute.String("usergroup.result", "conflict_retry"))
			return s.GetUserByAuthIdentifier(ctx, identity.AuthIdentifier)
		}

		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "create user failed", "error", err)
		return api.User{}, fmt.Errorf("create user: %w", err)
	}

	user, mapErr := mapUser(created)
	if mapErr != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map created user failed", "error", mapErr)
		return api.User{}, mapErr
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "created"))
	return user, nil
}

// ListUsers fetches all local users.
func (s *Service) ListUsers(ctx context.Context, actor api.User) ([]api.UserSummary, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "list_users"))

	if !actor.IsSuperAdmin {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return nil, fmt.Errorf("list users: %w", domain.ErrForbidden)
	}

	users, _, err := s.usergroupClient.UsersAPI.ApiV2UsersGet(ctx).Execute()
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "list users failed", "error", err)
		return nil, fmt.Errorf("list users from usergroup API: %w", err)
	}

	mapped := make([]api.UserSummary, 0, len(users))
	for i := range users {
		user, err := mapUserSummary(&users[i])
		if err != nil {
			annotateSpan(ctx, attribute.String("usergroup.result", "error"))
			s.log.ErrorContext(ctx, "list users map failed", "error", err)
			return nil, fmt.Errorf("list users: map user %d: %w", i, err)
		}
		mapped = append(mapped, user)
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}

func mapUserSummary(user *usergroupclient.User) (api.UserSummary, error) {
	if user == nil {
		return api.UserSummary{}, fmt.Errorf("get user: %w", domain.ErrNotFound)
	}

	status, err := api.NewUserStatusFromValue(string(user.GetStatus()))
	if err != nil {
		return api.UserSummary{}, fmt.Errorf("parse user status: %w", err)
	}

	return api.UserSummary{
		ID:             user.GetId(),
		AuthIdentifier: user.GetAuthIdentifier(),
		Email:          user.GetEmail(),
		GivenName:      user.GetGivenName(),
		FamilyName:     user.GetFamilyName(),
		Status:         status,
		IsSuperAdmin:   user.GetIsSuperAdmin(),
		CreatedAt:      formatTime(user.GetCreatedAtOk()),
		UpdatedAt:      formatTime(user.GetUpdatedAtOk()),
	}, nil
}

// GetUser fetches local user by internal id.
func (s *Service) GetUser(ctx context.Context, actor api.User, id string) (api.User, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "get_user"))

	if !actor.IsSuperAdmin && actor.ID != id {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return api.User{}, fmt.Errorf("get user: %w", domain.ErrForbidden)
	}

	user, resp, err := s.usergroupClient.UsersAPI.ApiV2UsersUserIdGet(ctx, id).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return api.User{}, domain.ErrNotFound
		}

		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "get user failed", "error", err)
		return api.User{}, fmt.Errorf("get user: %w", err)
	}

	mapped, err := mapUser(user)
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map user failed", "error", err)
		return api.User{}, err
	}

	groups, _, err := s.usergroupClient.GroupsAPI.ApiV2GroupsGet(ctx).Execute()
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "list user memberships failed", "error", err)
		return api.User{}, fmt.Errorf("list groups for memberships: %w", err)
	}

	memberships := make([]api.UserMembershipSummary, 0)
	for i := range groups {
		group := &groups[i]
		groupID := group.GetId()
		groupMemberships, _, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsGet(ctx, groupID).Execute()
		if err != nil {
			annotateSpan(ctx, attribute.String("usergroup.result", "error"))
			s.log.ErrorContext(ctx, "list user memberships failed", "error", err)
			return api.User{}, fmt.Errorf("list memberships for group %s: %w", groupID, err)
		}

		for j := range groupMemberships {
			membership := &groupMemberships[j]
			if membership.GetUserId() != id {
				continue
			}

			entry, err := mapUserMembershipSummary(group, membership)
			if err != nil {
				annotateSpan(ctx, attribute.String("usergroup.result", "error"))
				s.log.ErrorContext(ctx, "list user memberships failed", "error", err)
				return api.User{}, err
			}
			memberships = append(memberships, entry)
		}
	}
	mapped.Memberships = memberships

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}

func mapUserMembershipSummary(group *usergroupclient.Group, membership *usergroupclient.Membership) (api.UserMembershipSummary, error) {
	if group == nil || membership == nil {
		return api.UserMembershipSummary{}, fmt.Errorf("map membership summary: %w", domain.ErrNotFound)
	}

	groupID := group.GetId()
	if groupID == "" {
		return api.UserMembershipSummary{}, fmt.Errorf("map membership summary: %w", domain.ErrBadRequest)
	}

	return api.UserMembershipSummary{
		Group: api.ResourceRef{
			ID:   groupID,
			Name: group.GetName(),
			Href: fmt.Sprintf("v1/groups/%s", groupID),
		},
		Role:   api.MembershipRole(membership.GetRole()),
		Status: api.MembershipStatus(membership.GetStatus()),
	}, nil
}

// ListGroups fetches all groups and caller membership preview when present.
func (s *Service) ListGroups(ctx context.Context, actor api.User) ([]api.GroupSummary, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "list_groups"))

	groups, _, err := s.usergroupClient.GroupsAPI.ApiV2GroupsGet(ctx).Execute()
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "list groups failed", "error", err)
		return nil, fmt.Errorf("list groups from usergroup API: %w", err)
	}

	mapped := make([]api.GroupSummary, 0, len(groups))
	for i := range groups {
		group := &groups[i]
		groupSummary, err := mapGroupSummary(group)
		if err != nil {
			annotateSpan(ctx, attribute.String("usergroup.result", "error"))
			s.log.ErrorContext(ctx, "list groups map failed", "error", err)
			return nil, fmt.Errorf("list groups: map group %d: %w", i, err)
		}

		// TODO: see if we need to do this in a goroutine
		groupMemberships, _, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsGet(ctx, groupSummary.ID).Execute()
		if err != nil {
			annotateSpan(ctx, attribute.String("usergroup.result", "error"))
			s.log.ErrorContext(ctx, "list group memberships failed", "error", err)
			return nil, fmt.Errorf("list memberships for group %s: %w", groupSummary.ID, err)
		}

		for j := range groupMemberships {
			membership := &groupMemberships[j]
			if membership.GetUserId() != actor.ID {
				continue
			}

			viewerMembership, err := mapViewerMembershipSummary(membership)
			if err != nil {
				annotateSpan(ctx, attribute.String("usergroup.result", "error"))
				s.log.ErrorContext(ctx, "list group membership map failed", "error", err)
				return nil, err
			}
			groupSummary.Membership = &viewerMembership
			break
		}

		mapped = append(mapped, groupSummary)
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}

func mapGroupSummary(group *usergroupclient.Group) (api.GroupSummary, error) {
	if group == nil {
		return api.GroupSummary{}, fmt.Errorf("get group: %w", domain.ErrNotFound)
	}

	groupID := group.GetId()
	if groupID == "" {
		return api.GroupSummary{}, fmt.Errorf("get group: %w", domain.ErrNotFound)
	}

	return api.GroupSummary{
		ID:          groupID,
		Slug:        group.GetSlug(),
		Name:        group.GetName(),
		Description: group.GetDescription(),
		IsSystem:    group.GetIsSystem(),
		Membership:  nil,
	}, nil
}

func mapViewerMembershipSummary(membership *usergroupclient.Membership) (api.ViewerMembershipSummary, error) {
	if membership == nil {
		return api.ViewerMembershipSummary{}, fmt.Errorf("map membership summary: %w", domain.ErrNotFound)
	}

	return api.ViewerMembershipSummary{
		Role:   api.MembershipRole(membership.GetRole()),
		Status: api.MembershipStatus(membership.GetStatus()),
	}, nil
}

func mapMembership(membership *usergroupclient.Membership) (api.Membership, error) {
	if membership == nil {
		return api.Membership{}, fmt.Errorf("map membership: %w", domain.ErrNotFound)
	}

	userID := membership.GetUserId()
	groupID := membership.GetGroupId()
	if userID == "" || groupID == "" {
		return api.Membership{}, fmt.Errorf("map membership: %w", domain.ErrNotFound)
	}

	return api.Membership{
		UserID:    userID,
		GroupID:   groupID,
		Role:      api.MembershipRole(membership.GetRole()),
		Status:    api.MembershipStatus(membership.GetStatus()),
		CreatedAt: formatTime(membership.GetCreatedAtOk()),
		UpdatedAt: formatTime(membership.GetUpdatedAtOk()),
	}, nil
}

func (s *Service) mapGroupMembershipSummary(ctx context.Context, membership *usergroupclient.Membership) (api.GroupMembershipSummary, error) {
	if membership == nil {
		return api.GroupMembershipSummary{}, fmt.Errorf("map group membership summary: %w", domain.ErrNotFound)
	}

	userID := membership.GetUserId()
	if userID == "" {
		return api.GroupMembershipSummary{}, fmt.Errorf("map group membership summary: %w", domain.ErrBadRequest)
	}

	userRef := api.ResourceRef{
		ID:   userID,
		Href: fmt.Sprintf("v1/users/%s", userID),
		Name: "",
	}

	user, resp, err := s.usergroupClient.UsersAPI.ApiV2UsersUserIdGet(ctx, userID).Execute()
	if err == nil {
		name := strings.TrimSpace(user.GetGivenName() + " " + user.GetFamilyName())
		if name == "" {
			name = strings.TrimSpace(user.GetEmail())
		}
		userRef.Name = name
	} else if resp != nil && resp.StatusCode != http.StatusNotFound {
		s.log.WarnContext(ctx, "lookup group membership user failed", "user_id", userID, "error", err)
	}

	return api.GroupMembershipSummary{
		User:   userRef,
		Role:   api.MembershipRole(membership.GetRole()),
		Status: api.MembershipStatus(membership.GetStatus()),
	}, nil
}

func (s *Service) isActiveGroupAdmin(ctx context.Context, actor api.User, groupID string) (bool, error) {
	if actor.IsSuperAdmin {
		return true, nil
	}

	memberships, resp, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsGet(ctx, groupID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			return false, domain.ErrBadRequest
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, domain.ErrNotFound
		}
		return false, fmt.Errorf("list memberships for group %s: %w", groupID, err)
	}

	for i := range memberships {
		membership := &memberships[i]
		if membership.GetUserId() != actor.ID {
			continue
		}
		if membership.GetRole() == usergroupclient.ADMIN && membership.GetStatus() == usergroupclient.MembershipStatusActive {
			return true, nil
		}
	}

	return false, nil
}

// PatchMembership updates role or status for an existing membership.
func (s *Service) PatchMembership(ctx context.Context, actor api.User, groupID, userID string, patch api.PatchMembershipRequest) (api.Membership, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "patch_membership"))

	allowed, err := s.isActiveGroupAdmin(ctx, actor, groupID)
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "group admin check failed", "error", err)
		return api.Membership{}, err
	}
	if !allowed {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return api.Membership{}, fmt.Errorf("patch membership: %w", domain.ErrForbidden)
	}

	if patch.IsEmpty() {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.Membership{}, fmt.Errorf("patch membership: %w", domain.ErrBadRequest)
	}
	if patch.Role != nil && !patch.Role.IsValid() {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.Membership{}, fmt.Errorf("patch membership: %w", domain.ErrBadRequest)
	}
	if patch.Status != nil && !patch.Status.IsValid() {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.Membership{}, fmt.Errorf("patch membership: %w", domain.ErrBadRequest)
	}

	memberships, resp, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsGet(ctx, groupID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return api.Membership{}, domain.ErrNotFound
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return api.Membership{}, domain.ErrBadRequest
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			annotateSpan(ctx, attribute.String("usergroup.result", "timeout"))
			s.log.ErrorContext(ctx, "list group memberships timeout", "error", err)
			return api.Membership{}, fmt.Errorf("list group memberships timeout: %w: %w", domain.ErrUpstreamTimeout, err)
		}

		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "list group memberships failed", "error", err)
		return api.Membership{}, fmt.Errorf("list memberships for group %s: %w", groupID, err)
	}

	var current *usergroupclient.Membership
	for i := range memberships {
		if memberships[i].GetUserId() == userID {
			current = &memberships[i]
			break
		}
	}
	if current == nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
		return api.Membership{}, domain.ErrNotFound
	}

	role := api.MembershipRole(current.GetRole())
	if patch.Role != nil {
		role = *patch.Role
	}
	status := api.MembershipStatus(current.GetStatus())
	if patch.Status != nil {
		status = *patch.Status
	}

	request := usergroupclient.NewUpsertMembershipRequest(usergroupclient.MembershipRole(role), usergroupclient.MembershipStatus(status))
	updated, resp, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsUserIdPut(ctx, groupID, userID).UpsertMembershipRequest(*request).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return api.Membership{}, domain.ErrNotFound
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return api.Membership{}, domain.ErrBadRequest
		}
		return api.Membership{}, s.upstreamError(ctx, err, "patch membership")
	}

	mapped, mapErr := mapMembership(updated)
	if mapErr != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map patched membership failed", "error", mapErr)
		return api.Membership{}, mapErr
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}

// DeleteMembership removes a membership from a group.
func (s *Service) DeleteMembership(ctx context.Context, actor api.User, groupID, userID string) error {
	annotateSpan(ctx, attribute.String("usergroup.operation", "delete_membership"))

	if actor.ID != userID {
		allowed, err := s.isActiveGroupAdmin(ctx, actor, groupID)
		if err != nil {
			annotateSpan(ctx, attribute.String("usergroup.result", "error"))
			s.log.ErrorContext(ctx, "group admin check failed", "error", err)
			return err
		}
		if !allowed {
			annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
			return fmt.Errorf("delete membership: %w", domain.ErrForbidden)
		}
	}

	resp, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsUserIdDelete(ctx, groupID, userID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return domain.ErrNotFound
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return domain.ErrBadRequest
		}
		return s.upstreamError(ctx, err, "delete membership")
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return nil
}

// ListGroupMemberships fetches all memberships for a group.
func (s *Service) ListGroupMemberships(ctx context.Context, actor api.User, groupID string) ([]api.GroupMembershipSummary, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "list_group_memberships"))

	allowed, err := s.isActiveGroupAdmin(ctx, actor, groupID)
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "group admin check failed", "error", err)
		return nil, err
	}
	if !allowed {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return nil, fmt.Errorf("list group memberships: %w", domain.ErrForbidden)
	}

	memberships, resp, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsGet(ctx, groupID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return nil, domain.ErrBadRequest
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return nil, domain.ErrNotFound
		}
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "list group memberships failed", "error", err)
		return nil, fmt.Errorf("list memberships for group %s: %w", groupID, err)
	}

	mapped := make([]api.GroupMembershipSummary, 0, len(memberships))
	for i := range memberships {
		entry, mapErr := s.mapGroupMembershipSummary(ctx, &memberships[i])
		if mapErr != nil {
			annotateSpan(ctx, attribute.String("usergroup.result", "error"))
			s.log.ErrorContext(ctx, "map group membership failed", "error", mapErr)
			return nil, mapErr
		}
		mapped = append(mapped, entry)
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}

// GetGroup fetches group detail and caller membership preview.
func (s *Service) GetGroup(ctx context.Context, actor api.User, groupID string) (api.GroupSummary, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "get_group"))

	group, resp, err := s.usergroupClient.GroupsAPI.ApiV2GroupsGroupIdGet(ctx, groupID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return api.GroupSummary{}, domain.ErrBadRequest
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return api.GroupSummary{}, domain.ErrNotFound
		}

		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "get group failed", "error", err)
		return api.GroupSummary{}, fmt.Errorf("get group: %w", err)
	}

	groupSummary, err := mapGroupSummary(group)
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map group failed", "error", err)
		return api.GroupSummary{}, err
	}

	memberships, _, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsGet(ctx, groupID).Execute()
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "list group memberships failed", "error", err)
		return api.GroupSummary{}, fmt.Errorf("list memberships for group %s: %w", groupID, err)
	}

	for i := range memberships {
		membership := &memberships[i]
		if membership.GetUserId() != actor.ID {
			continue
		}

		viewerMembership, err := mapViewerMembershipSummary(membership)
		if err != nil {
			annotateSpan(ctx, attribute.String("usergroup.result", "error"))
			s.log.ErrorContext(ctx, "map group membership failed", "error", err)
			return api.GroupSummary{}, err
		}
		groupSummary.Membership = &viewerMembership
		break
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return groupSummary, nil
}

// PatchGroup updates mutable group fields.
func (s *Service) PatchGroup(ctx context.Context, actor api.User, groupID string, patch api.PatchGroupRequest) (api.Group, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "patch_group"))

	allowed, err := s.isActiveGroupAdmin(ctx, actor, groupID)
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "group admin check failed", "error", err)
		return api.Group{}, err
	}
	if !allowed {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return api.Group{}, fmt.Errorf("patch group: %w", domain.ErrForbidden)
	}

	request := usergroupclient.NewPatchGroupRequest()
	if patch.Slug != nil {
		request.SetSlug(*patch.Slug)
	}
	if patch.Name != nil {
		request.SetName(*patch.Name)
	}
	if patch.Description != nil {
		request.SetDescription(*patch.Description)
	}
	if patch.IsSystem != nil {
		request.SetIsSystem(*patch.IsSystem)
	}

	group, resp, err := s.usergroupClient.GroupsAPI.ApiV2GroupsGroupIdPatch(ctx, groupID).PatchGroupRequest(*request).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusConflict {
			annotateSpan(ctx, attribute.String("usergroup.result", "conflict"))
			return api.Group{}, domain.ErrConflict
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return api.Group{}, domain.ErrNotFound
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return api.Group{}, domain.ErrBadRequest
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			annotateSpan(ctx, attribute.String("usergroup.result", "timeout"))
			s.log.ErrorContext(ctx, "patch group timeout", "error", err)
			return api.Group{}, fmt.Errorf("patch group timeout: %w: %w", domain.ErrUpstreamTimeout, err)
		}

		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "patch group failed", "error", err)
		return api.Group{}, fmt.Errorf("patch group: %w", err)
	}

	mapped, mapErr := mapGroup(group)
	if mapErr != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map patched group failed", "error", mapErr)
		return api.Group{}, mapErr
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}

// DeleteGroup removes group record.
func (s *Service) DeleteGroup(ctx context.Context, actor api.User, groupID string) error {
	annotateSpan(ctx, attribute.String("usergroup.operation", "delete_group"))

	if !actor.IsSuperAdmin {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return fmt.Errorf("delete group: %w", domain.ErrForbidden)
	}

	resp, err := s.usergroupClient.GroupsAPI.ApiV2GroupsGroupIdDelete(ctx, groupID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return domain.ErrBadRequest
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return domain.ErrNotFound
		}
		return s.upstreamError(ctx, err, "delete group")
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return nil
}

// RequestMembership creates pending membership request for authenticated user.
func (s *Service) RequestMembership(ctx context.Context, actor api.User, groupID string, req api.CreateMembershipRequest) (api.Membership, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "request_membership"))

	if strings.TrimSpace(actor.ID) == "" {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return api.Membership{}, fmt.Errorf("request membership: %w", domain.ErrForbidden)
	}

	if !req.Role.IsValid() {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.Membership{}, fmt.Errorf("request membership: %w", domain.ErrBadRequest)
	}

	status := usergroupclient.MembershipStatusPending
	if actor.IsSuperAdmin {
		status = usergroupclient.MembershipStatusActive
	}
	request := usergroupclient.NewUpsertMembershipRequest(usergroupclient.MembershipRole(req.Role), status)
	created, resp, err := s.usergroupClient.MembershipsAPI.ApiV2GroupsGroupIdMembershipsUserIdPut(ctx, groupID, actor.ID).UpsertMembershipRequest(*request).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return api.Membership{}, domain.ErrNotFound
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return api.Membership{}, domain.ErrBadRequest
		}
		return api.Membership{}, s.upstreamError(ctx, err, "request membership")
	}

	mapped, mapErr := mapMembership(created)
	if mapErr != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map requested membership failed", "error", mapErr)
		return api.Membership{}, mapErr
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "created"))
	return mapped, nil
}

// CreateGroup creates a new group record.
func (s *Service) CreateGroup(ctx context.Context, actor api.User, req api.CreateGroupRequest) (api.Group, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "create_group"))

	if !actor.IsSuperAdmin {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return api.Group{}, fmt.Errorf("create group: %w", domain.ErrForbidden)
	}

	if strings.TrimSpace(req.Slug) == "" || strings.TrimSpace(req.Name) == "" {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.Group{}, fmt.Errorf("create group: %w", domain.ErrBadRequest)
	}

	createRequest := usergroupclient.NewCreateGroupRequest(req.Slug, req.Name)
	if req.Description != nil {
		createRequest.SetDescription(*req.Description)
	}
	if req.IsSystem != nil {
		createRequest.SetIsSystem(*req.IsSystem)
	}

	created, resp, err := s.usergroupClient.GroupsAPI.ApiV2GroupsPost(ctx).CreateGroupRequest(*createRequest).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusConflict {
			annotateSpan(ctx, attribute.String("usergroup.result", "conflict"))
			return api.Group{}, domain.ErrConflict
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return api.Group{}, domain.ErrBadRequest
		}
		return api.Group{}, s.upstreamError(ctx, err, "create group")
	}

	group, mapErr := mapGroup(created)
	if mapErr != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map created group failed", "error", mapErr)
		return api.Group{}, mapErr
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "created"))
	return group, nil
}

func mapGroup(group *usergroupclient.Group) (api.Group, error) {
	if group == nil {
		return api.Group{}, fmt.Errorf("get group: %w", domain.ErrNotFound)
	}

	groupID := group.GetId()
	if groupID == "" {
		return api.Group{}, fmt.Errorf("get group: %w", domain.ErrNotFound)
	}

	return api.Group{
		ID:          groupID,
		Slug:        group.GetSlug(),
		Name:        group.GetName(),
		Description: group.GetDescription(),
		IsSystem:    group.GetIsSystem(),
		Membership:  nil,
	}, nil
}

// PatchUser updates mutable local user fields.
func (s *Service) PatchUser(ctx context.Context, actor api.User, id string, patch api.PatchUserRequest) (api.User, error) {
	annotateSpan(ctx, attribute.String("usergroup.operation", "patch_user"))

	if !actor.IsSuperAdmin && actor.ID != id {
		annotateSpan(ctx, attribute.String("usergroup.result", "forbidden"))
		return api.User{}, fmt.Errorf("patch user: %w", domain.ErrForbidden)
	}

	if !actor.IsSuperAdmin && (patch.IsSuperAdmin != nil || patch.Status != nil) {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.User{}, fmt.Errorf("patch user: %w", domain.ErrBadRequest)
	}

	if patch.Status != nil && !patch.Status.IsValid() {
		annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
		return api.User{}, fmt.Errorf("patch user: %w", domain.ErrBadRequest)
	}

	request := usergroupclient.NewPatchUserRequest()
	if patch.Email != nil {
		request.SetEmail(*patch.Email)
	}
	if patch.GivenName != nil {
		request.SetGivenName(*patch.GivenName)
	}
	if patch.FamilyName != nil {
		request.SetFamilyName(*patch.FamilyName)
	}
	if patch.IsSuperAdmin != nil {
		request.SetIsSuperAdmin(*patch.IsSuperAdmin)
	}
	if patch.Status != nil {
		request.SetStatus(usergroupclient.UserStatus(*patch.Status))
	}

	user, resp, err := s.usergroupClient.UsersAPI.ApiV2UsersUserIdPatch(ctx, id).PatchUserRequest(*request).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			annotateSpan(ctx, attribute.String("usergroup.result", "not_found"))
			return api.User{}, domain.ErrNotFound
		}

		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			annotateSpan(ctx, attribute.String("usergroup.result", "bad_request"))
			return api.User{}, domain.ErrBadRequest
		}
		return api.User{}, s.upstreamError(ctx, err, "patch user")
	}

	mapped, err := mapUser(user)
	if err != nil {
		annotateSpan(ctx, attribute.String("usergroup.result", "error"))
		s.log.ErrorContext(ctx, "map patched user failed", "error", err)
		return api.User{}, err
	}

	annotateSpan(ctx, attribute.String("usergroup.result", "success"))
	return mapped, nil
}
