package softwareapplication

import (
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwarelifecycle"
	"github.com/google/uuid"
)

type applicationRequest struct {
	SourceIRI        string               `json:"source_iri"`
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
	SourceIRI string                  `json:"source_iri"`
	Type      metadata.IdentifierType `json:"type"`
	Value     string                  `json:"value"`
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
	SourceIRI     string    `json:"source_iri"`
}

func (r applicationRequest) domain() metadata.SoftwareApplication {
	result := metadata.SoftwareApplication{
		ID:               uuid.Nil,
		SourceIRI:        r.SourceIRI,
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
			SourceIRI:     "",
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
			ID:        uuid.Nil,
			SourceIRI: identifier.SourceIRI,
			Type:      identifier.Type,
			Value:     identifier.Value,
		})
	}
	for _, category := range r.Categories {
		result.Categories = append(result.Categories, metadata.Category{
			ID:        category.ID,
			SourceIRI: "",
			Name:      "",
			Scheme: metadata.CategoryScheme{
				ID:          uuid.Nil,
				SourceIRI:   "",
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
			ID:        contact.ID,
			SourceIRI: "",
			Email:     "",
			Role:      "",
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
	Group      categorySchemeResponse         `json:"group"`
	Metadata   softwareApplicationResponseDTO `json:"metadata"`
}

type softwareApplicationResponseDTO struct {
	ID               uuid.UUID               `json:"id"`
	SourceIRI        string                  `json:"source_iri"`
	Name             string                  `json:"name"`
	Description      string                  `json:"description"`
	MainEntityOfPage *string                 `json:"main_entity_of_page"`
	License          *string                 `json:"license"`
	Version          *string                 `json:"version"`
	Keywords         string                  `json:"keywords"`
	DownloadURL      *string                 `json:"download_url"`
	Requirements     *string                 `json:"requirements"`
	OperatingSystem  *string                 `json:"operating_system"`
	Creators         []organizationResponse  `json:"creators"`
	Identifiers      []identifierResponse    `json:"identifiers"`
	Categories       []categoryResponse      `json:"categories"`
	ContactPoints    *[]contactPointResponse `json:"contact_points"`
}

type organizationResponse struct {
	ID            uuid.UUID               `json:"id"`
	SourceIRI     string                  `json:"source_iri"`
	Name          string                  `json:"name"`
	Street        *string                 `json:"street"`
	Locality      *string                 `json:"locality"`
	PostalCode    *string                 `json:"postal_code"`
	Country       *string                 `json:"country"`
	LogoURL       *string                 `json:"logo_url"`
	PageURL       *string                 `json:"page_url"`
	Email         *string                 `json:"email"`
	LEICode       *string                 `json:"lei_code"`
	ContactPoints *[]contactPointResponse `json:"contact_points"`
	Identifiers   []identifierResponse    `json:"identifiers"`
}

type identifierResponse struct {
	ID        uuid.UUID               `json:"id"`
	SourceIRI string                  `json:"source_iri"`
	Type      metadata.IdentifierType `json:"type"`
	Value     string                  `json:"value"`
}

type categoryResponse struct {
	ID        uuid.UUID              `json:"id"`
	SourceIRI string                 `json:"source_iri"`
	Name      string                 `json:"name"`
	Scheme    categorySchemeResponse `json:"scheme"`
	Children  []categoryResponse     `json:"children"`
}

type categorySchemeResponse struct {
	ID        uuid.UUID                `json:"id"`
	SourceIRI string                   `json:"source_iri"`
	Name      string                   `json:"name"`
	Label     *string                  `json:"label"`
	Logo      *string                  `json:"logo"`
	Page      *string                  `json:"page"`
	Color     *string                  `json:"color"`
	Position  int32                    `json:"position"`
	Concept   metadata.TopLevelConcept `json:"concept"`
}

func applicationResponse(value softwarelifecycle.Application, includeContacts bool) record {
	return record{
		ResourceID: value.ApplicationID,
		VersionID:  value.VersionID,
		Revision:   value.Revision,
		Status:     value.Status,
		CreatedBy:  value.CreatedBy,
		Group:      categorySchemeDTO(value.Group),
		Metadata:   applicationMetadataDTO(value.Metadata, includeContacts),
	}
}

func applicationMetadataDTO(value metadata.SoftwareApplication, includeContacts bool) softwareApplicationResponseDTO {
	result := softwareApplicationResponseDTO{
		ID:               value.ID,
		SourceIRI:        value.SourceIRI,
		Name:             value.Name,
		Description:      value.Description,
		MainEntityOfPage: value.MainEntityOfPage,
		License:          value.License,
		Version:          value.Version,
		Keywords:         value.Keywords,
		DownloadURL:      value.DownloadURL,
		Requirements:     value.Requirements,
		OperatingSystem:  value.OperatingSystem,
		Creators:         make([]organizationResponse, 0, len(value.Creators)),
		Identifiers:      make([]identifierResponse, 0, len(value.Identifiers)),
		Categories:       make([]categoryResponse, 0, len(value.Categories)),
		ContactPoints:    nil,
	}
	for _, creator := range value.Creators {
		result.Creators = append(result.Creators, organizationDTO(creator, includeContacts))
	}
	for _, identifier := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierDTO(identifier))
	}
	for _, category := range value.Categories {
		result.Categories = append(result.Categories, categoryDTO(category))
	}
	if includeContacts {
		contacts := make([]contactPointResponse, 0, len(value.ContactPoints))
		for _, contact := range value.ContactPoints {
			contacts = append(contacts, contactPointDTO(contact))
		}
		result.ContactPoints = &contacts
	}
	return result
}

type contactPointResponse struct {
	ID        uuid.UUID                 `json:"id"`
	SourceIRI string                    `json:"source_iri"`
	Email     string                    `json:"email"`
	Role      metadata.ContactPointRole `json:"role"`
}

func contactPointDTO(value metadata.ContactPoint) contactPointResponse {
	return contactPointResponse{ID: value.ID, SourceIRI: value.SourceIRI, Email: value.Email, Role: value.Role}
}

func organizationDTO(value metadata.Organization, includeContacts bool) organizationResponse {
	result := organizationResponse{
		ID:            value.ID,
		SourceIRI:     value.SourceIRI,
		Name:          value.Name,
		Street:        value.Street,
		Locality:      value.Locality,
		PostalCode:    value.PostalCode,
		Country:       value.Country,
		LogoURL:       value.LogoURL,
		PageURL:       value.PageURL,
		Email:         value.Email,
		LEICode:       value.LEICode,
		ContactPoints: nil,
		Identifiers:   make([]identifierResponse, 0, len(value.Identifiers)),
	}
	for _, identifier := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierDTO(identifier))
	}
	for _, contact := range value.ContactPoints {
		if includeContacts {
			if result.ContactPoints == nil {
				result.ContactPoints = new([]contactPointResponse)
			}
			*result.ContactPoints = append(*result.ContactPoints, contactPointDTO(contact))
		}
	}
	return result
}

func identifierDTO(value metadata.Identifier) identifierResponse {
	return identifierResponse{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Type:      value.Type,
		Value:     value.Value,
	}
}

func categoryDTO(value metadata.Category) categoryResponse {
	result := categoryResponse{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Name:      value.Name,
		Scheme:    categorySchemeDTO(value.Scheme),
		Children:  make([]categoryResponse, 0, len(value.Children)),
	}
	for _, child := range value.Children {
		result.Children = append(result.Children, categoryDTO(child))
	}
	return result
}

func categorySchemeDTO(value metadata.CategoryScheme) categorySchemeResponse {
	return categorySchemeResponse{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Name:      value.Name,
		Label:     value.Label,
		Logo:      value.Logo,
		Page:      value.Page,
		Color:     value.Color,
		Position:  value.Position,
		Concept:   value.Concept,
	}
}
