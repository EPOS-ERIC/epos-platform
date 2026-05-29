package api

type Problem struct {
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Detail      string `json:"detail,omitempty"`
	DebugDetail string `json:"debug_detail,omitempty"`
	Instance    string `json:"instance,omitempty"`
}

type BaseDetailResource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
