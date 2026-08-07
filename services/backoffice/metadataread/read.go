// Package metadataread reads canonical metadata materialized in PostgreSQL.
package metadataread

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("metadata resource not found")

type ResourceType string

const (
	ResourceTypeDataset             ResourceType = "dataset"
	ResourceTypeDistribution        ResourceType = "distribution"
	ResourceTypeWebService          ResourceType = "webservice"
	ResourceTypeOrganization        ResourceType = "organization"
	ResourceTypeConcept             ResourceType = "concept"
	ResourceTypeConceptScheme       ResourceType = "conceptscheme"
	ResourceTypeSoftwareApplication ResourceType = "softwareapplication"
	ResourceTypeSoftwareSourceCode  ResourceType = "softwaresourcecode"
)

func ParseResourceType(value string) (ResourceType, error) {
	resourceType := ResourceType(value)
	switch resourceType {
	case ResourceTypeDataset, ResourceTypeDistribution, ResourceTypeWebService, ResourceTypeOrganization, ResourceTypeConcept, ResourceTypeConceptScheme, ResourceTypeSoftwareApplication, ResourceTypeSoftwareSourceCode:
		return resourceType, nil
	default:
		return "", fmt.Errorf("unsupported resource type: %s", value)
	}
}

type Resource struct {
	Dataset             *metadata.Dataset
	Distribution        *metadata.Distribution
	WebService          *metadata.WebService
	Organization        *metadata.Organization
	Concept             *metadata.Category
	ConceptScheme       *metadata.CategoryScheme
	SoftwareApplication *metadata.SoftwareApplication
	SoftwareSourceCode  *metadata.SoftwareSourceCode
}

type repository interface {
	Get(context.Context, ResourceType, uuid.UUID) (Resource, error)
}

type Reader struct {
	repository repository
}

func New(repository repository) *Reader {
	return &Reader{repository: repository}
}

func (r *Reader) Get(ctx context.Context, resourceType ResourceType, id uuid.UUID) (Resource, error) {
	resource, err := r.repository.Get(ctx, resourceType, id)
	if err != nil {
		return Resource{}, fmt.Errorf("get %s %q: %w", resourceType, id, err)
	}

	return resource, nil
}
