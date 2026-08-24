// Package handlers contains Gateway HTTP handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	backofficeclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/lifecycle"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

//nolint:dupl // Generated request wrappers require distinct concrete field types.
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
	resourceType, ok := resourceTypeParam(ctx, w, r)
	if !ok {
		return
	}

	body, ok := decodeJSONBody(ctx, w, r)
	if !ok {
		return
	}

	var request backofficeclient.CreateResourceRequest
	switch resourceType {
	case datasetType:
		var value backofficeclient.DatasetDraftMetadata
		if err := json.Unmarshal(body, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
			return
		}
		request = backofficeclient.CreateResourceRequest{
			GroupId:  groupID,
			Metadata: backofficeclient.DatasetDraftMetadataAsCreateResourceRequestMetadata(&value),
		}
	case softwareApplicationType:
		var value backofficeclient.SoftwareApplicationDraftMetadata
		if err := json.Unmarshal(body, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
			return
		}
		request = backofficeclient.CreateResourceRequest{
			GroupId:  groupID,
			Metadata: backofficeclient.SoftwareApplicationDraftMetadataAsCreateResourceRequestMetadata(&value),
		}
	case softwareSourceCodeType:
		var value backofficeclient.SoftwareSourceCodeDraftMetadata
		if err := json.Unmarshal(body, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
			return
		}
		request = backofficeclient.CreateResourceRequest{
			GroupId:  groupID,
			Metadata: backofficeclient.SoftwareSourceCodeDraftMetadataAsCreateResourceRequestMetadata(&value),
		}
	}

	record, err := h.service.Create(ctx, actor.ID, groupID, resourceType, request)
	if err != nil {
		if responseErr, ok := errors.AsType[*lifecycle.ResponseError](err); ok {
			if responseErr.ContentType != "" {
				w.Header().Set("Content-Type", responseErr.ContentType)
			}
			w.WriteHeader(responseErr.StatusCode)
			_, _ = w.Write(responseErr.Body)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
			return
		}
		log.ErrorContext(ctx, "lifecycle request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(api.WriteResult{
		ResourceType: api.ResourceType(resourceType),
		ID:           record.GetResourceId(),
		Status:       api.VersioningStatus(strings.ToLower(string(record.GetStatus()))),
	})
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

	body, err := h.service.Get(ctx, actorID, resourceType, id, versionID)
	if err != nil {
		if responseErr, ok := errors.AsType[*lifecycle.ResponseError](err); ok {
			if responseErr.ContentType != "" {
				w.Header().Set("Content-Type", responseErr.ContentType)
			}
			w.WriteHeader(responseErr.StatusCode)
			_, _ = w.Write(responseErr.Body)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
			return
		}
		log.ErrorContext(ctx, "lifecycle request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
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

	body, ok := decodeJSONBody(ctx, w, r)
	if !ok {
		return
	}

	var request backofficeclient.ReplaceResourceDraftRequest
	switch resourceType {
	case datasetType:
		var value backofficeclient.ReplaceDraftRequest
		if err := json.Unmarshal(body, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
			return
		}
		request = backofficeclient.ReplaceResourceDraftRequest{
			ReplaceDraftRequest:                    &value,
			ReplaceSoftwareApplicationDraftRequest: nil,
			ReplaceSoftwareSourceCodeDraftRequest:  nil,
		}
	case softwareApplicationType:
		var value backofficeclient.ReplaceSoftwareApplicationDraftRequest
		if err := json.Unmarshal(body, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
			return
		}
		request = backofficeclient.ReplaceResourceDraftRequest{
			ReplaceDraftRequest:                    nil,
			ReplaceSoftwareApplicationDraftRequest: &value,
			ReplaceSoftwareSourceCodeDraftRequest:  nil,
		}
	case softwareSourceCodeType:
		var value backofficeclient.ReplaceSoftwareSourceCodeDraftRequest
		if err := json.Unmarshal(body, &value); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
			return
		}
		request = backofficeclient.ReplaceResourceDraftRequest{
			ReplaceDraftRequest:                    nil,
			ReplaceSoftwareApplicationDraftRequest: nil,
			ReplaceSoftwareSourceCodeDraftRequest:  &value,
		}
	}

	record, err := h.service.ReplaceDraft(ctx, actor.ID, resourceType, id, versionID, request)
	if err != nil {
		if responseErr, ok := errors.AsType[*lifecycle.ResponseError](err); ok {
			if responseErr.ContentType != "" {
				w.Header().Set("Content-Type", responseErr.ContentType)
			}
			w.WriteHeader(responseErr.StatusCode)
			_, _ = w.Write(responseErr.Body)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
			return
		}
		log.ErrorContext(ctx, "lifecycle request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(api.WriteResult{
		ResourceType: api.ResourceType(resourceType),
		ID:           record.GetResourceId(),
		Status:       api.VersioningStatus(strings.ToLower(string(record.GetStatus()))),
	}); err != nil {
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
		if responseErr, ok := errors.AsType[*lifecycle.ResponseError](err); ok {
			if responseErr.ContentType != "" {
				w.Header().Set("Content-Type", responseErr.ContentType)
			}
			w.WriteHeader(responseErr.StatusCode)
			_, _ = w.Write(responseErr.Body)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
			return
		}
		log.ErrorContext(ctx, "lifecycle request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(api.WriteResult{
		ResourceType: api.ResourceType(resourceType),
		ID:           id,
		Status:       api.DiscardedStatus,
	}); err != nil {
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
	var record *backofficeclient.ResourceRecord
	var err error
	switch action {
	case "reject":
		body, ok := decodeJSONBody(ctx, w, r)
		if !ok {
			return
		}

		var request backofficeclient.RejectRequest
		if err := json.Unmarshal(body, &request); err != nil {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
			return
		}
		record, err = h.service.Reject(ctx, actor.ID, resourceType, id, versionID, request)
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
		if responseErr, ok := errors.AsType[*lifecycle.ResponseError](err); ok {
			if responseErr.ContentType != "" {
				w.Header().Set("Content-Type", responseErr.ContentType)
			}
			w.WriteHeader(responseErr.StatusCode)
			_, _ = w.Write(responseErr.Body)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
			return
		}
		log.ErrorContext(ctx, "lifecycle request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(api.WriteResult{
		ResourceType: api.ResourceType(resourceType),
		ID:           record.GetResourceId(),
		Status:       api.VersioningStatus(strings.ToLower(string(record.GetStatus()))),
	}); err != nil {
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
// calls Backoffice to create a draft from the selected source.
func (h *Lifecycle) createDraft(w http.ResponseWriter, r *http.Request, create func(context.Context, string, string, string) (*backofficeclient.ResourceRecord, error)) {
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
		if responseErr, ok := errors.AsType[*lifecycle.ResponseError](err); ok {
			if responseErr.ContentType != "" {
				w.Header().Set("Content-Type", responseErr.ContentType)
			}
			w.WriteHeader(responseErr.StatusCode)
			_, _ = w.Write(responseErr.Body)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
			return
		}
		log.ErrorContext(ctx, "lifecycle request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(api.WriteResult{
		ResourceType: api.ResourceType(resourceType),
		ID:           record.GetResourceId(),
		Status:       api.VersioningStatus(strings.ToLower(string(record.GetStatus()))),
	}); err != nil {
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

	body, ok := decodeJSONBody(ctx, w, r)
	if !ok {
		return
	}
	var request backofficeclient.CopyResourceRequest
	if err := json.Unmarshal(body, &request); err != nil {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
		return
	}

	record, err := h.service.Copy(ctx, actor.ID, groupID, resourceType, id, request)
	if err != nil {
		if responseErr, ok := errors.AsType[*lifecycle.ResponseError](err); ok {
			if responseErr.ContentType != "" {
				w.Header().Set("Content-Type", responseErr.ContentType)
			}
			w.WriteHeader(responseErr.StatusCode)
			_, _ = w.Write(responseErr.Body)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
			return
		}
		log.ErrorContext(ctx, "lifecycle request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(api.WriteResult{
		ResourceType: api.ResourceType(resourceType),
		ID:           record.GetResourceId(),
		Status:       api.VersioningStatus(strings.ToLower(string(record.GetStatus()))),
	}); err != nil {
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
	if _, err := uuid.Parse(value); err != nil {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid "+name, nil)
		return "", false
	}
	return value, true
}

func decodeJSONBody(ctx context.Context, w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil || !json.Valid(body) {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
		return nil, false
	}
	return body, true
}
