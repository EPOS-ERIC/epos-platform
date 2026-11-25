package metadatasync

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

const organizationsPath = "api/organizations"

// Remote API payload types
type remoteInt struct {
	Value int64 `json:"value"`
}
type remoteString struct {
	Value string `json:"value"`
}
type remoteLink struct {
	URI string `json:"uri"`
}
type remoteParent struct {
	ID int64 `json:"target_id"`
}
type remoteAddress struct {
	Street     string `json:"address_line1"`
	Locality   string `json:"locality"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country_code"`
}

type remoteOrganization struct {
	NID     []remoteInt     `json:"nid"`
	Title   []remoteString  `json:"title"`
	Address []remoteAddress `json:"field_full_address"`
	Logo    []remoteLink    `json:"field_logo"`
	Website []remoteLink    `json:"field_website"`
	Email   []remoteString  `json:"field_email"`
	Parent  []remoteParent  `json:"field_parent_org"`
	DOI     []remoteString  `json:"field_doi"`
	ISNI    []remoteString  `json:"field_isni"`
	MID     []remoteString  `json:"field_mid"`
	ROR     []remoteString  `json:"field_ror"`
	PIC     []remoteString  `json:"field_pic"`
	VIAF    []remoteString  `json:"field_viaf"`
}

func (w *Worker) fetchOrganizations(ctx context.Context) ([]domain.SyncedOrganization, int, error) {
	rows, pages, err := fetchSnapshot[remoteOrganization](ctx, w, organizationsPath)
	if err != nil {
		return nil, pages, err
	}

	values := make([]domain.SyncedOrganization, 0, len(rows))
	for i, row := range rows {
		org, err := row.toDomain()
		if err != nil {
			return values, pages, fmt.Errorf("invalid organization at row %d: %w", i, err)
		}

		values = append(values, org)
	}

	return values, pages, nil
}

// toDomain validates the remote payload and converts it to a domain value.
func (n remoteOrganization) toDomain() (domain.SyncedOrganization, error) {
	if len(n.NID) != 1 || len(n.Title) != 1 {
		return domain.SyncedOrganization{}, errors.New("missing NID or title")
	}
	if len(n.Address) > 1 || len(n.Logo) > 1 || len(n.Website) > 1 || len(n.Email) > 1 {
		return domain.SyncedOrganization{}, errors.New("unexpected multiple values")
	}

	org := domain.SyncedOrganization{
		NID:     n.NID[0].Value,
		Name:    n.Title[0].Value,
		LogoURL: linkURI(n.Logo),
		PageURL: linkURI(n.Website),
		Email:   stringValue(n.Email),
	}

	if len(n.Address) == 1 {
		addr := n.Address[0]
		org.Street = nonBlank(addr.Street)
		org.Locality = nonBlank(addr.Locality)
		org.PostalCode = nonBlank(addr.PostalCode)
		org.Country = nonBlank(addr.Country)
	}

	for _, parent := range n.Parent {
		org.ParentNIDs = append(org.ParentNIDs, parent.ID)
	}

	identifiers := map[string][]remoteString{
		"doi":  n.DOI,
		"isni": n.ISNI,
		"mid":  n.MID,
		"ror":  n.ROR,
		"pic":  n.PIC,
		"viaf": n.VIAF,
	}
	for typeName, values := range identifiers {
		if len(values) > 1 {
			return domain.SyncedOrganization{}, fmt.Errorf("multiple %s identifiers", typeName)
		}

		if len(values) == 1 {
			org.Identifiers = append(org.Identifiers, domain.OrganizationIdentifier{
				Type:  typeName,
				Value: values[0].Value,
			})
		}
	}

	return org, nil
}

func nonBlank(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

// linkURI returns the URI of the only link in links, or nil if there is none.
func linkURI(links []remoteLink) *string {
	if len(links) != 1 {
		return nil
	}

	return &links[0].URI
}

// stringValue returns the value of the only string in values, or nil if there is none.
func stringValue(values []remoteString) *string {
	if len(values) != 1 {
		return nil
	}

	return &values[0].Value
}
