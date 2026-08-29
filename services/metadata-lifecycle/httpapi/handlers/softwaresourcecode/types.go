package softwaresourcecode

import (
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type sourceCodeRequest struct {
	Name                 string              `json:"name"`
	Description          string              `json:"description"`
	ProgrammingLanguage  *string             `json:"programming_language"`
	Keywords             string              `json:"keywords"`
	Repository           *string             `json:"repository"`
	License              *string             `json:"license"`
	SoftwareRequirements *string             `json:"software_requirements"`
	RuntimePlatform      *string             `json:"runtime_platform"`
	MainEntityOfPage     *string             `json:"main_entity_of_page"`
	Version              *string             `json:"version"`
	Creators             []uuid.UUID         `json:"creators"`
	Identifiers          []identifierRequest `json:"identifiers"`
	Categories           []uuid.UUID         `json:"categories"`
	ContactPoints        []uuid.UUID         `json:"contact_points"`
}

type createRequest struct {
	ResourceType string            `json:"resource_type"`
	Metadata     sourceCodeRequest `json:"metadata"`
}

type replaceDraftRequest struct {
	ResourceType string            `json:"resource_type"`
	Revision     int32             `json:"revision"`
	Metadata     sourceCodeRequest `json:"metadata"`
}

type rejectionRequest struct {
	Note string `json:"note"`
}

type identifierRequest struct {
	Type  domain.IdentifierType `json:"type"`
	Value string                `json:"value"`
}

type sourceCodeRecord struct {
	ResourceID uuid.UUID                  `json:"resource_id"`
	VersionID  uuid.UUID                  `json:"version_id"`
	Revision   int32                      `json:"revision"`
	Status     domain.Status              `json:"status"`
	CreatedBy  string                     `json:"created_by"`
	GroupID    uuid.UUID                  `json:"group_id"`
	GroupName  string                     `json:"group_name"`
	CreatedAt  time.Time                  `json:"created_at"`
	UpdatedAt  time.Time                  `json:"updated_at"`
	Metadata   sourceCodeMetadataResponse `json:"metadata"`
}

type sourceCodeMetadataResponse struct {
	ID                   uuid.UUID            `json:"id"`
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
	Creators             []resourceReference  `json:"creators"`
	Identifiers          []identifierResponse `json:"identifiers"`
	Categories           []resourceReference  `json:"categories"`
	ContactPoints        *[]uuid.UUID         `json:"contact_points"`
}

type resourceReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (r sourceCodeRequest) rawInput() domain.SoftwareSourceCodeInput {
	identifiers := make([]domain.IdentifierInput, 0, len(r.Identifiers))
	for _, identifier := range r.Identifiers {
		identifiers = append(identifiers, domain.IdentifierInput{Type: identifier.Type, Value: identifier.Value})
	}
	return domain.SoftwareSourceCodeInput{
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
		CreatorIDs:           r.Creators,
		ContactPointIDs:      r.ContactPoints,
		CategoryIDs:          r.Categories,
		Identifiers:          identifiers,
	}
}

func (r sourceCodeRequest) input() (domain.SoftwareSourceCodeInput, error) {
	value, err := domain.NewSoftwareSourceCodeInput(r.rawInput())
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, fmt.Errorf("parse software source code input: %w", err)
	}
	return value, nil
}

func (r replaceDraftRequest) input() (domain.SoftwareSourceCodeDraftInput, error) {
	value, err := domain.NewSoftwareSourceCodeDraftInput(domain.SoftwareSourceCodeDraftInput{
		Revision: r.Revision,
		Metadata: r.Metadata.rawInput(),
	})
	if err != nil {
		return domain.SoftwareSourceCodeDraftInput{}, fmt.Errorf("parse software source code draft input: %w", err)
	}
	return value, nil
}

func (r rejectionRequest) input() (domain.SoftwareSourceCodeRejectionInput, error) {
	value, err := domain.NewSoftwareSourceCodeRejectionInput(domain.SoftwareSourceCodeRejectionInput{Note: r.Note})
	if err != nil {
		return domain.SoftwareSourceCodeRejectionInput{}, fmt.Errorf("parse software source code rejection input: %w", err)
	}
	return value, nil
}

func sourceCodeResponse(value domain.SoftwareSourceCodeVersion, includeContacts bool) sourceCodeRecord {
	return sourceCodeRecord{
		ResourceID: value.SoftwareSourceCodeID,
		VersionID:  value.VersionID,
		Revision:   value.Revision,
		Status:     value.Status,
		CreatedBy:  string(value.CreatedBy),
		GroupID:    value.Group.ID,
		GroupName:  value.Group.Name,
		CreatedAt:  value.CreatedAt,
		UpdatedAt:  value.UpdatedAt,
		Metadata:   sourceCodeMetadataDTO(value.Metadata, includeContacts),
	}
}

func sourceCodeMetadataDTO(value domain.SoftwareSourceCodeMetadata, includeContacts bool) sourceCodeMetadataResponse {
	result := sourceCodeMetadataResponse{
		ID:                   value.ID,
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
		Creators:             make([]resourceReference, 0, len(value.Creators)),
		Identifiers:          make([]identifierResponse, 0, len(value.Identifiers)),
		Categories:           make([]resourceReference, 0, len(value.Categories)),
		ContactPoints:        nil,
	}
	for _, creator := range value.Creators {
		result.Creators = append(result.Creators, resourceReferenceDTO(creator.ID, creator.Name))
	}
	for _, item := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierDTO(item))
	}
	for _, item := range value.Categories {
		result.Categories = append(result.Categories, resourceReferenceDTO(item.ID, item.Name))
	}
	if includeContacts {
		contacts := make([]uuid.UUID, 0, len(value.ContactPoints))
		for _, item := range value.ContactPoints {
			contacts = append(contacts, item.ID)
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

type identifierResponse struct {
	ID    uuid.UUID             `json:"id"`
	Type  domain.IdentifierType `json:"type"`
	Value string                `json:"value"`
}
