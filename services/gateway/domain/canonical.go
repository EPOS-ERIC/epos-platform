package domain

type ContactPoint struct {
	ID    string
	Email string
	Role  string
}

type OrganizationAddress struct {
	Street     *string
	Locality   *string
	PostalCode *string
	Country    *string
}

type CanonicalOrganization struct {
	ID              string
	Name            string
	Address         OrganizationAddress
	LogoURL         *string
	PageURL         *string
	Email           *string
	LEICode         *string
	Identifiers     []Identifier
	Parents         []string
	ContactPointIDs []string
}

type OrganizationIdentifierInput struct {
	Type  string
	Value string
}

type CreateOrganizationInput struct {
	Name            string
	Address         *OrganizationAddress
	LogoURL         *string
	PageURL         *string
	Email           *string
	LEICode         *string
	Identifiers     []OrganizationIdentifierInput
	Parents         []string
	ContactPointIDs []string
}

type ReplaceOrganizationInput struct {
	Name            string
	Address         OrganizationAddress
	LogoURL         *string
	PageURL         *string
	Email           *string
	LEICode         *string
	Identifiers     []OrganizationIdentifierInput
	Parents         []string
	ContactPointIDs []string
}

type CreateContactPointInput struct {
	Email string
	Role  string
}

type ReplaceContactPointInput struct {
	Email string
	Role  string
}
