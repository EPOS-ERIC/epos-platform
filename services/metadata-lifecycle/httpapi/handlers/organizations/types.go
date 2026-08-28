package organizations

import "github.com/google/uuid"

type organizationResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	// TODO: no reason to have the address be a sub object, let's inline the fields and keep it simple
	Address               address      `json:"address"`
	LogoURL               *string      `json:"logo_url"`
	PageURL               *string      `json:"page_url"`
	Email                 *string      `json:"email"`
	LEICode               *string      `json:"lei_code"`
	Identifiers           []identifier `json:"identifiers"`
	ParentOrganizationIDs []uuid.UUID  `json:"parents"`
	ContactPointIDs       []uuid.UUID  `json:"contact_point_ids"`
}
type address struct {
	Street     *string `json:"street"`
	Locality   *string `json:"locality"`
	PostalCode *string `json:"postal_code"`
	Country    *string `json:"country"`
}
type identifier struct {
	ID    uuid.UUID `json:"id"`
	Type  string    `json:"type"`
	Value string    `json:"value"`
}

type organizationRequest struct {
	Name            string                     `json:"name"`
	Address         organizationRequestAddress `json:"address"`
	LogoURL         *string                    `json:"logo_url"`
	PageURL         *string                    `json:"page_url"`
	Email           *string                    `json:"email"`
	LEICode         *string                    `json:"lei_code"`
	Identifiers     []identifierRequest        `json:"identifiers"`
	Parents         []uuid.UUID                `json:"parents"`
	ContactPointIDs []uuid.UUID                `json:"contact_point_ids"`
}
type organizationRequestAddress struct {
	Street     *string `json:"street"`
	Locality   *string `json:"locality"`
	PostalCode *string `json:"postal_code"`
	Country    *string `json:"country"`
}
type identifierRequest struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
