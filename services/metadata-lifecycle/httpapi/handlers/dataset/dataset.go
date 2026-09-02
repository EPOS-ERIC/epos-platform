// Package dataset exposes dataset HTTP handlers.
package dataset

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

type datasetService interface {
	CreateDraft(context.Context, domain.Actor, uuid.UUID, domain.DatasetInput) (domain.DatasetVersion, error)
	Read(context.Context, *domain.Actor, uuid.UUID, uuid.UUID) (domain.DatasetVersion, bool, error)
	ReplaceDraft(context.Context, domain.Actor, uuid.UUID, uuid.UUID, domain.DatasetDraftInput) (domain.DatasetVersion, error)
	Submit(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Withdraw(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Reject(context.Context, domain.Actor, uuid.UUID, uuid.UUID, domain.DatasetRejectionInput) (domain.DatasetVersion, error)
	Reopen(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Publish(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Unpublish(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
	Discard(context.Context, domain.Actor, uuid.UUID, uuid.UUID) error
	DraftFromPublication(context.Context, domain.Actor, uuid.UUID) (domain.DatasetVersion, error)
	DraftFromLatestArchive(context.Context, domain.Actor, uuid.UUID) (domain.DatasetVersion, error)
	CopyPublished(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error)
}

type handler struct {
	service datasetService
	log     *slog.Logger
}

// New creates a dataset HTTP handler.
func New(log *slog.Logger, service datasetService) *handler {
	return &handler{
		service: service,
		log:     log.With("component", "datasets-handler"),
	}
}

func (h *handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(h.log, w, r)
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
	value, err := domain.NewDatasetInput(request.Metadata.domain())
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	dataset, err := h.service.CreateDraft(r.Context(), actor, groupID, value)
	h.writeDataset(r, w, dataset, err, http.StatusCreated)
}

func (h *handler) Get(w http.ResponseWriter, r *http.Request) {
	datasetID, ok := datasetID(h.log, w, r)
	if !ok {
		return
	}
	actor, ok := transport.OptionalActor(w, r)
	if !ok {
		return
	}
	versionID, ok := versionID(h.log, w, r)
	if !ok {
		return
	}
	dataset, includeContacts, err := h.service.Read(r.Context(), actor, datasetID, versionID)
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	if !includeContacts {
		transport.WriteJSON(h.log, w, http.StatusOK, datasetResponse(dataset, false))
		return
	}
	transport.WriteJSON(h.log, w, http.StatusOK, datasetResponse(dataset, true))
}

func (h *handler) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(h.log, w, r)
	if !ok {
		return
	}
	datasetID, ok := datasetID(h.log, w, r)
	if !ok {
		return
	}
	var request replaceDraftRequest
	if !transport.Decode(h.log, w, r, &request) {
		return
	}
	versionID, ok := versionID(h.log, w, r)
	if !ok {
		return
	}
	value, err := domain.NewDatasetDraftInput(domain.DatasetDraftInput{
		Revision: request.Revision,
		Metadata: request.Metadata.domain(),
	})
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	dataset, err := h.service.ReplaceDraft(r.Context(), actor, datasetID, versionID, value)
	h.writeDataset(r, w, dataset, err, http.StatusOK)
}

func (h *handler) transition(
	w http.ResponseWriter,
	r *http.Request,
	action func(context.Context, domain.Actor, uuid.UUID, uuid.UUID) (domain.DatasetVersion, error),
) {
	actor, ok := actor(h.log, w, r)
	if !ok {
		return
	}
	datasetID, ok := datasetID(h.log, w, r)
	if !ok {
		return
	}
	versionID, ok := versionID(h.log, w, r)
	if !ok {
		return
	}
	dataset, err := action(r.Context(), actor, datasetID, versionID)
	h.writeDataset(r, w, dataset, err, http.StatusOK)
}

func (h *handler) Submit(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Submit)
}

func (h *handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Withdraw)
}

func (h *handler) Reject(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(h.log, w, r)
	if !ok {
		return
	}
	datasetID, ok := datasetID(h.log, w, r)
	if !ok {
		return
	}
	versionID, ok := versionID(h.log, w, r)
	if !ok {
		return
	}
	var request rejectionRequest
	if !transport.Decode(h.log, w, r, &request) {
		return
	}
	value, err := domain.NewDatasetRejectionInput(domain.DatasetRejectionInput{Note: request.Note})
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	dataset, err := h.service.Reject(r.Context(), actor, datasetID, versionID, value)
	h.writeDataset(r, w, dataset, err, http.StatusOK)
}

func (h *handler) Reopen(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Reopen)
}

func (h *handler) Publish(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Publish)
}

func (h *handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Unpublish)
}

func (h *handler) createDraft(
	w http.ResponseWriter,
	r *http.Request,
	action func(context.Context, domain.Actor, uuid.UUID) (domain.DatasetVersion, error),
) {
	actor, ok := actor(h.log, w, r)
	if !ok {
		return
	}
	datasetID, ok := datasetID(h.log, w, r)
	if !ok {
		return
	}
	dataset, err := action(r.Context(), actor, datasetID)
	h.writeDataset(r, w, dataset, err, http.StatusCreated)
}

func (h *handler) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	h.createDraft(w, r, h.service.DraftFromPublication)
}

func (h *handler) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	h.createDraft(w, r, h.service.DraftFromLatestArchive)
}

func (h *handler) Discard(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(h.log, w, r)
	if !ok {
		return
	}
	datasetID, ok := datasetID(h.log, w, r)
	if !ok {
		return
	}
	versionID, ok := versionID(h.log, w, r)
	if !ok {
		return
	}
	if err := h.service.Discard(r.Context(), actor, datasetID, versionID); err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) Copy(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(h.log, w, r)
	if !ok {
		return
	}
	sourceID, ok := datasetID(h.log, w, r)
	if !ok {
		return
	}
	targetGroupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid group_id")
		return
	}
	dataset, err := h.service.CopyPublished(r.Context(), actor, sourceID, targetGroupID)
	h.writeDataset(r, w, dataset, err, http.StatusCreated)
}

func (h *handler) writeDataset(r *http.Request, w http.ResponseWriter, dataset domain.DatasetVersion, err error, status int) {
	if err != nil {
		writeLifecycleError(h.log, r, w, err)
		return
	}
	transport.WriteJSON(h.log, w, status, datasetResponse(dataset, true))
}

func actor(log *slog.Logger, w http.ResponseWriter, r *http.Request) (domain.Actor, bool) {
	return transport.Actor(w, r)
}

func datasetID(log *slog.Logger, w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(log, w, chi.URLParam(r, "resource_id"), "dataset")
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
	case errors.Is(err, domain.ErrDatasetForbidden):
		transport.WriteProblem(log, w, http.StatusForbidden, "forbidden", "access denied")
	case errors.Is(err, domain.ErrDatasetNotFound):
		transport.WriteProblem(log, w, http.StatusNotFound, "dataset_not_found", "dataset or version not found")
	case errors.Is(err, domain.ErrDatasetConflict):
		transport.WriteProblem(log, w, http.StatusConflict, "conflict", "dataset lifecycle conflict")
	case errors.Is(err, domain.ErrInvalidDataset):
		detail := err.Error()
		if index := strings.LastIndex(detail, ": "); index > 0 {
			detail = detail[:index]
		}
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", detail)
	default:
		log.ErrorContext(r.Context(), "dataset request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", "dataset request failed")
	}
}
