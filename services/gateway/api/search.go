// Package api defines gateway request and response models shared across handlers and domain services.
package api

type SearchResponse struct {
	PageNumber int          `json:"page_number"`
	PageSize   int          `json:"page_size"`
	Total      int          `json:"total"`
	TotalPages int          `json:"total_pages"`
	Items      []SearchItem `json:"items"`
}

type SearchItem interface {
	searchItem()
}

type BaseSearchItem struct {
	Type ResourceType `json:"type"`
	ID   string       `json:"id"`
	Name string       `json:"name,omitempty"`
	Href string       `json:"href"`
}

type ResourceRef struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Href string `json:"href"`
}

type DatasetDistributionSearchItem struct {
	// TODO: add more fields here when necessary necessary
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Href string `json:"href"`
}

type DatasetItem struct {
	BaseSearchItem
	Providers     []ResourceRef                   `json:"providers,omitempty"`
	Concepts      []ResourceRef                   `json:"concepts,omitempty"`
	Distributions []DatasetDistributionSearchItem `json:"distributions,omitempty"`
}

func (DatasetItem) searchItem() {}

type SoftwareSourceCodeItem struct {
	BaseSearchItem
	Provider []ResourceRef `json:"provider,omitempty"`
}

func (SoftwareSourceCodeItem) searchItem() {}

type SoftwareApplicationItem struct {
	BaseSearchItem
	Provider []ResourceRef `json:"provider,omitempty"`
}

func (SoftwareApplicationItem) searchItem() {}

type WebServiceItem struct {
	BaseSearchItem
	Provider []ResourceRef `json:"provider,omitempty"`
	Concepts []ResourceRef `json:"concepts,omitempty"`
}

func (WebServiceItem) searchItem() {}

type OrganizationItem struct {
	BaseSearchItem
}

func (OrganizationItem) searchItem() {}
