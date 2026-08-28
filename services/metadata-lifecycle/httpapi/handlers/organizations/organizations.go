// Package organizations exposes public canonical organization endpoints.
package organizations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	organizationdomain "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/organizations"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("organizations-handler")

type service interface {
	List(context.Context) ([]organizationdomain.Organization, error)
	Get(context.Context, uuid.UUID) (organizationdomain.Organization, error)
	Create(context.Context, string, organizationdomain.CreateRequest) (organizationdomain.Organization, error)
	Replace(context.Context, string, uuid.UUID, organizationdomain.ReplaceRequest) (organizationdomain.Organization, error)
	Delete(context.Context, string, uuid.UUID) error
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	var body createOrganizationRequest
	if !transport.Decode(w, r, &body) {
		return
	}

	identifiers := make([]organizationdomain.IdentifierRequest, 0, len(body.Identifiers))
	for _, item := range body.Identifiers {
		identifiers = append(identifiers, organizationdomain.IdentifierRequest{
			Type:  item.Type,
			Value: item.Value,
		})
	}
	item, err := h.service.Create(r.Context(), actorID, organizationdomain.CreateRequest{
		Name:            body.Name,
		Street:          body.Address.Street,
		Locality:        body.Address.Locality,
		PostalCode:      body.Address.PostalCode,
		Country:         body.Address.Country,
		LogoURL:         body.LogoURL,
		PageURL:         body.PageURL,
		Email:           body.Email,
		LEICode:         body.LEICode,
		Identifiers:     identifiers,
		Parents:         body.Parents,
		ContactPointIDs: body.ContactPointIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, organizationdomain.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		case errors.Is(err, organizationdomain.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "organization management permission required")
		case errors.Is(err, organizationdomain.ErrConflict):
			transport.WriteProblem(w, http.StatusConflict, "conflict", "organization already exists")
		case errors.Is(err, organizationdomain.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "organization_not_found", "referenced organization or contact point not found")
		default:
			log.ErrorContext(r.Context(), "create organization failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "create organization failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusCreated, response(item))
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

func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization id")
		return
	}

	var body replaceOrganizationRequest
	if !transport.Decode(w, r, &body) {
		return
	}
	if !body.valid() {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "all organization fields are required")
		return
	}

	identifiers := make([]organizationdomain.IdentifierRequest, 0, len(*body.Identifiers))
	for _, item := range *body.Identifiers {
		identifiers = append(identifiers, organizationdomain.IdentifierRequest{
			Type:  item.Type,
			Value: item.Value,
		})
	}
	item, err := h.service.Replace(r.Context(), actorID, id, organizationdomain.ReplaceRequest{
		Name:            *body.Name,
		Street:          body.Address.Street.Value,
		Locality:        body.Address.Locality.Value,
		PostalCode:      body.Address.PostalCode.Value,
		Country:         body.Address.Country.Value,
		LogoURL:         body.LogoURL.Value,
		PageURL:         body.PageURL.Value,
		Email:           body.Email.Value,
		LEICode:         body.LEICode.Value,
		Identifiers:     identifiers,
		Parents:         *body.Parents,
		ContactPointIDs: *body.ContactPointIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, organizationdomain.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		case errors.Is(err, organizationdomain.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "organization management permission required")
		case errors.Is(err, organizationdomain.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "organization_not_found", "organization not found")
		case errors.Is(err, organizationdomain.ErrConflict):
			transport.WriteProblem(w, http.StatusConflict, "conflict", "organization already exists")
		default:
			log.ErrorContext(r.Context(), "replace organization failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "replace organization failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusOK, response(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization id")
		return
	}

	if err := h.service.Delete(r.Context(), actorID, id); err != nil {
		switch {
		case errors.Is(err, organizationdomain.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		case errors.Is(err, organizationdomain.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "organization management permission required")
		case errors.Is(err, organizationdomain.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "organization_not_found", "organization not found")
		default:
			log.ErrorContext(r.Context(), "delete organization failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "delete organization failed")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type organizationResponse struct {
	ID                    uuid.UUID    `json:"id"`
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
	ID    uuid.UUID `json:"id"`
	Type  string    `json:"type"`
	Value string    `json:"value"`
}

type createOrganizationRequest struct {
	Name            string                    `json:"name"`
	Address         organizationAddress       `json:"address"`
	LogoURL         *string                   `json:"logo_url"`
	PageURL         *string                   `json:"page_url"`
	Email           *string                   `json:"email"`
	LEICode         *string                   `json:"lei_code"`
	Identifiers     []createIdentifierRequest `json:"identifiers"`
	Parents         []uuid.UUID               `json:"parents"`
	ContactPointIDs []uuid.UUID               `json:"contact_point_ids"`
}

type organizationAddress struct {
	Street     *string `json:"street"`
	Locality   *string `json:"locality"`
	PostalCode *string `json:"postal_code"`
	Country    *string `json:"country"`
}

type createIdentifierRequest struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type replaceOrganizationRequest struct {
	Name            *string                     `json:"name"`
	Address         *replaceOrganizationAddress `json:"address"`
	LogoURL         optionalString              `json:"logo_url"`
	PageURL         optionalString              `json:"page_url"`
	Email           optionalString              `json:"email"`
	LEICode         optionalString              `json:"lei_code"`
	Identifiers     *[]createIdentifierRequest  `json:"identifiers"`
	Parents         *[]uuid.UUID                `json:"parents"`
	ContactPointIDs *[]uuid.UUID                `json:"contact_point_ids"`
}

type replaceOrganizationAddress struct {
	Street     optionalString `json:"street"`
	Locality   optionalString `json:"locality"`
	PostalCode optionalString `json:"postal_code"`
	Country    optionalString `json:"country"`
}

type optionalString struct {
	Set   bool
	Value *string
}

func (s *optionalString) UnmarshalJSON(data []byte) error {
	s.Set = true
	if string(data) == "null" {
		s.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode string: %w", err)
	}
	s.Value = &value
	return nil
}

func (r replaceOrganizationRequest) valid() bool {
	return r.Name != nil && r.Address != nil &&
		r.Address.Street.Set && r.Address.Locality.Set && r.Address.PostalCode.Set && r.Address.Country.Set &&
		r.LogoURL.Set && r.PageURL.Set && r.Email.Set && r.LEICode.Set &&
		r.Identifiers != nil && r.Parents != nil && r.ContactPointIDs != nil
}

func response(value organizationdomain.Organization) organizationResponse {
	identifiers := make([]identifier, 0, len(value.Identifiers))
	for _, item := range value.Identifiers {
		identifiers = append(identifiers, identifier{
			ID:    item.ID,
			Type:  item.Type,
			Value: item.Value,
		})
	}

	return organizationResponse{
		ID:   value.ID,
		Name: value.Name,
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
