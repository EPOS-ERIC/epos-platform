package handlers

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ListPermissions(ctx context.Context) ([]gatewayapi.Permission, error) {
	response, err := h.metadataLifecycle.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("list permissions: response is empty")
	}

	permissions := make([]gatewayapi.Permission, len(response.GetPermissions()))
	for i, permission := range response.GetPermissions() {
		permissions[i], err = fromProtoPermission(permission)
		if err != nil {
			return nil, err
		}
	}

	return permissions, nil
}

func fromProtoPermission(value *metadatalifecyclev1.Permission) (gatewayapi.Permission, error) {
	if value == nil {
		return gatewayapi.Permission{}, fmt.Errorf("list permissions: response contains nil permission")
	}

	scope, err := fromProtoPermissionScope(value.GetScope())
	if err != nil {
		return gatewayapi.Permission{}, err
	}

	return gatewayapi.Permission{
		Key:         value.GetKey(),
		Label:       value.GetLabel(),
		Description: value.GetDescription(),
		Scope:       scope,
	}, nil
}

func fromProtoPermissionScope(value metadatalifecyclev1.PermissionScope) (gatewayapi.PermissionScope, error) {
	switch value {
	case metadatalifecyclev1.PermissionScope_PERMISSION_SCOPE_GLOBAL:
		return gatewayapi.PermissionScopeGLOBAL, nil
	case metadatalifecyclev1.PermissionScope_PERMISSION_SCOPE_GROUP:
		return gatewayapi.PermissionScopeGROUP, nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown permission scope: %v", value)
	}
}
