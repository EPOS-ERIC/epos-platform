// Package clients provides outbound service clients for the gateway.
package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

type Backoffice struct {
	identity  client.InternalIdentityAPI
	users     client.UsersAPI
	groups    client.GroupsAPI
	access    client.GroupAccessAPI
	resources client.ResourcesAPI
	imports   client.ImportsAPI
}

type ActorIdentity struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

func NewBackoffice(baseURL string, upstreamCfg config.Upstream) (*Backoffice, error) {
	normalizedBaseURL, err := urlutil.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize backoffice base URL: %w", err)
	}

	upstreamClient, err := upstream.NewClient(upstream.Config{
		Service:     "backoffice",
		Timeout:     upstreamCfg.Timeout,
		RealTimeout: upstreamCfg.RealTimeout,
		CacheTTL:    0,
		MaxEntries:  upstreamCfg.MaxEntries,
		Transport:   nil,
	})
	if err != nil {
		return nil, fmt.Errorf("create backoffice client: %w", err)
	}

	clientConfig := client.NewConfiguration()
	clientConfig.Servers[0].URL = normalizedBaseURL
	clientConfig.HTTPClient = upstreamClient.StandardClient()

	apiClient := client.NewAPIClient(clientConfig)
	return &Backoffice{
		identity:  apiClient.InternalIdentityAPI,
		users:     apiClient.UsersAPI,
		groups:    apiClient.GroupsAPI,
		access:    apiClient.GroupAccessAPI,
		resources: apiClient.ResourcesAPI,
		imports:   apiClient.ImportsAPI,
	}, nil
}

func (c *Backoffice) ListGroups(ctx context.Context) ([]domain.Group, error) {
	if c == nil || c.groups == nil {
		return nil, fmt.Errorf("backoffice group client is not configured")
	}

	groups, response, err := c.groups.ListGroups(ctx).Execute()
	if err != nil {
		return nil, wrapGroupError("list backoffice groups", response, err)
	}

	result := make([]domain.Group, 0, len(groups))
	for _, group := range groups {
		mapped, err := toDomainGroup(&group)
		if err != nil {
			return nil, err
		}

		result = append(result, mapped)
	}

	return result, nil
}

func (c *Backoffice) GetGroup(ctx context.Context, id string) (domain.Group, error) {
	if c == nil || c.groups == nil {
		return domain.Group{}, fmt.Errorf("backoffice group client is not configured")
	}

	group, response, err := c.groups.GetGroup(ctx, id).Execute()
	if err != nil {
		return domain.Group{}, wrapGroupError("get backoffice group", response, err)
	}

	return toDomainGroup(group)
}

func (c *Backoffice) WithdrawMembershipRequest(ctx context.Context, groupID, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("backoffice group access client is not configured")
	}

	response, err := c.access.WithdrawGroupMembershipRequest(ctx, groupID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("withdraw backoffice membership request", response, err)
	}

	return nil
}

func (c *Backoffice) RequestMembership(ctx context.Context, groupID, actorID string, roles []string) (domain.GroupMembership, error) {
	if c == nil || c.access == nil {
		return domain.GroupMembership{}, fmt.Errorf("backoffice group access client is not configured")
	}

	membership, response, err := c.access.RequestGroupMembership(ctx, groupID).
		XEPOSUserID(actorID).
		CreateMembershipRequest(client.CreateMembershipRequest{Roles: roles}).
		Execute()
	if err != nil {
		return domain.GroupMembership{}, wrapGroupAccessError("request backoffice membership", response, err)
	}

	return toDomainGroupMembership(membership)
}

func (c *Backoffice) DecideMembership(ctx context.Context, groupID, userID, status, actorID string) (domain.GroupMembership, error) {
	if c == nil || c.access == nil {
		return domain.GroupMembership{}, fmt.Errorf("backoffice group access client is not configured")
	}

	membership, response, err := c.access.DecideGroupMembership(ctx, groupID, userID).
		XEPOSUserID(actorID).
		MembershipDecisionRequest(client.MembershipDecisionRequest{Status: status}).
		Execute()
	if err != nil {
		return domain.GroupMembership{}, wrapGroupAccessError("decide backoffice membership", response, err)
	}

	return toDomainGroupMembership(membership)
}

func (c *Backoffice) RevokeMembership(ctx context.Context, groupID, userID, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("backoffice group access client is not configured")
	}

	response, err := c.access.RevokeGroupMembership(ctx, groupID, userID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("revoke backoffice membership", response, err)
	}

	return nil
}

func (c *Backoffice) RequestRole(ctx context.Context, groupID, role, actorID string) (domain.RoleRequest, error) {
	if c == nil || c.access == nil {
		return domain.RoleRequest{}, fmt.Errorf("backoffice group access client is not configured")
	}

	request, response, err := c.access.RequestGroupRole(ctx, groupID).
		XEPOSUserID(actorID).
		CreateRoleRequest(client.CreateRoleRequest{Role: role}).
		Execute()
	if err != nil {
		return domain.RoleRequest{}, wrapGroupAccessError("request backoffice role", response, err)
	}
	if request == nil {
		return domain.RoleRequest{}, fmt.Errorf("decode backoffice role request: response is empty")
	}

	return domain.RoleRequest{
		ID:          request.Id,
		Role:        request.Role,
		Status:      request.Status,
		RequestedAt: request.RequestedAt,
	}, nil
}

func (c *Backoffice) ApplyRoleRequestDecision(ctx context.Context, groupID, requestID, status, actorID string) (domain.GroupMembership, error) {
	if c == nil || c.access == nil {
		return domain.GroupMembership{}, fmt.Errorf("backoffice group access client is not configured")
	}

	membership, response, err := c.access.ApplyRoleRequestDecision(ctx, groupID, requestID).
		XEPOSUserID(actorID).
		RoleRequestDecisionRequest(client.RoleRequestDecisionRequest{Status: status}).
		Execute()
	if err != nil {
		return domain.GroupMembership{}, wrapGroupAccessError("decide backoffice role request", response, err)
	}

	return toDomainGroupMembership(membership)
}

func (c *Backoffice) WithdrawRoleRequest(ctx context.Context, groupID, requestID, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("backoffice group access client is not configured")
	}

	response, err := c.access.WithdrawGroupRoleRequest(ctx, groupID, requestID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("withdraw backoffice role request", response, err)
	}

	return nil
}

func (c *Backoffice) AssignGroupRole(ctx context.Context, groupID, userID, role, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("backoffice group access client is not configured")
	}

	response, err := c.access.AssignGroupRole(ctx, groupID, userID, role).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("assign backoffice group role", response, err)
	}

	return nil
}

func (c *Backoffice) RemoveGroupRole(ctx context.Context, groupID, userID, role, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("backoffice group access client is not configured")
	}

	response, err := c.access.RemoveGroupRole(ctx, groupID, userID, role).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("remove backoffice group role", response, err)
	}

	return nil
}

func (c *Backoffice) GrantGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("backoffice group access client is not configured")
	}

	response, err := c.access.GrantGroupPermission(ctx, groupID, userID, permission).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("grant backoffice group permission", response, err)
	}

	return nil
}

func (c *Backoffice) RevokeGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("backoffice group access client is not configured")
	}

	response, err := c.access.RevokeGroupPermission(ctx, groupID, userID, permission).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("revoke backoffice group permission", response, err)
	}

	return nil
}

// Resources exposes typed Backoffice resource lifecycle operations.
func (c *Backoffice) Resources() client.ResourcesAPI {
	if c == nil {
		return nil
	}
	return c.resources
}

func (c *Backoffice) GetUserByID(ctx context.Context, userID, actorID string) (domain.User, error) {
	if c == nil || c.users == nil {
		return domain.User{}, fmt.Errorf("backoffice identity client is not configured")
	}

	user, response, err := c.users.GetUser(ctx, userID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return domain.User{}, wrapUserError("get backoffice user", response, err)
	}

	return toDomainUser(user)
}

func (c *Backoffice) ListUsers(ctx context.Context, actorID string) ([]domain.UserSummary, error) {
	if c == nil || c.users == nil {
		return nil, fmt.Errorf("backoffice identity client is not configured")
	}

	users, response, err := c.users.ListUsers(ctx).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapUserError("list backoffice users", response, err)
	}

	result := make([]domain.UserSummary, 0, len(users))
	for _, user := range users {
		result = append(result, domain.UserSummary{
			ID:                user.Id,
			Email:             user.Email,
			GivenName:         user.GivenName,
			FamilyName:        user.FamilyName,
			Status:            user.Status,
			GlobalRoles:       append([]string{}, user.GlobalRoles...),
			GlobalPermissions: append([]string{}, user.GlobalPermissions...),
			CreatedAt:         "",
			UpdatedAt:         "",
		})
	}

	return result, nil
}

func (c *Backoffice) GrantGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	if c == nil || c.users == nil {
		return fmt.Errorf("backoffice identity client is not configured")
	}

	response, err := c.users.GrantGlobalPermission(ctx, userID, permission).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapUserError("grant backoffice global permission", response, err)
	}

	return nil
}

func (c *Backoffice) RevokeGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	if c == nil || c.users == nil {
		return fmt.Errorf("backoffice identity client is not configured")
	}

	response, err := c.users.RevokeGlobalPermission(ctx, userID, permission).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapUserError("revoke backoffice global permission", response, err)
	}

	return nil
}

func (c *Backoffice) EnsureUser(ctx context.Context, identity ActorIdentity) (domain.User, error) {
	user, err := c.GetUserByID(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
	if errors.Is(err, domain.ErrNotFound) {
		user, err = c.CreateUser(ctx, identity)
		if errors.Is(err, domain.ErrConflict) {
			user, err = c.GetUserByID(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
		}
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("ensure backoffice user: %w", err)
	}

	return user, nil
}

func (c *Backoffice) CreateUser(ctx context.Context, identity ActorIdentity) (domain.User, error) {
	if c == nil || c.identity == nil {
		return domain.User{}, fmt.Errorf("backoffice identity client is not configured")
	}

	user, response, err := c.identity.ProvisionUser(ctx).CreateUserRequest(client.CreateUserRequest{
		Id:         identity.AuthIdentifier,
		Email:      identity.Email,
		GivenName:  identity.GivenName,
		FamilyName: identity.FamilyName,
	}).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusConflict {
			return domain.User{}, domain.ErrConflict
		}

		return domain.User{}, wrapConnectionError("create backoffice user", err)
	}

	resolved, err := toDomainUser(user)
	if err != nil {
		return domain.User{}, err
	}

	return resolved, nil
}

func (c *Backoffice) ImportTurtle(ctx context.Context, actorID string, body []byte) (*domain.ImportResult, error) {
	if c == nil || c.imports == nil {
		return nil, fmt.Errorf("backoffice import client is not configured")
	}

	result, response, err := c.imports.ImportTurtleMetadata(ctx).XEPOSUserID(actorID).Body(string(body)).Execute()
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusBadRequest:
				return nil, fmt.Errorf("backoffice import rejected request: %w", domain.ErrBadRequest)
			case http.StatusForbidden:
				return nil, fmt.Errorf("backoffice import forbidden: %w", domain.ErrForbidden)
			case http.StatusConflict:
				return nil, fmt.Errorf("backoffice import conflict: %w", domain.ErrConflict)
			}
			return nil, fmt.Errorf("backoffice import failed with status %d: %w", response.StatusCode, err)
		}
		return nil, wrapConnectionError("backoffice import request", err)
	}

	if result == nil {
		return nil, fmt.Errorf("decode backoffice import result: response is empty")
	}

	responseResult := &domain.ImportResult{
		Groups:               result.Groups,
		Categories:           result.Categories,
		Organizations:        result.Organizations,
		ContactPoints:        result.ContactPoints,
		Datasets:             result.Datasets,
		SoftwareApplications: result.SoftwareApplications,
		SoftwareSourceCodes:  result.SoftwareSourceCodes,
		Warnings:             make([]domain.ImportDiagnostic, 0, len(result.Warnings)),
	}
	for _, warning := range result.Warnings {
		responseResult.Warnings = append(responseResult.Warnings, domain.ImportDiagnostic{
			Severity:  warning.Severity,
			Subject:   warning.Subject,
			Predicate: warning.Predicate,
			Message:   warning.Message,
		})
	}

	return responseResult, nil
}

func wrapUserError(operation string, response *http.Response, err error) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusBadRequest:
			return domain.ErrBadRequest
		case http.StatusNotFound:
			return domain.ErrNotFound
		case http.StatusForbidden:
			return domain.ErrForbidden
		}
	}
	return wrapConnectionError(operation, err)
}

func wrapGroupError(operation string, response *http.Response, err error) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusBadRequest:
			return fmt.Errorf("%s: %w", operation, domain.ErrBadRequest)
		case http.StatusNotFound:
			return fmt.Errorf("%s: %w", operation, domain.ErrNotFound)
		}
	}
	return wrapConnectionError(operation, err)
}

func wrapGroupAccessError(operation string, response *http.Response, err error) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusBadRequest:
			return fmt.Errorf("%s: %w", operation, domain.ErrBadRequest)
		case http.StatusForbidden:
			return fmt.Errorf("%s: %w", operation, domain.ErrForbidden)
		case http.StatusNotFound:
			return fmt.Errorf("%s: %w", operation, domain.ErrNotFound)
		case http.StatusConflict:
			return fmt.Errorf("%s: %w", operation, domain.ErrConflict)
		}
	}
	return wrapConnectionError(operation, err)
}

func wrapConnectionError(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func toDomainUser(value *client.User) (domain.User, error) {
	if value == nil {
		return domain.User{}, fmt.Errorf("decode backoffice user: response is empty")
	}

	memberships := make([]domain.Membership, 0, len(value.Memberships))
	for _, membership := range value.Memberships {
		memberships = append(memberships, domain.Membership{
			Group: domain.Group{
				ID:          membership.Group.Id,
				Name:        membership.Group.Name,
				SourceIRI:   "",
				Description: nil,
				Label:       nil,
				Logo:        nil,
				Page:        nil,
				Color:       nil,
				Position:    0,
				Concept:     "",
			},
			Roles:       append([]string{}, membership.Roles...),
			Status:      membership.Status,
			Permissions: append([]string{}, membership.Permissions...),
			CreatedAt:   membership.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:   membership.UpdatedAt.Format(time.RFC3339Nano),
		})
	}

	return domain.User{
		ID:                value.Id,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            value.Status,
		GlobalRoles:       append([]string{}, value.GlobalRoles...),
		GlobalPermissions: append([]string{}, value.GlobalPermissions...),
		CreatedAt:         value.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         value.UpdatedAt.Format(time.RFC3339Nano),
		Memberships:       memberships,
	}, nil
}

func toDomainGroup(value *client.Group) (domain.Group, error) {
	if value == nil {
		return domain.Group{}, fmt.Errorf("decode backoffice group: response is empty")
	}

	return domain.Group{
		ID:          value.Id,
		SourceIRI:   value.SourceIri,
		Name:        value.Name,
		Description: value.Description.Get(),
		Label:       value.Label.Get(),
		Logo:        value.Logo.Get(),
		Page:        value.Page.Get(),
		Color:       value.Color.Get(),
		Position:    value.Position,
		Concept:     value.Concept,
	}, nil
}

func toDomainGroupMembership(value *client.GroupMembership) (domain.GroupMembership, error) {
	if value == nil {
		return domain.GroupMembership{}, fmt.Errorf("decode backoffice membership: response is empty")
	}

	roleRequests := make([]domain.RoleRequest, 0, len(value.RoleRequests))
	for _, request := range value.RoleRequests {
		roleRequests = append(roleRequests, domain.RoleRequest{
			ID:          request.Id,
			Role:        request.Role,
			Status:      request.Status,
			RequestedAt: request.RequestedAt,
		})
	}

	return domain.GroupMembership{
		User: domain.MembershipUser{
			ID:         value.User.Id,
			GivenName:  value.User.GivenName,
			FamilyName: value.User.FamilyName,
			Email:      value.User.Email,
		},
		Status:       value.Status,
		Roles:        append([]string{}, value.Roles...),
		Permissions:  append([]string{}, value.Permissions...),
		RoleRequests: roleRequests,
		CreatedAt:    value.CreatedAt,
		UpdatedAt:    value.UpdatedAt,
	}, nil
}
