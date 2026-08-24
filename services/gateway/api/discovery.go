package api

type DiscoveryItem struct {
	Type          string               `json:"type"`
	ResourceID    string               `json:"resource_id"`
	VersionID     string               `json:"version_id"`
	SourceIRI     string               `json:"source_iri"`
	Name          string               `json:"name"`
	Status        string               `json:"status"`
	Group         DiscoveryReference   `json:"group"`
	Providers     []DiscoveryReference `json:"providers,omitempty"`
	Creators      []DiscoveryReference `json:"creators,omitempty"`
	Categories    []DiscoveryReference `json:"categories,omitempty"`
	Distributions []DiscoveryReference `json:"distributions,omitempty"`
}

type DiscoveryReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
