package domain

// Capabilities describes instance-wide client configuration.
type Capabilities struct {
	OrganizationsSynced bool
	AuthRootURL         string
	EmailEnabled        bool
}
