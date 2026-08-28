package contactpoints

import "github.com/google/uuid"

type contactPointResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

type contactPointRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}
