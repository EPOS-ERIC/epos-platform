package domain

import "net/url"

// Capabilities describes instance-wide client configuration.
type Capabilities struct {
	OrganizationsSynced bool
	AuthRootURL         string
	EmailEnabled        bool
	IssueTrackerURL     *url.URL
	IssueTrackerOpenNew *url.URL
}
