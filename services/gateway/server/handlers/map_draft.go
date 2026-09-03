//nolint:dupl,exhaustruct_v5,wrapcheck
package handlers

import (
	"errors"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func mapDraftParameterMetadata(v gatewayapi.ParameterDraftMetadata) domain.ParameterDraftMetadata {
	return domain.ParameterDraftMetadata{
		Kind:           string(v.Kind),
		Variable:       v.Variable,
		Property:       v.Property,
		Label:          v.Label,
		Required:       v.Required,
		MultipleValues: v.MultipleValues,
		ReadOnly:       v.ReadOnly,
	}
}

func mapDraftParameter(v gatewayapi.ParameterDraft) domain.ParameterDraft {
	switch {
	case v.IsStringParameterDraft():
		x := v.StringParameterDraft
		return domain.StringParameterDraft{
			Metadata:      mapDraftParameterMetadata(x.Metadata),
			Default:       fromNilString(x.Default),
			AllowedValues: x.AllowedValues,
			Pattern:       x.Pattern,
		}
	case v.IsBooleanParameterDraft():
		x := v.BooleanParameterDraft
		return domain.BooleanParameterDraft{
			Metadata: mapDraftParameterMetadata(x.Metadata),
			Default:  fromNilBool(x.Default),
		}
	case v.IsIntegerParameterDraft():
		x := v.IntegerParameterDraft
		return domain.IntegerParameterDraft{
			Metadata:      mapDraftParameterMetadata(x.Metadata),
			Minimum:       fromNilInt64(x.Minimum),
			Maximum:       fromNilInt64(x.Maximum),
			Default:       fromNilInt64(x.Default),
			AllowedValues: x.AllowedValues,
		}
	case v.IsFloatParameterDraft():
		x := v.FloatParameterDraft
		return domain.FloatParameterDraft{
			Metadata:      mapDraftParameterMetadata(x.Metadata),
			Minimum:       fromNilFloat64(x.Minimum),
			Maximum:       fromNilFloat64(x.Maximum),
			Default:       fromNilFloat64(x.Default),
			AllowedValues: x.AllowedValues,
		}
	case v.IsDateTimeParameterDraft():
		x := v.DateTimeParameterDraft
		return domain.DateTimeParameterDraft{
			Metadata: mapDraftParameterMetadata(x.Metadata),
			Minimum:  fromNilTime(x.Minimum),
			Maximum:  fromNilTime(x.Maximum),
			Default:  fromNilTime(x.Default),
			Pattern:  x.Pattern,
		}
	default:
		return nil
	}
}

func fromNilString(v gatewayapi.NilString) *string {
	if v.Null {
		return nil
	}
	return &v.Value
}

func fromNilBool(v gatewayapi.NilBool) *bool {
	if v.Null {
		return nil
	}
	return &v.Value
}

func fromNilInt64(v gatewayapi.NilInt64) *int64 {
	if v.Null {
		return nil
	}
	return &v.Value
}

func fromNilFloat64(v gatewayapi.NilFloat64) *float64 {
	if v.Null {
		return nil
	}
	return &v.Value
}

func fromNilTime(v gatewayapi.NilDateTime) *time.Time {
	if v.Null {
		return nil
	}
	return &v.Value
}

func fromOptTime(v gatewayapi.OptNilDateTime) *time.Time {
	if !v.Set || v.Null {
		return nil
	}
	return &v.Value
}

func fromOptInt64(v gatewayapi.OptNilInt64) *int64 {
	if !v.Set || v.Null {
		return nil
	}
	return &v.Value
}

func mapDraftDistribution(v gatewayapi.DistributionDraft) domain.DistributionDraft {
	r := domain.DistributionDraft{Type: string(v.Type)}
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		r.Title = x.Title
		r.Description = x.Description.Value
		r.Issued = fromOptTime(x.Issued)
		r.Modified = fromOptTime(x.Modified)
		r.ByteSize = fromOptInt64(x.ByteSize)
		if x.License.Set && !x.License.Null {
			r.License = &x.License.Value
		}
		w := mapWebServiceDraft(x.WebService)
		if x.WebService.Provider.Set {
			s := x.WebService.Provider.Value.String()
			w.Provider = &s
		}
		r.WebService = &w
	}
	if v.IsDownloadableFileDistributionDraft() {
		x := v.DownloadableFileDistributionDraft
		r.Title = x.Title
		r.Description = x.Description.Value
		r.Issued = fromOptTime(x.Issued)
		r.Modified = fromOptTime(x.Modified)
		r.ByteSize = fromOptInt64(x.ByteSize)
		if x.License.Set && !x.License.Null {
			r.License = &x.License.Value
		}
		r.DownloadableFile = &domain.DownloadableFile{
			URL:    string(x.DownloadableFile.URL),
			Format: string(x.DownloadableFile.Format),
		}
	}
	return r
}

func fromOptTemporalCoverage(v gatewayapi.OptTemporalCoverage) *domain.TemporalCoverage {
	if !v.Set {
		return nil
	}
	r := domain.TemporalCoverage{}
	if v.Value.Start.Set && !v.Value.Start.Null {
		r.Start = &v.Value.Start.Value
	}
	if v.Value.End.Set && !v.Value.End.Null {
		r.End = &v.Value.End.Value
	}
	return &r
}

func fromOptString(v gatewayapi.OptString) *string {
	if !v.Set {
		return nil
	}
	return &v.Value
}

func fromURI(v gatewayapi.OptURI) *string {
	if !v.Set {
		return nil
	}
	s := v.Value.String()
	return &s
}

func mapDraftDocumentation(v gatewayapi.OptDocumentation) *domain.Documentation {
	if !v.Set {
		return nil
	}
	d := v.Value
	return &domain.Documentation{
		URI:         string(d.URI),
		Title:       fromOptNilString(d.Title),
		Description: fromOptNilString(d.Description),
	}
}

func mapDraftCommonParameters(v []gatewayapi.ParameterDraft) []domain.ParameterDraft {
	if v == nil {
		return nil
	}
	out := make([]domain.ParameterDraft, 0, len(v))
	for _, x := range v {
		if mapped := mapDraftParameter(x); mapped != nil {
			out = append(out, mapped)
		}
	}
	return out
}

func mapWebServiceDraft(v gatewayapi.WebServiceDraft) domain.WebServiceDraft {
	r := domain.WebServiceDraft{
		Name:             string(v.Name),
		Description:      v.Description.Value,
		TemplateURL:      string(v.TemplateURL),
		Parameters:       mapDraftCommonParameters(v.Parameters),
		Format:           string(v.Format),
		SpatialCoverage:  fromOptNilString(v.SpatialCoverage),
		TemporalCoverage: fromOptTemporalCoverage(v.TemporalCoverage),
		Documentation:    mapDraftDocumentation(v.Documentation),
	}
	if v.Provider.Set {
		s := v.Provider.Value.String()
		r.Provider = &s
	}
	for _, x := range v.ContactPoints {
		r.ContactPoints = append(r.ContactPoints, x.String())
	}
	return r
}

func mapDatasetDraft(v gatewayapi.DatasetDraftMetadata) domain.DatasetDraftMetadata {
	r := domain.DatasetDraftMetadata{
		Title:              string(v.Title),
		Description:        fromOptString(v.Description),
		Keywords:           fromOptString(v.Keywords),
		AccrualPeriodicity: fromOptNilString(v.AccrualPeriodicity),
		QualityAnnotation:  fromOptNilString(v.QualityAnnotation),
		AccessRight:        fromOptNilString(v.AccessRight),
		SpatialCoverage:    fromOptNilString(v.SpatialCoverage),
		TemporalCoverage:   fromOptTemporalCoverage(v.TemporalCoverage),
		SourceCreated:      fromOptTime(v.Created),
		SourceUpdated:      fromOptTime(v.Updated),
	}
	for _, x := range v.Distributions {
		r.Distributions = append(r.Distributions, mapDraftDistribution(x))
	}
	for _, x := range v.PersistentIdentifiers {
		r.PersistentIdentifiers = append(r.PersistentIdentifiers, domain.IdentifierDraft{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	for _, x := range v.ContactPoints {
		r.ContactPoints = append(r.ContactPoints, x.String())
	}
	for _, x := range v.Providers {
		r.Providers = append(r.Providers, x.String())
	}
	for _, x := range v.Categories {
		r.Categories = append(r.Categories, x.String())
	}
	return r
}

func mapSoftwareApplicationDraft(v gatewayapi.SoftwareApplicationDraftMetadata) domain.SoftwareApplicationDraftMetadata {
	r := domain.SoftwareApplicationDraftMetadata{
		Name:             string(v.Name),
		Description:      fromOptString(v.Description),
		Keywords:         fromOptString(v.Keywords),
		License:          fromOptNilString(v.License),
		Version:          fromOptNilString(v.Version),
		DownloadURL:      fromURI(v.DownloadURL),
		Requirements:     fromOptNilString(v.Requirements),
		OperatingSystem:  fromOptNilString(v.OperatingSystem),
		MainEntityOfPage: fromURI(v.MainEntityOfPage),
		Categories:       make([]string, 0, len(v.Categories)),
		ContactPoints:    make([]string, 0, len(v.ContactPoints)),
	}
	for _, x := range v.Creators {
		r.Creators = append(r.Creators, x.String())
	}
	for _, x := range v.Identifiers {
		r.Identifiers = append(r.Identifiers, domain.IdentifierDraft{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	for _, x := range v.Categories {
		r.Categories = append(r.Categories, x.String())
	}
	for _, x := range v.ContactPoints {
		r.ContactPoints = append(r.ContactPoints, x.String())
	}
	return r
}

func mapSoftwareSourceCodeDraft(v gatewayapi.SoftwareSourceCodeDraftMetadata) domain.SoftwareSourceCodeDraftMetadata {
	r := domain.SoftwareSourceCodeDraftMetadata{
		Name:                 string(v.Name),
		Description:          fromOptString(v.Description),
		Keywords:             fromOptString(v.Keywords),
		Repository:           fromURI(v.Repository),
		ProgrammingLanguage:  fromOptString(v.ProgrammingLanguage),
		License:              fromOptNilString(v.License),
		SoftwareRequirements: fromOptNilString(v.SoftwareRequirements),
		RuntimePlatform:      fromOptNilString(v.RuntimePlatform),
		MainEntityOfPage:     fromURI(v.MainEntityOfPage),
		Version:              fromOptNilString(v.Version),
	}
	for _, x := range v.Creators {
		r.Creators = append(r.Creators, x.String())
	}
	for _, x := range v.Identifiers {
		r.Identifiers = append(r.Identifiers, domain.IdentifierDraft{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	for _, x := range v.Categories {
		r.Categories = append(r.Categories, x.String())
	}
	for _, x := range v.ContactPoints {
		r.ContactPoints = append(r.ContactPoints, x.String())
	}
	return r
}

func mapAddress(v gatewayapi.OrganizationAddress) domain.OrganizationAddress {
	r := domain.OrganizationAddress{}
	if !v.Street.Null {
		r.Street = &v.Street.Value
	}
	if !v.Locality.Null {
		r.Locality = &v.Locality.Value
	}
	if !v.PostalCode.Null {
		r.PostalCode = &v.PostalCode.Value
	}
	if !v.Country.Null {
		r.Country = &v.Country.Value
	}
	return r
}

func mapCreateOrganization(v *gatewayapi.CreateOrganizationRequest) domain.CreateOrganizationInput {
	r := domain.CreateOrganizationInput{
		Name:            string(v.Name),
		LogoURL:         fromOptNilString(v.LogoURL),
		PageURL:         fromOptNilString(v.PageURL),
		Email:           fromOptNilString(v.Email),
		LEICode:         fromOptNilString(v.LeiCode),
		Parents:         make([]string, len(v.Parents)),
		ContactPointIDs: make([]string, len(v.ContactPointIds)),
	}
	if v.Address.Set {
		x := mapAddress(v.Address.Value)
		r.Address = &x
	}
	for i := range v.Parents {
		r.Parents[i] = v.Parents[i].String()
	}
	for i := range v.ContactPointIds {
		r.ContactPointIDs[i] = v.ContactPointIds[i].String()
	}
	for _, x := range v.Identifiers {
		r.Identifiers = append(r.Identifiers, domain.OrganizationIdentifierInput{
			Type:  string(x.Type),
			Value: string(x.Value),
		})
	}
	return r
}

func createDraft(q gatewayapi.CreateResourceDraftRequest) (string, domain.CreateResourceInput, error) {
	switch {
	case q.IsCreateDatasetResourceRequest():
		return "dataset", domain.CreateResourceInput{Metadata: mapDatasetDraft(q.CreateDatasetResourceRequest.Metadata)}, nil
	case q.IsCreateSoftwareApplicationResourceRequest():
		return "software-application", domain.CreateResourceInput{Metadata: mapSoftwareApplicationDraft(q.CreateSoftwareApplicationResourceRequest.Metadata)}, nil
	case q.IsCreateSoftwareSourceCodeResourceRequest():
		return "software-source-code", domain.CreateResourceInput{Metadata: mapSoftwareSourceCodeDraft(q.CreateSoftwareSourceCodeResourceRequest.Metadata)}, nil
	default:
		return "", domain.CreateResourceInput{}, errors.New("unsupported resource draft discriminator")
	}
}

func replaceDraft(q gatewayapi.ReplaceResourceDraftRequest) (string, domain.ReplaceResourceDraftInput, error) {
	switch {
	case q.IsReplaceDatasetDraftRequest():
		return "dataset", domain.ReplaceResourceDraftInput{
			Revision: q.ReplaceDatasetDraftRequest.Revision,
			Metadata: mapDatasetDraft(q.ReplaceDatasetDraftRequest.Metadata),
		}, nil
	case q.IsReplaceSoftwareApplicationDraftRequest():
		return "software-application", domain.ReplaceResourceDraftInput{
			Revision: q.ReplaceSoftwareApplicationDraftRequest.Revision,
			Metadata: mapSoftwareApplicationDraft(q.ReplaceSoftwareApplicationDraftRequest.Metadata),
		}, nil
	case q.IsReplaceSoftwareSourceCodeDraftRequest():
		return "software-source-code", domain.ReplaceResourceDraftInput{
			Revision: q.ReplaceSoftwareSourceCodeDraftRequest.Revision,
			Metadata: mapSoftwareSourceCodeDraft(q.ReplaceSoftwareSourceCodeDraftRequest.Metadata),
		}, nil
	default:
		return "", domain.ReplaceResourceDraftInput{}, errors.New("unsupported resource draft discriminator")
	}
}
