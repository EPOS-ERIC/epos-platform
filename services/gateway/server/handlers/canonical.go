package handlers

import (
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/canonical"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Canonical struct {
	service *canonical.Service
}

func NewCanonical(service *canonical.Service) *Canonical {
	return &Canonical{service: service}
}

func (h *Canonical) ListContactPoints(w http.ResponseWriter, r *http.Request) {
	actor, ok := authenticatedActor(r, w)
	if !ok {
		return
	}
	points, err := h.service.ListContactPoints(r.Context(), actor.ID)
	if err != nil {
		writeCanonicalError(r, w, err, "list contact points")
		return
	}

	result := mapSlice(points, toAPIContactPoint)
	writeJSON(w, http.StatusOK, result)
}

func (h *Canonical) GetContactPoint(w http.ResponseWriter, r *http.Request) {
	actor, ok := authenticatedActor(r, w)
	if !ok {
		return
	}
	id, ok := uuidParam(r.Context(), w, r, "contactPointId")
	if !ok {
		return
	}
	point, err := h.service.GetContactPoint(r.Context(), actor.ID, id)
	if err != nil {
		writeCanonicalError(r, w, err, "get contact point")
		return
	}
	writeJSON(w, http.StatusOK, toAPIContactPoint(point))
}

func (h *Canonical) CreateContactPoint(w http.ResponseWriter, r *http.Request) {
	actor, ok := authenticatedActor(r, w)
	if !ok {
		return
	}
	var request api.CreateContactPointRequest
	if !decodeJSON(r, w, &request) {
		return
	}

	point, err := h.service.CreateContactPoint(r.Context(), actor.ID, toDomainCreateContactPointInput(request))
	if err != nil {
		writeCanonicalError(r, w, err, "create contact point")
		return
	}
	writeJSON(w, http.StatusCreated, toAPIContactPoint(point))
}

func (h *Canonical) ReplaceContactPoint(w http.ResponseWriter, r *http.Request) {
	actor, ok := authenticatedActor(r, w)
	if !ok {
		return
	}
	id, ok := uuidParam(r.Context(), w, r, "contactPointId")
	if !ok {
		return
	}
	var request api.ReplaceContactPointRequest
	if !decodeJSON(r, w, &request) {
		return
	}

	point, err := h.service.ReplaceContactPoint(r.Context(), actor.ID, id, toDomainReplaceContactPointInput(request))
	if err != nil {
		writeCanonicalError(r, w, err, "replace contact point")
		return
	}
	writeJSON(w, http.StatusOK, toAPIContactPoint(point))
}

func (h *Canonical) DeleteContactPoint(w http.ResponseWriter, r *http.Request) {
	actor, ok := authenticatedActor(r, w)
	if !ok {
		return
	}
	id, ok := uuidParam(r.Context(), w, r, "contactPointId")
	if !ok {
		return
	}
	if err := h.service.DeleteContactPoint(r.Context(), actor.ID, id); err != nil {
		writeCanonicalError(r, w, err, "delete contact point")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Canonical) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	organizations, err := h.service.ListOrganizations(r.Context())
	if err != nil {
		writeCanonicalError(r, w, err, "list organizations")
		return
	}

	result := mapSlice(organizations, toAPICanonicalOrganization)
	writeJSON(w, http.StatusOK, result)
}

func (h *Canonical) GetOrganization(w http.ResponseWriter, r *http.Request) {
	id, ok := uuidParam(r.Context(), w, r, "organizationId")
	if !ok {
		return
	}
	organization, err := h.service.GetOrganization(r.Context(), id)
	if err != nil {
		writeCanonicalError(r, w, err, "get organization")
		return
	}
	writeJSON(w, http.StatusOK, toAPICanonicalOrganization(organization))
}

func (h *Canonical) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	actor, ok := authenticatedActor(r, w)
	if !ok {
		return
	}
	var request api.CreateOrganizationRequest
	if !decodeJSON(r, w, &request) {
		return
	}
	organization, err := h.service.CreateOrganization(r.Context(), actor.ID, toDomainCreateOrganizationInput(request))
	if err != nil {
		writeCanonicalError(r, w, err, "create organization")
		return
	}
	writeJSON(w, http.StatusCreated, toAPICanonicalOrganization(organization))
}

func (h *Canonical) ReplaceOrganization(w http.ResponseWriter, r *http.Request) {
	actor, ok := authenticatedActor(r, w)
	if !ok {
		return
	}
	id, ok := uuidParam(r.Context(), w, r, "organizationId")
	if !ok {
		return
	}
	var request api.ReplaceOrganizationRequest
	if !decodeJSON(r, w, &request) {
		return
	}
	organization, err := h.service.ReplaceOrganization(r.Context(), actor.ID, id, toDomainReplaceOrganizationInput(request))
	if err != nil {
		writeCanonicalError(r, w, err, "replace organization")
		return
	}
	writeJSON(w, http.StatusOK, toAPICanonicalOrganization(organization))
}

func (h *Canonical) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	actor, ok := authenticatedActor(r, w)
	if !ok {
		return
	}
	id, ok := uuidParam(r.Context(), w, r, "organizationId")
	if !ok {
		return
	}
	if err := h.service.DeleteOrganization(r.Context(), actor.ID, id); err != nil {
		writeCanonicalError(r, w, err, "delete organization")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func authenticatedActor(r *http.Request, w http.ResponseWriter) (api.User, bool) {
	actor, ok := AuthenticatedUserFromContext(r.Context())
	if !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
	}
	return actor, ok
}

//nolint:dupl // Error mapping intentionally mirrors other gateway handler boundaries.
func writeCanonicalError(r *http.Request, w http.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid canonical metadata request", err)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "canonical metadata operation is not permitted", nil)
	case errors.Is(err, domain.ErrNotFound):
		WriteError(r.Context(), w, http.StatusNotFound, "not_found", "canonical metadata resource not found", nil)
	case errors.Is(err, domain.ErrConflict):
		WriteError(r.Context(), w, http.StatusConflict, "conflict", "canonical metadata operation conflicts with existing data", nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "canonical metadata request timed out", err)
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
	}
}
