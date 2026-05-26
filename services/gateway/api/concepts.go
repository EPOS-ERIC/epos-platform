package api

type Concept struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Children    []Concept `json:"children,omitempty"`
}

type ConceptScheme struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Logo        string    `json:"logo,omitempty"`
	Homepage    string    `json:"homepage,omitempty"`
	Color       string    `json:"color,omitempty"`
	OrderNumber int32     `json:"order_number,omitempty"`
	Type        string    `json:"type"`
	Children    []Concept `json:"children,omitempty"`
}

type ConceptTreeResponse struct {
	Schemes []ConceptScheme `json:"schemes"`
}
