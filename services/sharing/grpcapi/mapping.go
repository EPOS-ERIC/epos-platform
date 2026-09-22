package grpcapi

import (
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toDomainConfigurationInput(configuration []byte, title, description *string, services []string, isOfficial, isPublic bool) (domain.ConfigurationInput, error) {
	return domain.NewConfigurationInput(domain.ConfigurationInput{
		Configuration:  configuration,
		Title:          title,
		Description:    description,
		ListOfServices: services,
		IsOfficial:     isOfficial,
		IsPublic:       isPublic,
	})
}

func toProtoConfiguration(record repository.ConfigurationRecord) *configurationv1.Configuration {
	result := &configurationv1.Configuration{
		Id:            record.ID.String(),
		Configuration: []byte(record.Configuration),
		IsOfficial:    record.IsOfficial,
		IsPublic:      record.IsPublic,
		OwnerId:       record.OwnerID,
		CreatedAt:     timestamppb.New(record.CreatedAt),
		UpdatedAt:     timestamppb.New(record.UpdatedAt),
	}
	if record.Title != nil {
		result.Title = record.Title
	}
	if record.Description != nil {
		result.Description = record.Description
	}
	result.ListOfServices = append([]string(nil), record.ListOfServices...)
	return result
}

func toProtoConfigurations(records []repository.ConfigurationRecord) []*configurationv1.Configuration {
	result := make([]*configurationv1.Configuration, 0, len(records))
	for _, record := range records {
		result = append(result, toProtoConfiguration(record))
	}
	return result
}
