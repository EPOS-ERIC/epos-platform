package api

type ContactPoint struct {
	ID        string `json:"id"`
	SourceIRI string `json:"source_iri"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

type OrganizationAddress struct {
	Street     *string `json:"street"`
	Locality   *string `json:"locality"`
	PostalCode *string `json:"postal_code"`
	Country    *string `json:"country"`
}

type OrganizationIdentifier struct {
	ID        string `json:"id"`
	SourceIRI string `json:"source_iri"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

type CanonicalOrganization struct {
	ID              string                   `json:"id"`
	SourceIRI       string                   `json:"source_iri"`
	Name            string                   `json:"name"`
	Address         OrganizationAddress      `json:"address"`
	LogoURL         *string                  `json:"logo_url"`
	PageURL         *string                  `json:"page_url"`
	Email           *string                  `json:"email"`
	LEICode         *string                  `json:"lei_code"`
	Identifiers     []OrganizationIdentifier `json:"identifiers"`
	Parents         []string                 `json:"parents"`
	ContactPointIDs []string                 `json:"contact_point_ids"`
}

type CreateContactPointRequest struct {
	SourceIRI *string `json:"source_iri,omitempty"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
}

type ReplaceContactPointRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type OrganizationIdentifierRequest struct {
	SourceIRI *string `json:"source_iri,omitempty"`
	Type      string  `json:"type"`
	Value     string  `json:"value"`
}

type CreateOrganizationRequest struct {
	SourceIRI       *string                         `json:"source_iri,omitempty"`
	Name            string                          `json:"name"`
	Address         *OrganizationAddress            `json:"address,omitempty"`
	LogoURL         *string                         `json:"logo_url"`
	PageURL         *string                         `json:"page_url"`
	Email           *string                         `json:"email"`
	LEICode         *string                         `json:"lei_code"`
	Identifiers     []OrganizationIdentifierRequest `json:"identifiers,omitempty"`
	Parents         []string                        `json:"parents,omitempty"`
	ContactPointIDs []string                        `json:"contact_point_ids,omitempty"`
}

type ReplaceOrganizationRequest struct {
	Name            string                          `json:"name"`
	Address         OrganizationAddress             `json:"address"`
	LogoURL         *string                         `json:"logo_url"`
	PageURL         *string                         `json:"page_url"`
	Email           *string                         `json:"email"`
	LEICode         *string                         `json:"lei_code"`
	Identifiers     []OrganizationIdentifierRequest `json:"identifiers"`
	Parents         []string                        `json:"parents"`
	ContactPointIDs []string                        `json:"contact_point_ids"`
}
