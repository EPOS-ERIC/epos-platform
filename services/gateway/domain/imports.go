package domain

type ImportResult struct {
	Groups               int32
	Categories           int32
	Organizations        int32
	ContactPoints        int32
	Datasets             int32
	SoftwareApplications int32
	SoftwareSourceCodes  int32
	Warnings             []ImportDiagnostic
}

type ImportDiagnostic struct {
	Severity  string
	Subject   string
	Predicate string
	Message   string
}
