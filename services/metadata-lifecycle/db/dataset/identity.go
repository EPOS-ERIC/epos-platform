package dataset

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type distributionIdentity struct {
	id                uuid.UUID
	resourceID        uuid.UUID
	serviceID         uuid.UUID
	serviceResourceID uuid.UUID
}

func layerIdentities(inputs []domain.DistributionInput, existing []domain.Distribution) (map[uuid.UUID]uuid.UUID, error) {
	byID := make(map[uuid.UUID]domain.Distribution, len(existing))
	for _, distribution := range existing {
		byID[distribution.ResourceID] = distribution
	}
	result := make(map[uuid.UUID]uuid.UUID)
	for i, distribution := range inputs {
		if distribution.WebService == nil {
			continue
		}
		var previous domain.WebService
		common := distribution.WebService.CommonInput()
		if distribution.ResourceID != nil && common.ResourceID != nil {
			old := byID[*distribution.ResourceID]
			if old.WebService != nil && old.WebService.Common().ResourceID == *common.ResourceID {
				previous = old.WebService
			}
		}
		check := func(id *uuid.UUID, old map[uuid.UUID]uuid.UUID, position int) error {
			if id == nil {
				return nil
			}
			identity, ok := old[*id]
			_, duplicate := result[*id]
			if !ok || duplicate || *id == uuid.Nil {
				return apperror.Field(fmt.Sprintf("metadata.distributions[%d].web_service.layers[%d].resource_id", i, position), "Layer must belong to this web service and appear only once.", nil)
			}
			result[*id] = identity
			return nil
		}
		switch configuration := distribution.WebService.(type) {
		case *domain.WMSWebServiceInput:
			old, _ := previous.(*domain.WMSWebService)
			byLayerID := make(map[uuid.UUID]uuid.UUID)
			if old != nil {
				for _, layer := range old.Layers {
					byLayerID[layer.ResourceID] = layer.ID
				}
			}
			for j, layer := range configuration.Layers {
				if err := check(layer.ResourceID, byLayerID, j); err != nil {
					return nil, err
				}
			}
		case *domain.WMTSWebServiceInput:
			old, _ := previous.(*domain.WMTSWebService)
			byLayerID := make(map[uuid.UUID]uuid.UUID)
			if old != nil {
				for _, layer := range old.Layers {
					byLayerID[layer.ResourceID] = layer.ID
				}
			}
			for j, layer := range configuration.Layers {
				if err := check(layer.ResourceID, byLayerID, j); err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
}

func layerIDs(resourceID *uuid.UUID, identities map[uuid.UUID]uuid.UUID) (uuid.UUID, uuid.UUID) {
	if resourceID != nil {
		if identity, ok := identities[*resourceID]; ok {
			return identity, *resourceID
		}
	}
	return uuid.New(), uuid.New()
}

func distributionIdentities(inputs []domain.DistributionInput, existing []domain.Distribution) ([]distributionIdentity, error) {
	byResourceID := make(map[uuid.UUID]domain.Distribution, len(existing))
	for _, value := range existing {
		byResourceID[value.ResourceID] = value
	}

	seen := make(map[uuid.UUID]struct{}, len(inputs))
	result := make([]distributionIdentity, len(inputs))
	for i, input := range inputs {
		identity := distributionIdentity{id: uuid.New(), resourceID: uuid.New()}
		var previous domain.Distribution
		if input.ResourceID != nil {
			id := *input.ResourceID
			var ok bool
			previous, ok = byResourceID[id]
			_, duplicate := seen[id]
			if !ok || duplicate || id == uuid.Nil {
				return nil, apperror.Field(fmt.Sprintf("metadata.distributions[%d].resource_id", i), "Distribution must belong to this version and appear only once.", nil)
			}

			seen[id] = struct{}{}
			identity.id, identity.resourceID = previous.ID, previous.ResourceID
		}

		if input.WebService != nil {
			identity.serviceID, identity.serviceResourceID = uuid.New(), uuid.New()
			common := input.WebService.CommonInput()
			if common.ResourceID != nil {
				id := *common.ResourceID
				if previous.WebService == nil || id == uuid.Nil || previous.WebService.Common().ResourceID != id {
					return nil, apperror.Field(fmt.Sprintf("metadata.distributions[%d].web_service.resource_id", i), "Web service must belong to this distribution.", nil)
				}

				previousCommon := previous.WebService.Common()
				identity.serviceID, identity.serviceResourceID = previousCommon.ID, previousCommon.ResourceID
			}
		}
		result[i] = identity
	}

	return result, nil
}

// parameterIdentities validates identity within the original service and parameter collection.
func parameterIdentities(inputs []domain.DistributionInput, existing []domain.Distribution) (map[uuid.UUID]domain.Parameter, error) {
	byID := make(map[uuid.UUID]domain.Distribution, len(existing))
	for _, distribution := range existing {
		byID[distribution.ResourceID] = distribution
	}
	result := make(map[uuid.UUID]domain.Parameter)
	for i, distribution := range inputs {
		if distribution.WebService == nil {
			continue
		}
		var previous domain.WebService
		common := distribution.WebService.CommonInput()
		if distribution.ResourceID != nil && common.ResourceID != nil {
			old := byID[*distribution.ResourceID]
			if old.WebService != nil && old.WebService.Common().ResourceID == *common.ResourceID {
				previous = old.WebService
			}
		}
		target := fmt.Sprintf("metadata.distributions[%d].web_service", i)
		match := func(values []domain.WebServiceParameterInput, old []domain.WebServiceParameter, path string) error {
			inputs := make([]domain.ParameterInput, len(values))
			existing := make([]domain.Parameter, len(old))
			for j := range values {
				inputs[j] = values[j].ParameterInput
			}
			for j := range old {
				existing[j] = old[j].Parameter
			}
			return matchParameters(inputs, existing, target+path, result)
		}
		generic := genericWebServiceInput(distribution.WebService)
		if generic != nil {
			var old domain.GenericWebService
			switch service := previous.(type) {
			case *domain.HTTPWebService:
				old = service.GenericWebService
			case *domain.DynamicWebService:
				old = service.GenericWebService
			}
			if err := match(generic.Parameters, old.Parameters, ".parameters"); err != nil {
				return nil, err
			}
			if generic.Body != nil {
				var oldBody []domain.Parameter
				if old.Body != nil {
					oldBody = old.Body.Parameters
				}
				if err := matchParameters(generic.Body.Parameters, oldBody, target+".body_parameters", result); err != nil {
					return nil, err
				}
			}
		}
		switch configuration := distribution.WebService.(type) {
		case *domain.WFSWebServiceInput:
			var oldParameters []domain.WebServiceParameter
			if old, ok := previous.(*domain.WFSWebService); ok {
				oldParameters = old.Parameters
			}
			if err := match(configuration.Parameters, oldParameters, ".parameters"); err != nil {
				return nil, err
			}
		case *domain.WMSWebServiceInput:
			var old domain.WMSWebService
			if service, ok := previous.(*domain.WMSWebService); ok {
				old = *service
			}
			if err := match(configuration.GetMapParameters, old.GetMapParameters, ".get_map_parameters"); err != nil {
				return nil, err
			}
			if err := match(configuration.GetFeatureInfoParameters, old.GetFeatureInfoParameters, ".get_feature_info_parameters"); err != nil {
				return nil, err
			}
		case *domain.WMTSWebServiceInput:
			var old domain.WMTSWebService
			if service, ok := previous.(*domain.WMTSWebService); ok {
				old = *service
			}
			if err := match(configuration.GetTileParameters, old.GetTileParameters, ".get_tile_parameters"); err != nil {
				return nil, err
			}
			if err := match(configuration.GetFeatureInfoParameters, old.GetFeatureInfoParameters, ".get_feature_info_parameters"); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func matchParameters(inputs []domain.ParameterInput, existing []domain.Parameter, target string, result map[uuid.UUID]domain.Parameter) error {
	byID := make(map[uuid.UUID]domain.Parameter, len(existing))
	for _, parameter := range existing {
		byID[parameter.ResourceID] = parameter
	}
	for i, input := range inputs {
		var previous domain.Parameter
		if input.ResourceID != nil {
			id := *input.ResourceID
			var ok bool
			previous, ok = byID[id]
			_, duplicate := result[id]
			if !ok || duplicate || id == uuid.Nil {
				return apperror.Field(fmt.Sprintf("%s[%d].resource_id", target, i), "Parameter must belong to this collection and appear only once.", nil)
			}
			result[id] = previous
		}
		if schema, ok := input.Schema.(domain.TemplateSchemaInput); ok {
			old, _ := previous.Schema.(domain.TemplateSchema)
			if err := matchParameters(schema.Parameters, old.Parameters, fmt.Sprintf("%s[%d].parameters", target, i), result); err != nil {
				return err
			}
		}
	}
	return nil
}
