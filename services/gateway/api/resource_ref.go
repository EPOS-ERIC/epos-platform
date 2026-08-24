package api

type ResourceRef struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Href string `json:"href"`
}
