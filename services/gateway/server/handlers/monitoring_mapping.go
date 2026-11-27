package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toAPIMonitoringConcepts(values []*metadatalifecyclev1.MonitoringConcept) ([]gatewayapi.MonitoringConcept, error) {
	result := make([]gatewayapi.MonitoringConcept, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, fmt.Errorf("missing monitoring concept")
		}
		id, err := parseUUID("concept_id", value.GetConceptId())
		if err != nil {
			return nil, err
		}
		concept := gatewayapi.MonitoringConcept{Concept: value.GetConcept(), ConceptID: id, Groups: make([]gatewayapi.MonitoringGroup, 0, len(value.Groups))}
		for _, group := range value.Groups {
			if group == nil {
				return nil, fmt.Errorf("missing monitoring group")
			}
			id, err := parseUUID("group id", group.GetId())
			if err != nil {
				return nil, err
			}
			mappedGroup := gatewayapi.MonitoringGroup{ID: id, Name: group.GetName(), Distributions: make([]gatewayapi.MonitoringDistribution, 0, len(group.Distributions))}
			for _, distribution := range group.Distributions {
				mapped, err := toAPIMonitoringDistribution(distribution)
				if err != nil {
					return nil, err
				}
				mappedGroup.Distributions = append(mappedGroup.Distributions, mapped)
			}
			concept.Groups = append(concept.Groups, mappedGroup)
		}
		result = append(result, concept)
	}
	return result, nil
}

func toAPIMonitoringDistribution(value *metadatalifecyclev1.MonitoringDistribution) (gatewayapi.MonitoringDistribution, error) {
	if value == nil || value.Request == nil {
		return gatewayapi.MonitoringDistribution{}, fmt.Errorf("missing monitoring distribution or request")
	}
	id, err := parseUUID("version_id", value.VersionId)
	if err != nil {
		return gatewayapi.MonitoringDistribution{}, err
	}
	resourceID, err := parseUUID("resource_id", value.ResourceId)
	if err != nil {
		return gatewayapi.MonitoringDistribution{}, err
	}
	url, err := parseURI(value.Request.Url)
	if err != nil {
		return gatewayapi.MonitoringDistribution{}, fmt.Errorf("invalid monitoring request URL")
	}
	method := gatewayapi.MonitoringRequestMethod(value.Request.Method)
	if err := method.Validate(); err != nil {
		return gatewayapi.MonitoringDistribution{}, fmt.Errorf("invalid monitoring request method")
	}
	request := gatewayapi.MonitoringRequest{
		Method: method, URL: url,
		Headers: gatewayapi.NilMonitoringRequestHeaders{Null: true},
		Body:    gatewayapi.NilString{Null: true},
	}
	if len(value.Request.Headers) != 0 {
		request.Headers = gatewayapi.NewNilMonitoringRequestHeaders(value.Request.Headers)
	}
	if value.Request.Body != nil {
		request.Body = gatewayapi.NewNilString(string(value.Request.Body))
	}
	contacts := make([]gatewayapi.MonitoringContactPoint, 0, len(value.ContactPoints))
	for _, contact := range value.ContactPoints {
		if contact == nil {
			return gatewayapi.MonitoringDistribution{}, fmt.Errorf("missing monitoring contact")
		}
		contactID, err := parseUUID("contact id", contact.Id)
		if err != nil {
			return gatewayapi.MonitoringDistribution{}, err
		}
		contacts = append(contacts, gatewayapi.MonitoringContactPoint{ID: contactID, Email: contact.Email})
	}
	return gatewayapi.MonitoringDistribution{
		VersionID:     id,
		ResourceID:    resourceID,
		Name:          value.Name,
		Request:       request,
		ContactPoints: contacts,
	}, nil
}
