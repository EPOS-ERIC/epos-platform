package groups

import "github.com/google/uuid"

type groupResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Label       *string   `json:"label"`
	Logo        *string   `json:"logo"`
	Page        *string   `json:"page"`
	Color       *string   `json:"color"`
	Position    int32     `json:"position"`
	Concept     string    `json:"concept"`
}

type groupRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Label       *string `json:"label"`
	Logo        *string `json:"logo"`
	Page        *string `json:"page"`
	Color       *string `json:"color"`
	Position    int32   `json:"position"`
	Concept     string  `json:"concept"`
}
