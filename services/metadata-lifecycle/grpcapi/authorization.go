package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

type AuthorizationServer struct {
	metadatalifecyclev1.UnimplementedAuthorizationServiceServer
	authorizer *authz.Authorizer
}

func NewAuthorization(authorizer *authz.Authorizer) *AuthorizationServer {
	return &AuthorizationServer{authorizer: authorizer}
}

func (s *AuthorizationServer) ListPermissions(ctx context.Context, _ *metadatalifecyclev1.ListPermissionsRequest) (*metadatalifecyclev1.ListPermissionsResponse, error) {
	permissions, err := s.authorizer.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	result, err := mapSliceErr(permissions, toProtoPermission)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.ListPermissionsResponse{Permissions: result}, nil
}

func toProtoPermission(value domain.Permission) (*metadatalifecyclev1.Permission, error) {
	scope, err := toProtoPermissionScope(value.Scope)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.Permission{
		Key:         value.Key,
		Label:       value.Label,
		Description: value.Description,
		Scope:       scope,
	}, nil
}

func toProtoPermissionScope(value domain.PermissionScope) (metadatalifecyclev1.PermissionScope, error) {
	switch value {
	case domain.PermissionScopeGlobal:
		return metadatalifecyclev1.PermissionScope_PERMISSION_SCOPE_GLOBAL, nil
	case domain.PermissionScopeGroup:
		return metadatalifecyclev1.PermissionScope_PERMISSION_SCOPE_GROUP, nil
	default:
		return 0, apperror.Default(apperror.CodeValidationFailed)
	}
}
