package domain

import (
	"fmt"

	"github.com/google/uuid"
)

// MonitoringDistribution contains the persisted data needed to build a target.
type MonitoringDistribution struct {
	ConceptID     uuid.UUID
	ConceptName   string
	GroupID       uuid.UUID
	GroupName     string
	Distribution  Distribution
	ContactPoints []ContactPoint
}

type MonitoringConcept struct {
	ID     uuid.UUID
	Name   string
	Groups []MonitoringGroup
}

type MonitoringGroup struct {
	ID            uuid.UUID
	Name          string
	Distributions []MonitoringTarget
}

type MonitoringTarget struct {
	ID            uuid.UUID
	ResourceID    uuid.UUID
	Name          string
	Request       PreparedHTTPRequest
	ContactPoints []ContactPoint
}

// MonitoringRequest prepares a provider request without contacting the provider.
func (value Distribution) MonitoringRequest() (PreparedHTTPRequest, error) {
	if value.Type == DistributionTypeDownloadableFile && value.DownloadableFile != nil {
		return PreparedHTTPRequest{Method: HTTPMethodGet, URL: value.DownloadableFile.URL}, nil
	}

	switch service := value.WebService.(type) {
	case *HTTPWebService:
		return service.PrepareAccess(DistributionAccessInput{Input: HTTPAccessInput{}})
	case *WFSWebService:
		return service.PrepareAccess(DistributionAccessInput{Input: WFSAccessInput{}})
	case *DynamicWebService:
		longitude, err := dynamicServiceMonitoringCoordinate(service.parameterWithRole(WebServiceParameterRoleLongitude))
		if err != nil {
			return PreparedHTTPRequest{}, fmt.Errorf("longitude default: %w", err)
		}

		latitude, err := dynamicServiceMonitoringCoordinate(service.parameterWithRole(WebServiceParameterRoleLatitude))
		if err != nil {
			return PreparedHTTPRequest{}, fmt.Errorf("latitude default: %w", err)
		}

		return service.PrepareAccess(DistributionAccessInput{
			Input: DynamicAccessInput{Longitude: longitude, Latitude: latitude},
		})
	case *WMSWebService:
		return PreparedHTTPRequest{Method: HTTPMethodGet, URL: service.CapabilitiesURL}, nil
	case *WMTSWebService:
		return PreparedHTTPRequest{Method: HTTPMethodGet, URL: service.CapabilitiesURL}, nil
	}

	return PreparedHTTPRequest{}, fmt.Errorf("unsupported distribution configuration")
}

func dynamicServiceMonitoringCoordinate(parameter *Parameter) (float64, error) {
	if parameter == nil {
		return 0, fmt.Errorf("missing coordinate parameter")
	}

	schema, ok := parameter.Schema.(FloatSchema)
	if !ok || len(schema.Default) != 1 || schema.MultipleValues {
		return 0, fmt.Errorf("invalid coordinate default")
	}

	return schema.Default[0], nil
}
