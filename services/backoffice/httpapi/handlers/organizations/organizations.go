// Package organizations exposes public canonical organization endpoints.
package organizations

import (
	"context"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	organizationdomain "github.com/EPOS-ERIC/epos-platform/services/backoffice/organizations"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("organizations-handler")

type service interface {
	List(context.Context) ([]organizationdomain.Organization, error)
	Get(context.Context, uuid.UUID) (organizationdomain.Organization, error)
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		log.ErrorContext(r.Context(), "list organizations failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "list organizations failed")
		return
	}

	result := make([]organizationResponse, 0, len(items))
	for _, item := range items {
		result = append(result, response(item))
	}

	transport.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization id")
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, organizationdomain.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		case errors.Is(err, organizationdomain.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "organization_not_found", "organization not found")
		default:
			log.ErrorContext(r.Context(), "get organization failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "get organization failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusOK, response(item))
}

type organizationResponse struct {
	ID                    uuid.UUID    `json:"id"`
	SourceIRI             string       `json:"source_iri"`
	Name                  string       `json:"name"`
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
	ID        uuid.UUID `json:"id"`
	SourceIRI string    `json:"source_iri"`
	Type      string    `json:"type"`
	Value     string    `json:"value"`
}

func response(value organizationdomain.Organization) organizationResponse {
	identifiers := make([]identifier, 0, len(value.Identifiers))
	for _, item := range value.Identifiers {
		identifiers = append(identifiers, identifier{
			ID:        item.ID,
			SourceIRI: item.SourceIRI,
			Type:      item.Type,
			Value:     item.Value,
		})
	}

	return organizationResponse{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Name:      value.Name,
		Address: address{
			Street:     value.Street,
			Locality:   value.Locality,
			PostalCode: value.PostalCode,
			Country:    value.Country,
		},
		LogoURL:               value.LogoURL,
		PageURL:               value.PageURL,
		Email:                 value.Email,
		LEICode:               value.LEICode,
		Identifiers:           identifiers,
		ParentOrganizationIDs: value.ParentOrganizationIDs,
		ContactPointIDs:       value.ContactPointIDs,
	}
}
