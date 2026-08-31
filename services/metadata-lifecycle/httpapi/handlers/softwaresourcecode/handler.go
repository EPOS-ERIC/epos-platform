// Package softwaresourcecode exposes software source code HTTP handlers.
package softwaresourcecode

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type sourceCodeService interface {
	CreateDraft(context.Context, domain.Actor, uuid.UUID, domain.SoftwareSourceCodeInput) (domain.SoftwareSourceCodeVersion, error)
	Read(context.Context, *domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, bool, error)
	ReplaceDraft(context.Context, domain.Actor, uuid.UUID, uuid.UUID, domain.SoftwareSourceCodeDraftInput) (domain.SoftwareSourceCodeVersion, error)
	Submit(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Withdraw(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Reject(context.Context, domain.Actor, uuid.UUID, uuid.UUID, domain.SoftwareSourceCodeRejectionInput) (domain.SoftwareSourceCodeVersion, error)
	Reopen(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Publish(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Unpublish(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	DraftFromPublication(context.Context, domain.Actor, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	DraftFromLatestArchive(context.Context, domain.Actor, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
	Discard(context.Context, domain.Actor, uuid.UUID, uuid.UUID) error
	CopyPublished(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.SoftwareSourceCodeVersion, error)
}

type Handler struct {
	service sourceCodeService
	log     *slog.Logger
}

// New creates a software source code HTTP handler.
func New(log *slog.Logger, service sourceCodeService) *Handler {
	return &Handler{
		service: service,
		log: log.With("component", "software-source-codes-handler"),
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	var request createRequest
	if !transport.Decode(h.log, w, r, &request) {
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || groupID == uuid.Nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid group_id")
		return
	}

	input, err := request.Metadata.input()
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	value, err := h.service.CreateDraft(r.Context(), actor, groupID, input)
	h.writeSourceCode(r, w, value, true, err, http.StatusCreated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sourceCodeID, ok := sourceCodeID(h.log, w, r)
	if !ok {
		return
	}
	versionID, ok := versionID(h.log, w, r)
	if !ok {
		return
	}
	actor, ok := transport.OptionalActor(w, r)
	if !ok {
		return
	}

	value, includeContacts, err := h.service.Read(r.Context(), actor, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, includeContacts, err, http.StatusOK)
}

func (h *Handler) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, versionID, ok := h.lifecycleIDs(w, r)
	if !ok {
		return
	}

	var request replaceDraftRequest
	if !transport.Decode(h.log, w, r, &request) {
		return
	}

	input, err := request.input()
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	value, err := h.service.ReplaceDraft(r.Context(), actor, sourceCodeID, versionID, input)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, versionID, ok := h.lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Submit(r.Context(), actor, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, versionID, ok := h.lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Withdraw(r.Context(), actor, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, versionID, ok := h.lifecycleIDs(w, r)
	if !ok {
		return
	}

	var request rejectionRequest
	if !transport.Decode(h.log, w, r, &request) {
		return
	}

	rejection, err := request.input()
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	value, err := h.service.Reject(r.Context(), actor, sourceCodeID, versionID, rejection)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Reopen(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, versionID, ok := h.lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Reopen(r.Context(), actor, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, versionID, ok := h.lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Publish(r.Context(), actor, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, versionID, ok := h.lifecycleIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.Unpublish(r.Context(), actor, sourceCodeID, versionID)
	h.writeSourceCode(r, w, value, true, err, http.StatusOK)
}

func (h *Handler) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, ok := h.resourceIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.DraftFromPublication(r.Context(), actor, sourceCodeID)
	h.writeSourceCode(r, w, value, true, err, http.StatusCreated)
}

func (h *Handler) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, ok := h.resourceIDs(w, r)
	if !ok {
		return
	}

	value, err := h.service.DraftFromLatestArchive(r.Context(), actor, sourceCodeID)
	h.writeSourceCode(r, w, value, true, err, http.StatusCreated)
}

func (h *Handler) Discard(w http.ResponseWriter, r *http.Request) {
	actor, sourceCodeID, versionID, ok := h.lifecycleIDs(w, r)
	if !ok {
		return
	}

	if err := h.service.Discard(r.Context(), actor, sourceCodeID, versionID); err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Copy(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	sourceCodeID, ok := sourceCodeID(h.log, w, r)
	if !ok {
		return
	}
	targetGroupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid group_id")
		return
	}

	value, err := h.service.CopyPublished(r.Context(), actor, sourceCodeID, targetGroupID)
	h.writeSourceCode(r, w, value, true, err, http.StatusCreated)
}

func (h *Handler) writeSourceCode(r *http.Request, w http.ResponseWriter, value domain.SoftwareSourceCodeVersion, includeContacts bool, err error, status int) {
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	transport.WriteJSON(h.log, w, status, sourceCodeResponse(value, includeContacts))
}

func (h *Handler) lifecycleIDs(w http.ResponseWriter, r *http.Request) (domain.Actor, uuid.UUID, uuid.UUID, bool) {
	actor, sourceCodeID, ok := h.resourceIDs(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	versionID, ok := versionID(h.log, w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	return actor, sourceCodeID, versionID, true
}

func (h *Handler) resourceIDs(w http.ResponseWriter, r *http.Request) (domain.Actor, uuid.UUID, bool) {
	actor, ok := transport.Actor(w, r)
	if !ok {
		return "", uuid.Nil, false
	}
	sourceCodeID, ok := sourceCodeID(h.log, w, r)
	if !ok {
		return "", uuid.Nil, false
	}
	return actor, sourceCodeID, true
}

func sourceCodeID(log *slog.Logger, w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(log, w, chi.URLParam(r, "resource_id"), "software source code")
}

func versionID(log *slog.Logger, w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(log, w, chi.URLParam(r, "version_id"), "version")
}

func pathUUID(log *slog.Logger, w http.ResponseWriter, value, resource string) (uuid.UUID, bool) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid "+resource+" id")
		return uuid.Nil, false
	}
	return id, true
}

func writeLifecycleError(log *slog.Logger, r *http.Request, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrSoftwareSourceCodeForbidden):
		transport.WriteProblem(log, w, http.StatusForbidden, "forbidden", "access denied")
	case errors.Is(err, domain.ErrSoftwareSourceCodeNotFound):
		transport.WriteProblem(log, w, http.StatusNotFound, "software_source_code_not_found", "software source code or version not found")
	case errors.Is(err, domain.ErrSoftwareSourceCodeConflict):
		transport.WriteProblem(log, w, http.StatusConflict, "conflict", "software source code lifecycle conflict")
	case errors.Is(err, domain.ErrSoftwareSourceCodeInvalid):
		detail := err.Error()
		if index := strings.LastIndex(detail, ": "); index > 0 {
			detail = detail[:index]
		}
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", detail)
	case errors.Is(err, domain.ErrSoftwareSourceCodeInternal):
		log.ErrorContext(r.Context(), "software source code request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", "software source code request failed")
	default:
		log.ErrorContext(r.Context(), "software source code request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", "software source code request failed")
	}
}
