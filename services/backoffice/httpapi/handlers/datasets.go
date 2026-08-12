// Package handlers exposes Backoffice HTTP handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const actorHeader = "X-EPOS-User-ID"

type datasetService interface {
	CreateDraft(context.Context, uuid.UUID, uuid.UUID, metadata.Dataset) (datasetlifecycle.Dataset, error)
	Read(context.Context, *uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, bool, error)
	ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int32, metadata.Dataset) (datasetlifecycle.Dataset, error)
	Submit(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Withdraw(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Reject(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (datasetlifecycle.Dataset, error)
	Reopen(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Publish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Unpublish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	Discard(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	DraftFromPublication(context.Context, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	DraftFromLatestArchive(context.Context, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error)
	CopyPublished(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (datasetlifecycle.Dataset, error)
}

type Datasets struct {
	service datasetService
}

func NewDatasets(service datasetService) *Datasets {
	return &Datasets{service: service}
}

func (h *Datasets) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	groupID, ok := groupID(w, r)
	if !ok {
		return
	}

	var request metadata.Dataset
	if !decode(w, r, &request) {
		return
	}
	dataset, err := h.service.CreateDraft(r.Context(), actor, groupID, request)
	h.writeDataset(r, w, dataset, err, http.StatusCreated)
}

func (h *Datasets) Get(w http.ResponseWriter, r *http.Request) {
	datasetID, ok := datasetID(w, r)
	if !ok {
		return
	}
	actorID, present, ok := optionalActor(w, r)
	if !ok {
		return
	}
	var actor *uuid.UUID
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
		writePublicDataset(w, dataset)
		return
	}
	writeJSON(w, http.StatusOK, dataset)
}

func (h *Datasets) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	datasetID, ok := datasetID(w, r)
	if !ok {
		return
	}
	var request struct {
		Revision int32            `json:"revision"`
		Metadata metadata.Dataset `json:"metadata"`
	}
	if !decode(w, r, &request) {
		return
	}
	versionID, ok := versionID(w, r)
	if !ok {
		return
	}
	dataset, err := h.service.ReplaceDraft(r.Context(), actor, datasetID, versionID, request.Revision, request.Metadata)
	h.writeDataset(r, w, dataset, err, http.StatusOK)
}

func (h *Datasets) action(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := actor(w, r)
		if !ok {
			return
		}
		datasetID, ok := datasetID(w, r)
		if !ok {
			return
		}
		versionID, ok := versionID(w, r)
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
		if name == "reject" && !decode(w, r, &request) {
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

func (h *Datasets) Submit(w http.ResponseWriter, r *http.Request) {
	h.action("submit")(w, r)
}

func (h *Datasets) Withdraw(w http.ResponseWriter, r *http.Request) {
	h.action("withdraw")(w, r)
}

func (h *Datasets) Reject(w http.ResponseWriter, r *http.Request) {
	h.action("reject")(w, r)
}

func (h *Datasets) Reopen(w http.ResponseWriter, r *http.Request) {
	h.action("reopen")(w, r)
}

func (h *Datasets) Publish(w http.ResponseWriter, r *http.Request) {
	h.action("publish")(w, r)
}

func (h *Datasets) Unpublish(w http.ResponseWriter, r *http.Request) {
	h.action("unpublish")(w, r)
}

func (h *Datasets) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	h.action("draft-from-publication")(w, r)
}

func (h *Datasets) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	h.action("draft-from-archive")(w, r)
}

func (h *Datasets) Discard(w http.ResponseWriter, r *http.Request) {
	h.action("discard")(w, r)
}

func (h *Datasets) Copy(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	sourceID, ok := datasetID(w, r)
	if !ok {
		return
	}
	groupID, ok := groupID(w, r)
	if !ok {
		return
	}
	var request struct {
		SourceIRI string `json:"source_iri"`
	}
	if !decode(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.SourceIRI) == "" {
		http.Error(w, "source_iri is required", http.StatusBadRequest)
		return
	}
	dataset, err := h.service.CopyPublished(r.Context(), actor, sourceID, groupID, request.SourceIRI)
	h.writeDataset(r, w, dataset, err, http.StatusCreated)
}

func (h *Datasets) writeDataset(r *http.Request, w http.ResponseWriter, dataset datasetlifecycle.Dataset, err error, status int) {
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	writeJSON(w, status, dataset)
}

func actor(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, present, ok := optionalActor(w, r)
	if !ok {
		return uuid.Nil, false
	}
	if !present {
		http.Error(w, actorHeader+" is required", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}

func optionalActor(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool, bool) {
	value := strings.TrimSpace(r.Header.Get(actorHeader))
	if value == "" {
		return uuid.Nil, false, true
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		http.Error(w, "invalid "+actorHeader, http.StatusUnauthorized)
		return uuid.Nil, true, false
	}
	return id, true, true
}

func datasetID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(w, chi.URLParam(r, "id"), "dataset")
}

func versionID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(w, chi.URLParam(r, "version_id"), "version")
}

func groupID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return pathUUID(w, chi.URLParam(r, "group_id"), "group")
}

func pathUUID(w http.ResponseWriter, value, resource string) (uuid.UUID, bool) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		http.Error(w, "invalid "+resource+" id", http.StatusBadRequest)
		return uuid.Nil, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return false
	}
	return true
}

func writePublicDataset(w http.ResponseWriter, dataset datasetlifecycle.Dataset) {
	encoded, err := json.Marshal(dataset)
	if err != nil {
		http.Error(w, "encode response", http.StatusInternalServerError)
		return
	}
	var response map[string]any
	if err := json.Unmarshal(encoded, &response); err != nil {
		http.Error(w, "encode response", http.StatusInternalServerError)
		return
	}
	metadataValue, _ := response["metadata"].(map[string]any)
	delete(metadataValue, "contact_points")
	if distributions, ok := metadataValue["distributions"].([]any); ok {
		for _, distribution := range distributions {
			item, _ := distribution.(map[string]any)
			service, _ := item["web_service"].(map[string]any)
			delete(service, "contact_points")
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func writeLifecycleError(r *http.Request, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, datasetlifecycle.ErrDenied):
		writeProblem(w, http.StatusForbidden, "forbidden", "access denied")
	case errors.Is(err, datasetlifecycle.ErrNotFound):
		writeProblem(w, http.StatusNotFound, "dataset_not_found", "dataset or version not found")
	case errors.Is(err, datasetlifecycle.ErrConflict):
		writeProblem(w, http.StatusConflict, "conflict", "dataset lifecycle conflict")
	case errors.Is(err, datasetlifecycle.ErrInvalid):
		detail := err.Error()
		if index := strings.LastIndex(detail, ": "); index > 0 {
			detail = detail[:index]
		}
		writeProblem(w, http.StatusBadRequest, "invalid_request", detail)
	default:
		log.ErrorContext(r.Context(), "dataset request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeProblem(w, http.StatusInternalServerError, "internal_server_error", "dataset request failed")
	}
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}{Title: code, Status: status, Detail: detail})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
