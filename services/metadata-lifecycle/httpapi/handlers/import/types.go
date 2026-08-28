package imports

import "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"

type resultResponse struct {
	Groups               int          `json:"groups"`
	Categories           int          `json:"categories"`
	Organizations        int          `json:"organizations"`
	ContactPoints        int          `json:"contact_points"`
	Datasets             int          `json:"datasets"`
	SoftwareApplications int          `json:"software_applications"`
	SoftwareSourceCodes  int          `json:"software_source_codes"`
	Warnings             []diagnostic `json:"warnings"`
}

type diagnostic struct {
	Severity  domain.ImportSeverity `json:"severity"`
	Subject   string                `json:"subject"`
	Predicate string                `json:"predicate"`
	Message   string                `json:"message"`
}
