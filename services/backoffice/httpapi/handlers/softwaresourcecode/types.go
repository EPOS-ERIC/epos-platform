package softwaresourcecode

import (
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/softwarelifecycle"
	"github.com/google/uuid"
)

type sourceCodeRequest struct {
	SourceIRI            string               `json:"source_iri"`
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	ProgrammingLanguage  *string              `json:"programming_language"`
	Keywords             string               `json:"keywords"`
	Repository           *string              `json:"repository"`
	License              *string              `json:"license"`
	SoftwareRequirements *string              `json:"software_requirements"`
	RuntimePlatform      *string              `json:"runtime_platform"`
	MainEntityOfPage     *string              `json:"main_entity_of_page"`
	Version              *string              `json:"version"`
	Creators             []canonicalReference `json:"creators"`
	Identifiers          []identifierRequest  `json:"identifiers"`
	Categories           []canonicalReference `json:"categories"`
	ContactPoints        []canonicalReference `json:"contact_points"`
}

type createRequest struct {
	GroupID  uuid.UUID         `json:"group_id"`
	Metadata sourceCodeRequest `json:"metadata"`
}

type replaceDraftRequest struct {
	Revision int32             `json:"revision"`
	Metadata sourceCodeRequest `json:"metadata"`
}

type rejectionRequest struct {
	Note string `json:"note"`
}

type copyRequest struct {
	TargetGroupID uuid.UUID `json:"target_group_id"`
	SourceIRI     string    `json:"source_iri"`
}

type canonicalReference struct {
	ID uuid.UUID `json:"id"`
}

type identifierRequest struct {
	SourceIRI string                  `json:"source_iri"`
	Type      metadata.IdentifierType `json:"type"`
	Value     string                  `json:"value"`
}

type sourceCodeRecord struct {
	ResourceID uuid.UUID                  `json:"resource_id"`
	VersionID  uuid.UUID                  `json:"version_id"`
	Revision   int32                      `json:"revision"`
	Status     metadata.DatasetStatus     `json:"status"`
	CreatedBy  uuid.UUID                  `json:"created_by"`
	Group      categoryScheme             `json:"group"`
	Metadata   sourceCodeMetadataResponse `json:"metadata"`
}

type sourceCodeMetadataResponse struct {
	ID                   uuid.UUID            `json:"id"`
	SourceIRI            string               `json:"source_iri"`
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	ProgrammingLanguage  *string              `json:"programming_language"`
	Keywords             string               `json:"keywords"`
	Repository           *string              `json:"repository"`
	License              *string              `json:"license"`
	SoftwareRequirements *string              `json:"software_requirements"`
	RuntimePlatform      *string              `json:"runtime_platform"`
	MainEntityOfPage     *string              `json:"main_entity_of_page"`
	Version              *string              `json:"version"`
	Creators             []organization       `json:"creators"`
	Identifiers          []identifierResponse `json:"identifiers"`
	Categories           []category           `json:"categories"`
	ContactPoints        *[]contactPoint      `json:"contact_points"`
}

type organization struct {
	ID            uuid.UUID            `json:"id"`
	SourceIRI     string               `json:"source_iri"`
	Name          string               `json:"name"`
	Street        *string              `json:"street"`
	Locality      *string              `json:"locality"`
	PostalCode    *string              `json:"postal_code"`
	Country       *string              `json:"country"`
	LogoURL       *string              `json:"logo_url"`
	PageURL       *string              `json:"page_url"`
	Email         *string              `json:"email"`
	LEICode       *string              `json:"lei_code"`
	ContactPoints *[]contactPoint      `json:"contact_points"`
	Identifiers   []identifierResponse `json:"identifiers"`
}

type category struct {
	ID        uuid.UUID      `json:"id"`
	SourceIRI string         `json:"source_iri"`
	Name      string         `json:"name"`
	Scheme    categoryScheme `json:"scheme"`
	Children  []category     `json:"children"`
}

type categoryScheme struct {
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

func (r sourceCodeRequest) domain() metadata.SoftwareSourceCode {
	result := metadata.SoftwareSourceCode{
		ID:                   uuid.Nil,
		SourceIRI:            r.SourceIRI,
		Name:                 r.Name,
		Description:          r.Description,
		ProgrammingLanguage:  r.ProgrammingLanguage,
		Keywords:             r.Keywords,
		Repository:           r.Repository,
		License:              r.License,
		SoftwareRequirements: r.SoftwareRequirements,
		RuntimePlatform:      r.RuntimePlatform,
		MainEntityOfPage:     r.MainEntityOfPage,
		Version:              r.Version,
		Creators:             make([]metadata.Organization, 0, len(r.Creators)),
		Identifiers:          make([]metadata.Identifier, 0, len(r.Identifiers)),
		Categories:           make([]metadata.Category, 0, len(r.Categories)),
		ContactPoints:        make([]metadata.ContactPoint, 0, len(r.ContactPoints)),
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
	for _, value := range r.Identifiers {
		result.Identifiers = append(result.Identifiers, metadata.Identifier{
			ID:        uuid.Nil,
			SourceIRI: value.SourceIRI,
			Type:      value.Type,
			Value:     value.Value,
		})
	}
	for _, value := range r.Categories {
		result.Categories = append(result.Categories, metadata.Category{
			ID:        value.ID,
			SourceIRI: "",
			Name:      "",
			Scheme: metadata.CategoryScheme{
				ID:        uuid.Nil,
				SourceIRI: "",
				Name:      "",
				Label:     nil,
				Logo:      nil,
				Page:      nil,
				Color:     nil,
				Position:  0,
				Concept:   "",
			},
			Children: nil,
		})
	}
	for _, value := range r.ContactPoints {
		result.ContactPoints = append(result.ContactPoints, metadata.ContactPoint{
			ID:        value.ID,
			SourceIRI: "",
			Email:     "",
			Role:      "",
		})
	}
	return result
}

func sourceCodeResponse(value softwarelifecycle.SourceCode, includeContacts bool) sourceCodeRecord {
	return sourceCodeRecord{
		ResourceID: value.SourceCodeID,
		VersionID:  value.VersionID,
		Revision:   value.Revision,
		Status:     value.Status,
		CreatedBy:  value.CreatedBy,
		Group:      categorySchemeDTO(value.Group),
		Metadata:   sourceCodeMetadataDTO(value.Metadata, includeContacts),
	}
}

func sourceCodeMetadataDTO(value metadata.SoftwareSourceCode, includeContacts bool) sourceCodeMetadataResponse {
	result := sourceCodeMetadataResponse{
		ID:                   value.ID,
		SourceIRI:            value.SourceIRI,
		Name:                 value.Name,
		Description:          value.Description,
		ProgrammingLanguage:  value.ProgrammingLanguage,
		Keywords:             value.Keywords,
		Repository:           value.Repository,
		License:              value.License,
		SoftwareRequirements: value.SoftwareRequirements,
		RuntimePlatform:      value.RuntimePlatform,
		MainEntityOfPage:     value.MainEntityOfPage,
		Version:              value.Version,
		Creators:             make([]organization, 0, len(value.Creators)),
		Identifiers:          make([]identifierResponse, 0, len(value.Identifiers)),
		Categories:           make([]category, 0, len(value.Categories)),
		ContactPoints:        nil,
	}
	for _, creator := range value.Creators {
		result.Creators = append(result.Creators, organizationDTO(creator, includeContacts))
	}
	for _, item := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierDTO(item))
	}
	for _, item := range value.Categories {
		result.Categories = append(result.Categories, categoryDTO(item))
	}
	if includeContacts {
		contacts := make([]contactPoint, 0, len(value.ContactPoints))
		for _, item := range value.ContactPoints {
			contacts = append(contacts, contactPointDTO(item))
		}
		result.ContactPoints = &contacts
	}
	return result
}

type contactPoint struct {
	ID        uuid.UUID                 `json:"id"`
	SourceIRI string                    `json:"source_iri"`
	Email     string                    `json:"email"`
	Role      metadata.ContactPointRole `json:"role"`
}

func contactPointDTO(value metadata.ContactPoint) contactPoint {
	return contactPoint{ID: value.ID, SourceIRI: value.SourceIRI, Email: value.Email, Role: value.Role}
}

func organizationDTO(value metadata.Organization, includeContacts bool) organization {
	result := organization{
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
	for _, item := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierDTO(item))
	}
	for _, contact := range value.ContactPoints {
		if includeContacts {
			if result.ContactPoints == nil {
				result.ContactPoints = new([]contactPoint)
			}
			*result.ContactPoints = append(*result.ContactPoints, contactPointDTO(contact))
		}
	}
	return result
}

func categoryDTO(value metadata.Category) category {
	result := category{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Name:      value.Name,
		Scheme:    categorySchemeDTO(value.Scheme),
		Children:  make([]category, 0, len(value.Children)),
	}
	for _, item := range value.Children {
		result.Children = append(result.Children, categoryDTO(item))
	}
	return result
}

func categorySchemeDTO(value metadata.CategoryScheme) categoryScheme {
	return categoryScheme{
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

func identifierDTO(value metadata.Identifier) identifierResponse {
	return identifierResponse{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Type:      value.Type,
		Value:     value.Value,
	}
}

type identifierResponse struct {
	ID        uuid.UUID               `json:"id"`
	SourceIRI string                  `json:"source_iri"`
	Type      metadata.IdentifierType `json:"type"`
	Value     string                  `json:"value"`
}
