// Package clients provides outbound service clients for the gateway.
package clients

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/urlutil"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

type MetadataLifecycle struct {
	identity client.InternalIdentityAPI
	users    client.UsersAPI
	imports  client.ImportsAPI

	GRPC     *MetadataLifecycleGRPC
	Identity *MetadataLifecycleIdentity

	log *slog.Logger
}

type ActorIdentity struct {
	AuthIdentifier string
	Email          string
	GivenName      string
	FamilyName     string
}

func NewMetadataLifecycle(log *slog.Logger, baseURL string, upstreamCfg config.Upstream, grpcBaseURL string) (*MetadataLifecycle, error) {
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
	grpc, identity, err := NewMetadataLifecycleGRPC(log, grpcBaseURL)
	if err != nil {
		return nil, fmt.Errorf("TODO: %w", err)
	}

	return &MetadataLifecycle{
		identity: apiClient.InternalIdentityAPI,
		users:    apiClient.UsersAPI,
		imports:  apiClient.ImportsAPI,

		// TODO: we are doing this just to hook up the provisioning of users for now, later we will only have the grpc service
		GRPC:     grpc,
		Identity: identity,

		log: log,
	}, nil
}

func (c *MetadataLifecycle) ListGroups(ctx context.Context) (*metadatalifecyclev1.ListGroupsResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	response, err := c.GRPC.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *MetadataLifecycle) GetGroup(ctx context.Context, request *metadatalifecyclev1.GetGroupRequest) (*metadatalifecyclev1.GetGroupResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	response, err := c.GRPC.GetGroup(ctx, request)
	if err != nil {
		return nil, err
	}

	if response == nil || response.Group == nil {
		return nil, fmt.Errorf("decode metadata-lifecycle group: response is empty")
	}

	return response, nil
}

func (c *MetadataLifecycle) WithdrawMembershipRequest(ctx context.Context, groupID, actorID string) error {
	if c == nil || c.GRPC == nil {
		return fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	_, err := c.GRPC.WithdrawMembership(ctx, actorID, &metadatalifecyclev1.WithdrawMembershipRequest{GroupId: groupID})
	return err
}

func (c *MetadataLifecycle) RequestMembership(ctx context.Context, groupID, actorID string, roles []string) (*metadatalifecyclev1.RequestMembershipResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}
	return c.GRPC.RequestMembership(ctx, actorID, &metadatalifecyclev1.RequestMembershipRequest{GroupId: groupID, Roles: roles})
}

func (c *MetadataLifecycle) DecideMembership(ctx context.Context, groupID, userID, status, actorID string) (*metadatalifecyclev1.DecideMembershipResponse, error) {
	return c.GRPC.DecideMembership(ctx, actorID, &metadatalifecyclev1.DecideMembershipRequest{GroupId: groupID, UserId: userID, Status: status})
}

func (c *MetadataLifecycle) RevokeMembership(ctx context.Context, groupID, userID, actorID string) error {
	_, err := c.GRPC.RevokeMembership(ctx, actorID, &metadatalifecyclev1.RevokeMembershipRequest{GroupId: groupID, UserId: userID})
	return err
}

func (c *MetadataLifecycle) RequestRole(ctx context.Context, groupID, role, actorID string) (*metadatalifecyclev1.RequestRoleResponse, error) {
	return c.GRPC.RequestRole(ctx, actorID, &metadatalifecyclev1.RequestRoleRequest{GroupId: groupID, Role: role})
}

func (c *MetadataLifecycle) ApplyRoleRequestDecision(ctx context.Context, groupID, requestID, status, actorID string) (*metadatalifecyclev1.ApplyRoleRequestDecisionResponse, error) {
	return c.GRPC.ApplyRoleRequestDecision(ctx, actorID, &metadatalifecyclev1.ApplyRoleRequestDecisionRequest{GroupId: groupID, RequestId: requestID, Status: status})
}

func (c *MetadataLifecycle) WithdrawRoleRequest(ctx context.Context, groupID, requestID, actorID string) error {
	_, err := c.GRPC.WithdrawRoleRequest(ctx, actorID, &metadatalifecyclev1.WithdrawRoleRequestRequest{GroupId: groupID, RequestId: requestID})
	return err
}

func (c *MetadataLifecycle) AssignGroupRole(ctx context.Context, groupID, userID, role, actorID string) error {
	_, err := c.GRPC.AssignRole(ctx, actorID, &metadatalifecyclev1.AssignRoleRequest{GroupId: groupID, UserId: userID, Role: role})
	return err
}

func (c *MetadataLifecycle) RemoveGroupRole(ctx context.Context, groupID, userID, role, actorID string) error {
	_, err := c.GRPC.RemoveRole(ctx, actorID, &metadatalifecyclev1.RemoveRoleRequest{GroupId: groupID, UserId: userID, Role: role})
	return err
}

func (c *MetadataLifecycle) GrantGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	_, err := c.GRPC.GrantPermission(ctx, actorID, &metadatalifecyclev1.GrantPermissionRequest{GroupId: groupID, UserId: userID, Permission: permission})
	return err
}

func (c *MetadataLifecycle) RevokeGroupPermission(ctx context.Context, groupID, userID, permission, actorID string) error {
	_, err := c.GRPC.RevokePermission(ctx, actorID, &metadatalifecyclev1.RevokePermissionRequest{GroupId: groupID, UserId: userID, Permission: permission})
	return err
}

func (c *MetadataLifecycle) RevokeGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	if c == nil || c.GRPC == nil {
		return fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	return c.GRPC.RevokeGlobalPermission(ctx, actorID, userID, permission)
}

func (c *MetadataLifecycle) AssignGlobalRole(ctx context.Context, actorID, userID, role string) error {
	if c == nil || c.GRPC == nil {
		return fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	return c.GRPC.AssignGlobalRole(ctx, actorID, userID, role)
}

func (c *MetadataLifecycle) RemoveGlobalRole(ctx context.Context, actorID, userID, role string) error {
	if c == nil || c.GRPC == nil {
		return fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	return c.GRPC.RemoveGlobalRole(ctx, actorID, userID, role)
}

func (c *MetadataLifecycle) BlockUser(ctx context.Context, actorID, userID, reason string) error {
	if c == nil || c.GRPC == nil {
		return fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	return c.GRPC.BlockUser(ctx, actorID, userID, reason)
}

func (c *MetadataLifecycle) UnblockUser(ctx context.Context, actorID, userID string) error {
	if c == nil || c.GRPC == nil {
		return fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	return c.GRPC.UnblockUser(ctx, actorID, userID)
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

func wrapConnectionError(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
