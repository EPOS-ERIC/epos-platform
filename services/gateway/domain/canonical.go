package domain

type ContactPoint struct {
	ID        string
	SourceIRI string
	Email     string
	Role      string
}

type OrganizationAddress struct {
	Street     *string
	Locality   *string
	PostalCode *string
	Country    *string
}

type OrganizationIdentifier struct {
	ID        string
	SourceIRI string
	Type      string
	Value     string
}

type CanonicalOrganization struct {
	ID              string
	SourceIRI       string
	Name            string
	Address         OrganizationAddress
	LogoURL         *string
	PageURL         *string
	Email           *string
	LEICode         *string
	Identifiers     []OrganizationIdentifier
	Parents         []string
	ContactPointIDs []string
}

type OrganizationIdentifierInput struct {
	SourceIRI *string
	Type      string
	Value     string
}

type CreateOrganizationInput struct {
	SourceIRI       *string
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
	SourceIRI *string
	Email     string
	Role      string
}

type ReplaceContactPointInput struct {
	Email string
	Role  string
}
