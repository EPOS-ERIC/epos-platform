//nolint:dupl,exhaustruct_v5,wrapcheck // TODO
package handlers

import (
	"errors"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func mapGroup(v domain.Group) (gatewayapi.Group, error) {
	id, err := parseUUID("group.id", v.ID)
	if err != nil {
		return gatewayapi.Group{}, err
	}
	return gatewayapi.Group{
		ID:   id,
		Name: v.Name,
		Description: gatewayapi.OptNilString{
			Value: value(v.Description),
			Set:   v.Description != nil,
		},
		Label: gatewayapi.OptNilString{
			Value: value(v.Label),
			Set:   v.Label != nil,
		},
		Logo: gatewayapi.OptNilString{
			Value: value(v.Logo),
			Set:   v.Logo != nil,
		},
		Page: gatewayapi.OptNilString{
			Value: value(v.Page),
			Set:   v.Page != nil,
		},
		Color: gatewayapi.OptNilString{
			Value: value(v.Color),
			Set:   v.Color != nil,
		},
		Position: v.Position,
		Concept:  gatewayapi.GroupConcept(v.Concept),
	}, nil
}

func mapCategory(v domain.Category) (gatewayapi.Category, error) {
	id, err := parseUUID("category.id", v.ID)
	if err != nil {
		return gatewayapi.Category{}, err
	}
	groupID, err := parseUUID("category.group_id", v.GroupID)
	if err != nil {
		return gatewayapi.Category{}, err
	}
	r := gatewayapi.Category{ID: id, GroupID: groupID, Name: v.Name}
	if v.ParentCategoryID != nil {
		parentID, err := parseUUID("category.parent_category_id", *v.ParentCategoryID)
		if err != nil {
			return gatewayapi.Category{}, err
		}
		r.ParentCategoryID.SetTo(parentID)
	}
	return r, nil
}

func mapMembershipUser(v domain.MembershipUser) gatewayapi.MembershipUser {
	return gatewayapi.MembershipUser{
		ID:         v.ID,
		GivenName:  v.GivenName,
		FamilyName: v.FamilyName,
		Email:      v.Email,
	}
}

func mapRole(v domain.RoleRequest) (gatewayapi.RoleRequest, error) {
	id, err := parseUUID("role_request.id", v.ID)
	if err != nil {
		return gatewayapi.RoleRequest{}, err
	}
	return gatewayapi.RoleRequest{
		ID:          id,
		Role:        gatewayapi.RoleRequestRole(v.Role),
		Status:      gatewayapi.RoleRequestStatus(v.Status),
		RequestedAt: v.RequestedAt,
	}, nil
}

func mapMembership(v domain.GroupMembership) (gatewayapi.GroupMembership, error) {
	r := gatewayapi.GroupMembership{
		User:        mapMembershipUser(v.User),
		Status:      gatewayapi.MembershipStatus(v.Status),
		Permissions: v.Permissions,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
	for _, role := range v.Roles {
		r.Roles = append(r.Roles, gatewayapi.MembershipRole(role))
	}
	for _, role := range v.RoleRequests {
		mapped, err := mapRole(role)
		if err != nil {
			return gatewayapi.GroupMembership{}, err
		}
		r.RoleRequests = append(r.RoleRequests, mapped)
	}
	return r, nil
}

func mapContact(v domain.ContactPoint) (gatewayapi.ContactPoint, error) {
	id, err := parseUUID("contact_point.id", v.ID)
	if err != nil {
		return gatewayapi.ContactPoint{}, err
	}
	return gatewayapi.ContactPoint{
		ID:    id,
		Email: v.Email,
		Role:  v.Role,
	}, nil
}

func mapReference(v domain.DiscoveryReference) gatewayapi.Reference {
	return gatewayapi.Reference{
		ID:   v.ID,
		Name: v.Name,
	}
}

func mapDiscovery(v domain.DiscoveryItem) (gatewayapi.DiscoveryItem, error) {
	resourceID, err := parseUUID("discovery.resource_id", v.ResourceID)
	if err != nil {
		return gatewayapi.DiscoveryItem{}, err
	}
	versionID, err := parseUUID("discovery.version_id", v.VersionID)
	if err != nil {
		return gatewayapi.DiscoveryItem{}, err
	}
	r := gatewayapi.DiscoveryItem{
		Type:       gatewayapi.DiscoveryResourceType(v.Type),
		ResourceID: resourceID,
		VersionID:  versionID,
		Name:       v.Name,
		Status:     gatewayapi.DiscoveryItemStatus(v.Status),
		UpdatedAt:  v.UpdatedAt,
		Concept:    gatewayapi.DiscoveryItemConcept(v.Concept),
		Group:      mapReference(v.Group),
	}
	if v.CreatedBy != nil {
		r.CreatedBy.SetTo(mapReference(*v.CreatedBy))
	}
	for _, x := range v.Providers {
		r.Providers = append(r.Providers, mapReference(x))
	}
	for _, x := range v.Creators {
		r.Creators = append(r.Creators, mapReference(x))
	}
	for _, x := range v.Categories {
		r.Categories = append(r.Categories, mapReference(x))
	}
	for _, x := range v.Distributions {
		r.Distributions = append(r.Distributions, mapReference(x))
	}
	return r, nil
}

// TODO: delete this, we don't need to know exactly which field has an error while parsing to uuid, inline directly uuid.Parse
func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid upstream %s: %w", field, err)
	}
	return id, nil
}

// TODO: make this toOptString instead
func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func toOptString(v *string) gatewayapi.OptNilString {
	if v == nil {
		return gatewayapi.OptNilString{}
	}
	return gatewayapi.NewOptNilString(*v)
}

func toOptTime(v *time.Time) gatewayapi.OptNilDateTime {
	if v == nil {
		return gatewayapi.OptNilDateTime{}
	}
	return gatewayapi.NewOptNilDateTime(*v)
}

func toOptInt64(v *int64) gatewayapi.OptNilInt64 {
	if v == nil {
		return gatewayapi.OptNilInt64{}
	}
	return gatewayapi.NewOptNilInt64(*v)
}

func mapTemporalCoverage(v *domain.TemporalCoverage) gatewayapi.OptTemporalCoverage {
	if v == nil {
		return gatewayapi.OptTemporalCoverage{}
	}
	return gatewayapi.NewOptTemporalCoverage(gatewayapi.TemporalCoverage{
		Start: toOptTime(v.Start),
		End:   toOptTime(v.End),
	})
}

func mapResourceReference(v domain.ResourceVersionReference) (gatewayapi.ResourceVersionReference, error) {
	id, err := parseUUID("resource_reference.id", v.ID)
	if err != nil {
		return gatewayapi.ResourceVersionReference{}, err
	}
	return gatewayapi.ResourceVersionReference{
		ID:   id,
		Name: v.Name,
	}, nil
}

func mapResourceIdentifier(v domain.Identifier) (gatewayapi.ResourceIdentifier, error) {
	id, err := parseUUID("identifier.id", v.ID)
	if err != nil {
		return gatewayapi.ResourceIdentifier{}, err
	}
	return gatewayapi.ResourceIdentifier{
		ID:    id,
		Type:  v.Type,
		Value: v.Value,
	}, nil
}

func mapDocumentation(v *domain.Documentation) gatewayapi.OptDocumentation {
	if v == nil {
		return gatewayapi.OptDocumentation{}
	}
	r := gatewayapi.Documentation{URI: gatewayapi.NonBlankString(v.URI)}
	if v.Title != nil {
		r.Title.SetTo(*v.Title)
	}
	if v.Description != nil {
		r.Description.SetTo(*v.Description)
	}
	return gatewayapi.NewOptDocumentation(r)
}

func toNilString(v *string) gatewayapi.NilString {
	if v == nil {
		return gatewayapi.NilString{Null: true}
	}
	return gatewayapi.NewNilString(*v)
}

func toNilBool(v *bool) gatewayapi.NilBool {
	if v == nil {
		return gatewayapi.NilBool{Null: true}
	}
	return gatewayapi.NewNilBool(*v)
}

func toNilInt64(v *int64) gatewayapi.NilInt64 {
	if v == nil {
		return gatewayapi.NilInt64{Null: true}
	}
	return gatewayapi.NewNilInt64(*v)
}

func toNilFloat64(v *float64) gatewayapi.NilFloat64 {
	if v == nil {
		return gatewayapi.NilFloat64{Null: true}
	}
	return gatewayapi.NewNilFloat64(*v)
}

func toNilTime(v *time.Time) gatewayapi.NilDateTime {
	if v == nil {
		return gatewayapi.NilDateTime{Null: true}
	}
	return gatewayapi.NewNilDateTime(*v)
}

func mapParameterMetadata(v domain.ParameterMetadata) (gatewayapi.ParameterMetadata, error) {
	id, err := parseUUID("parameter.metadata.id", v.ID)
	if err != nil {
		return gatewayapi.ParameterMetadata{}, err
	}
	return gatewayapi.ParameterMetadata{
		ID:             id,
		Variable:       v.Variable,
		Property:       v.Property,
		Label:          v.Label,
		Required:       v.Required,
		MultipleValues: v.MultipleValues,
		ReadOnly:       v.ReadOnly,
	}, nil
}

func mapParameter(v domain.Parameter) (gatewayapi.Parameter, error) {
	switch x := v.(type) {
	case domain.StringParameter:
		metadata, err := mapParameterMetadata(x.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewStringParameterParameter(gatewayapi.StringParameter{
			Metadata:      metadata,
			Default:       toNilString(x.Default),
			AllowedValues: x.AllowedValues,
			Pattern:       x.Pattern,
		}), nil
	case domain.BooleanParameter:
		metadata, err := mapParameterMetadata(x.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewBooleanParameterParameter(gatewayapi.BooleanParameter{
			Metadata: metadata,
			Default:  toNilBool(x.Default),
		}), nil
	case domain.IntegerParameter:
		metadata, err := mapParameterMetadata(x.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewIntegerParameterParameter(gatewayapi.IntegerParameter{
			Metadata:      metadata,
			Minimum:       toNilInt64(x.Minimum),
			Maximum:       toNilInt64(x.Maximum),
			Default:       toNilInt64(x.Default),
			AllowedValues: x.AllowedValues,
		}), nil
	case domain.FloatParameter:
		metadata, err := mapParameterMetadata(x.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewFloatParameterParameter(gatewayapi.FloatParameter{
			Metadata:      metadata,
			Minimum:       toNilFloat64(x.Minimum),
			Maximum:       toNilFloat64(x.Maximum),
			Default:       toNilFloat64(x.Default),
			AllowedValues: x.AllowedValues,
		}), nil
	case domain.DateTimeParameter:
		metadata, err := mapParameterMetadata(x.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewDateTimeParameterParameter(gatewayapi.DateTimeParameter{
			Metadata: metadata,
			Minimum:  toNilTime(x.Minimum),
			Maximum:  toNilTime(x.Maximum),
			Default:  toNilTime(x.Default),
			Pattern:  x.Pattern,
		}), nil
	default:
		return gatewayapi.Parameter{}, fmt.Errorf("unsupported upstream parameter type %T", v)
	}
}

func mapWebService(v domain.WebService) (gatewayapi.WebService, error) {
	id, err := parseUUID("web_service.id", v.ID)
	if err != nil {
		return gatewayapi.WebService{}, err
	}
	r := gatewayapi.WebService{
		ID:               id,
		Name:             v.Name,
		Description:      v.Description,
		TemplateURL:      v.TemplateURL,
		Format:           v.Format,
		SpatialCoverage:  toOptString(v.SpatialCoverage),
		TemporalCoverage: mapTemporalCoverage(v.TemporalCoverage),
	}
	for _, x := range v.Parameters {
		parameter, err := mapParameter(x)
		if err != nil {
			return gatewayapi.WebService{}, err
		}
		r.Parameters = append(r.Parameters, parameter)
	}
	if v.Provider != nil {
		provider, err := mapResourceReference(*v.Provider)
		if err != nil {
			return gatewayapi.WebService{}, err
		}
		r.Provider.SetTo(provider)
	}
	r.Documentation = mapDocumentation(v.Documentation)
	for _, x := range v.ContactPoints {
		contactPointID, err := parseUUID("web_service.contact_point_id", x)
		if err != nil {
			return gatewayapi.WebService{}, err
		}
		r.ContactPoints = append(r.ContactPoints, contactPointID)
	}
	return r, nil
}

func mapDistribution(v domain.Distribution) (gatewayapi.Distribution, error) {
	switch x := v.(type) {
	case domain.WebServiceDistribution:
		id, err := parseUUID("web_service_distribution.id", x.ID)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		webService, err := mapWebService(x.WebService)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		return gatewayapi.NewWebServiceDistributionDistribution(gatewayapi.WebServiceDistribution{
			ID:          id,
			Title:       x.Title,
			Description: x.Description,
			License:     toOptString(x.License),
			Type:        gatewayapi.WebServiceDistributionType(x.Type),
			Issued:      toOptTime(x.Issued),
			Modified:    toOptTime(x.Modified),
			ByteSize:    toOptInt64(x.ByteSize),
			WebService:  webService,
		}), nil
	case domain.DownloadableFileDistribution:
		id, err := parseUUID("downloadable_file_distribution.id", x.ID)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		return gatewayapi.NewDownloadableFileDistributionDistribution(gatewayapi.DownloadableFileDistribution{
			ID:          id,
			Title:       x.Title,
			Description: x.Description,
			License:     toOptString(x.License),
			Type:        gatewayapi.DownloadableFileDistributionType(x.Type),
			Issued:      toOptTime(x.Issued),
			Modified:    toOptTime(x.Modified),
			ByteSize:    toOptInt64(x.ByteSize),
			DownloadableFile: gatewayapi.DownloadableFile{
				URL:    gatewayapi.NonBlankString(x.DownloadableFile.URL),
				Format: gatewayapi.NonBlankString(x.DownloadableFile.Format),
			},
		}), nil
	default:
		return gatewayapi.Distribution{}, fmt.Errorf("unsupported upstream distribution type %T", v)
	}
}

func mapResourceMetadata(v domain.ResourceVersionMetadata) (gatewayapi.ResourceVersionMetadata, bool, error) {
	var r gatewayapi.ResourceVersionMetadata
	switch x := v.(type) {
	case domain.DatasetResourceMetadata:
		id, err := parseUUID("dataset.id", x.ID)
		if err != nil {
			return gatewayapi.ResourceVersionMetadata{}, false, err
		}
		m := gatewayapi.DatasetResourceMetadata{
			ResourceType:       gatewayapi.DatasetResourceMetadataResourceTypeDataset,
			ID:                 id,
			Title:              x.Title,
			Description:        x.Description,
			Keywords:           x.Keywords,
			AccrualPeriodicity: toOptString(x.AccrualPeriodicity),
			QualityAnnotation:  toOptString(x.QualityAnnotation),
			AccessRight:        toOptString(x.AccessRight),
			SpatialCoverage:    toOptString(x.SpatialCoverage),
			TemporalCoverage:   mapTemporalCoverage(x.TemporalCoverage),
			Created:            toOptTime(x.SourceCreated),
			Updated:            toOptTime(x.SourceUpdated),
		}
		for _, d := range x.Distributions {
			mapped, err := mapDistribution(d)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Distributions = append(m.Distributions, mapped)
		}
		for _, id := range x.PersistentIdentifiers {
			mapped, err := mapResourceIdentifier(id)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.PersistentIdentifiers = append(m.PersistentIdentifiers, mapped)
		}
		for _, id := range x.ContactPoints {
			mapped, err := parseUUID("dataset.contact_point_id", id)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.ContactPoints = append(m.ContactPoints, mapped)
		}
		for _, ref := range x.Providers {
			mapped, err := mapResourceReference(ref)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Providers = append(m.Providers, mapped)
		}
		for _, ref := range x.Categories {
			mapped, err := mapResourceReference(ref)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Categories = append(m.Categories, mapped)
		}
		r.SetDatasetResourceMetadata(m)
	case domain.SoftwareApplicationResourceMetadata:
		id, err := parseUUID("software_application.id", x.ID)
		if err != nil {
			return gatewayapi.ResourceVersionMetadata{}, false, err
		}
		m := gatewayapi.SoftwareApplicationResourceMetadata{
			ResourceType:     gatewayapi.SoftwareApplicationResourceMetadataResourceTypeSoftwareApplication,
			ID:               id,
			Name:             x.Name,
			Description:      x.Description,
			MainEntityOfPage: toOptString(x.MainEntityOfPage),
			License:          toOptString(x.License),
			Version:          toOptString(x.Version),
			Keywords:         x.Keywords,
			DownloadURL:      toOptString(x.DownloadURL),
			Requirements:     toOptString(x.Requirements),
			OperatingSystem:  toOptString(x.OperatingSystem),
		}
		for _, ref := range x.Creators {
			mapped, err := mapResourceReference(ref)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Creators = append(m.Creators, mapped)
		}
		for _, id := range x.Identifiers {
			mapped, err := mapResourceIdentifier(id)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Identifiers = append(m.Identifiers, mapped)
		}
		for _, id := range x.ContactPoints {
			mapped, err := parseUUID("software_application.contact_point_id", id)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.ContactPoints = append(m.ContactPoints, mapped)
		}
		for _, ref := range x.Categories {
			mapped, err := mapResourceReference(ref)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Categories = append(m.Categories, mapped)
		}
		r.SetSoftwareApplicationResourceMetadata(m)
	case domain.SoftwareSourceCodeResourceMetadata:
		id, err := parseUUID("software_source_code.id", x.ID)
		if err != nil {
			return gatewayapi.ResourceVersionMetadata{}, false, err
		}
		m := gatewayapi.SoftwareSourceCodeResourceMetadata{
			ResourceType:         gatewayapi.SoftwareSourceCodeResourceMetadataResourceTypeSoftwareSourceCode,
			ID:                   id,
			Name:                 x.Name,
			Description:          x.Description,
			ProgrammingLanguage:  toOptString(x.ProgrammingLanguage),
			Keywords:             x.Keywords,
			Repository:           toOptString(x.Repository),
			License:              toOptString(x.License),
			SoftwareRequirements: toOptString(x.SoftwareRequirements),
			RuntimePlatform:      toOptString(x.RuntimePlatform),
			MainEntityOfPage:     toOptString(x.MainEntityOfPage),
			Version:              toOptString(x.Version),
		}
		for _, ref := range x.Creators {
			mapped, err := mapResourceReference(ref)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Creators = append(m.Creators, mapped)
		}
		for _, id := range x.Identifiers {
			mapped, err := mapResourceIdentifier(id)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Identifiers = append(m.Identifiers, mapped)
		}
		for _, id := range x.ContactPoints {
			mapped, err := parseUUID("software_source_code.contact_point_id", id)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.ContactPoints = append(m.ContactPoints, mapped)
		}
		for _, ref := range x.Categories {
			mapped, err := mapResourceReference(ref)
			if err != nil {
				return gatewayapi.ResourceVersionMetadata{}, false, err
			}
			m.Categories = append(m.Categories, mapped)
		}
		r.SetSoftwareSourceCodeResourceMetadata(m)
	default:
		return gatewayapi.ResourceVersionMetadata{}, false, nil
	}
	return r, true, nil
}

func mapResourceVersion(v domain.ResourceVersion) (gatewayapi.ResourceVersion, error) {
	resourceID, err := parseUUID("resource_version.resource_id", v.ResourceID)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	versionID, err := parseUUID("resource_version.version_id", v.VersionID)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	groupID, err := parseUUID("resource_version.group_id", v.GroupID)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	r := gatewayapi.ResourceVersion{
		ResourceID: resourceID,
		VersionID:  versionID,
		Revision:   v.Revision,
		Status:     v.Status,
		CreatedBy:  v.CreatedBy,
		GroupID:    groupID,
		GroupName:  v.GroupName,
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
	}
	metadata, ok, err := mapResourceMetadata(v.Metadata)
	if err != nil {
		return gatewayapi.ResourceVersion{}, err
	}
	if !ok {
		return gatewayapi.ResourceVersion{}, errors.New("resource version metadata is nil or unsupported")
	}
	r.Metadata = metadata
	return r, nil
}

func mapOrganization(v domain.CanonicalOrganization) (gatewayapi.CanonicalOrganization, error) {
	id, err := parseUUID("organization.id", v.ID)
	if err != nil {
		return gatewayapi.CanonicalOrganization{}, err
	}
	r := gatewayapi.CanonicalOrganization{
		ID:      id,
		Name:    v.Name,
		Address: gatewayapi.OrganizationAddress{},
	}
	if v.Address.Street != nil {
		r.Address.Street.SetTo(*v.Address.Street)
	}
	if v.Address.Locality != nil {
		r.Address.Locality.SetTo(*v.Address.Locality)
	}
	if v.Address.PostalCode != nil {
		r.Address.PostalCode.SetTo(*v.Address.PostalCode)
	}
	if v.Address.Country != nil {
		r.Address.Country.SetTo(*v.Address.Country)
	}
	if v.LogoURL != nil {
		r.LogoURL.SetTo(*v.LogoURL)
	}
	if v.PageURL != nil {
		r.PageURL.SetTo(*v.PageURL)
	}
	if v.Email != nil {
		r.Email.SetTo(*v.Email)
	}
	if v.LEICode != nil {
		r.LeiCode.SetTo(*v.LEICode)
	}
	for _, x := range v.Identifiers {
		identifierID, err := parseUUID("organization.identifier.id", x.ID)
		if err != nil {
			return gatewayapi.CanonicalOrganization{}, err
		}
		r.Identifiers = append(r.Identifiers, gatewayapi.OrganizationIdentifier{
			ID:    identifierID,
			Type:  x.Type,
			Value: x.Value,
		})
	}
	for _, x := range v.Parents {
		parentID, err := parseUUID("organization.parent_id", x)
		if err != nil {
			return gatewayapi.CanonicalOrganization{}, err
		}
		r.Parents = append(r.Parents, parentID)
	}
	for _, x := range v.ContactPointIDs {
		contactPointID, err := parseUUID("organization.contact_point_id", x)
		if err != nil {
			return gatewayapi.CanonicalOrganization{}, err
		}
		r.ContactPointIds = append(r.ContactPointIds, contactPointID)
	}
	return r, nil
}
