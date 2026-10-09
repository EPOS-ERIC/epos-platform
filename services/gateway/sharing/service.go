// Package sharing contains Gateway configuration-sharing use-cases.
package sharing

import (
	"context"
	"slices"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	client *clients.Sharing
}

func NewService(client *clients.Sharing) *Service {
	return &Service{client: client}
}

func (s *Service) CreateConfiguration(ctx context.Context, user domain.User, authenticated bool, request *configurationv1.CreateConfigurationRequest) (*configurationv1.CreateConfigurationResponse, error) {
	actor, manage := caller(user, authenticated)
	if actor == "" {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	return s.client.CreateConfiguration(ctx, actor, manage, request)
}

func (s *Service) GetConfiguration(ctx context.Context, user domain.User, authenticated bool, request *configurationv1.GetConfigurationRequest) (*configurationv1.GetConfigurationResponse, error) {
	actor, manage := caller(user, authenticated)

	return s.client.GetConfiguration(ctx, actor, manage, request)
}

func (s *Service) ListConfigurations(ctx context.Context, user domain.User, authenticated bool, request *configurationv1.ListConfigurationsRequest) (*configurationv1.ListConfigurationsResponse, error) {
	actor, manage := caller(user, authenticated)

	switch request.GetScope() {
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN:
		if actor == "" {
			return nil, apperror.Default(apperror.CodeUnauthenticated)
		}
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER,
		configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_ALL:
		if actor == "" {
			return nil, apperror.Default(apperror.CodeUnauthenticated)
		}
		if !manage {
			return nil, apperror.Default(apperror.CodePermissionDenied)
		}
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OFFICIAL:
	default:
		return nil, apperror.Default(apperror.CodeValidationFailed)
	}
	if request.GetScope() == configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER {
		if request.GetOwnerId() == "" {
			return nil, apperror.Field("owner_id", "Owner ID is required for this list scope.", nil)
		}
	} else if request.GetOwnerId() != "" {
		return nil, apperror.Field("owner_id", "Owner ID is only valid with the owner list scope.", nil)
	}

	return s.client.ListConfigurations(ctx, actor, manage, request)
}

func (s *Service) UpdateConfiguration(ctx context.Context, user domain.User, authenticated bool, request *configurationv1.UpdateConfigurationRequest) (*configurationv1.UpdateConfigurationResponse, error) {
	actor, manage := caller(user, authenticated)
	if actor == "" {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	return s.client.UpdateConfiguration(ctx, actor, manage, request)
}

func (s *Service) DeleteConfiguration(ctx context.Context, user domain.User, authenticated bool, request *configurationv1.DeleteConfigurationRequest) (*configurationv1.DeleteConfigurationResponse, error) {
	actor, manage := caller(user, authenticated)
	if actor == "" {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	return s.client.DeleteConfiguration(ctx, actor, manage, request)
}

func (s *Service) SetConfigurationOfficialStatus(ctx context.Context, user domain.User, authenticated bool, request *configurationv1.SetConfigurationOfficialStatusRequest) (*configurationv1.SetConfigurationOfficialStatusResponse, error) {
	actor, manage := caller(user, authenticated)
	if actor == "" {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	if !manage {
		return nil, apperror.Default(apperror.CodePermissionDenied)
	}
	return s.client.SetConfigurationOfficialStatus(ctx, actor, manage, request)
}

func (s *Service) CanManageConfigurations(user domain.User, authenticated bool) bool {
	_, manage := caller(user, authenticated)
	return manage
}

func caller(user domain.User, authenticated bool) (string, bool) {
	if !authenticated || user.ID == "" {
		return "", false
	}

	hasPermissions := slices.Contains(user.GlobalGrantedPermissions, domain.GlobalPermissionConfigurationManage) ||
		slices.Contains(user.GlobalRolePermissions, domain.GlobalPermissionConfigurationManage)
	return user.ID, hasPermissions
}
