// Package dataset exposes dataset HTTP handlers.
package dataset

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("dataset-handler")

type datasetService interface {
	CreateDraft(context.Context, string, uuid.UUID, metadata.Dataset) (datasetlifecycle.Dataset, error)
	Read(context.Context, *string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, bool, error)
	ReplaceDraft(context.Context, string, uuid.UUID, uuid.UUID, int32, metadata.Dataset) (datasetlifecycle.Dataset, error)
	Submit(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Withdraw(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Reject(context.Context, string, uuid.UUID, uuid.UUID, string) (datasetlifecycle.Dataset, error)
	Reopen(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Publish(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Discard(context.Context, string, uuid.UUID, uuid.UUID) error
	DraftFromPublication(context.Context, string, uuid.UUID) (datasetlifecycle.Dataset, error)
	DraftFromLatestArchive(context.Context, string, uuid.UUID) (datasetlifecycle.Dataset, error)
	CopyPublished(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
}

type handler struct {
	service datasetService
}

// New creates a dataset HTTP handler.
func New(service datasetService) *handler {
	return &handler{service: service}
}

func (h *handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	var request createRequest
	if !transport.Decode(w, r, &request) {
		return
	}
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
	dataset, err := h.service.CreateDraft(r.Context(), actor, groupID, request.Metadata.domain())
	h.writeDataset(r, w, dataset, err, http.StatusCreated)
}

func (h *handler) Get(w http.ResponseWriter, r *http.Request) {
	datasetID, ok := datasetID(w, r)
	if !ok {
		return
	}
	actorID, present, ok := optionalActor(w, r)
	if !ok {
		return
	}
	var actor *string
	if present {
		actor = &actorID
	}
	versionID, ok := versionID(w, r)
	if !ok {
		return
	}
	dataset, includeContacts, err := h.service.Read(r.Context(), actor, datasetID, versionID)
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	if !includeContacts {
		transport.WriteJSON(w, http.StatusOK, datasetResponse(dataset, false))
		return
	}
	transport.WriteJSON(w, http.StatusOK, datasetResponse(dataset, true))
}

func (h *handler) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	datasetID, ok := datasetID(w, r)
	if !ok {
		return
	}
	var request struct {
		ResourceType string              `json:"resource_type"`
		Revision     int32               `json:"revision"`
		Metadata     datasetDraftRequest `json:"metadata"`
	}
	if !transport.Decode(w, r, &request) {
		return
	}
	versionID, ok := versionID(w, r)
	if !ok {
		return
	}
	dataset, err := h.service.ReplaceDraft(r.Context(), actor, datasetID, versionID, request.Revision, request.Metadata.domain())
	h.writeDataset(r, w, dataset, err, http.StatusOK)
}

func (h *handler) action(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := actor(w, r)
		if !ok {
			return
		}
		datasetID, ok := datasetID(w, r)
		if !ok {
			return
		}
		if name == "draft-from-publication" || name == "draft-from-archive" {
			var dataset datasetlifecycle.Dataset
			var err error
			if name == "draft-from-publication" {
				dataset, err = h.service.DraftFromPublication(r.Context(), actor, datasetID)
			} else {
				dataset, err = h.service.DraftFromLatestArchive(r.Context(), actor, datasetID)
			}
			h.writeDataset(r, w, dataset, err, http.StatusCreated)
			return
		}
		versionID, ok := versionID(w, r)
		if !ok {
			return
		}
		if name == "discard" {
			if err := h.service.Discard(r.Context(), actor, datasetID, versionID); err != nil {
				writeLifecycleError(r, w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var request struct {
			Note string `json:"note"`
		}
		if name == "reject" && !transport.Decode(w, r, &request) {
			return
		}
		var dataset datasetlifecycle.Dataset
		var err error
		switch name {
		case "submit":
			dataset, err = h.service.Submit(r.Context(), actor, datasetID, versionID)
		case "withdraw":
			dataset, err = h.service.Withdraw(r.Context(), actor, datasetID, versionID)
		case "reject":
			dataset, err = h.service.Reject(r.Context(), actor, datasetID, versionID, request.Note)
		case "reopen":
			dataset, err = h.service.Reopen(r.Context(), actor, datasetID, versionID)
		case "publish":
			dataset, err = h.service.Publish(r.Context(), actor, datasetID, versionID)
		case "unpublish":
			dataset, err = h.service.Unpublish(r.Context(), actor, datasetID, versionID)
		}
		h.writeDataset(r, w, dataset, err, http.StatusOK)
	}
}

func (h *handler) Submit(w http.ResponseWriter, r *http.Request) {
	h.action("submit")(w, r)
}

func (h *handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	h.action("withdraw")(w, r)
}

func (h *handler) Reject(w http.ResponseWriter, r *http.Request) {
	h.action("reject")(w, r)
}

func (h *handler) Reopen(w http.ResponseWriter, r *http.Request) {
	h.action("reopen")(w, r)
}

func (h *handler) Publish(w http.ResponseWriter, r *http.Request) {
	h.action("publish")(w, r)
}

func (h *handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	h.action("unpublish")(w, r)
}

func (h *handler) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	h.action("draft-from-publication")(w, r)
}

func (h *handler) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	h.action("draft-from-archive")(w, r)
}

func (h *handler) Discard(w http.ResponseWriter, r *http.Request) {
	h.action("discard")(w, r)
}

func (h *handler) Copy(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	sourceID, ok := datasetID(w, r)
	if !ok {
		return
	}
	targetGroupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group_id")
		return
	}
	dataset, err := h.service.CopyPublished(r.Context(), actor, sourceID, targetGroupID)
	h.writeDataset(r, w, dataset, err, http.StatusCreated)
}

func (h *handler) writeDataset(r *http.Request, w http.ResponseWriter, dataset datasetlifecycle.Dataset, err error, status int) {
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	transport.WriteJSON(w, status, datasetResponse(dataset, true))
}

func actor(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, present, ok := optionalActor(w, r)
	if !ok {
		return "", false
	}
	if !present {
		http.Error(w, transport.ActorHeader+" is required", http.StatusUnauthorized)
		return "", false
	}
	return id, true
}

func optionalActor(w http.ResponseWriter, r *http.Request) (string, bool, bool) {
	return transport.OptionalActor(w, r)
}

func datasetID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(w, chi.URLParam(r, "resource_id"), "dataset")
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
	case errors.Is(err, datasetlifecycle.ErrDenied):
		transport.WriteProblem(w, http.StatusForbidden, "forbidden", "access denied")
	case errors.Is(err, datasetlifecycle.ErrNotFound):
		transport.WriteProblem(w, http.StatusNotFound, "dataset_not_found", "dataset or version not found")
	case errors.Is(err, datasetlifecycle.ErrConflict):
		transport.WriteProblem(w, http.StatusConflict, "conflict", "dataset lifecycle conflict")
	case errors.Is(err, datasetlifecycle.ErrInvalid):
		detail := err.Error()
		if index := strings.LastIndex(detail, ": "); index > 0 {
			detail = detail[:index]
		}
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", detail)
	default:
		log.ErrorContext(r.Context(), "dataset request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "dataset request failed")
	}
}
