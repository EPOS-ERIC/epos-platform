package categories

import "github.com/google/uuid"

type categoryResponse struct {
	ID               uuid.UUID  `json:"id"`
	GroupID          uuid.UUID  `json:"group_id"`
	ParentCategoryID *uuid.UUID `json:"parent_category_id"`
	Name             string     `json:"name"`
}

type categoryRequest struct {
	Name             string     `json:"name"`
	ParentCategoryID *uuid.UUID `json:"parent_category_id"`
}
