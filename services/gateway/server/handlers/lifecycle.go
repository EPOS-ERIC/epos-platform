//nolint:dupl,exhaustruct_v5,wrapcheck
package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) GetResourceVersion(ctx context.Context, q gatewayapi.GetResourceVersionParams) (*gatewayapi.ResourceVersion, error) {
	actor, _ := actorID(ctx)
	v, err := h.lifecycle.Get(ctx, actor, string(q.ResourceType), q.ID.String(), q.VersionId.String())
	if err != nil {
		return nil, err
	}
	r, err := mapResourceVersion(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func writeResult(v domain.ResourceVersion, resourceType string) *gatewayapi.WriteResult {
	return &gatewayapi.WriteResult{ResourceType: gatewayapi.WriteResourceType(resourceType), ID: v.ResourceID, Status: gatewayapi.VersioningStatus(strings.ToLower(v.Status))}
}

func (h *ogenHandler) CopyResource(ctx context.Context, p gatewayapi.CopyResourceParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.Copy(ctx, a, p.GroupId.String(), string(p.ResourceType), p.ID.String())
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) DraftFromPublication(ctx context.Context, p gatewayapi.DraftFromPublicationParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.DraftFromPublication(ctx, a, string(p.ResourceType), p.ID.String())
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) DraftFromArchive(ctx context.Context, p gatewayapi.DraftFromArchiveParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.DraftFromArchive(ctx, a, string(p.ResourceType), p.ID.String())
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) DiscardResourceDraftVersion(ctx context.Context, p gatewayapi.DiscardResourceDraftVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	if err := h.lifecycle.Discard(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String()); err != nil {
		return nil, err
	}
	return &gatewayapi.WriteResult{ResourceType: gatewayapi.WriteResourceType(p.ResourceType), ID: p.ID.String(), Status: gatewayapi.VersioningStatusDiscarded}, nil
}

func (h *ogenHandler) SubmitResourceVersion(ctx context.Context, p gatewayapi.SubmitResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.Submit(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) WithdrawResourceVersion(ctx context.Context, p gatewayapi.WithdrawResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.Withdraw(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) ReopenResourceVersion(ctx context.Context, p gatewayapi.ReopenResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.Reopen(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) PublishResourceVersion(ctx context.Context, p gatewayapi.PublishResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.Publish(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) UnpublishResourceVersion(ctx context.Context, p gatewayapi.UnpublishResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.Unpublish(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) RejectResourceVersion(ctx context.Context, q *gatewayapi.RejectResourceRequest, p gatewayapi.RejectResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.lifecycle.Reject(ctx, a, string(p.ResourceType), p.ID.String(), p.VersionId.String(), domain.RejectResourceInput{Note: string(q.Note)})
	if err != nil {
		return nil, err
	}
	return writeResult(v, string(p.ResourceType)), nil
}

func (h *ogenHandler) CreateResource(ctx context.Context, q gatewayapi.CreateResourceDraftRequest, p gatewayapi.CreateResourceParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	typ, in, err := createDraft(q)
	if err != nil {
		return nil, errors.Join(domain.ErrBadRequest, err)
	}
	v, err := h.lifecycle.Create(ctx, a, p.GroupId.String(), typ, in)
	if err != nil {
		return nil, err
	}
	return writeResult(v, typ), nil
}

func (h *ogenHandler) ReplaceResourceDraftVersion(ctx context.Context, q gatewayapi.ReplaceResourceDraftRequest, p gatewayapi.ReplaceResourceDraftVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	typ, in, err := replaceDraft(q)
	if err != nil {
		return nil, errors.Join(domain.ErrBadRequest, err)
	}
	v, err := h.lifecycle.ReplaceDraft(ctx, a, typ, p.ID.String(), p.VersionId.String(), in)
	if err != nil {
		return nil, err
	}
	return writeResult(v, typ), nil
}
