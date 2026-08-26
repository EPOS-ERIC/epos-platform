package api

type Problem struct {
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Detail      string `json:"detail,omitempty"`
	DebugDetail string `json:"debug_detail,omitempty"`
}

// Reference identifies another API resource by ID and display name.
type Reference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
