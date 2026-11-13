package metadatasync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

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

// syncOrganizations fetches every organization page from the remote API and
// applies the result as a single snapshot. It returns the number of
// organizations collected and the number of pages requested.
func (w *Worker) syncOrganizations(ctx context.Context) (int, int, error) {
	endpoint, err := url.JoinPath(w.config.BaseURL, organizationsPath)
	if err != nil {
		return 0, 0, fmt.Errorf("build organization URL: %w", err)
	}

	values := make([]domain.SyncedOrganization, 0)

	// TODO: when we will have a way to know the total number of pages we can make this nicer
	for page := 0; ; page++ {
		nodes, err := w.fetchOrganizationPage(ctx, endpoint, page)
		if err != nil {
			return len(values), page, err
		}

		pages := page + 1

		if len(nodes) == 0 {
			if err := w.metadata.SyncOrganizations(ctx, values); err != nil {
				return len(values), pages, fmt.Errorf("apply organization snapshot: %w", err)
			}

			return len(values), pages, nil
		}

		for _, node := range nodes {
			org, err := node.toDomain()
			if err != nil {
				return len(values), pages, fmt.Errorf("invalid organization on page %d: %w", page, err)
			}
			values = append(values, org)
		}
	}
}

// fetchOrganizationPage requests and decodes a single page of organizations.
func (w *Worker) fetchOrganizationPage(ctx context.Context, endpoint string, page int) ([]remoteOrganization, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse organization URL: %w", err)
	}

	query := parsed.Query()
	query.Set("_format", "json")
	query.Set("page", strconv.Itoa(page))
	parsed.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create organization request: %w", err)
	}
	req.SetBasicAuth(w.config.Username, w.password)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request organization page %d: %w", page, err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("organization page %d returned HTTP %d", page, resp.StatusCode)
	}

	nodes, decodeErr := decodeOrganizations(resp.Body)
	closeErr := resp.Body.Close()

	if decodeErr != nil {
		return nil, fmt.Errorf("decode organization page %d: invalid response", page)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close organization page %d: %w", page, closeErr)
	}

	return nodes, nil
}

// decodeOrganizations strictly decodes a JSON array of organizations,
// rejecting a null body and any data after the array.
func decodeOrganizations(r io.Reader) ([]remoteOrganization, error) {
	var nodes []remoteOrganization

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&nodes); err != nil {
		return nil, err
	}
	if nodes == nil {
		return nil, errors.New("null response")
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, errors.New("trailing data")
		}
		return nil, err
	}

	return nodes, nil
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
		org.Street = &addr.Street
		org.Locality = &addr.Locality
		org.PostalCode = &addr.PostalCode
		org.Country = &addr.Country
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
