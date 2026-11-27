package grpcapi

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toProtoMonitoringConcepts(values []domain.MonitoringConcept) []*metadatalifecyclev1.MonitoringConcept {
	result := make([]*metadatalifecyclev1.MonitoringConcept, 0, len(values))
	for _, value := range values {
		concept := &metadatalifecyclev1.MonitoringConcept{
			ConceptId: value.ID.String(), Concept: value.Name,
			Groups: make([]*metadatalifecyclev1.MonitoringGroup, 0, len(value.Groups)),
		}
		for _, group := range value.Groups {
			mappedGroup := &metadatalifecyclev1.MonitoringGroup{
				Id: group.ID.String(), Name: group.Name,
				Distributions: make([]*metadatalifecyclev1.MonitoringDistribution, 0, len(group.Distributions)),
			}
			for _, distribution := range group.Distributions {
				contacts := make([]*metadatalifecyclev1.MonitoringContactPoint, 0, len(distribution.ContactPoints))
				for _, contact := range distribution.ContactPoints {
					contacts = append(contacts, &metadatalifecyclev1.MonitoringContactPoint{
						Id: contact.ID.String(), Email: contact.Email,
					})
				}
				mappedGroup.Distributions = append(mappedGroup.Distributions, &metadatalifecyclev1.MonitoringDistribution{
					VersionId:  distribution.ID.String(),
					ResourceId: distribution.ResourceID.String(),
					Name:       distribution.Name,
					Request: &metadatalifecyclev1.MonitoringRequest{
						Method:  string(distribution.Request.Method),
						Url:     distribution.Request.URL,
						Headers: distribution.Request.Headers,
						Body:    distribution.Request.Body,
					},
					ContactPoints: contacts,
				})
			}
			concept.Groups = append(concept.Groups, mappedGroup)
		}
		result = append(result, concept)
	}
	return result
}
