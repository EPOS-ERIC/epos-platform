// Package handlers contains Gateway HTTP handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/lifecycle"
	"github.com/go-chi/chi/v5"
)

const (
	datasetType             = "dataset"
	softwareApplicationType = "software-application"
	softwareSourceCodeType  = "software-source-code"
)

type Lifecycle struct {
	service *lifecycle.Service
}

func NewLifecycle(service *lifecycle.Service) *Lifecycle {
	return &Lifecycle{service: service}
}

//nolint:dupl // TODO Generated request wrappers require distinct concrete field types.
func (h *Lifecycle) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID, ok := uuidParam(ctx, w, r, "groupId")
	if !ok {
		return
	}
	var envelope struct {
		ResourceType string          `json:"resource_type"`
		Metadata     json.RawMessage `json:"metadata"`
	}
	if !decodeJSON(r, w, &envelope) {
		return
	}
	resourceType := envelope.ResourceType
	var input domain.CreateResourceInput
	switch resourceType {
	case datasetType:
		var value api.DatasetDraftMetadata
		if err := json.Unmarshal(envelope.Metadata, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid dataset metadata", err)
			return
		}
		if value.Title == "" {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "title is required", nil)
			return
		}
		input.Metadata = toDomainDatasetDraftMetadata(value)
	case softwareApplicationType:
		var value api.SoftwareApplicationDraftMetadata
		if err := json.Unmarshal(envelope.Metadata, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid software application metadata", err)
			return
		}
		if value.Name == "" {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "name is required", nil)
			return
		}
		input.Metadata = toDomainSoftwareApplicationDraftMetadata(value)
	case softwareSourceCodeType:
		var value api.SoftwareSourceCodeDraftMetadata
		if err := json.Unmarshal(envelope.Metadata, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid software source code metadata", err)
			return
		}
		if value.Name == "" {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "name is required", nil)
			return
		}
		input.Metadata = toDomainSoftwareSourceCodeDraftMetadata(value)
	default:
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid resource_type", nil)
		return
	}

	record, err := h.service.Create(ctx, actor.ID, groupID, resourceType, input)
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(toAPIWriteResult(resourceType, record))
	if err != nil {
		log.ErrorContext(ctx, "encode lifecycle response failed", "error", err)
	}
}

func (h *Lifecycle) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actorID := ""
	if actor, ok := AuthenticatedUserFromContext(ctx); ok {
		actorID = actor.ID
	}

	resourceType, ok := resourceTypeParam(ctx, w, r)
	if !ok {
		return
	}
	id, ok := uuidParam(ctx, w, r, "id")
	if !ok {
		return
	}
	versionID, ok := uuidParam(ctx, w, r, "versionId")
	if !ok {
		return
	}

	record, err := h.service.Get(ctx, actorID, resourceType, id, versionID)
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toAPIResourceVersion(record)); err != nil {
		log.ErrorContext(ctx, "encode lifecycle response failed", "error", err)
	}
}

//nolint:dupl // Generated request wrappers require distinct concrete field types.
func (h *Lifecycle) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}
	id, ok := uuidParam(ctx, w, r, "id")
	if !ok {
		return
	}
	versionID, ok := uuidParam(ctx, w, r, "versionId")
	if !ok {
		return
	}

	var envelope struct {
		ResourceType string          `json:"resource_type"`
		Revision     int32           `json:"revision"`
		Metadata     json.RawMessage `json:"metadata"`
	}
	if !decodeJSON(r, w, &envelope) {
		return
	}
	resourceType := envelope.ResourceType
	var input domain.ReplaceResourceDraftInput
	switch resourceType {
	case datasetType:
		var metadata api.DatasetDraftMetadata
		if err := json.Unmarshal(envelope.Metadata, &metadata); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid dataset metadata", err)
			return
		}
		input = toDomainReplaceResourceDraftInput(envelope.Revision, toDomainDatasetDraftMetadata(metadata))
	case softwareApplicationType:
		var metadata api.SoftwareApplicationDraftMetadata
		if err := json.Unmarshal(envelope.Metadata, &metadata); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid software application metadata", err)
			return
		}
		input = toDomainReplaceResourceDraftInput(envelope.Revision, toDomainSoftwareApplicationDraftMetadata(metadata))
	case softwareSourceCodeType:
		var metadata api.SoftwareSourceCodeDraftMetadata
		if err := json.Unmarshal(envelope.Metadata, &metadata); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid software source code metadata", err)
			return
		}
		input = toDomainReplaceResourceDraftInput(envelope.Revision, toDomainSoftwareSourceCodeDraftMetadata(metadata))
	default:
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid resource_type", nil)
		return
	}

	record, err := h.service.ReplaceDraft(ctx, actor.ID, resourceType, id, versionID, input)
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toAPIWriteResult(resourceType, record)); err != nil {
		log.ErrorContext(ctx, "encode lifecycle response failed", "error", err)
	}
}

func (h *Lifecycle) Discard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}
	resourceType, ok := resourceTypeParam(ctx, w, r)
	if !ok {
		return
	}
	id, ok := uuidParam(ctx, w, r, "id")
	if !ok {
		return
	}
	versionID, ok := uuidParam(ctx, w, r, "versionId")
	if !ok {
		return
	}

	if err := h.service.Discard(ctx, actor.ID, resourceType, id, versionID); err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toAPIDiscardedWriteResult(resourceType, id)); err != nil {
		log.ErrorContext(ctx, "encode lifecycle response failed", "error", err)
	}
}

func (h *Lifecycle) Action(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}
	resourceType, ok := resourceTypeParam(ctx, w, r)
	if !ok {
		return
	}
	id, ok := uuidParam(ctx, w, r, "id")
	if !ok {
		return
	}
	versionID, ok := uuidParam(ctx, w, r, "versionId")
	if !ok {
		return
	}
	action := chi.URLParam(r, "action")
	var record domain.ResourceVersion
	var err error
	switch action {
	case "reject":
		var request api.RejectResourceRequest
		if !decodeJSON(r, w, &request) {
			return
		}
		if strings.TrimSpace(request.Note) == "" {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "note is required", nil)
			return
		}
		record, err = h.service.Reject(ctx, actor.ID, resourceType, id, versionID, toDomainRejectResourceInput(request))
	case "submit":
		record, err = h.service.Submit(ctx, actor.ID, resourceType, id, versionID)
	case "withdraw":
		record, err = h.service.Withdraw(ctx, actor.ID, resourceType, id, versionID)
	case "reopen":
		record, err = h.service.Reopen(ctx, actor.ID, resourceType, id, versionID)
	case "publish":
		record, err = h.service.Publish(ctx, actor.ID, resourceType, id, versionID)
	case "unpublish":
		record, err = h.service.Unpublish(ctx, actor.ID, resourceType, id, versionID)
	default:
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "unsupported lifecycle action", nil)
		return
	}
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toAPIWriteResult(resourceType, record)); err != nil {
		log.ErrorContext(ctx, "encode lifecycle response failed", "error", err)
	}
}

func (h *Lifecycle) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	h.createDraft(w, r, h.service.DraftFromPublication)
}

func (h *Lifecycle) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	h.createDraft(w, r, h.service.DraftFromArchive)
}

// createDraft handles HTTP plumbing shared by draft-from-publication and
// draft-from-archive endpoints. create is the operation-specific callback that
// calls Metadata Lifecycle to create a draft from the selected source.
func (h *Lifecycle) createDraft(w http.ResponseWriter, r *http.Request, create func(context.Context, string, string, string) (domain.ResourceVersion, error)) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}
	resourceType, ok := resourceTypeParam(ctx, w, r)
	if !ok {
		return
	}
	id, ok := uuidParam(ctx, w, r, "id")
	if !ok {
		return
	}

	record, err := create(ctx, actor.ID, resourceType, id)
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toAPIWriteResult(resourceType, record)); err != nil {
		log.ErrorContext(ctx, "encode lifecycle response failed", "error", err)
	}
}

func (h *Lifecycle) Copy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}
	groupID, ok := uuidParam(ctx, w, r, "groupId")
	if !ok {
		return
	}
	resourceType, ok := resourceTypeParam(ctx, w, r)
	if !ok {
		return
	}
	id, ok := uuidParam(ctx, w, r, "id")
	if !ok {
		return
	}

	record, err := h.service.Copy(ctx, actor.ID, groupID, resourceType, id)
	if err != nil {
		writeLifecycleError(r, w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toAPIWriteResult(resourceType, record)); err != nil {
		log.ErrorContext(ctx, "encode lifecycle response failed", "error", err)
	}
}

func resourceTypeParam(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, bool) {
	resourceType := chi.URLParam(r, "resourceType")
	if resourceType != datasetType && resourceType != softwareApplicationType && resourceType != softwareSourceCodeType {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("resource type '%s' not supported", resourceType), nil)
		return "", false
	}
	return resourceType, true
}

func uuidParam(ctx context.Context, w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	value := chi.URLParam(r, name)
	if !validUUID(value) {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid "+name, nil)
		return "", false
	}
	return value, true
}

func writeLifecycleError(r *http.Request, w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrUpstreamTimeout) {
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
		return
	}

	if responseErr, ok := errors.AsType[*domain.UpstreamError](err); ok {
		switch responseErr.StatusCode {
		case http.StatusBadRequest:
			WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid lifecycle request", err)
		case http.StatusForbidden:
			WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "lifecycle operation is not permitted", nil)
		case http.StatusNotFound:
			WriteError(r.Context(), w, http.StatusNotFound, "not_found", "resource version not found", nil)
		case http.StatusConflict:
			WriteError(r.Context(), w, http.StatusConflict, "conflict", "lifecycle operation conflicts with resource state", nil)
		default:
			WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
		}
		return
	}

	log.ErrorContext(r.Context(), "lifecycle request failed", "error", err)
	WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
}
