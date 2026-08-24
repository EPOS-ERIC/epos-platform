package api

type DiscoveryItem struct {
	Type          string      `json:"type"`
	ResourceID    string      `json:"resource_id"`
	VersionID     string      `json:"version_id"`
	SourceIRI     string      `json:"source_iri"`
	Name          string      `json:"name"`
	Status        string      `json:"status"`
	Group         Reference   `json:"group"`
	Providers     []Reference `json:"providers,omitempty"`
	Creators      []Reference `json:"creators,omitempty"`
	Categories    []Reference `json:"categories,omitempty"`
	Distributions []Reference `json:"distributions,omitempty"`
}
