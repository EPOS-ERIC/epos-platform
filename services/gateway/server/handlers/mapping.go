package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func mapPtr[T, K any](v *T, f func(T) K) *K {
	if v == nil {
		return nil
	}
	return new(f(*v))
}

func mapSlice[T, K any](in []T, f func(T) K) []K {
	if in == nil {
		return nil
	}
	out := make([]K, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func toDomainDatasetDraftMetadata(value api.DatasetDraftMetadata) domain.DatasetDraftMetadata {
	return domain.DatasetDraftMetadata{
		ID:                    value.ID,
		SourceIRI:             value.SourceIRI,
		Title:                 value.Title,
		Description:           value.Description,
		Keywords:              value.Keywords,
		AccrualPeriodicity:    value.AccrualPeriodicity,
		QualityAnnotation:     value.QualityAnnotation,
		AccessRight:           value.AccessRight,
		Distributions:         mapSlice(value.Distributions, toDomainDistribution),
		SpatialCoverage:       mapPtr(value.SpatialCoverage, toDomainSpatialCoverage),
		TemporalCoverage:      mapPtr(value.TemporalCoverage, toDomainTemporalCoverage),
		PersistentIdentifiers: mapSlice(value.PersistentIdentifiers, toDomainIdentifierDraft),
		ContactPoints:         mapSlice(value.ContactPoints, toDomainResourceReference),
		Providers:             mapSlice(value.Providers, toDomainResourceReference),
		Categories:            mapSlice(value.Categories, toDomainResourceReference),
		SourceCreated:         value.Created,
		SourceUpdated:         value.Updated,
	}
}

func toDomainSoftwareApplicationDraftMetadata(value api.SoftwareApplicationDraftMetadata) domain.SoftwareApplicationDraftMetadata {
	return domain.SoftwareApplicationDraftMetadata{
		SourceIRI:        value.SourceIRI,
		Name:             value.Name,
		Description:      value.Description,
		Keywords:         value.Keywords,
		License:          value.License,
		Version:          value.Version,
		DownloadURL:      value.DownloadURL,
		Requirements:     value.Requirements,
		OperatingSystem:  value.OperatingSystem,
		MainEntityOfPage: value.MainEntityOfPage,
		Creators:         mapSlice(value.Creators, toDomainResourceReference),
		Identifiers:      mapSlice(value.Identifiers, toDomainIdentifierDraft),
		Categories:       mapSlice(value.Categories, toDomainResourceReference),
		ContactPoints:    mapSlice(value.ContactPoints, toDomainResourceReference),
	}
}

func toDomainSoftwareSourceCodeDraftMetadata(value api.SoftwareSourceCodeDraftMetadata) domain.SoftwareSourceCodeDraftMetadata {
	return domain.SoftwareSourceCodeDraftMetadata{
		SourceIRI:            value.SourceIRI,
		Name:                 value.Name,
		Description:          value.Description,
		Keywords:             value.Keywords,
		Repository:           value.Repository,
		ProgrammingLanguage:  value.ProgrammingLanguage,
		License:              value.License,
		SoftwareRequirements: value.SoftwareRequirements,
		RuntimePlatform:      value.RuntimePlatform,
		MainEntityOfPage:     value.MainEntityOfPage,
		Version:              value.Version,
		Creators:             mapSlice(value.Creators, toDomainResourceReference),
		Identifiers:          mapSlice(value.Identifiers, toDomainIdentifierDraft),
		Categories:           mapSlice(value.Categories, toDomainResourceReference),
		ContactPoints:        mapSlice(value.ContactPoints, toDomainResourceReference),
	}
}

func toDomainResourceReference(value api.ResourceReference) domain.ResourceReference {
	return domain.ResourceReference(value)
}

func toDomainIdentifierDraft(value api.IdentifierDraft) domain.Identifier {
	return domain.Identifier(value)
}

func toDomainCoordinate(value api.Coordinate) domain.Coordinate {
	return domain.Coordinate(value)
}

func toDomainDocumentation(value api.Documentation) domain.Documentation {
	return domain.Documentation(value)
}

func toDomainDownloadableFile(value api.DownloadableFile) domain.DownloadableFile {
	return domain.DownloadableFile(value)
}

func toDomainDistribution(value api.DistributionDraft) domain.DistributionDraft {
	return domain.DistributionDraft{
		ID:               value.ID,
		Title:            value.Title,
		Description:      value.Description,
		License:          value.License,
		Type:             value.Type,
		Issued:           value.Issued,
		Modified:         value.Modified,
		ByteSize:         value.ByteSize,
		WebService:       mapPtr(value.WebService, toDomainWebServiceDraft),
		DownloadableFile: mapPtr(value.DownloadableFile, toDomainDownloadableFile),
	}
}

func toDomainWebServiceDraft(value api.WebServiceDraft) domain.WebServiceDraft {
	return domain.WebServiceDraft{
		ID:               value.ID,
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateURL,
		Parameters:       value.Parameters,
		Format:           value.Format,
		Provider:         mapPtr(value.Provider, toDomainResourceReference),
		Documentation:    mapPtr(value.Documentation, toDomainDocumentation),
		SpatialCoverage:  mapPtr(value.SpatialCoverage, toDomainSpatialCoverage),
		TemporalCoverage: mapPtr(value.TemporalCoverage, toDomainTemporalCoverage),
		ContactPoints:    mapSlice(value.ContactPoints, toDomainResourceReference),
	}
}

func toDomainSpatialCoverage(value api.SpatialCoverage) domain.SpatialCoverage {
	return domain.SpatialCoverage{
		Point:   mapPtr(value.Point, toDomainCoordinate),
		Polygon: mapSlice(value.Polygon, toDomainCoordinate),
	}
}

func toDomainTemporalCoverage(value api.TemporalCoverage) domain.TemporalCoverage {
	return domain.TemporalCoverage(value)
}

func toDomainRejectResourceInput(value api.RejectResourceRequest) domain.RejectResourceInput {
	return domain.RejectResourceInput(value)
}

func toDomainCopyResourceInput(value api.CopyResourceRequest) domain.CopyResourceInput {
	return domain.CopyResourceInput(value)
}

func toDomainReplaceResourceDraftInput(revision int32, metadata domain.DraftMetadata) domain.ReplaceResourceDraftInput {
	return domain.ReplaceResourceDraftInput{Revision: revision, Metadata: metadata}
}

func toAPIWriteResult(resourceType string, value domain.ResourceVersion) api.WriteResult {
	return api.WriteResult{
		ResourceType: api.ResourceType(resourceType),
		ID:           value.ResourceID,
		Status:       api.VersioningStatus(strings.ToLower(value.Status)),
	}
}

func toAPIDiscardedWriteResult(resourceType, resourceID string) api.WriteResult {
	return api.WriteResult{
		ResourceType: api.ResourceType(resourceType),
		ID:           resourceID,
		Status:       api.DiscardedStatus,
	}
}

func toAPIResourceVersion(value domain.ResourceVersion) api.ResourceVersion {
	result := api.ResourceVersion{
		ResourceID: value.ResourceID,
		VersionID:  value.VersionID,
		Revision:   value.Revision,
		Status:     value.Status,
		CreatedBy:  value.CreatedBy,
		GroupID:    value.GroupID,
		GroupName:  value.GroupName,
		CreatedAt:  value.CreatedAt,
		UpdatedAt:  value.UpdatedAt,
		Metadata:   nil,
	}

	switch metadata := value.Metadata.(type) {
	case domain.DatasetResourceMetadata:
		result.Metadata = toAPIDatasetResourceMetadata(metadata)
	case domain.SoftwareApplicationResourceMetadata:
		result.Metadata = toAPISoftwareApplicationResourceMetadata(metadata)
	case domain.SoftwareSourceCodeResourceMetadata:
		result.Metadata = toAPISoftwareSourceCodeResourceMetadata(metadata)
	}

	return result
}

func toAPIDatasetResourceMetadata(value domain.DatasetResourceMetadata) api.DatasetResourceMetadata {
	return api.DatasetResourceMetadata{
		ID:                    value.ID,
		SourceIRI:             value.SourceIRI,
		Title:                 value.Title,
		Description:           value.Description,
		Keywords:              value.Keywords,
		AccrualPeriodicity:    value.AccrualPeriodicity,
		QualityAnnotation:     value.QualityAnnotation,
		AccessRight:           value.AccessRight,
		Distributions:         toAPIDistributions(value.Distributions),
		SpatialCoverage:       mapPtr(value.SpatialCoverage, toAPISpatialCoverage),
		TemporalCoverage:      mapPtr(value.TemporalCoverage, toAPITemporalCoverage),
		PersistentIdentifiers: mapSlice(value.PersistentIdentifiers, toAPIIdentifierDraft),
		ContactPoints:         mapSlice(value.ContactPoints, toAPIContactPointReference),
		Providers:             mapSlice(value.Providers, toAPIResourceVersionReference),
		Categories:            mapSlice(value.Categories, toAPIResourceVersionReference),
		Created:               value.SourceCreated,
		Updated:               value.SourceUpdated,
	}
}

func toAPISoftwareApplicationResourceMetadata(value domain.SoftwareApplicationResourceMetadata) api.SoftwareApplicationResourceMetadata {
	return api.SoftwareApplicationResourceMetadata{
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
		Creators:         mapSlice(value.Creators, toAPIResourceVersionReference),
		Identifiers:      mapSlice(value.Identifiers, toAPIIdentifierDraft),
		ContactPoints:    mapSlice(value.ContactPoints, toAPIContactPointReference),
		Categories:       mapSlice(value.Categories, toAPIResourceVersionReference),
	}
}

func toAPISoftwareSourceCodeResourceMetadata(value domain.SoftwareSourceCodeResourceMetadata) api.SoftwareSourceCodeResourceMetadata {
	return api.SoftwareSourceCodeResourceMetadata{
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
		Creators:             mapSlice(value.Creators, toAPIResourceVersionReference),
		Identifiers:          mapSlice(value.Identifiers, toAPIIdentifierDraft),
		ContactPoints:        mapSlice(value.ContactPoints, toAPIContactPointReference),
		Categories:           mapSlice(value.Categories, toAPIResourceVersionReference),
	}
}

func toAPIIdentifierDraft(value domain.Identifier) api.IdentifierDraft {
	return api.IdentifierDraft{ID: value.ID, SourceIRI: value.SourceIRI, Type: value.Type, Value: value.Value}
}

func toAPIResourceVersionReference(value domain.ResourceVersionReference) api.ResourceVersionReference {
	return api.ResourceVersionReference{ID: value.ID, SourceIRI: value.SourceIRI, Name: value.Name}
}

func toAPIContactPointReference(value domain.ContactPointReference) api.ContactPointReference {
	return api.ContactPointReference{ID: value.ID, SourceIRI: value.SourceIRI}
}

func toAPISpatialCoverage(value domain.SpatialCoverage) api.SpatialCoverage {
	return api.SpatialCoverage{Point: mapPtr(value.Point, toAPICoordinate), Polygon: mapSlice(value.Polygon, toAPICoordinate)}
}

func toAPICoordinate(value domain.Coordinate) api.Coordinate {
	return api.Coordinate{Latitude: value.Latitude, Longitude: value.Longitude}
}

func toAPITemporalCoverage(value domain.TemporalCoverage) api.TemporalCoverage {
	return api.TemporalCoverage{Start: value.Start, End: value.End}
}

func toAPIDistributions(values []domain.Distribution) []api.Distribution {
	if values == nil {
		return nil
	}
	result := make([]api.Distribution, 0, len(values))
	for _, value := range values {
		switch value := value.(type) {
		case domain.WebServiceDistribution:
			result = append(result, api.WebServiceDistribution{
				ID:          value.ID,
				Title:       value.Title,
				Description: value.Description,
				License:     value.License,
				Type:        value.Type,
				Issued:      value.Issued,
				Modified:    value.Modified,
				ByteSize:    value.ByteSize,
				WebService:  toAPIWebService(value.WebService),
			})
		case domain.DownloadableFileDistribution:
			result = append(result, api.DownloadableFileDistribution{
				ID:          value.ID,
				Title:       value.Title,
				Description: value.Description,
				License:     value.License,
				Type:        value.Type,
				Issued:      value.Issued,
				Modified:    value.Modified,
				ByteSize:    value.ByteSize,
				DownloadableFile: api.DownloadableFile{
					URL:    value.DownloadableFile.URL,
					Format: value.DownloadableFile.Format,
				},
			})
		}
	}
	return result
}

func toAPIWebService(value domain.WebService) api.WebService {
	return api.WebService{
		ID:               value.ID,
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateURL,
		Parameters:       toAPIParameters(value.Parameters),
		Format:           value.Format,
		Provider:         mapPtr(value.Provider, toAPIResourceVersionReference),
		Documentation:    mapPtr(value.Documentation, func(value domain.Documentation) api.Documentation { return api.Documentation(value) }),
		SpatialCoverage:  mapPtr(value.SpatialCoverage, toAPISpatialCoverage),
		TemporalCoverage: mapPtr(value.TemporalCoverage, toAPITemporalCoverage),
		ContactPoints:    mapSlice(value.ContactPoints, toAPIContactPointReference),
	}
}

func toAPIParameters(values []domain.Parameter) []api.Parameter {
	if values == nil {
		return nil
	}
	result := make([]api.Parameter, 0, len(values))
	for _, value := range values {
		switch value := value.(type) {
		case domain.StringParameter:
			result = append(result, api.StringParameter{Kind: value.Metadata.Kind, Metadata: toAPIParameterMetadata(value.Metadata), Default: value.Default, AllowedValues: value.AllowedValues, Pattern: value.Pattern})
		case domain.BooleanParameter:
			result = append(result, api.BooleanParameter{Kind: value.Metadata.Kind, Metadata: toAPIParameterMetadata(value.Metadata), Default: value.Default})
		case domain.IntegerParameter:
			result = append(result, api.IntegerParameter{Kind: value.Metadata.Kind, Metadata: toAPIParameterMetadata(value.Metadata), Minimum: value.Minimum, Maximum: value.Maximum, Default: value.Default, AllowedValues: value.AllowedValues})
		case domain.FloatParameter:
			result = append(result, api.FloatParameter{Kind: value.Metadata.Kind, Metadata: toAPIParameterMetadata(value.Metadata), Minimum: value.Minimum, Maximum: value.Maximum, Default: value.Default, AllowedValues: value.AllowedValues})
		case domain.DateTimeParameter:
			result = append(result, api.DateTimeParameter{Kind: value.Metadata.Kind, Metadata: toAPIParameterMetadata(value.Metadata), Minimum: value.Minimum, Maximum: value.Maximum, Default: value.Default, Pattern: value.Pattern})
		}
	}
	return result
}

func toAPIParameterMetadata(value domain.ParameterMetadata) api.ParameterMetadata {
	return api.ParameterMetadata{
		ID:             value.ID,
		Variable:       value.Variable,
		Property:       value.Property,
		Label:          value.Label,
		Required:       value.Required,
		MultipleValues: value.MultipleValues,
		ReadOnly:       value.ReadOnly,
	}
}

func toAPIContactPoint(value domain.ContactPoint) api.ContactPoint {
	return api.ContactPoint(value)
}

func toAPIOrganizationIdentifier(value domain.Identifier) api.OrganizationIdentifier {
	return api.OrganizationIdentifier(value)
}

func toAPICanonicalOrganization(value domain.CanonicalOrganization) api.CanonicalOrganization {
	return api.CanonicalOrganization{
		ID:              value.ID,
		SourceIRI:       value.SourceIRI,
		Name:            value.Name,
		Address:         api.OrganizationAddress(value.Address),
		LogoURL:         value.LogoURL,
		PageURL:         value.PageURL,
		Email:           value.Email,
		LEICode:         value.LEICode,
		Identifiers:     mapSlice(value.Identifiers, toAPIOrganizationIdentifier),
		Parents:         value.Parents,
		ContactPointIDs: value.ContactPointIDs,
	}
}

func toDomainOrganizationAddress(value api.OrganizationAddress) domain.OrganizationAddress {
	return domain.OrganizationAddress(value)
}

func toDomainOrganizationIdentifierInput(value api.OrganizationIdentifierRequest) domain.OrganizationIdentifierInput {
	return domain.OrganizationIdentifierInput(value)
}

func toDomainCreateContactPointInput(value api.CreateContactPointRequest) domain.CreateContactPointInput {
	return domain.CreateContactPointInput(value)
}

func toDomainReplaceContactPointInput(value api.ReplaceContactPointRequest) domain.ReplaceContactPointInput {
	return domain.ReplaceContactPointInput(value)
}

func toDomainCreateOrganizationInput(value api.CreateOrganizationRequest) domain.CreateOrganizationInput {
	return domain.CreateOrganizationInput{
		SourceIRI:       value.SourceIRI,
		Name:            value.Name,
		Address:         mapPtr(value.Address, toDomainOrganizationAddress),
		LogoURL:         value.LogoURL,
		PageURL:         value.PageURL,
		Email:           value.Email,
		LEICode:         value.LEICode,
		Identifiers:     mapSlice(value.Identifiers, toDomainOrganizationIdentifierInput),
		Parents:         value.Parents,
		ContactPointIDs: value.ContactPointIDs,
	}
}

func toDomainReplaceOrganizationInput(value api.ReplaceOrganizationRequest) domain.ReplaceOrganizationInput {
	return domain.ReplaceOrganizationInput{
		Name:            value.Name,
		Address:         toDomainOrganizationAddress(value.Address),
		LogoURL:         value.LogoURL,
		PageURL:         value.PageURL,
		Email:           value.Email,
		LEICode:         value.LEICode,
		Identifiers:     mapSlice(value.Identifiers, toDomainOrganizationIdentifierInput),
		Parents:         value.Parents,
		ContactPointIDs: value.ContactPointIDs,
	}
}

func toAPIImportDiagnostic(value domain.ImportDiagnostic) api.ImportDiagnostic {
	return api.ImportDiagnostic(value)
}

func toAPIImportResult(value domain.ImportResult) api.ImportResult {
	return api.ImportResult{
		Groups:               value.Groups,
		Categories:           value.Categories,
		Organizations:        value.Organizations,
		ContactPoints:        value.ContactPoints,
		Datasets:             value.Datasets,
		SoftwareApplications: value.SoftwareApplications,
		SoftwareSourceCodes:  value.SoftwareSourceCodes,
		Warnings:             mapSlice(value.Warnings, toAPIImportDiagnostic),
	}
}

func toDomainCreateGroupInput(value api.CreateGroupRequest) domain.CreateGroupInput {
	return domain.CreateGroupInput(value)
}

func toDomainReplaceGroupInput(value api.ReplaceGroupRequest) domain.ReplaceGroupInput {
	return domain.ReplaceGroupInput(value)
}

func toDomainCreateCategoryInput(value api.CreateCategoryRequest) domain.CreateCategoryInput {
	return domain.CreateCategoryInput(value)
}

func toDomainReplaceCategoryInput(value api.ReplaceCategoryRequest) domain.ReplaceCategoryInput {
	return domain.ReplaceCategoryInput(value)
}

func toAPIGroup(value domain.Group) api.Group {
	return api.Group(value)
}

func toAPICategory(value domain.Category) api.Category {
	return api.Category(value)
}

func toAPIMembershipRole(value string) api.MembershipRole {
	return api.MembershipRole(value)
}

func toAPIMembershipStatus(value string) api.MembershipStatus {
	return api.MembershipStatus(value)
}

func toAPIMembershipUser(value domain.MembershipUser) api.MembershipUser {
	return api.MembershipUser(value)
}

func toAPIRoleRequest(value domain.RoleRequest) api.RoleRequest {
	return api.RoleRequest{
		ID:          value.ID,
		Role:        value.Role,
		Status:      value.Status,
		RequestedAt: value.RequestedAt.Format(time.RFC3339Nano),
	}
}

func toAPIRoleRequests(values []domain.RoleRequest) []api.RoleRequest {
	return mapSlice(values, toAPIRoleRequest)
}

func toAPIGroupMembership(value domain.GroupMembership) api.GroupMembership {
	return api.GroupMembership{
		User:         toAPIMembershipUser(value.User),
		Status:       toAPIMembershipStatus(value.Status),
		Roles:        mapSlice(value.Roles, toAPIMembershipRole),
		Permissions:  append([]string{}, value.Permissions...),
		RoleRequests: toAPIRoleRequests(value.RoleRequests),
		CreatedAt:    value.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:    value.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func toAPIReference(value domain.DiscoveryReference) api.Reference {
	return api.Reference(value)
}

func toAPIGroupReference(value domain.Group) api.Reference {
	return api.Reference{ID: value.ID, Name: value.Name}
}

func toAPIUserMembershipSummary(value domain.Membership) api.UserMembershipSummary {
	return api.UserMembershipSummary{
		Group:        toAPIGroupReference(value.Group),
		Roles:        mapSlice(value.Roles, toAPIMembershipRole),
		Status:       toAPIMembershipStatus(value.Status),
		Permissions:  append([]string{}, value.Permissions...),
		RoleRequests: toAPIRoleRequests(value.RoleRequests),
		CreatedAt:    value.CreatedAt,
		UpdatedAt:    value.UpdatedAt,
	}
}

func ToAPIUser(value domain.User) api.User {
	return api.User{
		ID:                value.ID,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            api.UserStatus(value.Status),
		GlobalRoles:       append([]string{}, value.GlobalRoles...),
		GlobalPermissions: append([]string{}, value.GlobalPermissions...),
		Memberships:       mapSlice(value.Memberships, toAPIUserMembershipSummary),
		CreatedAt:         value.CreatedAt,
		UpdatedAt:         value.UpdatedAt,
	}
}

func toAPIUserSummary(value domain.UserSummary) api.UserSummary {
	return api.UserSummary{
		ID:                value.ID,
		Email:             value.Email,
		GivenName:         value.GivenName,
		FamilyName:        value.FamilyName,
		Status:            api.UserStatus(value.Status),
		GlobalRoles:       append([]string{}, value.GlobalRoles...),
		GlobalPermissions: append([]string{}, value.GlobalPermissions...),
		CreatedAt:         value.CreatedAt,
		UpdatedAt:         value.UpdatedAt,
	}
}

func toAPIUserSummaries(values []domain.UserSummary) []api.UserSummary {
	return mapSlice(values, toAPIUserSummary)
}

func toAPIDiscoveryItem(value domain.DiscoveryItem) api.DiscoveryItem {
	return api.DiscoveryItem{
		Type:          value.Type,
		ResourceID:    value.ResourceID,
		VersionID:     value.VersionID,
		SourceIRI:     value.SourceIRI,
		Name:          value.Name,
		Status:        value.Status,
		Group:         toAPIReference(value.Group),
		Providers:     mapSlice(value.Providers, toAPIReference),
		Creators:      mapSlice(value.Creators, toAPIReference),
		Categories:    mapSlice(value.Categories, toAPIReference),
		Distributions: mapSlice(value.Distributions, toAPIReference),
	}
}

var discoveryTypeNames = map[string]struct{}{
	"dataset":              {},
	"software-application": {},
	"software-source-code": {},
}

// TODO: use a domain level enum
var discoveryStatusNames = map[string]struct{}{
	"PUBLISHED": {},
	"DRAFT":     {},
	"SUBMITTED": {},
	"ARCHIVED":  {},
	"REJECTED":  {},
}

func toDomainDiscoveryFilter(values url.Values) (domain.DiscoveryFilter, error) {
	dateFrom, err := parseDiscoveryTime(values.Get("dateFrom"), "dateFrom")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	dateTo, err := parseDiscoveryTime(values.Get("dateTo"), "dateTo")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	if dateFrom != nil && dateTo != nil && dateFrom.After(*dateTo) {
		return domain.DiscoveryFilter{}, fmt.Errorf("dateFrom must be less than or equal to dateTo")
	}

	types, err := discoveryCSV(values["type"], discoveryTypeNames, "type")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	statuses, err := discoveryStatuses(values["versioningStatus"])
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}

	return domain.DiscoveryFilter{
		Query:              strings.TrimSpace(values.Get("q")),
		Types:              types,
		SpatialFilter:      strings.TrimSpace(values.Get("spatialFilter")),
		DateFrom:           dateFrom,
		DateTo:             dateTo,
		Provider:           strings.Join(splitDiscoveryCSV(values["provider"]), ","),
		Keywords:           strings.Join(splitDiscoveryCSV(values["keywords"]), ","),
		VersioningStatuses: statuses,
	}, nil
}

func parseDiscoveryTime(value, name string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339", name)
	}
	return &parsed, nil
}

func discoveryCSV(values []string, allowed map[string]struct{}, name string) ([]string, error) {
	result := splitDiscoveryCSV(values)
	for _, value := range result {
		if _, ok := allowed[value]; !ok {
			return nil, fmt.Errorf("unsupported %s %q", name, value)
		}
	}
	return result, nil
}

func discoveryStatuses(values []string) ([]string, error) {
	rawStatuses := splitDiscoveryCSV(values)
	if len(rawStatuses) == 0 {
		// TODO: use an enum
		return []string{"PUBLISHED"}, nil
	}
	statuses := make([]string, 0, len(rawStatuses))
	for _, rawStatus := range rawStatuses {
		status := strings.ToUpper(rawStatus)
		if _, ok := discoveryStatusNames[status]; !ok {
			return nil, fmt.Errorf("unsupported versioningStatus %q", rawStatus)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func splitDiscoveryCSV(values []string) []string {
	var result []string
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}
