package softwareapplication

import (
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwarelifecycle"
	"github.com/google/uuid"
)

type applicationRequest struct {
	Name             string               `json:"name"`
	Description      string               `json:"description"`
	MainEntityOfPage *string              `json:"main_entity_of_page"`
	License          *string              `json:"license"`
	Version          *string              `json:"version"`
	Keywords         string               `json:"keywords"`
	DownloadURL      *string              `json:"download_url"`
	Requirements     *string              `json:"requirements"`
	OperatingSystem  *string              `json:"operating_system"`
	Creators         []canonicalReference `json:"creators"`
	Identifiers      []identifierRequest  `json:"identifiers"`
	Categories       []canonicalReference `json:"categories"`
	ContactPoints    []canonicalReference `json:"contact_points"`
}

type createRequest struct {
	GroupID      uuid.UUID          `json:"group_id,omitempty"`
	ResourceType string             `json:"resource_type"`
	Metadata     applicationRequest `json:"metadata"`
}

type canonicalReference struct {
	ID uuid.UUID `json:"id"`
}

type identifierRequest struct {
	Type  metadata.IdentifierType `json:"type"`
	Value string                  `json:"value"`
}

type replaceDraftRequest struct {
	ResourceType string             `json:"resource_type"`
	Revision     int32              `json:"revision"`
	Metadata     applicationRequest `json:"metadata"`
}

type rejectRequest struct {
	Note string `json:"note"`
}

type copyRequest struct {
	TargetGroupID uuid.UUID `json:"target_group_id"`
}

func (r applicationRequest) domain() metadata.SoftwareApplication {
	result := metadata.SoftwareApplication{
		ID:               uuid.Nil,
		Name:             r.Name,
		Description:      r.Description,
		MainEntityOfPage: r.MainEntityOfPage,
		License:          r.License,
		Version:          r.Version,
		Keywords:         r.Keywords,
		DownloadURL:      r.DownloadURL,
		Requirements:     r.Requirements,
		OperatingSystem:  r.OperatingSystem,
		Creators:         make([]metadata.Organization, 0, len(r.Creators)),
		Identifiers:      make([]metadata.Identifier, 0, len(r.Identifiers)),
		Categories:       make([]metadata.Category, 0, len(r.Categories)),
		ContactPoints:    make([]metadata.ContactPoint, 0, len(r.ContactPoints)),
	}
	for _, creator := range r.Creators {
		result.Creators = append(result.Creators, metadata.Organization{
			ID:            creator.ID,
			Name:          "",
			Street:        nil,
			Locality:      nil,
			PostalCode:    nil,
			Country:       nil,
			LogoURL:       nil,
			PageURL:       nil,
			Email:         nil,
			LEICode:       nil,
			ContactPoints: nil,
			Identifiers:   nil,
		})
	}
	for _, identifier := range r.Identifiers {
		result.Identifiers = append(result.Identifiers, metadata.Identifier{
			ID:    uuid.Nil,
			Type:  identifier.Type,
			Value: identifier.Value,
		})
	}
	for _, category := range r.Categories {
		result.Categories = append(result.Categories, metadata.Category{
			ID:   category.ID,
			Name: "",
			Scheme: metadata.CategoryScheme{
				ID:          uuid.Nil,
				Name:        "",
				Description: nil,
				Label:       nil,
				Logo:        nil,
				Page:        nil,
				Color:       nil,
				Position:    0,
				Concept:     "",
			},
			Children: nil,
		})
	}
	for _, contact := range r.ContactPoints {
		result.ContactPoints = append(result.ContactPoints, metadata.ContactPoint{
			ID:    contact.ID,
			Email: "",
			Role:  "",
		})
	}
	return result
}

type record struct {
	ResourceID uuid.UUID                      `json:"resource_id"`
	VersionID  uuid.UUID                      `json:"version_id"`
	Revision   int32                          `json:"revision"`
	Status     metadata.DatasetStatus         `json:"status"`
	CreatedBy  string                         `json:"created_by"`
	GroupID    uuid.UUID                      `json:"group_id"`
	GroupName  string                         `json:"group_name"`
	CreatedAt  time.Time                      `json:"created_at"`
	UpdatedAt  time.Time                      `json:"updated_at"`
	Metadata   softwareApplicationResponseDTO `json:"metadata"`
}

type softwareApplicationResponseDTO struct {
	ID               uuid.UUID                `json:"id"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	MainEntityOfPage *string                  `json:"main_entity_of_page"`
	License          *string                  `json:"license"`
	Version          *string                  `json:"version"`
	Keywords         string                   `json:"keywords"`
	DownloadURL      *string                  `json:"download_url"`
	Requirements     *string                  `json:"requirements"`
	OperatingSystem  *string                  `json:"operating_system"`
	Creators         []resourceReference      `json:"creators"`
	Identifiers      []identifierResponse     `json:"identifiers"`
	Categories       []resourceReference      `json:"categories"`
	ContactPoints    *[]contactPointReference `json:"contact_points"`
}

type resourceReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type identifierResponse struct {
	ID    uuid.UUID               `json:"id"`
	Type  metadata.IdentifierType `json:"type"`
	Value string                  `json:"value"`
}

func applicationResponse(value softwarelifecycle.Application, includeContacts bool) record {
	return record{
		ResourceID: value.ApplicationID,
		VersionID:  value.VersionID,
		Revision:   value.Revision,
		Status:     value.Status,
		CreatedBy:  value.CreatedBy,
		GroupID:    value.Group.ID,
		GroupName:  value.Group.Name,
		CreatedAt:  value.CreatedAt,
		UpdatedAt:  value.UpdatedAt,
		Metadata:   applicationMetadataDTO(value.Metadata, includeContacts),
	}
}

func applicationMetadataDTO(value metadata.SoftwareApplication, includeContacts bool) softwareApplicationResponseDTO {
	result := softwareApplicationResponseDTO{
		ID:               value.ID,
		Name:             value.Name,
		Description:      value.Description,
		MainEntityOfPage: value.MainEntityOfPage,
		License:          value.License,
		Version:          value.Version,
		Keywords:         value.Keywords,
		DownloadURL:      value.DownloadURL,
		Requirements:     value.Requirements,
		OperatingSystem:  value.OperatingSystem,
		Creators:         make([]resourceReference, 0, len(value.Creators)),
		Identifiers:      make([]identifierResponse, 0, len(value.Identifiers)),
		Categories:       make([]resourceReference, 0, len(value.Categories)),
		ContactPoints:    nil,
	}
	for _, creator := range value.Creators {
		result.Creators = append(result.Creators, resourceReferenceDTO(creator.ID, creator.Name))
	}
	for _, identifier := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierDTO(identifier))
	}
	for _, category := range value.Categories {
		result.Categories = append(result.Categories, resourceReferenceDTO(category.ID, category.Name))
	}
	if includeContacts {
		contacts := make([]contactPointReference, 0, len(value.ContactPoints))
		for _, contact := range value.ContactPoints {
			contacts = append(contacts, contactPointReferenceDTO(contact))
		}
		result.ContactPoints = &contacts
	}
	return result
}

type contactPointReference struct {
	ID uuid.UUID `json:"id"`
}

func resourceReferenceDTO(id uuid.UUID, name string) resourceReference {
	return resourceReference{ID: id, Name: name}
}

func contactPointReferenceDTO(value metadata.ContactPoint) contactPointReference {
	// TODO: we can probably skip the struct here and just use the id (also is this duplicated?)
	return contactPointReference{ID: value.ID}
}

func identifierDTO(value metadata.Identifier) identifierResponse {
	return identifierResponse{
		ID:    value.ID,
		Type:  value.Type,
		Value: value.Value,
	}
}
