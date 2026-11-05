package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func toProtoLifecycleResourceType(value gatewayapi.LifecycleResourceType) (metadatalifecyclev1.ResourceType, error) {
	switch value {
	case gatewayapi.LifecycleResourceTypeDataset:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET, nil
	case gatewayapi.LifecycleResourceTypeSoftwareApplication:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION, nil
	case gatewayapi.LifecycleResourceTypeSoftwareSourceCode:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE, nil
	default:
		return 0, fmt.Errorf("invalid lifecycle resource type %q", value)
	}
}

func (h *ogenHandler) GetResourceVersion(ctx context.Context, q gatewayapi.GetResourceVersionParams) (*gatewayapi.ResourceVersion, error) {
	actor, _ := authcontext.ActorID(ctx)
	v, resource, err := h.resource.GetVersion(ctx, actor, domain.ResourceType(q.ResourceType), q.ID, q.VersionId)
	if err != nil {
		return nil, fmt.Errorf("get resource version: %w", err)
	}
	r, err := toAPIResourceVersion(v.GetResource(), resource)
	if err != nil {
		return nil, fmt.Errorf("map resource version: %w", err)
	}
	return &r, nil
}

func (h *ogenHandler) ListResourceVersions(ctx context.Context, q gatewayapi.ListResourceVersionsParams) ([]gatewayapi.ResourceVersionSummary, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(q.ResourceType)
	if err != nil {
		return nil, err
	}

	response, err := h.metadataLifecycleClient.ListResourceVersions(ctx, actor, resourceType, q.ID.String())
	if err != nil {
		return nil, fmt.Errorf("list resource versions: %w", err)
	}
	versions, err := toAPIResourceVersionSummaries(response.GetVersions())
	if err != nil {
		return nil, fmt.Errorf("map resource version summaries: %w", err)
	}
	return versions, nil
}

func toAPIWriteResult(v *metadatalifecyclev1.ResourceRecord, resourceType gatewayapi.WriteResourceType) (*gatewayapi.WriteResult, error) {
	if v == nil {
		return nil, errors.New("resource operation returned empty response")
	}
	status, err := toAPIVersioningStatus(v.Status)
	if err != nil {
		return nil, fmt.Errorf("map versioning status: %w", err)
	}
	return &gatewayapi.WriteResult{
		ResourceType: resourceType,
		ID:           v.ResourceId,
		Status:       status,
	}, nil
}

func (h *ogenHandler) CopyResource(ctx context.Context, p gatewayapi.CopyResourceParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.resource.Copy(ctx, a, p.GroupId.String(), resourceType, p.ID.String())
	if err != nil {
		return nil, fmt.Errorf("copy resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) DraftFromPublication(ctx context.Context, p gatewayapi.DraftFromPublicationParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.resource.DraftPublication(ctx, a, resourceType, p.ID.String())
	if err != nil {
		return nil, fmt.Errorf("create draft from publication: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) DraftFromArchive(ctx context.Context, p gatewayapi.DraftFromArchiveParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.resource.DraftArchive(ctx, a, resourceType, p.ID.String())
	if err != nil {
		return nil, fmt.Errorf("create draft from archive: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) DiscardResourceDraftVersion(ctx context.Context, p gatewayapi.DiscardResourceDraftVersionParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return err
	}
	if err := h.resource.Discard(ctx, a, resourceType, p.ID.String(), p.VersionId.String()); err != nil {
		return fmt.Errorf("discard resource draft: %w", err)
	}
	return nil
}

func (h *ogenHandler) SubmitResourceVersion(ctx context.Context, p gatewayapi.SubmitResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.SubmitResource(ctx, a, resourceType, p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("submit resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) WithdrawResourceVersion(ctx context.Context, p gatewayapi.WithdrawResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.WithdrawResource(ctx, a, resourceType, p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("withdraw resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) ReopenResourceVersion(ctx context.Context, p gatewayapi.ReopenResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.ReopenResource(ctx, a, resourceType, p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("reopen resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) PublishResourceVersion(ctx context.Context, p gatewayapi.PublishResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.PublishResource(ctx, a, resourceType, p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("publish resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) UnpublishResourceVersion(ctx context.Context, p gatewayapi.UnpublishResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.UnpublishResource(ctx, a, resourceType, p.ID.String(), p.VersionId.String())
	if err != nil {
		return nil, fmt.Errorf("unpublish resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) RejectResourceVersion(ctx context.Context, q *gatewayapi.RejectResourceRequest, p gatewayapi.RejectResourceVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	resourceType, err := toProtoLifecycleResourceType(p.ResourceType)
	if err != nil {
		return nil, err
	}
	v, err := h.metadataLifecycleClient.RejectResource(ctx, a, resourceType, p.ID.String(), p.VersionId.String(), string(q.Note))
	if err != nil {
		return nil, fmt.Errorf("reject resource: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), gatewayapi.WriteResourceType(p.ResourceType))
}

func (h *ogenHandler) CreateResource(ctx context.Context, q gatewayapi.CreateResourceDraftRequest, p gatewayapi.CreateResourceParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	in, typ, err := toProtoCreateResource(p.GroupId.String(), q)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeValidationFailed, "", err)
	}
	v, err := h.metadataLifecycleClient.CreateResourceDraft(ctx, a, in)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}
	if v == nil || v.Resource == nil {
		return nil, errors.New("create resource returned empty response")
	}
	status, err := toAPIVersioningStatus(v.Resource.Status)
	if err != nil {
		return nil, fmt.Errorf("map versioning status: %w", err)
	}
	return &gatewayapi.WriteResult{
		ResourceType: typ,
		ID:           v.Resource.ResourceId,
		Status:       status,
	}, nil
}

func (h *ogenHandler) ReplaceResourceDraftVersion(ctx context.Context, q gatewayapi.ReplaceResourceDraftRequest, p gatewayapi.ReplaceResourceDraftVersionParams) (*gatewayapi.WriteResult, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	typ, in, err := toProtoReplaceResourceDraft(q)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeValidationFailed, "", err)
	}
	v, err := h.resource.Replace(ctx, a, p.ID.String(), p.VersionId.String(), in)
	if err != nil {
		return nil, fmt.Errorf("replace resource draft: %w", err)
	}
	return toAPIWriteResult(v.GetResource(), typ)
}
