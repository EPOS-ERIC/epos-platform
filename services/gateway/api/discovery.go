package api

type DiscoveryItem struct {
	Type          string      `json:"type"`
	ResourceID    string      `json:"resource_id"`
	VersionID     string      `json:"version_id"`
	Name          string      `json:"name"`
	Status        string      `json:"status"`
	UpdatedAt     string      `json:"updated_at"`
	Concept       string      `json:"concept"`
	CreatedBy     *Reference  `json:"created_by,omitempty"`
	Group         Reference   `json:"group"`
	Providers     []Reference `json:"providers,omitempty"`
	Creators      []Reference `json:"creators,omitempty"`
	Categories    []Reference `json:"categories,omitempty"`
	Distributions []Reference `json:"distributions,omitempty"`
}
