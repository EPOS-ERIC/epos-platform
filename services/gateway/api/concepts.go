package api

type Concept struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	InScheme    *ResourceRef  `json:"in_scheme,omitempty"`
	Children    []ResourceRef `json:"children,omitempty"`
}

type ConceptScheme struct {
	ID          string        `json:"id"`
	Label       string        `json:"label"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Logo        string        `json:"logo,omitempty"`
	Homepage    string        `json:"homepage,omitempty"`
	Color       string        `json:"color,omitempty"`
	OrderNumber int32         `json:"order_number,omitempty"`
	TopConcepts []ResourceRef `json:"top_concepts,omitempty"`
}

type ConceptTreeResponse struct {
	Schemes []ConceptScheme `json:"schemes"`
}
