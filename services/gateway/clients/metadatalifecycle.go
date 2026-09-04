// Package clients provides outbound service clients for the gateway.
package clients

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

type MetadataLifecycle struct {
	identity      client.InternalIdentityAPI
	users         client.UsersAPI
	groups        client.GroupsAPI
	access        client.GroupAccessAPI
	resources     client.ResourcesAPI
	imports       client.ImportsAPI
	contactPoints client.ContactPointsAPI
	organizations client.OrganizationsAPI
	categories    client.CategoriesAPI

	Proxy    *MetadataLifecycleProxy
	Identity *MetadataLifecycleIdentity

	log *slog.Logger
}

type ActorIdentity struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

func NewMetadataLifecycle(log *slog.Logger, baseURL string, upstreamCfg config.Upstream) (*MetadataLifecycle, error) {
	normalizedBaseURL, err := urlutil.NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize metadata-lifecycle base URL: %w", err)
	}

	upstreamClient, err := upstream.NewClient(log.With("component", "upstream"), upstream.Config{
		Service:     "metadata-lifecycle",
		Timeout:     upstreamCfg.Timeout,
		RealTimeout: upstreamCfg.RealTimeout,
		CacheTTL:    0,
		MaxEntries:  upstreamCfg.MaxEntries,
		Transport:   nil,
	})
	if err != nil {
		return nil, fmt.Errorf("create metadata-lifecycle client: %w", err)
	}

	clientConfig := client.NewConfiguration()
	clientConfig.Servers[0].URL = normalizedBaseURL
	clientConfig.HTTPClient = upstreamClient.StandardClient()

	apiClient := client.NewAPIClient(clientConfig)

	// TODO: we are doing this just to hook up the provisioning of users for now, later we will only have the grpc service
	proxy, identity, err := NewMetadataLifecycleGRPC(log, "metadata-lifecycle-service:5001")
	if err != nil {
		return nil, fmt.Errorf("TODO: %w", err)
	}

	return &MetadataLifecycle{
		identity:      apiClient.InternalIdentityAPI,
		users:         apiClient.UsersAPI,
		groups:        apiClient.GroupsAPI,
		access:        apiClient.GroupAccessAPI,
		resources:     apiClient.ResourcesAPI,
		imports:       apiClient.ImportsAPI,
		contactPoints: apiClient.ContactPointsAPI,
		organizations: apiClient.OrganizationsAPI,
		categories:    apiClient.CategoriesAPI,

		// TODO: we are doing this just to hook up the provisioning of users for now, later we will only have the grpc service
		Proxy:    proxy,
		Identity: identity,

		log: log,
	}, nil
}

func (c *MetadataLifecycle) ListGroups(ctx context.Context) ([]client.Group, error) {
	if c == nil || c.groups == nil {
		return nil, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	groups, response, err := c.groups.ListGroups(ctx).Execute()
	if err != nil {
		return nil, wrapGroupError("list metadata-lifecycle groups", response, err)
	}

	return groups, nil
}

func (c *MetadataLifecycle) GetGroup(ctx context.Context, id string) (*client.Group, error) {
	if c == nil || c.groups == nil {
		return nil, fmt.Errorf("metadata-lifecycle group client is not configured")
	}

	group, response, err := c.groups.GetGroup(ctx, id).Execute()
	if err != nil {
		return nil, wrapGroupError("get metadata-lifecycle group", response, err)
	}

	if group == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle group: response is empty")
	}

	return group, nil
}

func (c *MetadataLifecycle) WithdrawMembershipRequest(ctx context.Context, groupID, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("metadata-lifecycle group access client is not configured")
	}

	response, err := c.access.WithdrawGroupMembershipRequest(ctx, groupID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("withdraw metadata-lifecycle membership request", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) RequestMembership(ctx context.Context, groupID, actorID string, roles []string) (*client.GroupMembership, error) {
	if c == nil || c.access == nil {
		return nil, fmt.Errorf("metadata-lifecycle group access client is not configured")
	}
	membership, response, err := c.access.RequestGroupMembership(ctx, groupID).
		XEPOSUserID(actorID).
		CreateMembershipRequest(client.CreateMembershipRequest{Roles: roles}).
		Execute()
	if err != nil {
		return nil, wrapGroupAccessError("request metadata-lifecycle membership", response, err)
	}
	return membership, nil
}

func (c *MetadataLifecycle) DecideMembership(ctx context.Context, groupID, userID, status, actorID string) (*client.GroupMembership, error) {
	if c == nil || c.access == nil {
		return nil, fmt.Errorf("metadata-lifecycle group access client is not configured")
	}
	membership, response, err := c.access.DecideGroupMembership(ctx, groupID, userID).
		XEPOSUserID(actorID).
		MembershipDecisionRequest(client.MembershipDecisionRequest{Status: status}).
		Execute()
	if err != nil {
		return nil, wrapGroupAccessError("decide metadata-lifecycle membership", response, err)
	}
	return membership, nil
}

func (c *MetadataLifecycle) RevokeMembership(ctx context.Context, groupID, userID, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("metadata-lifecycle group access client is not configured")
	}

	response, err := c.access.RevokeGroupMembership(ctx, groupID, userID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("revoke metadata-lifecycle membership", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) RequestRole(ctx context.Context, groupID, role, actorID string) (*client.RoleRequest, error) {
	if c == nil || c.access == nil {
		return nil, fmt.Errorf("metadata-lifecycle group access client is not configured")
	}
	request, response, err := c.access.RequestGroupRole(ctx, groupID).
		XEPOSUserID(actorID).
		CreateRoleRequest(client.CreateRoleRequest{Role: role}).
		Execute()
	if err != nil {
		return nil, wrapGroupAccessError("request metadata-lifecycle role", response, err)
	}
	return request, nil
}

func (c *MetadataLifecycle) ApplyRoleRequestDecision(ctx context.Context, groupID, requestID, status, actorID string) (*client.GroupMembership, error) {
	if c == nil || c.access == nil {
		return nil, fmt.Errorf("metadata-lifecycle group access client is not configured")
	}
	membership, response, err := c.access.ApplyRoleRequestDecision(ctx, groupID, requestID).
		XEPOSUserID(actorID).
		RoleRequestDecisionRequest(client.RoleRequestDecisionRequest{Status: status}).
		Execute()
	if err != nil {
		return nil, wrapGroupAccessError("decide metadata-lifecycle role request", response, err)
	}
	return membership, nil
}

func (c *MetadataLifecycle) WithdrawRoleRequest(ctx context.Context, groupID, requestID, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("metadata-lifecycle group access client is not configured")
	}

	response, err := c.access.WithdrawGroupRoleRequest(ctx, groupID, requestID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("withdraw metadata-lifecycle role request", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) AssignGroupRole(ctx context.Context, groupID, userID, role, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("metadata-lifecycle group access client is not configured")
	}

	response, err := c.access.AssignGroupRole(ctx, groupID, userID, role).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("assign metadata-lifecycle group role", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) RemoveGroupRole(ctx context.Context, groupID, userID, role, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("metadata-lifecycle group access client is not configured")
	}

	response, err := c.access.RemoveGroupRole(ctx, groupID, userID, role).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("remove metadata-lifecycle group role", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) GrantGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("metadata-lifecycle group access client is not configured")
	}

	response, err := c.access.GrantGroupPermission(ctx, groupID, userID, permission).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("grant metadata-lifecycle group permission", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) RevokeGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	if c == nil || c.access == nil {
		return fmt.Errorf("metadata-lifecycle group access client is not configured")
	}

	response, err := c.access.RevokeGroupPermission(ctx, groupID, userID, permission).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapGroupAccessError("revoke metadata-lifecycle group permission", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) ListUsers(ctx context.Context, actorID string) ([]client.UserSummary, error) {
	if c == nil || c.users == nil {
		return nil, fmt.Errorf("metadata-lifecycle users client is not configured")
	}

	users, response, err := c.users.ListUsers(ctx).XEPOSUserID(actorID).Execute()
	if err != nil {
		return nil, wrapUserError("list metadata-lifecycle users", response, err)
	}

	return users, nil
}

func (c *MetadataLifecycle) GrantGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	if c == nil || c.users == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	response, err := c.users.GrantGlobalPermission(ctx, userID, permission).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapUserError("grant metadata-lifecycle global permission", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) RevokeGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	if c == nil || c.users == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	response, err := c.users.RevokeGlobalPermission(ctx, userID, permission).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapUserError("revoke metadata-lifecycle global permission", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) AssignGlobalRole(ctx context.Context, actorID, userID, role string) error {
	if c == nil || c.users == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	response, err := c.users.AssignGlobalRole(ctx, userID, role).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapUserError("assign metadata-lifecycle global role", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) RemoveGlobalRole(ctx context.Context, actorID, userID, role string) error {
	if c == nil || c.users == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	response, err := c.users.RemoveGlobalRole(ctx, userID, role).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapUserError("remove metadata-lifecycle global role", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) BlockUser(ctx context.Context, actorID, userID, reason string) error {
	if c == nil || c.users == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	response, err := c.users.BlockUser(ctx, userID).
		XEPOSUserID(actorID).
		BlockUserRequest(client.BlockUserRequest{Reason: reason}).
		Execute()
	if err != nil {
		return wrapUserError("block metadata-lifecycle user", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) UnblockUser(ctx context.Context, actorID, userID string) error {
	if c == nil || c.users == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	response, err := c.users.UnblockUser(ctx, userID).XEPOSUserID(actorID).Execute()
	if err != nil {
		return wrapUserError("unblock metadata-lifecycle user", response, err)
	}

	return nil
}

func (c *MetadataLifecycle) EnsureUser(ctx context.Context, identity ActorIdentity) (domain.User, error) {
	user, err := c.Identity.GetUserForAuth(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
	if errors.Is(err, domain.ErrNotFound) {
		err = c.Identity.ProvisionUser(ctx, identity)
		if err == nil || errors.Is(err, domain.ErrConflict) {
			user, err = c.Identity.GetUserForAuth(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
		}
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("ensure metadata-lifecycle user: %w", err)
	}

	return user, nil
}

func (c *MetadataLifecycle) ImportTurtle(ctx context.Context, actorID string, body []byte) (*client.ImportResult, error) {
	if c == nil || c.imports == nil {
		return nil, fmt.Errorf("metadata-lifecycle import client is not configured")
	}

	result, response, err := c.imports.ImportTurtleMetadata(ctx).XEPOSUserID(actorID).Body(string(body)).Execute()
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusBadRequest:
				return nil, fmt.Errorf("metadata-lifecycle import rejected request: %w", domain.ErrBadRequest)
			case http.StatusForbidden:
				return nil, fmt.Errorf("metadata-lifecycle import forbidden: %w", domain.ErrForbidden)
			case http.StatusConflict:
				return nil, fmt.Errorf("metadata-lifecycle import conflict: %w", domain.ErrConflict)
			}
			return nil, fmt.Errorf("metadata-lifecycle import failed with status %d: %w", response.StatusCode, err)
		}
		return nil, wrapConnectionError("metadata-lifecycle import request", err)
	}

	if result == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle import result: response is empty")
	}
	return result, nil
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
		case http.StatusConflict:
			return domain.ErrConflict
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
