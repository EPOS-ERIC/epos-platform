// Package api defines gateway request and response models shared across handlers and domain services.
package api

type SearchResponse struct {
	Total int          `json:"total"`
	Items []SearchItem `json:"items"`
}

type SearchItem interface {
	searchItem()
}

type BaseSearchItem struct {
	Type ResourceType `json:"type"`
	ID   string       `json:"id"`
	Name string       `json:"name"`
	Href string       `json:"href"`
}

type DatasetItem struct {
	BaseSearchItem
	DataProviders []string `json:"data_providers,omitempty"`
	Concepts      []string `json:"concepts,omitempty"`
	// TODO: the operations that are available for this dataset, to be aggregated from the details of the dataset -> distribution -> ids of the operations
	Operations []string `json:"operations"`
}

func (DatasetItem) searchItem() {}

type SoftwareSourceCodeItem struct {
	BaseSearchItem
	Concepts []string `json:"concepts,omitempty"`
}

func (SoftwareSourceCodeItem) searchItem() {}

type SoftwareApplicationItem struct {
	BaseSearchItem
	Concepts []string `json:"concepts,omitempty"`
}

func (SoftwareApplicationItem) searchItem() {}

type WebServiceItem struct {
	BaseSearchItem
	ServiceProviders []string `json:"service_providers,omitempty"`
	Concepts         []string `json:"concepts,omitempty"`
}

func (WebServiceItem) searchItem() {}

type OrganizationItem struct {
	BaseSearchItem
}

func (OrganizationItem) searchItem() {}
