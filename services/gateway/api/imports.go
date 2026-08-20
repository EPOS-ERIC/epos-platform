package api

type ImportResult struct {
	Groups               int32              `json:"groups"`
	Categories           int32              `json:"categories"`
	Organizations        int32              `json:"organizations"`
	ContactPoints        int32              `json:"contact_points"`
	Datasets             int32              `json:"datasets"`
	SoftwareApplications int32              `json:"software_applications"`
	SoftwareSourceCodes  int32              `json:"software_source_codes"`
	Warnings             []ImportDiagnostic `json:"warnings"`
}

type ImportDiagnostic struct {
	Severity  string `json:"severity"`
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Message   string `json:"message"`
}
