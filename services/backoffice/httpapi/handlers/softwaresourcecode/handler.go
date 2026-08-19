// Package softwaresourcecode exposes software source code HTTP handlers.
package softwaresourcecode

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/softwarelifecycle"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("software-source-code-handler")

type sourceCodeService interface {
	CreateDraft(context.Context, uuid.UUID, uuid.UUID, metadata.SoftwareSourceCode) (softwarelifecycle.SourceCode, error)
	Read(context.Context, *uuid.UUID, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, bool, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int32, metadata.SoftwareSourceCode) (softwarelifecycle.SourceCode, error)
	Submit(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, error)
	Reject(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (softwarelifecycle.SourceCode, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, error)
	Publish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, error)
	Unpublish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, error)
	DraftFromPublication(context.Context, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, error)
	DraftFromLatestArchive(context.Context, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, error)
	Discard(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	CopyPublished(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (softwarelifecycle.SourceCode, error)
}

type Handler struct {
	service sourceCodeService
}

// New creates a software source code HTTP handler.
func New(service sourceCodeService) *Handler {
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
	if request.GroupID == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "group_id is required")
		return
	}

	value, err := h.service.CreateDraft(r.Context(), actorID, request.GroupID, request.Metadata.domain())
	h.writeSourceCode(r, w, value, true, err, http.StatusCreated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sourceCodeID, ok := sourceCodeID(w, r)
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

	var actor *uuid.UUID
	if present {
		actor = &actorID
	}

	value, includeContacts, err := h.service.Read(r.Context(), actor, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, includeContacts, err, http.StatusOK)
}

func (h *Handler) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, versionID, ok := lifecycleIDs(w, r)
	if !ok {
		return
	}

	var request replaceDraftRequest
	if !transport.Decode(w, r, &request) {
		return
	}

	value, err := h.service.ReplaceDraft(r.Context(), actorID, sourceCodeID, versionID, request.Revision, request.Metadata.domain())
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, versionID, ok := lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Submit(r.Context(), actorID, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, versionID, ok := lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Withdraw(r.Context(), actorID, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, versionID, ok := lifecycleIDs(w, r)
	if !ok {
		return
	}

	var request rejectionRequest
	if !transport.Decode(w, r, &request) {
		return
	}

	value, err := h.service.Reject(r.Context(), actorID, sourceCodeID, versionID, request.Note)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Reopen(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, versionID, ok := lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Reopen(r.Context(), actorID, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, versionID, ok := lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Publish(r.Context(), actorID, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, versionID, ok := lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Unpublish(r.Context(), actorID, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, ok := resourceIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.DraftFromPublication(r.Context(), actorID, sourceCodeID)
	h.writeSourceCode(r, w, value, true, err, http.StatusCreated)
}

func (h *Handler) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, ok := resourceIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.DraftFromLatestArchive(r.Context(), actorID, sourceCodeID)
	h.writeSourceCode(r, w, value, true, err, http.StatusCreated)
}

func (h *Handler) Discard(w http.ResponseWriter, r *http.Request) {
	actorID, sourceCodeID, versionID, ok := lifecycleIDs(w, r)
	if !ok {
		return
	}

	if err := h.service.Discard(r.Context(), actorID, sourceCodeID, versionID); err != nil {
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
	sourceCodeID, ok := sourceCodeID(w, r)
	if !ok {
		return
	}
	var request copyRequest
	if !transport.Decode(w, r, &request) {
		return
	}
	if request.TargetGroupID == uuid.Nil || strings.TrimSpace(request.SourceIRI) == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "target_group_id and source_iri are required")
		return
	}

	value, err := h.service.CopyPublished(r.Context(), actorID, sourceCodeID, request.TargetGroupID, request.SourceIRI)
	h.writeSourceCode(r, w, value, true, err, http.StatusCreated)
}

func (h *Handler) writeSourceCode(r *http.Request, w http.ResponseWriter, value softwarelifecycle.SourceCode, includeContacts bool, err error, status int) {
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	transport.WriteJSON(w, status, sourceCodeResponse(value, includeContacts))
}

func lifecycleIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	actorID, sourceCodeID, ok := resourceIDs(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	versionID, ok := versionID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return actorID, sourceCodeID, versionID, true
}

func resourceIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	sourceCodeID, ok := sourceCodeID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return actorID, sourceCodeID, true
}

func sourceCodeID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(w, chi.URLParam(r, "resource_id"), "software source code")
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
		transport.WriteProblem(w, http.StatusNotFound, "software_source_code_not_found", "software source code or version not found")
	case errors.Is(err, softwarelifecycle.ErrConflict):
		transport.WriteProblem(w, http.StatusConflict, "conflict", "software source code lifecycle conflict")
	case errors.Is(err, softwarelifecycle.ErrInvalid):
		detail := err.Error()
		if index := strings.LastIndex(detail, ": "); index > 0 {
			detail = detail[:index]
		}
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", detail)
	default:
		log.ErrorContext(r.Context(), "software source code request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "software source code request failed")
	}
}
