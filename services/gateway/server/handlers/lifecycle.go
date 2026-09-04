package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) GetResourceVersion(ctx context.Context, q gatewayapi.GetResourceVersionParams) (*gatewayapi.ResourceVersion, error) {
	actor, _ := actorID(ctx)
	v, err := h.metadataLifecycle.GetResourceVersion(ctx, actor, string(q.ResourceType), q.ID.String(), q.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("get resource version: %w", err)
	}
	r, err := toAPIResourceVersion(v.GetResource())
	if err != nil {
		return nil, fmt.Errorf("map resource version: %w", err)
	}
	return &r, nil
}

func toAPIWriteResult(v *metadatalifecyclev1.ResourceRecord, resourceType string) (*gatewayapi.WriteResult, error) {
	if v == nil {
		return nil, errors.New("resource operation returned empty response")
	}
	return &gatewayapi.WriteResult{
		ResourceType: gatewayapi.WriteResourceType(resourceType),
		ID:           v.ResourceId,
		Status:       gatewayapi.VersioningStatus(strings.ToLower(string(v.Status))),
	}, nil
}

func (h *ogenHandler) CopyResource(ctx context.Context, p gatewayapi.CopyResourceParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.CopyResource(ctx, a, p.GroupId.String(), string(p.ResourceType), p.ID.String())
	if err != nil {
		return nil, fmt.Errorf("copy resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) DraftFromPublication(ctx context.Context, p gatewayapi.DraftFromPublicationParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.CreateResourceDraftFromPublication(ctx, a, string(p.ResourceType), p.ID.String())
	if err != nil {
		return nil, fmt.Errorf("create draft from publication: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) DraftFromArchive(ctx context.Context, p gatewayapi.DraftFromArchiveParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.CreateResourceDraftFromArchive(ctx, a, string(p.ResourceType), p.ID.String())
	if err != nil {
		return nil, fmt.Errorf("create draft from archive: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) DiscardResourceDraftVersion(ctx context.Context, p gatewayapi.DiscardResourceDraftVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	if err := h.metadataLifecycle.DiscardResourceDraft(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String()); err != nil {
		return nil, fmt.Errorf("discard resource draft: %w", err)
	}
	return &gatewayapi.WriteResult{
		ResourceType: gatewayapi.WriteResourceType(p.ResourceType),
		ID:           p.ID.String(),
		Status:       gatewayapi.VersioningStatusDiscarded,
	}, nil
}

func (h *ogenHandler) SubmitResourceVersion(ctx context.Context, p gatewayapi.SubmitResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.SubmitResource(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("submit resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) WithdrawResourceVersion(ctx context.Context, p gatewayapi.WithdrawResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.WithdrawResource(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("withdraw resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) ReopenResourceVersion(ctx context.Context, p gatewayapi.ReopenResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.ReopenResource(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("reopen resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) PublishResourceVersion(ctx context.Context, p gatewayapi.PublishResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.PublishResource(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("publish resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) UnpublishResourceVersion(ctx context.Context, p gatewayapi.UnpublishResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.UnpublishResource(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("unpublish resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) RejectResourceVersion(ctx context.Context, q *gatewayapi.RejectResourceRequest, p gatewayapi.RejectResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.metadataLifecycle.RejectResource(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String(), string(q.Note))
	if err != nil {
		return nil, fmt.Errorf("reject resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), string(p.ResourceType))
}

func (h *ogenHandler) CreateResource(ctx context.Context, q gatewayapi.CreateResourceDraftRequest, p gatewayapi.CreateResourceParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	in, typ, err := toProtoCreateResource(p.GroupId.String(), q)
	if err != nil {
		return nil, errors.Join(domain.ErrBadRequest, err)
	}
	v, err := h.metadataLifecycle.CreateResourceDraft(ctx, a, in)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}
	if v == nil || v.Resource == nil {
		return nil, errors.New("create resource returned empty response")
	}
	return &gatewayapi.WriteResult{ResourceType: gatewayapi.WriteResourceType(typ), ID: v.Resource.ResourceId, Status: gatewayapi.VersioningStatus(strings.ToLower(v.Resource.Status))}, nil
}

func (h *ogenHandler) ReplaceResourceDraftVersion(ctx context.Context, q gatewayapi.ReplaceResourceDraftRequest, p gatewayapi.ReplaceResourceDraftVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	typ, in, err := toProtoReplaceResourceDraft(q)
	if err != nil {
		return nil, errors.Join(domain.ErrBadRequest, err)
	}
	v, err := h.metadataLifecycle.ReplaceResourceDraft(ctx, a, p.ID.String(), p.VersionId.String(), in)
	if err != nil {
		return nil, fmt.Errorf("replace resource draft: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), typ)
}
