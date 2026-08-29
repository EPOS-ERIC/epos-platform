// Package softwareapplication exposes software application HTTP handlers.
package softwareapplication

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("software-application-handler")

// Service is the software application lifecycle behavior used by Handler.
type Service interface {
	CreateDraft(context.Context, domain.Actor, uuid.UUID, domain.SoftwareApplicationInput) (domain.SoftwareApplicationVersion, error)
	Read(context.Context, *domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, bool, error)
	ReplaceDraft(context.Context, domain.Actor, uuid.UUID, uuid.UUID, domain.SoftwareApplicationDraftInput) (domain.SoftwareApplicationVersion, error)
	Submit(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Withdraw(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Reject(context.Context, domain.Actor, uuid.UUID, uuid.UUID, domain.SoftwareApplicationRejectionInput) (domain.SoftwareApplicationVersion, error)
	Reopen(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Publish(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Unpublish(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	DraftFromPublication(context.Context, domain.Actor, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	DraftFromLatestArchive(context.Context, domain.Actor, uuid.UUID) (domain.SoftwareApplicationVersion, error)
	Discard(context.Context, domain.Actor, uuid.UUID, uuid.UUID) error
	CopyPublished(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareApplicationVersion, error)
}

type Handler struct {
	service Service
}

// New creates a software application HTTP handler.
func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.ActorValue(w, r)
	if !ok {
		return
	}
	var request createRequest
	if !transport.Decode(w, r, &request) {
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || groupID == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group_id")
		return
	}
	value, err := request.Metadata.input()
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	application, err := h.service.CreateDraft(r.Context(), actor, groupID, value)
	h.writeApplication(r, w, application, true, err, http.StatusCreated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	applicationID, ok := applicationID(w, r)
	if !ok {
		return
	}
	versionID, ok := versionID(w, r)
	if !ok {
		return
	}
	actor, ok := transport.OptionalActorValue(w, r)
	if !ok {
		return
	}

	application, includeContacts, err := h.service.Read(r.Context(), actor, applicationID, versionID)
	h.writeApplication(r, w, application, includeContacts, err, http.StatusOK)
}

func (h *Handler) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	actor, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	var request replaceDraftRequest
	if !transport.Decode(w, r, &request) {
		return
	}

	value, err := request.input()
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	application, err := h.service.ReplaceDraft(r.Context(), actor, applicationID, versionID, value)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	actor, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Submit(r.Context(), actor, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	actor, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Withdraw(r.Context(), actor, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	actor, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	var request rejectRequest
	if !transport.Decode(w, r, &request) {
		return
	}
	value, err := request.input()
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	application, err := h.service.Reject(r.Context(), actor, applicationID, versionID, value)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Reopen(w http.ResponseWriter, r *http.Request) {
	actor, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Reopen(r.Context(), actor, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	actor, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Publish(r.Context(), actor, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	actor, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Unpublish(r.Context(), actor, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.ActorValue(w, r)
	if !ok {
		return
	}
	applicationID, ok := applicationID(w, r)
	if !ok {
		return
	}
	application, err := h.service.DraftFromPublication(r.Context(), actor, applicationID)
	h.writeApplication(r, w, application, true, err, http.StatusCreated)
}

func (h *Handler) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.ActorValue(w, r)
	if !ok {
		return
	}
	applicationID, ok := applicationID(w, r)
	if !ok {
		return
	}
	application, err := h.service.DraftFromLatestArchive(r.Context(), actor, applicationID)
	h.writeApplication(r, w, application, true, err, http.StatusCreated)
}

func (h *Handler) Discard(w http.ResponseWriter, r *http.Request) {
	actor, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	if err := h.service.Discard(r.Context(), actor, applicationID, versionID); err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Copy(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.ActorValue(w, r)
	if !ok {
		return
	}
	sourceID, ok := applicationID(w, r)
	if !ok {
		return
	}
	targetGroupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group_id")
		return
	}
	application, err := h.service.CopyPublished(r.Context(), actor, sourceID, targetGroupID)
	h.writeApplication(r, w, application, true, err, http.StatusCreated)
}

func (h *Handler) writeApplication(r *http.Request, w http.ResponseWriter, application domain.SoftwareApplicationVersion, includeContacts bool, err error, status int) {
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	transport.WriteJSON(w, status, applicationResponse(application, includeContacts))
}

func commandIDs(w http.ResponseWriter, r *http.Request) (domain.Actor, uuid.UUID, uuid.UUID, bool) {
	actor, ok := transport.ActorValue(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	applicationID, ok := applicationID(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	versionID, ok := versionID(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	return actor, applicationID, versionID, true
}

func applicationID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(w, chi.URLParam(r, "resource_id"), "software application")
}

func versionID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(w, chi.URLParam(r, "version_id"), "version")
}

func pathUUID(w http.ResponseWriter, value, resource string) (uuid.UUID, bool) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid "+resource+" id")
		return uuid.Nil, false
	}
	return id, true
}

func writeLifecycleError(r *http.Request, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrSoftwareApplicationForbidden):
		transport.WriteProblem(w, http.StatusForbidden, "forbidden", "access denied")
	case errors.Is(err, domain.ErrSoftwareApplicationNotFound):
		transport.WriteProblem(w, http.StatusNotFound, "software_application_not_found", "software application or version not found")
	case errors.Is(err, domain.ErrSoftwareApplicationConflict):
		transport.WriteProblem(w, http.StatusConflict, "conflict", "software application lifecycle conflict")
	case errors.Is(err, domain.ErrSoftwareApplicationInvalid):
		detail := err.Error()
		if index := strings.LastIndex(detail, ": "); index > 0 {
			detail = detail[:index]
		}
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", detail)
	case errors.Is(err, domain.ErrSoftwareApplicationInternal):
		log.ErrorContext(r.Context(), "software application request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "software application request failed")
	default:
		log.ErrorContext(r.Context(), "software application request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "software application request failed")
	}
}
