package grpcapi

import (
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toDomainConfigurationInput(configuration []byte, title string, description *string, services []string, isPublic *bool) (domain.ConfigurationInput, error) {
	return domain.NewConfigurationInput(domain.ConfigurationInput{
		Configuration:  configuration,
		Title:          title,
		Description:    description,
		ListOfServices: services,
		IsPublic:       isPublic,
	})
}

func toProtoConfiguration(record domain.Configuration) *configurationv1.Configuration {
	return &configurationv1.Configuration{
		Id:             record.ID.String(),
		Configuration:  []byte(record.Configuration),
		Title:          record.Title,
		Description:    record.Description,
		ListOfServices: append([]string{}, record.ListOfServices...),
		IsPublic:       record.IsPublic,
		OwnerId:        string(record.OwnerID),
		CreatedAt:      timestamppb.New(record.CreatedAt),
		UpdatedAt:      timestamppb.New(record.UpdatedAt),
		IsOfficial:     record.IsOfficial,
	}
}

func toProtoConfigurationSummary(record domain.ConfigurationSummary, includeOwner bool) *configurationv1.ConfigurationSummary {
	result := &configurationv1.ConfigurationSummary{
		Id:             record.ID.String(),
		Title:          record.Title,
		Description:    record.Description,
		ListOfServices: append([]string{}, record.ListOfServices...),
		IsOfficial:     record.IsOfficial,
		IsPublic:       record.IsPublic,
		CreatedAt:      timestamppb.New(record.CreatedAt),
		UpdatedAt:      timestamppb.New(record.UpdatedAt),
	}
	if includeOwner {
		result.OwnerId = new(string(record.OwnerID))
	}
	return result
}

func toProtoConfigurationSummaries(records []domain.ConfigurationSummary, includeOwner bool) []*configurationv1.ConfigurationSummary {
	result := make([]*configurationv1.ConfigurationSummary, 0, len(records))
	for _, record := range records {
		result = append(result, toProtoConfigurationSummary(record, includeOwner))
	}
	return result
}

func toDomainListScope(value configurationv1.ConfigurationListScope) (domain.ListScope, error) {
	switch value {
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN:
		return domain.ListScopeOwn, nil
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER:
		return domain.ListScopeOwner, nil
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_ALL:
		return domain.ListScopeAll, nil
	case configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OFFICIAL:
		return domain.ListScopeOfficial, nil
	default:
		return domain.NewListScope("")
	}
}
