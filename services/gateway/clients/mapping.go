package clients

import (
	"fmt"
	"slices"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

// Optional scalar, passed straight through
func setIfPresent[T any](v *T, set func(T)) {
	if v != nil {
		set(*v)
	}
}

// Optional value that needs converting first (pointer -> struct conversion)
func setPtrConv[T, K any](v *T, conv func(T) K, set func(K)) {
	if v != nil {
		set(conv(*v))
	}
}

// Non-empty slice that needs converting first
func setSliceConv[T, K any](v []T, conv func([]T) K, set func(K)) {
	if len(v) > 0 {
		set(conv(v))
	}
}

func mapSlice[T, K any](values []T, f func(T) K) []K {
	if values == nil {
		return nil
	}
	result := make([]K, len(values))
	for i, value := range values {
		result[i] = f(value)
	}
	return result
}

func mapSliceErr[T, K any](values []T, f func(T) (K, error)) ([]K, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]K, len(values))
	for i, value := range values {
		mapped, err := f(value)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
	}
	return result, nil
}

func toClientDatasetDraftMetadata(metadata domain.DatasetDraftMetadata) client.DatasetDraftMetadata {
	result := client.DatasetDraftMetadata{ //nolint:exhaustruct_v5 // we set the values manually later
		SourceIri: metadata.SourceIRI,
		Title:     metadata.Title,
	}

	setIfPresent(metadata.Description, result.SetDescription)
	setIfPresent(metadata.Keywords, result.SetKeywords)
	setIfPresent(metadata.AccrualPeriodicity, result.SetAccrualPeriodicity)
	setIfPresent(metadata.QualityAnnotation, result.SetQualityAnnotation)
	setIfPresent(metadata.AccessRight, result.SetAccessRight)
	setIfPresent(metadata.SourceCreated, result.SetSourceCreated)
	setIfPresent(metadata.SourceUpdated, result.SetSourceUpdated)

	if metadata.ID != "" {
		result.SetId(metadata.ID)
	}

	setPtrConv(metadata.SpatialCoverage, toClientSpatialCoverage, result.SetSpatialCoverage)
	setPtrConv(metadata.TemporalCoverage, toClientTemporalCoverage, result.SetTemporalCoverage)

	setSliceConv(metadata.Distributions, toClientDistributionDrafts, result.SetDistributions)
	setSliceConv(metadata.PersistentIdentifiers, toClientIdentifierDrafts, result.SetPersistentIdentifiers)
	setSliceConv(metadata.ContactPoints, toClientResourceReferences, result.SetContactPoints)
	setSliceConv(metadata.Providers, toClientResourceReferences, result.SetProviders)
	setSliceConv(metadata.Categories, toClientResourceReferences, result.SetCategories)

	return result
}

func toClientCreateResourceRequest(resourceType string, metadata domain.DraftMetadata) (client.CreateResourceRequest, error) {
	switch metadata := metadata.(type) {
	case domain.DatasetDraftMetadata:
		return client.CreateDatasetResourceRequestAsCreateResourceRequest(&client.CreateDatasetResourceRequest{
			ResourceType: resourceType,
			Metadata:     toClientDatasetDraftMetadata(metadata),
		}), nil
	case domain.SoftwareApplicationDraftMetadata:
		return client.CreateSoftwareApplicationResourceRequestAsCreateResourceRequest(&client.CreateSoftwareApplicationResourceRequest{
			ResourceType: resourceType,
			Metadata:     toClientSoftwareApplicationDraftMetadata(metadata),
		}), nil
	case domain.SoftwareSourceCodeDraftMetadata:
		return client.CreateSoftwareSourceCodeResourceRequestAsCreateResourceRequest(&client.CreateSoftwareSourceCodeResourceRequest{
			ResourceType: resourceType,
			Metadata:     toClientSoftwareSourceCodeDraftMetadata(metadata),
		}), nil
	default:
		return client.CreateResourceRequest{}, fmt.Errorf("unsupported lifecycle metadata type %T", metadata)
	}
}

func toClientReplaceResourceDraftRequest(input domain.ReplaceResourceDraftInput) (client.ReplaceResourceDraftRequest, error) {
	switch metadata := input.Metadata.(type) {
	case domain.DatasetDraftMetadata:
		return client.ReplaceResourceDraftRequest{ //nolint:exhaustruct_v5 // union request sets one variant
			ReplaceDraftRequest: &client.ReplaceDraftRequest{
				ResourceType: "dataset",
				Revision:     input.Revision,
				Metadata:     toClientDatasetDraftMetadata(metadata),
			},
		}, nil
	case domain.SoftwareApplicationDraftMetadata:
		return client.ReplaceResourceDraftRequest{ //nolint:exhaustruct_v5 // union request sets one variant
			ReplaceSoftwareApplicationDraftRequest: &client.ReplaceSoftwareApplicationDraftRequest{
				ResourceType: "software-application",
				Revision:     input.Revision,
				Metadata:     toClientSoftwareApplicationDraftMetadata(metadata),
			},
		}, nil
	case domain.SoftwareSourceCodeDraftMetadata:
		return client.ReplaceResourceDraftRequest{ //nolint:exhaustruct_v5 // union request sets one variant
			ReplaceSoftwareSourceCodeDraftRequest: &client.ReplaceSoftwareSourceCodeDraftRequest{
				ResourceType: "software-source-code",
				Revision:     input.Revision,
				Metadata:     toClientSoftwareSourceCodeDraftMetadata(metadata),
			},
		}, nil
	default:
		return client.ReplaceResourceDraftRequest{}, fmt.Errorf("unsupported lifecycle metadata type %T", input.Metadata)
	}
}

func toClientSoftwareApplicationDraftMetadata(metadata domain.SoftwareApplicationDraftMetadata) client.SoftwareApplicationDraftMetadata {
	result := client.SoftwareApplicationDraftMetadata{ //nolint:exhaustruct_v5 // optional values are set below
		SourceIri: metadata.SourceIRI,
		Name:      metadata.Name,
	}

	setIfPresent(metadata.Description, result.SetDescription)
	setIfPresent(metadata.Keywords, result.SetKeywords)
	setIfPresent(metadata.License, result.SetLicense)
	setIfPresent(metadata.Version, result.SetVersion)
	setIfPresent(metadata.DownloadURL, result.SetDownloadUrl)
	setIfPresent(metadata.MainEntityOfPage, result.SetMainEntityOfPage)
	setIfPresent(metadata.Requirements, result.SetRequirements)
	setIfPresent(metadata.OperatingSystem, result.SetOperatingSystem)
	setSliceConv(metadata.Creators, toClientResourceReferences, result.SetCreators)
	setSliceConv(metadata.Identifiers, toClientIdentifierDrafts, result.SetIdentifiers)
	setSliceConv(metadata.Categories, toClientResourceReferences, result.SetCategories)
	setSliceConv(metadata.ContactPoints, toClientResourceReferences, result.SetContactPoints)

	return result
}

func toClientSoftwareSourceCodeDraftMetadata(metadata domain.SoftwareSourceCodeDraftMetadata) client.SoftwareSourceCodeDraftMetadata {
	result := client.SoftwareSourceCodeDraftMetadata{ //nolint:exhaustruct_v5 // optional values are set below
		SourceIri: metadata.SourceIRI,
		Name:      metadata.Name,
	}

	setIfPresent(metadata.Description, result.SetDescription)
	setIfPresent(metadata.Keywords, result.SetKeywords)
	setIfPresent(metadata.Repository, result.SetRepository)
	setIfPresent(metadata.ProgrammingLanguage, result.SetProgrammingLanguage)
	setIfPresent(metadata.License, result.SetLicense)
	setIfPresent(metadata.SoftwareRequirements, result.SetSoftwareRequirements)
	setIfPresent(metadata.RuntimePlatform, result.SetRuntimePlatform)
	setIfPresent(metadata.MainEntityOfPage, result.SetMainEntityOfPage)
	setIfPresent(metadata.Version, result.SetVersion)
	setSliceConv(metadata.Creators, toClientResourceReferences, result.SetCreators)
	setSliceConv(metadata.Identifiers, toClientIdentifierDrafts, result.SetIdentifiers)
	setSliceConv(metadata.Categories, toClientResourceReferences, result.SetCategories)
	setSliceConv(metadata.ContactPoints, toClientResourceReferences, result.SetContactPoints)

	return result
}

func toClientResourceReferences(values []domain.ResourceReference) []client.CanonicalReference {
	return mapSlice(values, func(value domain.ResourceReference) client.CanonicalReference {
		return client.CanonicalReference{Id: value.ID}
	})
}

func toClientIdentifierDrafts(values []domain.Identifier) []client.IdentifierDraft {
	return mapSlice(values, func(value domain.Identifier) client.IdentifierDraft {
		result := client.IdentifierDraft{ //nolint:exhaustruct_v5 // optional ID is set below
			SourceIri: value.SourceIRI,
			Type:      value.Type,
			Value:     value.Value,
		}
		result.SetId(value.ID)
		return result
	})
}

func toClientSpatialCoverage(value domain.SpatialCoverage) client.Spatial {
	return client.Spatial{
		Point:   toClientCoordinatePtr(value.Point),
		Polygon: mapSlice(value.Polygon, toClientCoordinate),
	}
}

func toClientCoordinatePtr(value *domain.Coordinate) *client.Coordinate {
	if value == nil {
		return nil
	}
	return &client.Coordinate{Latitude: float32(value.Latitude), Longitude: float32(value.Longitude)}
}

func toClientCoordinate(value domain.Coordinate) client.Coordinate {
	return client.Coordinate{Latitude: float32(value.Latitude), Longitude: float32(value.Longitude)}
}

func toClientTemporalCoverage(value domain.TemporalCoverage) client.Temporal {
	result := client.Temporal{} //nolint:exhaustruct_v5 // optional values are set below
	setIfPresent(value.Start, result.SetStart)
	setIfPresent(value.End, result.SetEnd)
	return result
}

func toClientDistributionDrafts(values []domain.DistributionDraft) []client.DistributionDraft {
	return mapSlice(values, toClientDistributionDraft)
}

func toClientDistributionDraft(value domain.DistributionDraft) client.DistributionDraft {
	result := client.DistributionDraft{ //nolint:exhaustruct_v5 // optional values are set below
		Title:       value.Title,
		Description: new(value.Description),
		Type:        client.DistributionType(value.Type),
	}
	result.SetId(value.ID)
	setIfPresent(value.License, result.SetLicense)
	setIfPresent(value.Issued, result.SetIssued)
	setIfPresent(value.Modified, result.SetModified)
	setIfPresent(value.ByteSize, result.SetByteSize)
	if value.DownloadableFile != nil {
		result.SetDownloadableFile(client.DownloadableFile{Url: value.DownloadableFile.URL, Format: value.DownloadableFile.Format})
	}
	return result
}

func toDomainResourceVersion(record *client.ResourceRecord) (domain.ResourceVersion, error) {
	if record == nil {
		return domain.ResourceVersion{}, fmt.Errorf("decode metadata-lifecycle resource record: response is empty")
	}

	result := domain.ResourceVersion{ //nolint:exhaustruct_v5 // metadata is set from response variant below
		ResourceID: record.ResourceId,
		VersionID:  record.VersionId,
		Revision:   record.Revision,
		Status:     string(record.Status),
		CreatedBy:  record.CreatedBy,
		Group:      domain.ResourceGroupReference{ID: record.Group.Id, Name: record.Group.Name},
	}

	switch {
	case record.Metadata.DatasetMetadata != nil:
		metadata := record.Metadata.DatasetMetadata
		result.Metadata = domain.DatasetResourceMetadata{
			ID:          metadata.Id,
			SourceIRI:   metadata.SourceIri,
			Title:       metadata.Title,
			Description: metadata.Description,
			Keywords:    metadata.Keywords,
			Status:      string(metadata.Status),
			Created:     metadata.Created.Format(time.RFC3339Nano),
			Updated:     metadata.Updated.Format(time.RFC3339Nano),
		}
	case record.Metadata.SoftwareApplicationMetadata != nil:
		metadata := record.Metadata.SoftwareApplicationMetadata
		result.Metadata = domain.SoftwareApplicationResourceMetadata{
			ID:          metadata.Id,
			SourceIRI:   metadata.SourceIri,
			Name:        metadata.Name,
			Description: metadata.Description,
			Keywords:    metadata.Keywords,
		}
	case record.Metadata.SoftwareSourceCodeMetadata != nil:
		metadata := record.Metadata.SoftwareSourceCodeMetadata
		result.Metadata = domain.SoftwareSourceCodeResourceMetadata{
			ID:          metadata.Id,
			SourceIRI:   metadata.SourceIri,
			Name:        metadata.Name,
			Description: metadata.Description,
			Keywords:    metadata.Keywords,
		}
	default:
		return domain.ResourceVersion{}, fmt.Errorf("decode metadata-lifecycle resource metadata: response is empty")
	}

	return result, nil
}

func toDomainContactPoint(value client.ContactPoint) domain.ContactPoint {
	return domain.ContactPoint{
		ID:        value.Id,
		SourceIRI: value.SourceIri,
		Email:     value.Email,
		Role:      value.Role,
	}
}

func toDomainCanonicalOrganization(value client.CanonicalOrganization) domain.CanonicalOrganization {
	return domain.CanonicalOrganization{
		ID:              value.Id,
		SourceIRI:       value.SourceIri,
		Name:            value.Name,
		Address:         toDomainOrganizationAddress(value.Address),
		LogoURL:         value.LogoUrl.Get(),
		PageURL:         value.PageUrl.Get(),
		Email:           value.Email.Get(),
		LEICode:         value.LeiCode.Get(),
		Parents:         slices.Clone(value.Parents),
		ContactPointIDs: slices.Clone(value.ContactPointIds),
		Identifiers:     mapSlice(value.Identifiers, toDomainIdentifier),
	}
}

func toDomainIdentifier(value client.Identifier) domain.Identifier {
	return domain.Identifier{
		ID:        value.Id,
		SourceIRI: value.SourceIri,
		Type:      value.Type,
		Value:     value.Value,
	}
}

func toDomainCategory(value client.CanonicalCategory) domain.Category {
	return domain.Category{
		ID:               value.Id,
		GroupID:          value.GroupId,
		ParentCategoryID: value.ParentCategoryId.Get(),
		SourceIRI:        value.SourceIri,
		Name:             value.Name,
	}
}

func toClientCreateContactPointRequest(value domain.CreateContactPointInput) client.CreateContactPointRequest {
	return client.CreateContactPointRequest{SourceIri: value.SourceIRI, Email: value.Email, Role: value.Role}
}

func toClientReplaceContactPointRequest(value domain.ReplaceContactPointInput) client.ReplaceContactPointRequest {
	return client.ReplaceContactPointRequest{Email: value.Email, Role: value.Role}
}

func toClientOrganizationIdentifierInput(value domain.OrganizationIdentifierInput) client.CreateOrganizationIdentifierRequest {
	return client.CreateOrganizationIdentifierRequest{SourceIri: value.SourceIRI, Type: value.Type, Value: value.Value}
}

func toClientOrganizationIdentifierInputs(values []domain.OrganizationIdentifierInput) []client.CreateOrganizationIdentifierRequest {
	return mapSlice(values, toClientOrganizationIdentifierInput)
}

func toClientCreateCanonicalOrganizationRequest(value domain.CreateOrganizationInput) client.CreateCanonicalOrganizationRequest {
	result := client.CreateCanonicalOrganizationRequest{ //nolint:exhaustruct_v5 // optional values are set below
		SourceIri:       value.SourceIRI,
		Name:            value.Name,
		Identifiers:     toClientOrganizationIdentifierInputs(value.Identifiers),
		Parents:         value.Parents,
		ContactPointIds: value.ContactPointIDs,
	}
	setPtrConv(value.Address, toClientCreateOrganizationAddress, func(address *client.CreateCanonicalOrganizationRequestAddress) {
		result.Address = address
	})
	setNullable(&result.LogoUrl, value.LogoURL)
	setNullable(&result.PageUrl, value.PageURL)
	setNullable(&result.Email, value.Email)
	setNullable(&result.LeiCode, value.LEICode)
	return result
}

func toClientReplaceCanonicalOrganizationRequest(value domain.ReplaceOrganizationInput) client.ReplaceCanonicalOrganizationRequest {
	result := client.ReplaceCanonicalOrganizationRequest{ //nolint:exhaustruct_v5 // optional values are set below
		Name:            value.Name,
		Address:         toClientOrganizationAddress(value.Address),
		Identifiers:     toClientOrganizationIdentifierInputs(value.Identifiers),
		Parents:         value.Parents,
		ContactPointIds: value.ContactPointIDs,
	}
	setNullable(&result.LogoUrl, value.LogoURL)
	setNullable(&result.PageUrl, value.PageURL)
	setNullable(&result.Email, value.Email)
	setNullable(&result.LeiCode, value.LEICode)
	return result
}

func toClientCreateOrganizationAddress(value domain.OrganizationAddress) *client.CreateCanonicalOrganizationRequestAddress {
	result := &client.CreateCanonicalOrganizationRequestAddress{} //nolint:exhaustruct_v5 // optional values are set below
	setNullable(&result.Street, value.Street)
	setNullable(&result.Locality, value.Locality)
	setNullable(&result.PostalCode, value.PostalCode)
	setNullable(&result.Country, value.Country)
	return result
}

func toClientOrganizationAddress(value domain.OrganizationAddress) client.CanonicalOrganizationAddress {
	result := client.CanonicalOrganizationAddress{} //nolint:exhaustruct_v5 // optional values are set below
	setNullable(&result.Street, value.Street)
	setNullable(&result.Locality, value.Locality)
	setNullable(&result.PostalCode, value.PostalCode)
	setNullable(&result.Country, value.Country)
	return result
}

func toDomainOrganizationAddress(value client.CanonicalOrganizationAddress) domain.OrganizationAddress {
	return domain.OrganizationAddress{
		Street:     value.Street.Get(),
		Locality:   value.Locality.Get(),
		PostalCode: value.PostalCode.Get(),
		Country:    value.Country.Get(),
	}
}

func setNullable(value *client.NullableString, input *string) {
	if input == nil {
		value.Set(nil)
		return
	}
	value.Set(input)
}

func toClientCreateGroupRequest(value domain.CreateGroupInput) client.CreateGroupRequest {
	result := client.CreateGroupRequest{ //nolint:exhaustruct_v5 // optional values are set below
		SourceIri: value.SourceIRI,
		Name:      value.Name,
		Position:  value.Position,
		Concept:   value.Concept,
	}
	setNullable(&result.Description, value.Description)
	setNullable(&result.Label, value.Label)
	setNullable(&result.Logo, value.Logo)
	setNullable(&result.Page, value.Page)
	setNullable(&result.Color, value.Color)
	return result
}

func toClientReplaceGroupRequest(value domain.ReplaceGroupInput) client.ReplaceGroupRequest {
	result := client.ReplaceGroupRequest{ //nolint:exhaustruct_v5 // optional values are set below
		Name:     value.Name,
		Position: value.Position,
		Concept:  value.Concept,
	}
	setNullable(&result.Description, value.Description)
	setNullable(&result.Label, value.Label)
	setNullable(&result.Logo, value.Logo)
	setNullable(&result.Page, value.Page)
	setNullable(&result.Color, value.Color)
	return result
}

func toClientCreateCategoryRequest(value domain.CreateCategoryInput) client.CreateCategoryRequest {
	return client.CreateCategoryRequest{SourceIri: value.SourceIRI, Name: value.Name, ParentCategoryId: value.ParentCategoryID}
}

func toClientReplaceCategoryRequest(value domain.ReplaceCategoryInput) client.ReplaceCategoryRequest {
	result := client.ReplaceCategoryRequest{ //nolint:exhaustruct_v5 // optional parent ID is set below
		Name: value.Name,
	}
	setNullable(&result.ParentCategoryId, value.ParentCategoryID)
	return result
}

func toClientResourceTypes(values []string) []client.ResourceType {
	return mapSlice(values, func(value string) client.ResourceType { return client.ResourceType(value) })
}

func toDomainDiscoveryItem(value client.DiscoveryItem) domain.DiscoveryItem {
	return domain.DiscoveryItem{
		Type:          value.Type,
		ResourceID:    value.ResourceId,
		VersionID:     value.VersionId,
		SourceIRI:     value.SourceIri,
		Name:          value.Name,
		Status:        string(value.Status),
		Group:         toDomainDiscoveryReference(value.Group),
		Providers:     toDomainDiscoveryReferences(value.Providers),
		Creators:      toDomainDiscoveryReferences(value.Creators),
		Categories:    toDomainDiscoveryReferences(value.Categories),
		Distributions: toDomainDiscoveryReferences(value.Distributions),
	}
}

func toDomainDiscoveryReferences(values []client.DiscoveryReference) []domain.DiscoveryReference {
	return mapSlice(values, toDomainDiscoveryReference)
}

func toDomainDiscoveryReference(value client.DiscoveryReference) domain.DiscoveryReference {
	return domain.DiscoveryReference{ID: value.Id, Name: value.Name}
}

func toDomainUser(value *client.User) (domain.User, error) {
	if value == nil {
		return domain.User{}, fmt.Errorf("decode metadata-lifecycle user: response is empty")
	}

	memberships := mapSlice(value.Memberships, func(membership client.UserMembership) domain.Membership {
		return domain.Membership{
			Group: domain.Group{ //nolint:exhaustruct_v5 // membership reference only carries ID and name
				ID:   membership.Group.Id,
				Name: membership.Group.Name,
			},
			Roles:        slices.Clone(membership.Roles),
			Status:       membership.Status,
			Permissions:  slices.Clone(membership.Permissions),
			RoleRequests: toDomainRoleRequests(membership.RoleRequests),
			CreatedAt:    membership.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:    membership.UpdatedAt.Format(time.RFC3339Nano),
		}
	})

	return domain.User{
		ID:                value.Id,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            value.Status,
		GlobalRoles:       slices.Clone(value.GlobalRoles),
		GlobalPermissions: slices.Clone(value.GlobalPermissions),
		CreatedAt:         value.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         value.UpdatedAt.Format(time.RFC3339Nano),
		Memberships:       memberships,
	}, nil
}

func toDomainRoleRequests(values []client.RoleRequest) []domain.RoleRequest {
	return mapSlice(values, toDomainRoleRequest)
}

func toDomainRoleRequest(value client.RoleRequest) domain.RoleRequest {
	return domain.RoleRequest{
		ID:          value.Id,
		Role:        value.Role,
		Status:      value.Status,
		RequestedAt: value.RequestedAt,
	}
}

func toDomainGroup(value *client.Group) (domain.Group, error) {
	if value == nil {
		return domain.Group{}, fmt.Errorf("decode metadata-lifecycle group: response is empty")
	}

	return domain.Group{
		ID:          value.Id,
		SourceIRI:   value.SourceIri,
		Name:        value.Name,
		Description: value.Description.Get(),
		Label:       value.Label.Get(),
		Logo:        value.Logo.Get(),
		Page:        value.Page.Get(),
		Color:       value.Color.Get(),
		Position:    value.Position,
		Concept:     value.Concept,
	}, nil
}

func toDomainGroups(values []client.Group) ([]domain.Group, error) {
	return mapSliceErr(values, func(value client.Group) (domain.Group, error) {
		return toDomainGroup(new(value))
	})
}

func toDomainGroupMembership(value *client.GroupMembership) (domain.GroupMembership, error) {
	if value == nil {
		return domain.GroupMembership{}, fmt.Errorf("decode metadata-lifecycle membership: response is empty")
	}

	return domain.GroupMembership{
		User: domain.MembershipUser{
			ID:         value.User.Id,
			GivenName:  value.User.GivenName,
			FamilyName: value.User.FamilyName,
			Email:      value.User.Email,
		},
		Status:       value.Status,
		Roles:        slices.Clone(value.Roles),
		Permissions:  slices.Clone(value.Permissions),
		RoleRequests: toDomainRoleRequests(value.RoleRequests),
		CreatedAt:    value.CreatedAt,
		UpdatedAt:    value.UpdatedAt,
	}, nil
}

func toDomainGroupMembershipValue(value client.GroupMembership) (domain.GroupMembership, error) {
	return toDomainGroupMembership(new(value))
}

func toDomainUserSummary(value client.UserSummary) domain.UserSummary {
	return domain.UserSummary{ //nolint:exhaustruct_v5 // summary response has no timestamps
		ID:                value.Id,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            value.Status,
		GlobalRoles:       slices.Clone(value.GlobalRoles),
		GlobalPermissions: slices.Clone(value.GlobalPermissions),
	}
}

func toDomainUserSummaries(values []client.UserSummary) []domain.UserSummary {
	return mapSlice(values, toDomainUserSummary)
}

func toDomainImportDiagnostic(value client.ImportDiagnostic) domain.ImportDiagnostic {
	return domain.ImportDiagnostic(value)
}

func toDomainImportResult(value client.ImportResult) domain.ImportResult {
	return domain.ImportResult{
		Groups:               value.Groups,
		Categories:           value.Categories,
		Organizations:        value.Organizations,
		ContactPoints:        value.ContactPoints,
		Datasets:             value.Datasets,
		SoftwareApplications: value.SoftwareApplications,
		SoftwareSourceCodes:  value.SoftwareSourceCodes,
		Warnings:             mapSlice(value.Warnings, toDomainImportDiagnostic),
	}
}

func toClientRejectRequest(value domain.RejectResourceInput) client.RejectRequest {
	return client.RejectRequest{Note: value.Note}
}

func toClientCopyResourceRequest(groupID string, value domain.CopyResourceInput) client.CopyResourceRequest {
	return client.CopyResourceRequest{TargetGroupId: groupID, SourceIri: value.SourceIRI}
}

func toClientCreateMembershipRequest(roles []string) client.CreateMembershipRequest {
	return client.CreateMembershipRequest{Roles: roles}
}

func toClientMembershipDecisionRequest(status string) client.MembershipDecisionRequest {
	return client.MembershipDecisionRequest{Status: status}
}

func toClientCreateRoleRequest(role string) client.CreateRoleRequest {
	return client.CreateRoleRequest{Role: role}
}

func toClientRoleRequestDecisionRequest(status string) client.RoleRequestDecisionRequest {
	return client.RoleRequestDecisionRequest{Status: status}
}

func toClientBlockUserRequest(reason string) client.BlockUserRequest {
	return client.BlockUserRequest{Reason: reason}
}

func toClientCreateUserRequest(identity ActorIdentity) client.CreateUserRequest {
	return client.CreateUserRequest{
		Id:         identity.AuthIdentifier,
		Email:      identity.Email,
		GivenName:  identity.GivenName,
		FamilyName: identity.FamilyName,
	}
}
