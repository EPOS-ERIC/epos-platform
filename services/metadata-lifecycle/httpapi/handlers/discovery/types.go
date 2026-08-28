package discovery

import (
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type discoveryItemResponse struct {
	Type          domain.ResourceType          `json:"type"`
	ResourceID    uuid.UUID                    `json:"resource_id"`
	VersionID     uuid.UUID                    `json:"version_id"`
	Name          string                       `json:"name"`
	Status        domain.Status                `json:"status"`
	Group         discoveryReferenceResponse   `json:"group"`
	Providers     []discoveryReferenceResponse `json:"providers,omitempty"`
	Creators      []discoveryReferenceResponse `json:"creators,omitempty"`
	Categories    []discoveryReferenceResponse `json:"categories,omitempty"`
	Distributions []discoveryReferenceResponse `json:"distributions,omitempty"`
}

type discoveryReferenceResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
