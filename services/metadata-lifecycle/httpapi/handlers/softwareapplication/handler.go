// Package softwareapplication exposes software application HTTP handlers.
package softwareapplication

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwarelifecycle"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("software-application-handler")

// Service is the software application lifecycle behavior used by Handler.
type Service interface {
	CreateDraft(context.Context, string, uuid.UUID, metadata.SoftwareApplication) (softwarelifecycle.Application, error)
	Read(context.Context, *string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, bool, error)
	ReplaceDraft(context.Context, string, uuid.UUID, uuid.UUID, int32, metadata.SoftwareApplication) (softwarelifecycle.Application, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, error)
	Withdraw(context.Context, string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, string) (softwarelifecycle.Application, error)
	Reopen(context.Context, string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, error)
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, error)
	DraftFromPublication(context.Context, string, uuid.UUID) (softwarelifecycle.Application, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (softwarelifecycle.Application, error)
	Discard(context.Context, string, uuid.UUID, uuid.UUID) error
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, error)
}

type Handler struct {
	service Service
}

// New creates a software application HTTP handler.
func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	var request createRequest
	if !transport.Decode(w, r, &request) {
		return
	}
	// TODO: use an helper for stuff like this
	groupID := request.GroupID
	if groupIDPath := chi.URLParam(r, "group_id"); groupIDPath != "" {
		parsed, parseErr := uuid.Parse(groupIDPath)
		if parseErr != nil {
			groupID = uuid.Nil
		} else {
			groupID = parsed
		}
	}
	if groupID == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group_id")
		return
	}
	application, err := h.service.CreateDraft(r.Context(), actorID, groupID, request.Metadata.domain())
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
	actorID, present, ok := transport.OptionalActor(w, r)
	if !ok {
		return
	}
	var actor *string
	if present {
		actor = &actorID
	}

	application, includeContacts, err := h.service.Read(r.Context(), actor, applicationID, versionID)
	h.writeApplication(r, w, application, includeContacts, err, http.StatusOK)
}

func (h *Handler) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	actorID, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	var request replaceDraftRequest
	if !transport.Decode(w, r, &request) {
		return
	}

	application, err := h.service.ReplaceDraft(r.Context(), actorID, applicationID, versionID, request.Revision, request.Metadata.domain())
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	actorID, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Submit(r.Context(), actorID, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	actorID, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Withdraw(r.Context(), actorID, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	actorID, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	var request rejectRequest
	if !transport.Decode(w, r, &request) {
		return
	}
	application, err := h.service.Reject(r.Context(), actorID, applicationID, versionID, request.Note)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Reopen(w http.ResponseWriter, r *http.Request) {
	actorID, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Reopen(r.Context(), actorID, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	actorID, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Publish(r.Context(), actorID, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	actorID, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	application, err := h.service.Unpublish(r.Context(), actorID, applicationID, versionID)
	h.writeApplication(r, w, application, true, err, http.StatusOK)
}

func (h *Handler) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	applicationID, ok := applicationID(w, r)
	if !ok {
		return
	}
	application, err := h.service.DraftFromPublication(r.Context(), actorID, applicationID)
	h.writeApplication(r, w, application, true, err, http.StatusCreated)
}

func (h *Handler) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	applicationID, ok := applicationID(w, r)
	if !ok {
		return
	}
	application, err := h.service.DraftFromLatestArchive(r.Context(), actorID, applicationID)
	h.writeApplication(r, w, application, true, err, http.StatusCreated)
}

func (h *Handler) Discard(w http.ResponseWriter, r *http.Request) {
	actorID, applicationID, versionID, ok := commandIDs(w, r)
	if !ok {
		return
	}
	if err := h.service.Discard(r.Context(), actorID, applicationID, versionID); err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Copy(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
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
	application, err := h.service.CopyPublished(r.Context(), actorID, sourceID, targetGroupID)
	h.writeApplication(r, w, application, true, err, http.StatusCreated)
}

func (h *Handler) writeApplication(r *http.Request, w http.ResponseWriter, application softwarelifecycle.Application, includeContacts bool, err error, status int) {
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	transport.WriteJSON(w, status, applicationResponse(application, includeContacts))
}

func commandIDs(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, uuid.UUID, bool) {
	actorID, ok := transport.Actor(w, r)
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
	return actorID, applicationID, versionID, true
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
	case errors.Is(err, softwarelifecycle.ErrDenied):
		transport.WriteProblem(w, http.StatusForbidden, "forbidden", "access denied")
	case errors.Is(err, softwarelifecycle.ErrNotFound):
		transport.WriteProblem(w, http.StatusNotFound, "software_application_not_found", "software application or version not found")
	case errors.Is(err, softwarelifecycle.ErrConflict):
		transport.WriteProblem(w, http.StatusConflict, "conflict", "software application lifecycle conflict")
	case errors.Is(err, softwarelifecycle.ErrInvalid):
		detail := err.Error()
		if index := strings.LastIndex(detail, ": "); index > 0 {
			detail = detail[:index]
		}
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", detail)
	case errors.Is(err, softwarelifecycle.ErrInternal):
		log.ErrorContext(r.Context(), "software application request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "software application request failed")
	default:
		log.ErrorContext(r.Context(), "software application request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "software application request failed")
	}
}
