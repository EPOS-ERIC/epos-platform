package handlers

import (
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func toClientTemporal(v gatewayapi.OptTemporalCoverage) *metadatalifecycleclient.Temporal {
	if !v.Set {
		return nil
	}
	r := &metadatalifecycleclient.Temporal{}
	if v.Value.Start.Set && !v.Value.Start.Null {
		r.SetStart(v.Value.Start.Value)
	}
	if v.Value.End.Set && !v.Value.End.Null {
		r.SetEnd(v.Value.End.Value)
	}
	return r
}

func toClientDocumentation(v gatewayapi.OptDocumentation) *metadatalifecycleclient.Documentation {
	if !v.Set {
		return nil
	}
	d := v.Value
	r := &metadatalifecycleclient.Documentation{Uri: string(d.URI)}
	if d.Title.Set && !d.Title.Null {
		r.SetTitle(d.Title.Value)
	}
	if d.Description.Set && !d.Description.Null {
		r.SetDescription(d.Description.Value)
	}
	return r
}

func toClientWebServiceDraft(v gatewayapi.WebServiceDraft) metadatalifecycleclient.WebServiceDraft {
	r := metadatalifecycleclient.WebServiceDraft{
		Name:        string(v.Name),
		Description: &v.Description.Value,
		TemplateUrl: string(v.TemplateURL),
		Format:      string(v.Format),
	}
	if v.SpatialCoverage.Set && !v.SpatialCoverage.Null {
		r.SetSpatialCoverage(v.SpatialCoverage.Value)
	}
	if temporal := toClientTemporal(v.TemporalCoverage); temporal != nil {
		r.SetTemporalCoverage(*temporal)
	}
	if documentation := toClientDocumentation(v.Documentation); documentation != nil {
		r.SetDocumentation(*documentation)
	}
	if v.Provider.Set {
		r.SetProvider(v.Provider.Value.String())
	}
	for _, x := range v.ContactPoints {
		r.ContactPoints = append(r.ContactPoints, x.String())
	}
	for _, x := range v.Parameters {
		r.Parameters = append(r.Parameters, toClientParameterDraft(x))
	}
	return r
}

func toClientDistributionDraft(v gatewayapi.DistributionDraft) metadatalifecycleclient.DistributionDraft {
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		r := &metadatalifecycleclient.WebServiceDistributionDraft{
			Title:       x.Title,
			Description: &x.Description.Value,
			Type:        string(v.Type),
			WebService:  toClientWebServiceDraft(x.WebService),
		}
		if x.License.Set && !x.License.Null {
			r.SetLicense(x.License.Value)
		}
		if x.Issued.Set && !x.Issued.Null {
			r.SetIssued(x.Issued.Value)
		}
		if x.Modified.Set && !x.Modified.Null {
			r.SetModified(x.Modified.Value)
		}
		if x.ByteSize.Set && !x.ByteSize.Null {
			r.SetByteSize(x.ByteSize.Value)
		}
		return metadatalifecycleclient.WebServiceDistributionDraftAsDistributionDraft(r)
	}
	x := v.DownloadableFileDistributionDraft
	r := &metadatalifecycleclient.DownloadableFileDistributionDraft{
		Title:       x.Title,
		Description: &x.Description.Value,
		Type:        string(v.Type),
		DownloadableFile: metadatalifecycleclient.DownloadableFile{
			Url:    string(x.DownloadableFile.URL),
			Format: string(x.DownloadableFile.Format),
		},
	}
	if x.License.Set && !x.License.Null {
		r.SetLicense(x.License.Value)
	}
	if x.Issued.Set && !x.Issued.Null {
		r.SetIssued(x.Issued.Value)
	}
	if x.Modified.Set && !x.Modified.Null {
		r.SetModified(x.Modified.Value)
	}
	if x.ByteSize.Set && !x.ByteSize.Null {
		r.SetByteSize(x.ByteSize.Value)
	}
	return metadatalifecycleclient.DownloadableFileDistributionDraftAsDistributionDraft(r)
}
