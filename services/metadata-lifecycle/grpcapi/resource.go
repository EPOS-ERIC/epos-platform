package grpcapi

// TODO: format this file, possibly split it in multiple files and most likely simplify the logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/datasets"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwareapplications"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwaresourcecodes"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ResourceServer struct {
	metadatalifecyclev1.UnimplementedResourceServiceServer
	datasets     *datasets.Service
	applications *softwareapplications.Service
	sourceCodes  *softwaresourcecodes.Service
}

func NewResource(datasets *datasets.Service, applications *softwareapplications.Service, sourceCodes *softwaresourcecodes.Service) *ResourceServer {
	return &ResourceServer{
		datasets:     datasets,
		applications: applications,
		sourceCodes:  sourceCodes,
	}
}

func (s *ResourceServer) CreateResource(ctx context.Context, request *metadatalifecyclev1.CreateResourceRequest) (*metadatalifecyclev1.CreateResourceResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "resource request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseUUIDAt(request.GetGroupId(), "group id", "groupId")
	if err != nil {
		return nil, err
	}

	switch value := request.GetMetadata().(type) {
	case *metadatalifecyclev1.CreateResourceRequest_Dataset:
		if value.Dataset == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, "dataset metadata is required")
		}
		input, err := datasetInput(value.Dataset)
		if err != nil {
			return nil, apperror.PrefixTarget(err, "metadata")
		}
		created, err := s.datasets.CreateDraft(ctx, actor, groupID, input)
		if err != nil {
			return nil, err
		}
		return createResponse(created.DatasetID, created.VersionID, created.Revision, created.Status), nil
	case *metadatalifecyclev1.CreateResourceRequest_SoftwareApplication:
		if value.SoftwareApplication == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, "software application metadata is required")
		}
		input, err := softwareApplicationInput(value.SoftwareApplication)
		if err != nil {
			return nil, apperror.PrefixTarget(err, "metadata")
		}
		created, err := s.applications.CreateDraft(ctx, actor, groupID, input)
		if err != nil {
			return nil, err
		}
		return createResponse(created.SoftwareApplicationID, created.VersionID, created.Revision, created.Status), nil
	case *metadatalifecyclev1.CreateResourceRequest_SoftwareSourceCode:
		if value.SoftwareSourceCode == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, "software source code metadata is required")
		}
		input, err := softwareSourceCodeInput(value.SoftwareSourceCode)
		if err != nil {
			return nil, apperror.PrefixTarget(err, "metadata")
		}
		created, err := s.sourceCodes.CreateDraft(ctx, actor, groupID, input)
		if err != nil {
			return nil, err
		}
		return createResponse(created.SoftwareSourceCodeID, created.VersionID, created.Revision, created.Status), nil
	default:
		return nil, apperror.New(apperror.CodeValidationFailed, "resource metadata is required")
	}
}

func (s *ResourceServer) GetResourceVersion(ctx context.Context, request *metadatalifecyclev1.GetResourceVersionRequest) (*metadatalifecyclev1.GetResourceVersionResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "resource version request is required")
	}
	resourceID, err := parseUUIDAt(request.GetResourceId(), "resource id", "id")
	if err != nil {
		return nil, err
	}
	versionID, err := parseUUIDAt(request.GetVersionId(), "version id", "versionId")
	if err != nil {
		return nil, err
	}
	actor, err := optionalActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var response *metadatalifecyclev1.ResourceVersion
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		value, err := s.datasets.Read(ctx, actor, resourceID, versionID)
		if err != nil {
			return nil, err
		}
		response = toProtoDatasetVersion(value)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		value, err := s.applications.Read(ctx, actor, resourceID, versionID)
		if err != nil {
			return nil, err
		}
		response = toProtoSoftwareApplicationVersion(value)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		value, err := s.sourceCodes.Read(ctx, actor, resourceID, versionID)
		if err != nil {
			return nil, err
		}
		response = toProtoSoftwareSourceCodeVersion(value)
	default:
		return nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
	}
	return &metadatalifecyclev1.GetResourceVersionResponse{Resource: response}, nil
}

func (s *ResourceServer) PrepareDistributionDownload(ctx context.Context, request *metadatalifecyclev1.PrepareDistributionDownloadRequest) (*metadatalifecyclev1.PrepareDistributionDownloadResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "distribution download request is required")
	}

	distributionID, err := parseUUIDAt(request.GetDistributionId(), "distribution id", "distributionId")
	if err != nil {
		return nil, err
	}

	actor, err := optionalActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	download, err := s.datasets.PrepareDownload(ctx, actor, distributionID)
	if err != nil {
		return nil, err
	}

	return toProtoDistributionDownload(download), nil
}

func (s *ResourceServer) PrepareDistributionAccess(ctx context.Context, request *metadatalifecyclev1.PrepareDistributionAccessRequest) (*metadatalifecyclev1.PrepareDistributionAccessResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "distribution access request is required")
	}

	distributionID, err := parseUUIDAt(request.GetDistributionId(), "distribution id", "distributionId")
	if err != nil {
		return nil, err
	}

	input, err := toDomainDistributionAccessInput(request)
	if err != nil {
		return nil, err
	}

	actor, err := optionalActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	prepared, err := s.datasets.PrepareAccess(ctx, actor, distributionID, input)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.PrepareDistributionAccessResponse{
		ResourceId: prepared.ResourceID.String(),
		VersionId:  prepared.VersionID.String(),
		Method:     webServiceMethodOutput(prepared.Method),
		Url:        prepared.URL,
		Headers:    prepared.Headers,
		Body:       prepared.Body,
	}, nil
}

func (s *ResourceServer) ListResourceVersions(ctx context.Context, request *metadatalifecyclev1.ListResourceVersionsRequest) (*metadatalifecyclev1.ListResourceVersionsResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "resource versions request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := parseUUIDAt(request.GetResourceId(), "resource id", "id")
	if err != nil {
		return nil, err
	}

	var summaries []domain.ResourceVersionSummary
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		summaries, err = s.datasets.ListVersions(ctx, actor, resourceID)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		summaries, err = s.applications.ListVersions(ctx, actor, resourceID)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		summaries, err = s.sourceCodes.ListVersions(ctx, actor, resourceID)
	default:
		return nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
	}
	if err != nil {
		return nil, err
	}

	response := &metadatalifecyclev1.ListResourceVersionsResponse{Versions: make([]*metadatalifecyclev1.ResourceVersionSummary, 0, len(summaries))}
	for _, summary := range summaries {
		response.Versions = append(response.Versions, resourceVersionSummaryOutput(summary))
	}
	return response, nil
}

func (s *ResourceServer) ReplaceResourceDraft(ctx context.Context, request *metadatalifecyclev1.ReplaceResourceDraftRequest) (*metadatalifecyclev1.ReplaceResourceDraftResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "resource draft request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := parseUUIDAt(request.GetResourceId(), "resource id", "id")
	if err != nil {
		return nil, err
	}
	versionID, err := parseUUIDAt(request.GetVersionId(), "version id", "versionId")
	if err != nil {
		return nil, err
	}

	switch value := request.GetMetadata().(type) {
	case *metadatalifecyclev1.ReplaceResourceDraftRequest_Dataset:
		if value.Dataset == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, "dataset metadata is required")
		}
		metadata, err := datasetInput(value.Dataset)
		if err != nil {
			return nil, apperror.PrefixTarget(err, "metadata")
		}
		input := domain.DatasetDraftInput{
			Revision: request.GetRevision(),
			Metadata: metadata,
		}
		if err := input.Normalize(); err != nil {
			return nil, err
		}
		result, removed, err := s.datasets.ReplaceDraft(ctx, actor, resourceID, versionID, input)
		if err != nil {
			return nil, err
		}
		return &metadatalifecyclev1.ReplaceResourceDraftResponse{
			RemovedDistributionIds: uuidStrings(removed),
			Resource: &metadatalifecyclev1.ResourceRecord{
				ResourceId: result.DatasetID.String(),
				VersionId:  result.VersionID.String(),
				Revision:   result.Revision,
				Status:     toGRPCVersioningStatus(result.Status),
			},
		}, nil
	case *metadatalifecyclev1.ReplaceResourceDraftRequest_SoftwareApplication:
		if value.SoftwareApplication == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, "software application metadata is required")
		}
		metadata, err := softwareApplicationInput(value.SoftwareApplication)
		if err != nil {
			return nil, apperror.PrefixTarget(err, "metadata")
		}
		input := domain.SoftwareApplicationDraftInput{
			Revision: request.GetRevision(),
			Metadata: metadata,
		}
		if err := input.Normalize(); err != nil {
			return nil, err
		}
		result, err := s.applications.ReplaceDraft(ctx, actor, resourceID, versionID, input)
		if err != nil {
			return nil, err
		}
		return &metadatalifecyclev1.ReplaceResourceDraftResponse{
			Resource: &metadatalifecyclev1.ResourceRecord{
				ResourceId: result.SoftwareApplicationID.String(),
				VersionId:  result.VersionID.String(),
				Revision:   result.Revision,
				Status:     toGRPCVersioningStatus(result.Status),
			},
		}, nil
	case *metadatalifecyclev1.ReplaceResourceDraftRequest_SoftwareSourceCode:
		if value.SoftwareSourceCode == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, "software source code metadata is required")
		}
		metadata, err := softwareSourceCodeInput(value.SoftwareSourceCode)
		if err != nil {
			return nil, apperror.PrefixTarget(err, "metadata")
		}
		input := domain.SoftwareSourceCodeDraftInput{
			Revision: request.GetRevision(),
			Metadata: metadata,
		}
		if err := input.Normalize(); err != nil {
			return nil, err
		}
		result, err := s.sourceCodes.ReplaceDraft(ctx, actor, resourceID, versionID, input)
		if err != nil {
			return nil, err
		}
		return &metadatalifecyclev1.ReplaceResourceDraftResponse{
			Resource: &metadatalifecyclev1.ResourceRecord{
				ResourceId: result.SoftwareSourceCodeID.String(),
				VersionId:  result.VersionID.String(),
				Revision:   result.Revision,
				Status:     toGRPCVersioningStatus(result.Status),
			},
		}, nil
	default:
		return nil, apperror.New(apperror.CodeValidationFailed, "resource metadata is required")
	}
}

func (s *ResourceServer) DiscardResourceDraft(ctx context.Context, request *metadatalifecyclev1.DiscardResourceDraftRequest) (*metadatalifecyclev1.DiscardResourceDraftResponse, error) {
	actor, resourceID, versionID, err := s.lifecycleRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	var removed []uuid.UUID
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		removed, err = s.datasets.Discard(ctx, actor, resourceID, versionID)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		err = s.applications.Discard(ctx, actor, resourceID, versionID)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		err = s.sourceCodes.Discard(ctx, actor, resourceID, versionID)
	default:
		return nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
	}
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DiscardResourceDraftResponse{
		RemovedDistributionIds: uuidStrings(removed),
	}, nil
}

//nolint:dupl // Resource lifecycle transitions intentionally mirror per-resource services.
func (s *ResourceServer) SubmitResource(ctx context.Context, request *metadatalifecyclev1.SubmitResourceRequest) (*metadatalifecyclev1.SubmitResourceResponse, error) {
	record, notifications, err := s.lifecycleNotificationTransition(ctx, request, func(actor string, id, version uuid.UUID) (any, []domain.Notification, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Submit(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Submit(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Submit(ctx, actor, id, version)
		default:
			return nil, nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}

	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.SubmitResourceResponse{
		Resource:      record,
		Notifications: protoNotifications,
	}, nil
}

func (s *ResourceServer) WithdrawResource(ctx context.Context, request *metadatalifecyclev1.WithdrawResourceRequest) (*metadatalifecyclev1.WithdrawResourceResponse, error) {
	record, err := s.lifecycleTransition(ctx, request, func(actor string, id, version uuid.UUID) (any, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Withdraw(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Withdraw(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Withdraw(ctx, actor, id, version)
		default:
			return nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.WithdrawResourceResponse{Resource: record}, nil
}

func (s *ResourceServer) RejectResource(ctx context.Context, request *metadatalifecyclev1.RejectResourceRequest) (*metadatalifecyclev1.RejectResourceResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "resource request is required")
	}
	record, notifications, err := s.lifecycleNotificationTransition(ctx, request, func(actor string, resourceID, versionID uuid.UUID) (any, []domain.Notification, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET,
			metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION,
			metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		default:
			return nil, nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
		}
		value := domain.RejectionInput{Note: request.GetNote()}
		if err := value.Normalize(); err != nil {
			return nil, nil, err
		}
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Reject(ctx, actor, resourceID, versionID, value)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Reject(ctx, actor, resourceID, versionID, value)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Reject(ctx, actor, resourceID, versionID, value)
		default:
			return nil, nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}
	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.RejectResourceResponse{Resource: record, Notifications: protoNotifications}, nil
}

func (s *ResourceServer) ReopenResource(ctx context.Context, request *metadatalifecyclev1.ReopenResourceRequest) (*metadatalifecyclev1.ReopenResourceResponse, error) {
	record, err := s.lifecycleTransition(ctx, request, func(actor string, id, version uuid.UUID) (any, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Reopen(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Reopen(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Reopen(ctx, actor, id, version)
		default:
			return nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.ReopenResourceResponse{Resource: record}, nil
}

//nolint:dupl // Resource lifecycle transitions intentionally mirror per-resource services.
func (s *ResourceServer) PublishResource(ctx context.Context, request *metadatalifecyclev1.PublishResourceRequest) (*metadatalifecyclev1.PublishResourceResponse, error) {
	record, notifications, err := s.lifecycleNotificationTransition(ctx, request, func(actor string, id, version uuid.UUID) (any, []domain.Notification, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Publish(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Publish(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Publish(ctx, actor, id, version)
		default:
			return nil, nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}

	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.PublishResourceResponse{Resource: record, Notifications: protoNotifications}, nil
}

//nolint:dupl // Resource lifecycle transitions intentionally mirror per-resource services.
func (s *ResourceServer) UnpublishResource(ctx context.Context, request *metadatalifecyclev1.UnpublishResourceRequest) (*metadatalifecyclev1.UnpublishResourceResponse, error) {
	record, notifications, err := s.lifecycleNotificationTransition(ctx, request, func(actor string, id, version uuid.UUID) (any, []domain.Notification, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Unpublish(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Unpublish(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Unpublish(ctx, actor, id, version)
		default:
			return nil, nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}

	protoNotifications, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.UnpublishResourceResponse{Resource: record, Notifications: protoNotifications}, nil
}

func (s *ResourceServer) DraftResourceFromPublication(ctx context.Context, request *metadatalifecyclev1.DraftResourceFromPublicationRequest) (*metadatalifecyclev1.DraftResourceFromPublicationResponse, error) {
	record, pairs, err := s.draftResource(ctx, request, true)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DraftResourceFromPublicationResponse{
		Resource:            record,
		CopiedDistributions: toProtoDistributionIDPairs(pairs),
	}, nil
}

func (s *ResourceServer) DraftResourceFromLatestArchive(ctx context.Context, request *metadatalifecyclev1.DraftResourceFromLatestArchiveRequest) (*metadatalifecyclev1.DraftResourceFromLatestArchiveResponse, error) {
	record, pairs, err := s.draftResource(ctx, request, false)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DraftResourceFromLatestArchiveResponse{
		Resource:            record,
		CopiedDistributions: toProtoDistributionIDPairs(pairs),
	}, nil
}

func (s *ResourceServer) CopyPublishedResource(ctx context.Context, request *metadatalifecyclev1.CopyPublishedResourceRequest) (*metadatalifecyclev1.CopyPublishedResourceResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "resource request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	sourceID, err := parseUUIDAt(request.GetResourceId(), "resource id", "id")
	if err != nil {
		return nil, err
	}
	groupID, err := parseUUIDAt(request.GetGroupId(), "group id", "groupId")
	if err != nil {
		return nil, err
	}
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		result, pairs, e := s.datasets.CopyPublished(ctx, actor, sourceID, groupID)
		if e != nil {
			return nil, e
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.CopyPublishedResourceResponse{
			Resource:            record,
			CopiedDistributions: toProtoDistributionIDPairs(pairs),
		}, e
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		result, e := s.applications.CopyPublished(ctx, actor, sourceID, groupID)
		if e != nil {
			return nil, e
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.CopyPublishedResourceResponse{Resource: record}, e
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		result, e := s.sourceCodes.CopyPublished(ctx, actor, sourceID, groupID)
		if e != nil {
			return nil, e
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.CopyPublishedResourceResponse{Resource: record}, e
	default:
		return nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
	}
}

type resourceVersionRequest interface {
	GetResourceId() string
	GetVersionId() string
}

func (s *ResourceServer) lifecycleRequest(ctx context.Context, request resourceVersionRequest) (string, uuid.UUID, uuid.UUID, error) {
	if request == nil {
		return "", uuid.Nil, uuid.Nil, apperror.New(apperror.CodeValidationFailed, "resource request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	id, err := parseUUIDAt(request.GetResourceId(), "resource id", "id")
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	version, err := parseUUIDAt(request.GetVersionId(), "version id", "versionId")
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	return actor, id, version, nil
}

func (s *ResourceServer) lifecycleTransition(ctx context.Context, request resourceVersionRequest, call func(string, uuid.UUID, uuid.UUID) (any, error)) (*metadatalifecyclev1.ResourceRecord, error) {
	actor, id, version, err := s.lifecycleRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	result, err := call(actor, id, version)
	if err != nil {
		return nil, err
	}
	return lifecycleResponse(result)
}

func (s *ResourceServer) lifecycleNotificationTransition(ctx context.Context, request resourceVersionRequest, call func(string, uuid.UUID, uuid.UUID) (any, []domain.Notification, error)) (*metadatalifecyclev1.ResourceRecord, []domain.Notification, error) {
	// TODO: to simplify this whole file with the submit/publis/reject actions which are basically the same, we could create an interface and use that to switch on the functions?
	actor, id, version, err := s.lifecycleRequest(ctx, request)
	if err != nil {
		return nil, nil, err
	}
	result, notifications, err := call(actor, id, version)
	if err != nil {
		return nil, nil, err
	}
	record, err := lifecycleResponse(result)
	if err != nil {
		return nil, nil, err
	}
	return record, notifications, nil
}

type draftResourceRequest interface {
	GetResourceType() metadatalifecyclev1.ResourceType
	GetResourceId() string
}

func (s *ResourceServer) draftResource(ctx context.Context, request draftResourceRequest, fromPublication bool) (*metadatalifecyclev1.ResourceRecord, map[uuid.UUID]uuid.UUID, error) {
	if request == nil {
		return nil, nil, apperror.New(apperror.CodeValidationFailed, "resource request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	id, err := parseUUIDAt(request.GetResourceId(), "resource id", "id")
	if err != nil {
		return nil, nil, err
	}
	var result any
	var pairs map[uuid.UUID]uuid.UUID
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		if fromPublication {
			result, pairs, err = s.datasets.DraftFromPublication(ctx, actor, id)
		} else {
			result, pairs, err = s.datasets.DraftFromLatestArchive(ctx, actor, id)
		}
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		if fromPublication {
			result, err = s.applications.DraftFromPublication(ctx, actor, id)
		} else {
			result, err = s.applications.DraftFromLatestArchive(ctx, actor, id)
		}
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		if fromPublication {
			result, err = s.sourceCodes.DraftFromPublication(ctx, actor, id)
		} else {
			result, err = s.sourceCodes.DraftFromLatestArchive(ctx, actor, id)
		}
	default:
		return nil, nil, apperror.New(apperror.CodeValidationFailed, "resource type is required")
	}
	if err != nil {
		return nil, nil, err
	}
	record, err := lifecycleResponse(result)

	return record, pairs, err
}

func lifecycleResponse(value any) (*metadatalifecyclev1.ResourceRecord, error) {
	switch result := value.(type) {
	case domain.DatasetVersion:
		return &metadatalifecyclev1.ResourceRecord{ResourceId: result.DatasetID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: toGRPCVersioningStatus(result.Status)}, nil
	case domain.SoftwareApplicationVersion:
		return &metadatalifecyclev1.ResourceRecord{ResourceId: result.SoftwareApplicationID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: toGRPCVersioningStatus(result.Status)}, nil
	case domain.SoftwareSourceCodeVersion:
		return &metadatalifecyclev1.ResourceRecord{ResourceId: result.SoftwareSourceCodeID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: toGRPCVersioningStatus(result.Status)}, nil
	default:
		return nil, apperror.Default(apperror.CodeInternal)
	}
}

func optionalActorFromContext(ctx context.Context) (*string, error) {
	md, ok := grpcmetadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil
	}
	values := md.Get(actorMetadataKey)
	if len(values) == 0 {
		return nil, nil
	}
	if len(values) != 1 {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid actor")
	}
	actor, err := domain.NormalizeActor(values[0])
	if err != nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid actor")
	}
	return &actor, nil
}

func datasetResourceMetadata(value domain.DatasetMetadata) *metadatalifecyclev1.DatasetResourceMetadata {
	result := &metadatalifecyclev1.DatasetResourceMetadata{
		Title:                     value.Title,
		Description:               value.Description,
		Keywords:                  value.Keywords,
		AccrualPeriodicity:        toProtoAccrualPeriodicity(value.AccrualPeriodicity),
		QualityAnnotation:         value.QualityAnnotation,
		ConformsTo:                value.ConformsTo,
		Provenance:                value.Provenance,
		Type:                      toProtoDatasetType(value.Type),
		Version:                   value.Version,
		SpatialResolutionInMeters: value.SpatialResolutionInMeters,
		TemporalResolution:        value.TemporalResolution,
		AccessRight:               datasetAccessRightOutput(value.AccessRight),
		SpatialCoverage:           geometryString(value.SpatialCoverage),
		Created:                   timestampOutput(value.SourceCreated),
		Updated:                   timestampOutput(value.SourceUpdated),
	}
	if value.TemporalCoverage != nil {
		result.TemporalRange = &metadatalifecyclev1.DatasetResourceMetadata_TemporalCoverage{TemporalCoverage: temporalOutput(value.TemporalCoverage)}
	}
	if value.GeologicalTemporalCoverage != nil {
		result.TemporalRange = &metadatalifecyclev1.DatasetResourceMetadata_GeologicalTemporalCoverage{GeologicalTemporalCoverage: geologicalTemporalOutput(value.GeologicalTemporalCoverage)}
	}
	for _, item := range value.PersistentIdentifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, identifierOutput(item))
	}
	for _, item := range value.Providers {
		result.Providers = append(result.Providers, referenceOutput(item.ID, item.Name))
	}
	result.Categories = categoryReferences(value.Categories)
	for _, item := range value.ContactPoints {
		result.ContactPoints = append(result.ContactPoints, item.ID.String())
	}
	for _, item := range value.Distributions {
		result.Distributions = append(result.Distributions, distributionOutput(item))
	}
	return result
}

func applicationResourceMetadata(value domain.SoftwareApplicationMetadata) *metadatalifecyclev1.SoftwareApplicationResourceMetadata {
	result := &metadatalifecyclev1.SoftwareApplicationResourceMetadata{
		Name:             value.Name,
		Description:      value.Description,
		MainEntityOfPage: value.MainEntityOfPage,
		License:          value.License,
		Version:          value.Version,
		Keywords:         value.Keywords,
		DownloadUrl:      value.DownloadURL,
		Requirements:     value.Requirements,
		OperatingSystem:  value.OperatingSystem,
	}
	for _, item := range value.Creators {
		result.Creators = append(result.Creators, referenceOutput(item.ID, item.Name))
	}
	for _, item := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierOutput(item))
	}
	result.Categories = categoryReferences(value.Categories)
	for _, item := range value.ContactPoints {
		result.ContactPoints = append(result.ContactPoints, item.ID.String())
	}
	return result
}

func sourceCodeResourceMetadata(value domain.SoftwareSourceCodeMetadata) *metadatalifecyclev1.SoftwareSourceCodeResourceMetadata {
	result := &metadatalifecyclev1.SoftwareSourceCodeResourceMetadata{
		Name:                 value.Name,
		Description:          value.Description,
		ProgrammingLanguage:  value.ProgrammingLanguage,
		Keywords:             value.Keywords,
		Repository:           value.Repository,
		License:              value.License,
		SoftwareRequirements: value.SoftwareRequirements,
		RuntimePlatform:      value.RuntimePlatform,
		MainEntityOfPage:     value.MainEntityOfPage,
		Version:              value.Version,
	}
	for _, item := range value.Creators {
		result.Creators = append(result.Creators, referenceOutput(item.ID, item.Name))
	}
	for _, item := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierOutput(item))
	}
	result.Categories = categoryReferences(value.Categories)
	for _, item := range value.ContactPoints {
		result.ContactPoints = append(result.ContactPoints, item.ID.String())
	}
	return result
}

func referenceOutput(id uuid.UUID, name string) *metadatalifecyclev1.EntityReference {
	return &metadatalifecyclev1.EntityReference{Id: id.String(), Name: name}
}

func categoryReferences(values []domain.CategoryMetadata) []*metadatalifecyclev1.EntityReference {
	var result []*metadatalifecyclev1.EntityReference
	for _, value := range values {
		result = append(result, referenceOutput(value.ID, value.Name))
	}
	return result
}

func identifierOutput(value domain.Identifier) *metadatalifecyclev1.VersionedIdentifier {
	return &metadatalifecyclev1.VersionedIdentifier{
		Id:         value.ID.String(),
		ResourceId: value.ResourceID.String(),
		Type:       string(value.Type),
		Value:      value.Value,
	}
}

func geometryString(value geom.T) *string {
	if value == nil {
		return nil
	}
	result, err := wkt.Marshal(value)
	if err != nil {
		return nil
	}
	return &result
}

func timestampOutput(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

func temporalOutput(value *domain.Temporal) *metadatalifecyclev1.Temporal {
	if value == nil {
		return nil
	}
	return &metadatalifecyclev1.Temporal{Start: timestampOutput(value.Start), End: timestampOutput(value.End)}
}

func geologicalTemporalOutput(value *domain.GeologicalTemporal) *metadatalifecyclev1.GeologicalTemporalCoverage {
	if value == nil {
		return nil
	}
	return &metadatalifecyclev1.GeologicalTemporalCoverage{StartMya: value.StartMYA, EndMya: value.EndMYA}
}

func distributionOutput(value domain.Distribution) *metadatalifecyclev1.Distribution {
	result := &metadatalifecyclev1.Distribution{
		Id:          value.ID.String(),
		ResourceId:  value.ResourceID.String(),
		Title:       value.Title,
		Description: value.Description,
		License:     value.License,
		Type:        toProtoDistributionType(value.Type),
		ByteSize:    value.ByteSize,
	}
	if value.DownloadableFile != nil {
		result.Delivery = &metadatalifecyclev1.Distribution_DownloadableFile{DownloadableFile: &metadatalifecyclev1.DownloadableFile{Url: value.DownloadableFile.URL, Format: value.DownloadableFile.Format}}
	}
	if value.WebService != nil {
		result.Delivery = &metadatalifecyclev1.Distribution_WebService{
			WebService: toProtoWebService(value.WebService),
		}
	}
	return result
}

func parameterOutput(value domain.WebServiceParameter) *metadatalifecyclev1.Parameter {
	result := &metadatalifecyclev1.Parameter{
		Id:          value.ID.String(),
		ResourceId:  value.ResourceID.String(),
		Position:    int32(value.Position),
		Name:        value.Name,
		Label:       value.Label,
		Description: value.Description,
		Required:    value.Required,
		ReadOnly:    value.ReadOnly,
		Role:        webServiceParameterRoleOutput(value.Role),
		Location:    webServiceParameterLocationOutput(value.Location),
		Explode:     value.Explode,
	}
	switch item := value.Schema.(type) {
	case domain.StringSchema:
		result.Value = &metadatalifecyclev1.Parameter_String_{
			String_: &metadatalifecyclev1.StringParameter{
				Default:        item.Default,
				Enum:           stringEnumOutput(item.Enum),
				Pattern:        item.Pattern,
				MultipleValues: item.MultipleValues,
			},
		}
	case domain.BooleanSchema:
		result.Value = &metadatalifecyclev1.Parameter_Boolean{
			Boolean: &metadatalifecyclev1.BooleanParameter{
				Default: item.Default,
			},
		}
	case domain.IntegerSchema:
		result.Value = &metadatalifecyclev1.Parameter_Integer{
			Integer: &metadatalifecyclev1.IntegerParameter{
				Minimum:        item.Minimum,
				Maximum:        item.Maximum,
				Default:        item.Default,
				Enum:           integerEnumOutput(item.Enum),
				MultipleValues: item.MultipleValues,
			},
		}
	case domain.FloatSchema:
		result.Value = &metadatalifecyclev1.Parameter_Float{
			Float: &metadatalifecyclev1.FloatParameter{
				Minimum:        item.Minimum,
				Maximum:        item.Maximum,
				Default:        item.Default,
				Enum:           floatEnumOutput(item.Enum),
				MultipleValues: item.MultipleValues,
			},
		}
	case domain.DateTimeSchema:
		result.Value = &metadatalifecyclev1.Parameter_DateTime{
			DateTime: &metadatalifecyclev1.DateTimeParameter{
				Minimum: timestampOutput(item.Minimum),
				Maximum: timestampOutput(item.Maximum),
				Default: timestampOutput(item.Default),
				Layout:  item.Layout,
			},
		}
	case domain.TemplateSchema:
		template := &metadatalifecyclev1.TemplateParameter{Template: item.Template}
		for _, parameter := range item.Parameters {
			template.Parameters = append(template.Parameters, scalarParameterOutput(parameter))
		}
		result.Value = &metadatalifecyclev1.Parameter_Template{Template: template}
	}
	return result
}

func scalarParameterOutput(value domain.Parameter) *metadatalifecyclev1.ScalarParameter {
	result := &metadatalifecyclev1.ScalarParameter{
		ResourceId:  value.ResourceID.String(),
		Position:    int32(value.Position),
		Name:        value.Name,
		Label:       value.Label,
		Description: value.Description,
		Role:        webServiceParameterRoleOutput(value.Role),
		Required:    value.Required,
		ReadOnly:    value.ReadOnly,
		Id:          value.ID.String(),
	}
	switch item := value.Schema.(type) {
	case domain.StringSchema:
		result.Value = &metadatalifecyclev1.ScalarParameter_String_{
			String_: &metadatalifecyclev1.ScalarStringParameter{
				Default: item.Default,
				Enum:    stringEnumOutput(item.Enum),
				Pattern: item.Pattern,
			},
		}
	case domain.BooleanSchema:
		result.Value = &metadatalifecyclev1.ScalarParameter_Boolean{
			Boolean: &metadatalifecyclev1.ScalarBooleanParameter{
				Default: item.Default,
			},
		}
	case domain.IntegerSchema:
		result.Value = &metadatalifecyclev1.ScalarParameter_Integer{
			Integer: &metadatalifecyclev1.ScalarIntegerParameter{
				Minimum: item.Minimum,
				Maximum: item.Maximum,
				Default: item.Default,
				Enum:    integerEnumOutput(item.Enum),
			},
		}
	case domain.FloatSchema:
		result.Value = &metadatalifecyclev1.ScalarParameter_Float{
			Float: &metadatalifecyclev1.ScalarFloatParameter{
				Minimum: item.Minimum,
				Maximum: item.Maximum,
				Default: item.Default,
				Enum:    floatEnumOutput(item.Enum),
			},
		}
	case domain.DateTimeSchema:
		result.Value = &metadatalifecyclev1.ScalarParameter_DateTime{
			DateTime: &metadatalifecyclev1.ScalarDateTimeParameter{
				Minimum: timestampOutput(item.Minimum),
				Maximum: timestampOutput(item.Maximum),
				Default: timestampOutput(item.Default),
				Layout:  item.Layout,
			},
		}
	}
	return result
}

func stringEnumOutput(values []domain.StringEnumOption) []*metadatalifecyclev1.StringEnumOption {
	if values == nil {
		return nil
	}
	result := make([]*metadatalifecyclev1.StringEnumOption, len(values))
	for i, value := range values {
		result[i] = &metadatalifecyclev1.StringEnumOption{Value: value.Value, Label: value.Label}
	}
	return result
}

func integerEnumOutput(values []domain.IntegerEnumOption) []*metadatalifecyclev1.IntegerEnumOption {
	if values == nil {
		return nil
	}
	result := make([]*metadatalifecyclev1.IntegerEnumOption, len(values))
	for i, value := range values {
		result[i] = &metadatalifecyclev1.IntegerEnumOption{Value: value.Value, Label: value.Label}
	}
	return result
}

func floatEnumOutput(values []domain.FloatEnumOption) []*metadatalifecyclev1.FloatEnumOption {
	if values == nil {
		return nil
	}
	result := make([]*metadatalifecyclev1.FloatEnumOption, len(values))
	for i, value := range values {
		result[i] = &metadatalifecyclev1.FloatEnumOption{Value: value.Value, Label: value.Label}
	}
	return result
}

func stringEnumInput(values []*metadatalifecyclev1.StringEnumOption) ([]domain.StringEnumOption, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]domain.StringEnumOption, len(values))
	for i, value := range values {
		if value == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, fmt.Sprintf("string enum option %d is required", i))
		}
		result[i] = domain.StringEnumOption{Value: value.Value, Label: value.Label}
	}
	return result, nil
}

func integerEnumInput(values []*metadatalifecyclev1.IntegerEnumOption) ([]domain.IntegerEnumOption, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]domain.IntegerEnumOption, len(values))
	for i, value := range values {
		if value == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, fmt.Sprintf("integer enum option %d is required", i))
		}
		result[i] = domain.IntegerEnumOption{Value: value.Value, Label: value.Label}
	}
	return result, nil
}

func floatEnumInput(values []*metadatalifecyclev1.FloatEnumOption) ([]domain.FloatEnumOption, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]domain.FloatEnumOption, len(values))
	for i, value := range values {
		if value == nil {
			return nil, apperror.New(apperror.CodeValidationFailed, fmt.Sprintf("float enum option %d is required", i))
		}
		result[i] = domain.FloatEnumOption{Value: value.Value, Label: value.Label}
	}
	return result, nil
}

func datasetInput(value *metadatalifecyclev1.DatasetInput) (domain.DatasetInput, error) {
	contactPoints, err := resourceParseUUIDs(value.GetContactPoints(), "dataset contact point", "contact_points")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	providers, err := resourceParseUUIDs(value.GetProviders(), "dataset provider", "providers")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "dataset category", "categories")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	accessRight, err := datasetAccessRightInput(value.AccessRight)
	if err != nil {
		return domain.DatasetInput{}, err
	}
	accrualPeriodicity, err := toDomainAccrualPeriodicity(value.AccrualPeriodicity)
	if err != nil {
		return domain.DatasetInput{}, err
	}
	datasetType, err := toDomainDatasetType(value.Type)
	if err != nil {
		return domain.DatasetInput{}, err
	}
	temporal, err := temporalInput(value.GetTemporalCoverage())
	if err != nil {
		return domain.DatasetInput{}, fmt.Errorf("map temporal coverage: %w", apperror.PrefixTarget(err, "temporal_coverage"))
	}
	geologicalTemporal := geologicalTemporalInput(value.GetGeologicalTemporalCoverage())
	spatial, err := geometryInput(value.SpatialCoverage, "spatial_coverage")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	distributions := make([]domain.DistributionInput, 0, len(value.GetDistributions()))
	for i, item := range value.GetDistributions() {
		mapped, err := distributionInput(item)
		if err != nil {
			return domain.DatasetInput{}, fmt.Errorf("map distribution: %w", apperror.PrefixTarget(err, fmt.Sprintf("distributions[%d]", i)))
		}
		distributions = append(distributions, mapped)
	}
	identifiers, err := identifierInputs(value.GetPersistentIdentifiers(), "persistent_identifiers")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	created, err := timestampInput(value.GetCreated())
	if err != nil {
		return domain.DatasetInput{}, fmt.Errorf("map created timestamp: %w", apperror.PrefixTarget(err, "created"))
	}
	updated, err := timestampInput(value.GetUpdated())
	if err != nil {
		return domain.DatasetInput{}, fmt.Errorf("map updated timestamp: %w", apperror.PrefixTarget(err, "updated"))
	}
	result := domain.DatasetInput{
		Title:                      value.GetTitle(),
		Description:                value.GetDescription(),
		Keywords:                   value.GetKeywords(),
		AccrualPeriodicity:         accrualPeriodicity,
		QualityAnnotation:          value.QualityAnnotation,
		ConformsTo:                 value.ConformsTo,
		Provenance:                 value.Provenance,
		Type:                       datasetType,
		Version:                    value.Version,
		SpatialResolutionInMeters:  value.SpatialResolutionInMeters,
		TemporalResolution:         value.TemporalResolution,
		AccessRight:                accessRight,
		Distributions:              distributions,
		SpatialCoverage:            spatial,
		TemporalCoverage:           temporal,
		GeologicalTemporalCoverage: geologicalTemporal,
		PersistentIdentifiers:      identifiers,
		ContactPointIDs:            contactPoints,
		ProviderIDs:                providers,
		CategoryIDs:                categories,
		SourceCreated:              created,
		SourceUpdated:              updated,
	}
	if err := result.Normalize(); err != nil {
		return domain.DatasetInput{}, fmt.Errorf("validate dataset input: %w", err)
	}
	return result, nil
}

//nolint:dupl // Create and draft protobufs intentionally map identical domain fields.
func distributionInput(value *metadatalifecyclev1.DistributionInput) (domain.DistributionInput, error) {
	if value == nil {
		return domain.DistributionInput{}, apperror.New(apperror.CodeValidationFailed, "distribution is required")
	}
	result := domain.DistributionInput{
		Title:       value.GetTitle(),
		Description: value.GetDescription(),
		License:     value.License,
		ByteSize:    value.ByteSize,
	}
	distributionType, err := toDomainDistributionType(value.GetType())
	if err != nil {
		return domain.DistributionInput{}, err
	}
	result.Type = distributionType
	if value.ResourceId != nil {
		id, err := parseUUIDAt(*value.ResourceId, "distribution resource ID", "resource_id")
		if err != nil {
			return domain.DistributionInput{}, err
		}
		result.ResourceID = &id
	}
	switch delivery := value.GetDelivery().(type) {
	case *metadatalifecyclev1.DistributionInput_WebService:
		mapped, err := toDomainWebServiceInput(delivery.WebService)
		if err != nil {
			return domain.DistributionInput{}, fmt.Errorf("map web service: %w", apperror.PrefixTarget(err, "web_service"))
		}
		result.WebService = mapped
	case *metadatalifecyclev1.DistributionInput_DownloadableFile:
		if delivery.DownloadableFile == nil {
			return domain.DistributionInput{}, apperror.New(apperror.CodeValidationFailed, "downloadable file is required")
		}
		result.DownloadableFile = &domain.DownloadableFile{URL: delivery.DownloadableFile.GetUrl(), Format: delivery.DownloadableFile.GetFormat()}
	default:
		return domain.DistributionInput{}, apperror.New(apperror.CodeValidationFailed, "distribution delivery method is required")
	}
	return result, nil
}

func webServiceMethodInput(value metadatalifecyclev1.WebServiceMethod) (domain.HTTPMethod, error) {
	switch value {
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET:
		return domain.HTTPMethodGet, nil
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_POST:
		return domain.HTTPMethodPost, nil
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_PUT:
		return domain.HTTPMethodPut, nil
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_DELETE:
		return domain.HTTPMethodDelete, nil
	default:
		return "", apperror.New(apperror.CodeValidationFailed, "web service method is invalid")
	}
}

func webServiceMethodOutput(value domain.HTTPMethod) metadatalifecyclev1.WebServiceMethod {
	switch value {
	case domain.HTTPMethodGet:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET
	case domain.HTTPMethodPost:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_POST
	case domain.HTTPMethodPut:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_PUT
	case domain.HTTPMethodDelete:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_DELETE
	default:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_UNSPECIFIED
	}
}

func webServiceTypeOutput(value domain.WebServiceType) metadatalifecyclev1.WebServiceType {
	switch value {
	case domain.WebServiceTypeWMS:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMS
	case domain.WebServiceTypeHTTP:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_HTTP
	case domain.WebServiceTypeWMTS:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMTS
	case domain.WebServiceTypeWFS:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WFS
	case domain.WebServiceTypeDynamic:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_DYNAMIC
	default:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_UNSPECIFIED
	}
}

func webServiceVisibleOnInput(values []metadatalifecyclev1.WebServiceVisibleOn) ([]domain.WebServiceVisibleOn, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]domain.WebServiceVisibleOn, len(values))
	for i, value := range values {
		switch value {
		case metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH:
			result[i] = domain.WebServiceVisibleOnGraph
		case metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_TABLE:
			result[i] = domain.WebServiceVisibleOnTable
		case metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP:
			result[i] = domain.WebServiceVisibleOnMap
		default:
			return nil, apperror.New(apperror.CodeValidationFailed, "web service visible on is invalid")
		}
	}
	return result, nil
}

func webServiceVisibleOnOutput(values []domain.WebServiceVisibleOn) []metadatalifecyclev1.WebServiceVisibleOn {
	if values == nil {
		return nil
	}
	result := make([]metadatalifecyclev1.WebServiceVisibleOn, len(values))
	for i, value := range values {
		switch value {
		case domain.WebServiceVisibleOnGraph:
			result[i] = metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH
		case domain.WebServiceVisibleOnTable:
			result[i] = metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_TABLE
		case domain.WebServiceVisibleOnMap:
			result[i] = metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP
		default:
			result[i] = metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_UNSPECIFIED
		}
	}
	return result
}

func webServiceParameterInput(value *metadatalifecyclev1.ParameterInput) (domain.WebServiceParameterInput, error) {
	if value == nil {
		return domain.WebServiceParameterInput{}, apperror.New(apperror.CodeValidationFailed, "web service parameter is required")
	}
	parameter, err := parameterInput(value)
	if err != nil {
		return domain.WebServiceParameterInput{}, err
	}
	location, err := webServiceParameterLocationInput(value.GetLocation())
	if err != nil {
		return domain.WebServiceParameterInput{}, err
	}
	role, err := webServiceParameterRoleInput(value.GetRole())
	if err != nil {
		return domain.WebServiceParameterInput{}, err
	}
	parameter.Role = role
	return domain.WebServiceParameterInput{ParameterInput: parameter, Location: location, Explode: value.GetExplode()}, nil
}

func webServiceParameterRoleInput(value metadatalifecyclev1.WebServiceParameterRole) (*domain.WebServiceParameterRole, error) {
	var role domain.WebServiceParameterRole
	switch value {
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_UNSPECIFIED:
		return nil, nil
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_FORMAT:
		role = domain.WebServiceParameterRoleFormat
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_START_DATE:
		role = domain.WebServiceParameterRoleStartDate
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_END_DATE:
		role = domain.WebServiceParameterRoleEndDate
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MIN_LON:
		role = domain.WebServiceParameterRoleMinLon
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MAX_LON:
		role = domain.WebServiceParameterRoleMaxLon
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MIN_LAT:
		role = domain.WebServiceParameterRoleMinLat
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MAX_LAT:
		role = domain.WebServiceParameterRoleMaxLat
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_LONGITUDE:
		role = domain.WebServiceParameterRoleLongitude
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_LATITUDE:
		role = domain.WebServiceParameterRoleLatitude
	default:
		return nil, apperror.New(apperror.CodeValidationFailed, "parameter role is invalid")
	}
	return &role, nil
}

func webServiceParameterRoleOutput(value *domain.WebServiceParameterRole) metadatalifecyclev1.WebServiceParameterRole {
	if value == nil {
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_UNSPECIFIED
	}
	switch *value {
	case domain.WebServiceParameterRoleFormat:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_FORMAT
	case domain.WebServiceParameterRoleStartDate:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_START_DATE
	case domain.WebServiceParameterRoleEndDate:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_END_DATE
	case domain.WebServiceParameterRoleMinLon:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MIN_LON
	case domain.WebServiceParameterRoleMaxLon:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MAX_LON
	case domain.WebServiceParameterRoleMinLat:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MIN_LAT
	case domain.WebServiceParameterRoleMaxLat:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MAX_LAT
	case domain.WebServiceParameterRoleLongitude:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_LONGITUDE
	case domain.WebServiceParameterRoleLatitude:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_LATITUDE
	default:
		return metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_UNSPECIFIED
	}
}

func webServiceParameterLocationInput(value metadatalifecyclev1.WebServiceParameterLocation) (domain.ParameterLocation, error) {
	switch value {
	case metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_PATH:
		return domain.ParameterLocationPath, nil
	case metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_QUERY:
		return domain.ParameterLocationQuery, nil
	case metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_HEADER:
		return domain.ParameterLocationHeader, nil
	default:
		return "", apperror.New(apperror.CodeValidationFailed, "parameter location is invalid")
	}
}

func webServiceParameterLocationOutput(value domain.ParameterLocation) metadatalifecyclev1.WebServiceParameterLocation {
	switch value {
	case domain.ParameterLocationPath:
		return metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_PATH
	case domain.ParameterLocationQuery:
		return metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_QUERY
	case domain.ParameterLocationHeader:
		return metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_HEADER
	default:
		return metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_UNSPECIFIED
	}
}

func parameterInput(value *metadatalifecyclev1.ParameterInput) (domain.ParameterInput, error) {
	if value == nil {
		return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "parameter is required")
	}
	parameter := domain.ParameterInput{
		Position:    int(value.GetPosition()),
		Name:        value.GetName(),
		Label:       value.GetLabel(),
		Description: value.Description,
		Required:    value.GetRequired(),
		ReadOnly:    value.GetReadOnly(),
	}
	if value.ResourceId != nil {
		id, err := parseUUIDAt(*value.ResourceId, "parameter resource ID", "resource_id")
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.ResourceID = &id
	}

	switch v := value.GetValue().(type) {
	case *metadatalifecyclev1.ParameterInput_String_:
		if v.String_ == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "string parameter is required")
		}
		enum, err := stringEnumInput(v.String_.GetEnum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.Schema = domain.StringSchema{Default: v.String_.Default, Enum: enum, Pattern: v.String_.Pattern, MultipleValues: v.String_.MultipleValues}
	case *metadatalifecyclev1.ParameterInput_Boolean:
		if v.Boolean == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "boolean parameter is required")
		}
		parameter.Schema = domain.BooleanSchema{Default: v.Boolean.Default}
	case *metadatalifecyclev1.ParameterInput_Integer:
		if v.Integer == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "integer parameter is required")
		}
		enum, err := integerEnumInput(v.Integer.GetEnum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.Schema = domain.IntegerSchema{Minimum: v.Integer.Minimum, Maximum: v.Integer.Maximum, Default: v.Integer.Default, Enum: enum, MultipleValues: v.Integer.MultipleValues}
	case *metadatalifecyclev1.ParameterInput_Float:
		if v.Float == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "float parameter is required")
		}
		enum, err := floatEnumInput(v.Float.GetEnum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.Schema = domain.FloatSchema{Minimum: v.Float.Minimum, Maximum: v.Float.Maximum, Default: v.Float.Default, Enum: enum, MultipleValues: v.Float.MultipleValues}
	case *metadatalifecyclev1.ParameterInput_DateTime:
		if v.DateTime == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "date-time parameter is required")
		}
		minimum, err := timestampInput(v.DateTime.GetMinimum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		maximum, err := timestampInput(v.DateTime.GetMaximum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		def, err := timestampInput(v.DateTime.GetDefault())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.Schema = domain.DateTimeSchema{Minimum: minimum, Maximum: maximum, Default: def, Layout: v.DateTime.Layout}
	case *metadatalifecyclev1.ParameterInput_Template:
		if v.Template == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "template parameter is required")
		}
		parameters := make([]domain.ParameterInput, 0, len(v.Template.GetParameters()))
		for _, item := range v.Template.GetParameters() {
			mapped, err := scalarParameterInput(item)
			if err != nil {
				return domain.ParameterInput{}, err
			}
			parameters = append(parameters, mapped)
		}
		parameter.Schema = domain.TemplateSchemaInput{Template: v.Template.GetTemplate(), Parameters: parameters}
	default:
		return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "parameter value is required")
	}
	return parameter, nil
}

func scalarParameterInput(value *metadatalifecyclev1.ScalarParameterInput) (domain.ParameterInput, error) {
	if value == nil {
		return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "scalar parameter is required")
	}
	role, err := webServiceParameterRoleInput(value.GetRole())
	if err != nil {
		return domain.ParameterInput{}, err
	}
	parameter := domain.ParameterInput{
		Position:    int(value.GetPosition()),
		Name:        value.GetName(),
		Label:       value.GetLabel(),
		Description: value.Description,
		Role:        role,
		Required:    value.GetRequired(),
		ReadOnly:    value.GetReadOnly(),
	}
	if value.ResourceId != nil {
		id, err := parseUUIDAt(*value.ResourceId, "parameter resource ID", "resource_id")
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.ResourceID = &id
	}

	switch v := value.GetValue().(type) {
	case *metadatalifecyclev1.ScalarParameterInput_String_:
		if v.String_ == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "string parameter is required")
		}
		enum, err := stringEnumInput(v.String_.GetEnum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.Schema = domain.StringSchema{Default: v.String_.Default, Enum: enum, Pattern: v.String_.Pattern}
	case *metadatalifecyclev1.ScalarParameterInput_Boolean:
		if v.Boolean == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "boolean parameter is required")
		}
		parameter.Schema = domain.BooleanSchema{Default: v.Boolean.Default}
	case *metadatalifecyclev1.ScalarParameterInput_Integer:
		if v.Integer == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "integer parameter is required")
		}
		enum, err := integerEnumInput(v.Integer.GetEnum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.Schema = domain.IntegerSchema{Minimum: v.Integer.Minimum, Maximum: v.Integer.Maximum, Default: v.Integer.Default, Enum: enum}
	case *metadatalifecyclev1.ScalarParameterInput_Float:
		if v.Float == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "float parameter is required")
		}
		enum, err := floatEnumInput(v.Float.GetEnum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.Schema = domain.FloatSchema{Minimum: v.Float.Minimum, Maximum: v.Float.Maximum, Default: v.Float.Default, Enum: enum}
	case *metadatalifecyclev1.ScalarParameterInput_DateTime:
		if v.DateTime == nil {
			return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "date-time parameter is required")
		}
		minimum, err := timestampInput(v.DateTime.GetMinimum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		maximum, err := timestampInput(v.DateTime.GetMaximum())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		def, err := timestampInput(v.DateTime.GetDefault())
		if err != nil {
			return domain.ParameterInput{}, err
		}
		parameter.Schema = domain.DateTimeSchema{Minimum: minimum, Maximum: maximum, Default: def, Layout: v.DateTime.Layout}
	default:
		return domain.ParameterInput{}, apperror.New(apperror.CodeValidationFailed, "scalar parameter value is required")
	}
	return parameter, nil
}

func datasetAccessRightInput(value *metadatalifecyclev1.DatasetAccessRight) (*domain.AccessRight, error) {
	if value == nil {
		return nil, nil
	}
	var result domain.AccessRight
	switch *value {
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_NON_PUBLIC:
		result = domain.AccessRightNonPublic
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_NORMAL:
		result = domain.AccessRightNormal
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_OP_DATPRO:
		result = domain.AccessRightOpDatpro
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_PUBLIC:
		result = domain.AccessRightPublic
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_RESTRICTED:
		result = domain.AccessRightRestricted
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_CONFIDENTIAL:
		result = domain.AccessRightConfidential
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_SENSITIVE:
		result = domain.AccessRightSensitive
	default:
		return nil, apperror.Field("access_right", "Access right is not supported.", nil)
	}
	return &result, nil
}

func datasetAccessRightOutput(value *domain.AccessRight) *metadatalifecyclev1.DatasetAccessRight {
	if value == nil {
		return nil
	}
	var result metadatalifecyclev1.DatasetAccessRight
	switch *value {
	case domain.AccessRightNonPublic:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_NON_PUBLIC
	case domain.AccessRightNormal:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_NORMAL
	case domain.AccessRightOpDatpro:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_OP_DATPRO
	case domain.AccessRightPublic:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_PUBLIC
	case domain.AccessRightRestricted:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_RESTRICTED
	case domain.AccessRightConfidential:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_CONFIDENTIAL
	case domain.AccessRightSensitive:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_SENSITIVE
	default:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_UNSPECIFIED
	}
	return &result
}

//nolint:dupl // Create and draft protobufs intentionally map identical domain fields.
func softwareApplicationInput(value *metadatalifecyclev1.SoftwareApplicationInput) (domain.SoftwareApplicationInput, error) {
	creators, err := resourceParseUUIDs(value.GetCreators(), "software application creator", "creators")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "software application category", "categories")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	contacts, err := resourceParseUUIDs(value.GetContactPoints(), "software application contact point", "contact_points")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	identifiers, err := identifierInputs(value.GetIdentifiers(), "identifiers")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	result := domain.SoftwareApplicationInput{
		Name:             value.GetName(),
		Description:      value.GetDescription(),
		MainEntityOfPage: value.MainEntityOfPage,
		License:          value.License,
		Version:          value.Version,
		Keywords:         value.GetKeywords(),
		DownloadURL:      value.DownloadUrl,
		Requirements:     value.Requirements,
		OperatingSystem:  value.OperatingSystem,
		CreatorIDs:       creators,
		ContactPointIDs:  contacts,
		CategoryIDs:      categories,
		Identifiers:      identifiers,
	}
	if err := result.Normalize(); err != nil {
		return domain.SoftwareApplicationInput{}, fmt.Errorf("validate software application input: %w", err)
	}
	return result, nil
}

//nolint:dupl // Create and draft protobufs intentionally map identical domain fields.
func softwareSourceCodeInput(value *metadatalifecyclev1.SoftwareSourceCodeInput) (domain.SoftwareSourceCodeInput, error) {
	creators, err := resourceParseUUIDs(value.GetCreators(), "software source code creator", "creators")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "software source code category", "categories")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	contacts, err := resourceParseUUIDs(value.GetContactPoints(), "software source code contact point", "contact_points")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	identifiers, err := identifierInputs(value.GetIdentifiers(), "identifiers")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	result := domain.SoftwareSourceCodeInput{
		Name:                 value.GetName(),
		Description:          value.GetDescription(),
		ProgrammingLanguage:  value.ProgrammingLanguage,
		Keywords:             value.GetKeywords(),
		Repository:           value.Repository,
		License:              value.License,
		SoftwareRequirements: value.SoftwareRequirements,
		RuntimePlatform:      value.RuntimePlatform,
		MainEntityOfPage:     value.MainEntityOfPage,
		Version:              value.Version,
		CreatorIDs:           creators,
		ContactPointIDs:      contacts,
		CategoryIDs:          categories,
		Identifiers:          identifiers,
	}
	if err := result.Normalize(); err != nil {
		return domain.SoftwareSourceCodeInput{}, fmt.Errorf("validate software source code input: %w", err)
	}
	return result, nil
}

func identifierInputs(values []*metadatalifecyclev1.VersionedIdentifierInput, target string) ([]domain.IdentifierInput, error) {
	result := make([]domain.IdentifierInput, 0, len(values))
	for i, value := range values {
		if value == nil {
			return nil, apperror.Field(fmt.Sprintf("%s[%d]", target, i), "Identifier is required.", fmt.Errorf("missing identifier at index %d", i))
		}
		var resourceID *uuid.UUID
		if value.ResourceId != nil {
			id, err := parseUUIDAt(*value.ResourceId, "identifier resource ID", fmt.Sprintf("%s[%d].resource_id", target, i))
			if err != nil {
				return nil, err
			}
			resourceID = &id
		}
		result = append(result, domain.IdentifierInput{
			ResourceID: resourceID,
			Type:       domain.IdentifierType(value.GetType()),
			Value:      value.GetValue(),
		})
	}
	return result, nil
}

func parseUUIDAt(value, field, target string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		if err == nil {
			err = fmt.Errorf("%s must not be the zero UUID", field)
		}
		return uuid.Nil, apperror.Field(target, "Must be a valid UUID.", fmt.Errorf("invalid %s: %w", field, err))
	}
	return id, nil
}

func resourceParseUUIDs(values []string, field, target string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for i, value := range values {
		id, err := parseUUIDAt(value, field, fmt.Sprintf("%s[%d]", target, i))
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

func timestampInput(value *timestamppb.Timestamp) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	if err := value.CheckValid(); err != nil {
		return nil, apperror.Field("", "Timestamp must be valid.", fmt.Errorf("invalid timestamp: %w", err))
	}
	result := value.AsTime()
	return &result, nil
}

func temporalInput(value *metadatalifecyclev1.Temporal) (*domain.Temporal, error) {
	if value == nil {
		return nil, nil
	}
	start, err := timestampInput(value.GetStart())
	if err != nil {
		return nil, fmt.Errorf("map temporal start: %w", apperror.PrefixTarget(err, "start"))
	}
	end, err := timestampInput(value.GetEnd())
	if err != nil {
		return nil, fmt.Errorf("map temporal end: %w", apperror.PrefixTarget(err, "end"))
	}
	return &domain.Temporal{Start: start, End: end}, nil
}

func geologicalTemporalInput(value *metadatalifecyclev1.GeologicalTemporalCoverage) *domain.GeologicalTemporal {
	if value == nil {
		return nil
	}
	return &domain.GeologicalTemporal{StartMYA: value.StartMya, EndMYA: value.EndMya}
}

func toGRPCVersioningStatus(status domain.Status) metadatalifecyclev1.VersioningStatus {
	switch status {
	case domain.Published:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_PUBLISHED
	case domain.Draft:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_DRAFT
	case domain.Archived:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_ARCHIVED
	case domain.Rejected:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_REJECTED
	case domain.Submitted:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_SUBMITTED
	default:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_UNSPECIFIED
	}
}

func toGRPCResourceType(resourceType domain.ResourceType) metadatalifecyclev1.ResourceType {
	switch resourceType {
	case domain.Dataset:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET
	case domain.SoftwareApplication:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION
	case domain.SoftwareSourceCode:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE
	default:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_UNSPECIFIED
	}
}

func resourceVersionSummaryOutput(value domain.ResourceVersionSummary) *metadatalifecyclev1.ResourceVersionSummary {
	return &metadatalifecyclev1.ResourceVersionSummary{
		ResourceType:  toGRPCResourceType(value.ResourceType),
		ResourceId:    value.ResourceID.String(),
		Id:            value.VersionID.String(),
		Revision:      value.Revision,
		Status:        toGRPCVersioningStatus(value.Status),
		Name:          value.Name,
		CreatedBy:     value.CreatedBy,
		CreatedAt:     timestamppb.New(value.CreatedAt),
		UpdatedAt:     timestamppb.New(value.UpdatedAt),
		SubmittedBy:   value.SubmittedBy,
		SubmittedAt:   timestampOutput(value.SubmittedAt),
		PublishedBy:   value.PublishedBy,
		PublishedAt:   timestampOutput(value.PublishedAt),
		ArchivedBy:    value.ArchivedBy,
		ArchivedAt:    timestampOutput(value.ArchivedAt),
		RejectedBy:    value.RejectedBy,
		RejectedAt:    timestampOutput(value.RejectedAt),
		RejectionNote: value.RejectionNote,
	}
}

func geometryInput(value *string, target string) (geom.T, error) {
	if value == nil {
		return nil, nil
	}
	geometry, err := wkt.Unmarshal(strings.TrimSpace(*value))
	if err != nil {
		return nil, apperror.Field(target, "Value must be valid WKT geometry.", fmt.Errorf("parse geometry: %w", err))
	}
	geometry, err = geom.SetSRID(geometry, 4326)
	if err != nil {
		return nil, apperror.Field(target, "Geometry must use EPSG:4326 coordinates.", fmt.Errorf("set geometry SRID: %w", err))
	}
	return geometry, nil
}

func createResponse(resourceID, versionID uuid.UUID, revision int32, resourceStatus domain.Status) *metadatalifecyclev1.CreateResourceResponse {
	return &metadatalifecyclev1.CreateResourceResponse{
		Resource: &metadatalifecyclev1.ResourceRecord{
			ResourceId: resourceID.String(),
			VersionId:  versionID.String(),
			Revision:   revision,
			Status:     toGRPCVersioningStatus(resourceStatus),
		},
	}
}
