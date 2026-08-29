package softwareapplication

import (
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type applicationRequest struct {
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	MainEntityOfPage *string             `json:"main_entity_of_page"`
	License          *string             `json:"license"`
	Version          *string             `json:"version"`
	Keywords         string              `json:"keywords"`
	DownloadURL      *string             `json:"download_url"`
	Requirements     *string             `json:"requirements"`
	OperatingSystem  *string             `json:"operating_system"`
	Creators         []uuid.UUID         `json:"creators"`
	Identifiers      []identifierRequest `json:"identifiers"`
	Categories       []uuid.UUID         `json:"categories"`
	ContactPoints    []uuid.UUID         `json:"contact_points"`
}

type createRequest struct {
	ResourceType string             `json:"resource_type"`
	Metadata     applicationRequest `json:"metadata"`
}

type identifierRequest struct {
	Type  domain.IdentifierType `json:"type"`
	Value string                `json:"value"`
}

type replaceDraftRequest struct {
	ResourceType string             `json:"resource_type"`
	Revision     int32              `json:"revision"`
	Metadata     applicationRequest `json:"metadata"`
}

type rejectRequest struct {
	Note string `json:"note"`
}

func (r applicationRequest) rawInput() domain.SoftwareApplicationInput {
	identifiers := make([]domain.IdentifierInput, 0, len(r.Identifiers))
	for _, identifier := range r.Identifiers {
		identifiers = append(identifiers, domain.IdentifierInput{Type: identifier.Type, Value: identifier.Value})
	}
	return domain.SoftwareApplicationInput{
		Name:             r.Name,
		Description:      r.Description,
		MainEntityOfPage: r.MainEntityOfPage,
		License:          r.License,
		Version:          r.Version,
		Keywords:         r.Keywords,
		DownloadURL:      r.DownloadURL,
		Requirements:     r.Requirements,
		OperatingSystem:  r.OperatingSystem,
		CreatorIDs:       r.Creators,
		ContactPointIDs:  r.ContactPoints,
		CategoryIDs:      r.Categories,
		Identifiers:      identifiers,
	}
}

func (r applicationRequest) input() (domain.SoftwareApplicationInput, error) {
	value, err := domain.NewSoftwareApplicationInput(r.rawInput())
	if err != nil {
		return domain.SoftwareApplicationInput{}, fmt.Errorf("parse software application input: %w", err)
	}
	return value, nil
}

func (r replaceDraftRequest) input() (domain.SoftwareApplicationDraftInput, error) {
	value, err := domain.NewSoftwareApplicationDraftInput(domain.SoftwareApplicationDraftInput{
		Revision: r.Revision,
		Metadata: r.Metadata.rawInput(),
	})
	if err != nil {
		return domain.SoftwareApplicationDraftInput{}, fmt.Errorf("parse software application draft input: %w", err)
	}
	return value, nil
}

func (r rejectRequest) input() (domain.SoftwareApplicationRejectionInput, error) {
	value, err := domain.NewSoftwareApplicationRejectionInput(domain.SoftwareApplicationRejectionInput{Note: r.Note})
	if err != nil {
		return domain.SoftwareApplicationRejectionInput{}, fmt.Errorf("parse software application rejection input: %w", err)
	}
	return value, nil
}

type record struct {
	ResourceID uuid.UUID                      `json:"resource_id"`
	VersionID  uuid.UUID                      `json:"version_id"`
	Revision   int32                          `json:"revision"`
	Status     domain.Status                  `json:"status"`
	CreatedBy  string                         `json:"created_by"`
	GroupID    uuid.UUID                      `json:"group_id"`
	GroupName  string                         `json:"group_name"`
	CreatedAt  time.Time                      `json:"created_at"`
	UpdatedAt  time.Time                      `json:"updated_at"`
	Metadata   softwareApplicationResponseDTO `json:"metadata"`
}

type softwareApplicationResponseDTO struct {
	ID               uuid.UUID            `json:"id"`
	Name             string               `json:"name"`
	Description      string               `json:"description"`
	MainEntityOfPage *string              `json:"main_entity_of_page"`
	License          *string              `json:"license"`
	Version          *string              `json:"version"`
	Keywords         string               `json:"keywords"`
	DownloadURL      *string              `json:"download_url"`
	Requirements     *string              `json:"requirements"`
	OperatingSystem  *string              `json:"operating_system"`
	Creators         []resourceReference  `json:"creators"`
	Identifiers      []identifierResponse `json:"identifiers"`
	Categories       []resourceReference  `json:"categories"`
	ContactPoints    *[]uuid.UUID         `json:"contact_points"`
}

type resourceReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type identifierResponse struct {
	ID    uuid.UUID             `json:"id"`
	Type  domain.IdentifierType `json:"type"`
	Value string                `json:"value"`
}

func applicationResponse(value domain.SoftwareApplicationVersion, includeContacts bool) record {
	return record{
		ResourceID: value.SoftwareApplicationID,
		VersionID:  value.VersionID,
		Revision:   value.Revision,
		Status:     value.Status,
		CreatedBy:  string(value.CreatedBy),
		GroupID:    value.Group.ID,
		GroupName:  value.Group.Name,
		CreatedAt:  value.CreatedAt,
		UpdatedAt:  value.UpdatedAt,
		Metadata:   applicationMetadataDTO(value.Metadata, includeContacts),
	}
}

func applicationMetadataDTO(value domain.SoftwareApplicationMetadata, includeContacts bool) softwareApplicationResponseDTO {
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
		contacts := make([]uuid.UUID, 0, len(value.ContactPoints))
		for _, contact := range value.ContactPoints {
			contacts = append(contacts, contact.ID)
		}
		result.ContactPoints = &contacts
	}
	return result
}

func resourceReferenceDTO(id uuid.UUID, name string) resourceReference {
	return resourceReference{ID: id, Name: name}
}

func identifierDTO(value domain.Identifier) identifierResponse {
	return identifierResponse{
		ID:    value.ID,
		Type:  value.Type,
		Value: value.Value,
	}
}
