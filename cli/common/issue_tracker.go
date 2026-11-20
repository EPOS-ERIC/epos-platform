package common

import (
	"fmt"
	"net/url"
)

// IssueTracker configures the links exposed by the gateway.
type IssueTracker struct {
	URL     string `yaml:"url"`
	OpenNew string `yaml:"open_new"`
}

func (t IssueTracker) Validate() error {
	if t == (IssueTracker{}) {
		return nil
	}

	uri, err := url.Parse(t.URL)
	if err != nil || !uri.IsAbs() {
		return fmt.Errorf("components.gateway.issue_tracker.url must be an absolute URI")
	}

	uri, err = url.Parse(t.OpenNew)
	if err != nil || !uri.IsAbs() {
		return fmt.Errorf("components.gateway.issue_tracker.open_new must be an absolute URI")
	}

	return nil
}
