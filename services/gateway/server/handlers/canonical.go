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

	result := make([]api.ContactPoint, 0, len(points))
	for _, point := range points {
		result = append(result, contactPointFromDomain(point))
	}
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
	writeJSON(w, http.StatusOK, contactPointFromDomain(point))
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

	point, err := h.service.CreateContactPoint(r.Context(), actor.ID, domain.CreateContactPointInput{
		SourceIRI: request.SourceIRI,
		Email:     request.Email,
		Role:      request.Role,
	})
	if err != nil {
		writeCanonicalError(r, w, err, "create contact point")
		return
	}
	writeJSON(w, http.StatusCreated, contactPointFromDomain(point))
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

	point, err := h.service.ReplaceContactPoint(r.Context(), actor.ID, id, domain.ReplaceContactPointInput{
		Email: request.Email,
		Role:  request.Role,
	})
	if err != nil {
		writeCanonicalError(r, w, err, "replace contact point")
		return
	}
	writeJSON(w, http.StatusOK, contactPointFromDomain(point))
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

	result := make([]api.CanonicalOrganization, 0, len(organizations))
	for _, organization := range organizations {
		result = append(result, organizationFromDomain(organization))
	}
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
	writeJSON(w, http.StatusOK, organizationFromDomain(organization))
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
	organization, err := h.service.CreateOrganization(r.Context(), actor.ID, createOrganizationInput(request))
	if err != nil {
		writeCanonicalError(r, w, err, "create organization")
		return
	}
	writeJSON(w, http.StatusCreated, organizationFromDomain(organization))
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
	organization, err := h.service.ReplaceOrganization(r.Context(), actor.ID, id, replaceOrganizationInput(request))
	if err != nil {
		writeCanonicalError(r, w, err, "replace organization")
		return
	}
	writeJSON(w, http.StatusOK, organizationFromDomain(organization))
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

func contactPointFromDomain(point domain.ContactPoint) api.ContactPoint {
	return api.ContactPoint{ID: point.ID, SourceIRI: point.SourceIRI, Email: point.Email, Role: point.Role}
}

func organizationFromDomain(organization domain.CanonicalOrganization) api.CanonicalOrganization {
	identifiers := make([]api.OrganizationIdentifier, 0, len(organization.Identifiers))
	for _, identifier := range organization.Identifiers {
		identifiers = append(identifiers, api.OrganizationIdentifier{ID: identifier.ID, SourceIRI: identifier.SourceIRI, Type: identifier.Type, Value: identifier.Value})
	}
	return api.CanonicalOrganization{
		ID: organization.ID, SourceIRI: organization.SourceIRI, Name: organization.Name,
		Address: api.OrganizationAddress{Street: organization.Address.Street, Locality: organization.Address.Locality, PostalCode: organization.Address.PostalCode, Country: organization.Address.Country},
		LogoURL: organization.LogoURL, PageURL: organization.PageURL, Email: organization.Email, LEICode: organization.LEICode,
		Identifiers: identifiers, Parents: organization.Parents, ContactPointIDs: organization.ContactPointIDs,
	}
}

func createOrganizationInput(request api.CreateOrganizationRequest) domain.CreateOrganizationInput {
	identifiers := make([]domain.OrganizationIdentifierInput, 0, len(request.Identifiers))
	for _, identifier := range request.Identifiers {
		identifiers = append(identifiers, domain.OrganizationIdentifierInput{SourceIRI: identifier.SourceIRI, Type: identifier.Type, Value: identifier.Value})
	}
	return domain.CreateOrganizationInput{
		SourceIRI: request.SourceIRI, Name: request.Name, Address: organizationAddressToDomain(request.Address),
		LogoURL: request.LogoURL, PageURL: request.PageURL, Email: request.Email, LEICode: request.LEICode,
		Identifiers: identifiers, Parents: request.Parents, ContactPointIDs: request.ContactPointIDs,
	}
}

func replaceOrganizationInput(request api.ReplaceOrganizationRequest) domain.ReplaceOrganizationInput {
	identifiers := make([]domain.OrganizationIdentifierInput, 0, len(request.Identifiers))
	for _, identifier := range request.Identifiers {
		identifiers = append(identifiers, domain.OrganizationIdentifierInput{SourceIRI: identifier.SourceIRI, Type: identifier.Type, Value: identifier.Value})
	}
	return domain.ReplaceOrganizationInput{
		Name: request.Name, Address: organizationAddressToDomainValue(request.Address), LogoURL: request.LogoURL, PageURL: request.PageURL,
		Email: request.Email, LEICode: request.LEICode, Identifiers: identifiers,
		Parents: request.Parents, ContactPointIDs: request.ContactPointIDs,
	}
}

func organizationAddressToDomain(address *api.OrganizationAddress) *domain.OrganizationAddress {
	if address == nil {
		return nil
	}
	value := organizationAddressToDomainValue(*address)
	return &value
}

func organizationAddressToDomainValue(address api.OrganizationAddress) domain.OrganizationAddress {
	return domain.OrganizationAddress{Street: address.Street, Locality: address.Locality, PostalCode: address.PostalCode, Country: address.Country}
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
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
	}
}
