package grpcapi

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	accessservice "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/access"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AccessServer struct {
	metadatalifecyclev1.UnimplementedGroupAccessServiceServer
	access *accessservice.Service
}

func NewAccess(access *accessservice.Service) *AccessServer {
	return &AccessServer{access: access}
}

func (s *AccessServer) ListGroupMemberships(ctx context.Context, request *metadatalifecyclev1.ListGroupMembershipsRequest) (*metadatalifecyclev1.ListGroupMembershipsResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	values, err := s.access.ListMemberships(ctx, actor, groupID)
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	result := make([]*metadatalifecyclev1.GroupMembership, len(values))
	for i, value := range values {
		result[i] = toProtoGroupMembership(value)
	}
	return &metadatalifecyclev1.ListGroupMembershipsResponse{Memberships: result}, nil
}

func (s *AccessServer) RequestMembership(ctx context.Context, request *metadatalifecyclev1.RequestMembershipRequest) (*metadatalifecyclev1.RequestMembershipResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	roles := make([]domain.GroupRole, len(request.Roles))
	seen := make(map[domain.GroupRole]struct{}, len(request.Roles))
	for i, role := range request.Roles {
		parsedRole, err := toDomainGroupRole(role)
		if err == nil {
			parsedRole, err = domain.NewRequestedGroupRole(string(parsedRole))
		}
		if err != nil {
			return nil, wrapHandlerError(err)
		}
		if _, ok := seen[parsedRole]; ok {
			return nil, domain.ErrInvalidAccess
		}
		seen[parsedRole] = struct{}{}
		roles[i] = parsedRole
	}
	value, err := s.access.RequestMembership(ctx, actor, groupID, roles)
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	return &metadatalifecyclev1.RequestMembershipResponse{Membership: toProtoGroupMembership(value)}, nil
}

func (s *AccessServer) WithdrawMembership(ctx context.Context, request *metadatalifecyclev1.WithdrawMembershipRequest) (*metadatalifecyclev1.WithdrawMembershipResponse, error) {
	return &metadatalifecyclev1.WithdrawMembershipResponse{}, s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		return s.access.WithdrawMembershipRequest(ctx, actor, groupID)
	})
}

func (s *AccessServer) DecideMembership(ctx context.Context, request *metadatalifecyclev1.DecideMembershipRequest) (*metadatalifecyclev1.DecideMembershipResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	decision, err := toDomainDecision(request.Decision)
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	value, err := s.access.DecideMembership(ctx, actor, groupID, request.UserId, decision)
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	return &metadatalifecyclev1.DecideMembershipResponse{Membership: toProtoGroupMembership(value)}, nil
}

func (s *AccessServer) RevokeMembership(ctx context.Context, request *metadatalifecyclev1.RevokeMembershipRequest) (*metadatalifecyclev1.RevokeMembershipResponse, error) {
	return &metadatalifecyclev1.RevokeMembershipResponse{}, s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		return s.access.RevokeMembership(ctx, actor, groupID, request.UserId)
	})
}

func (s *AccessServer) RequestRole(ctx context.Context, request *metadatalifecyclev1.RequestRoleRequest) (*metadatalifecyclev1.RequestRoleResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	role, err := toDomainGroupRole(request.Role)
	if err == nil {
		role, err = domain.NewRequestedGroupRole(string(role))
	}
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	value, err := s.access.RequestRole(ctx, actor, groupID, role)
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	roleRequest, err := toProtoRoleRequest(value)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.RequestRoleResponse{Request: roleRequest}, nil
}

func (s *AccessServer) DecideRoleRequest(ctx context.Context, request *metadatalifecyclev1.DecideRoleRequestRequest) (*metadatalifecyclev1.DecideRoleRequestResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseAccessID(request.GroupId, "group id")
	if err != nil {
		return nil, err
	}
	requestID, err := parseAccessID(request.RequestId, "request id")
	if err != nil {
		return nil, err
	}
	decision, err := toDomainDecision(request.Decision)
	if err != nil {
		return nil, err
	}
	value, err := s.access.DecideRoleRequest(ctx, actor, groupID, requestID, decision)
	if err != nil {
		return nil, wrapHandlerError(err)
	}
	return &metadatalifecyclev1.DecideRoleRequestResponse{Membership: toProtoGroupMembership(value)}, nil
}

func (s *AccessServer) WithdrawRoleRequest(ctx context.Context, request *metadatalifecyclev1.WithdrawRoleRequestRequest) (*metadatalifecyclev1.WithdrawRoleRequestResponse, error) {
	return &metadatalifecyclev1.WithdrawRoleRequestResponse{}, s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		requestID, err := parseAccessID(request.RequestId, "request id")
		if err != nil {
			return err
		}
		return s.access.WithdrawRoleRequest(ctx, actor, groupID, requestID)
	})
}

func (s *AccessServer) AssignRole(ctx context.Context, request *metadatalifecyclev1.AssignRoleRequest) (*metadatalifecyclev1.AssignRoleResponse, error) {
	return &metadatalifecyclev1.AssignRoleResponse{}, s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		role, err := toDomainGroupRole(request.Role)
		if err != nil {
			return err
		}
		return s.access.AssignRole(ctx, actor, groupID, request.UserId, role)
	})
}

func (s *AccessServer) RemoveRole(ctx context.Context, request *metadatalifecyclev1.RemoveRoleRequest) (*metadatalifecyclev1.RemoveRoleResponse, error) {
	return &metadatalifecyclev1.RemoveRoleResponse{}, s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		role, err := toDomainGroupRole(request.Role)
		if err != nil {
			return err
		}
		return s.access.RemoveRole(ctx, actor, groupID, request.UserId, role)
	})
}

func (s *AccessServer) GrantPermission(ctx context.Context, request *metadatalifecyclev1.GrantPermissionRequest) (*metadatalifecyclev1.GrantPermissionResponse, error) {
	return &metadatalifecyclev1.GrantPermissionResponse{}, s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		return s.access.GrantPermission(ctx, actor, groupID, request.UserId, request.Permission)
	})
}

func (s *AccessServer) RevokePermission(ctx context.Context, request *metadatalifecyclev1.RevokePermissionRequest) (*metadatalifecyclev1.RevokePermissionResponse, error) {
	return &metadatalifecyclev1.RevokePermissionResponse{}, s.runActor(ctx, func(actor domain.Actor) error {
		groupID, err := parseAccessID(request.GroupId, "group id")
		if err != nil {
			return err
		}
		return s.access.RevokePermission(ctx, actor, groupID, request.UserId, request.Permission)
	})
}

// runActor TODO: delete this useless function, just inline it
func (s *AccessServer) runActor(ctx context.Context, f func(domain.Actor) error) error {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return err
	}
	return f(actor)
}

func parseAccessID(value, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid %s", name)
	}
	return id, nil
}

func toProtoGroupMembership(value domain.GroupMembership) *metadatalifecyclev1.GroupMembership {
	roles := make([]metadatalifecyclev1.GroupRole, len(value.Roles))
	for i, role := range value.Roles {
		roles[i], _ = toProtoGroupRole(role)
	}
	requests := make([]*metadatalifecyclev1.RoleRequest, len(value.RoleRequests))
	for i, request := range value.RoleRequests {
		mapped, _ := toProtoRoleRequest(request)
		requests[i] = mapped
	}
	status, _ := toProtoMembershipStatus(value.Status)
	return &metadatalifecyclev1.GroupMembership{User: &metadatalifecyclev1.MembershipUser{Id: value.User.ID, GivenName: value.User.GivenName, FamilyName: value.User.FamilyName, Email: value.User.Email}, Status: status, Roles: roles, Permissions: append([]string(nil), value.Permissions...), RoleRequests: requests, CreatedAt: timestamppb.New(value.CreatedAt), UpdatedAt: timestamppb.New(value.UpdatedAt)}
}

func toDomainGroupRole(value metadatalifecyclev1.GroupRole) (domain.GroupRole, error) {
	switch value {
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR:
		return domain.GroupRoleEditor, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER:
		return domain.GroupRoleReviewer, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_ADMIN:
		return domain.GroupRoleAdmin, nil
	default:
		return "", domain.ErrInvalidAccess
	}
}

func toDomainDecision(value metadatalifecyclev1.Decision) (domain.Decision, error) {
	switch value {
	case metadatalifecyclev1.Decision_DECISION_ACCEPT:
		return domain.DecisionAccept, nil
	case metadatalifecyclev1.Decision_DECISION_REJECT:
		return domain.DecisionReject, nil
	default:
		return "", domain.ErrInvalidAccess
	}
}
