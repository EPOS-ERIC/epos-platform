package domain

// OrganizationInput contains fields supplied by an organization editor.
type OrganizationInput struct {
	Name        string
	Street      *string
	Locality    *string
	PostalCode  *string
	Country     *string
	LogoURL     *string
	PageURL     *string
	Email       *string
	Identifiers []OrganizationIdentifier
	Parents     []string
}

type OrganizationIdentifier struct {
	Type  string
	Value string
}

// SyncedOrganization contains remote-owned organization fields.
type SyncedOrganization struct {
	NID         int64
	ParentNIDs  []int64
	Identifiers []OrganizationIdentifier
	Name        string
	Street      *string
	Locality    *string
	PostalCode  *string
	Country     *string
	LogoURL     *string
	PageURL     *string
	Email       *string
}
