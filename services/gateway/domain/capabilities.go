package domain

import "net/url"

// Capabilities describes instance-wide client configuration.
type Capabilities struct {
	MetadataSyncBaseURL string
	AuthRootURL         string
	EmailEnabled        bool
	IssueTrackerURL     *url.URL
	IssueTrackerOpenNew *url.URL
}
