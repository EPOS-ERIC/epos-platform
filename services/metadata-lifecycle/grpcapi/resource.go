package grpcapi

// TODO: format this file, possibly split it in multiple files and most likely simplify the logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/datasets"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwareapplications"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwaresourcecodes"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
	"google.golang.org/grpc/codes"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
		return nil, status.Error(codes.InvalidArgument, "resource request is required") //nolint:wrapcheck
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	groupID, err := parseUUID(request.GetGroupId(), "group id")
	if err != nil {
		return nil, err
	}

	switch value := request.GetMetadata().(type) {
	case *metadatalifecyclev1.CreateResourceRequest_Dataset:
		if value.Dataset == nil {
			return nil, status.Error(codes.InvalidArgument, "dataset metadata is required")
		}
		input, err := datasetInput(value.Dataset)
		if err != nil {
			return nil, err
		}
		created, err := s.datasets.CreateDraft(ctx, actor, groupID, input)
		if err != nil {
			return nil, resourceError(err)
		}
		return createResponse(created.DatasetID, created.VersionID, created.Revision, created.Status), nil
	case *metadatalifecyclev1.CreateResourceRequest_SoftwareApplication:
		if value.SoftwareApplication == nil {
			return nil, status.Error(codes.InvalidArgument, "software application metadata is required")
		}
		input, err := softwareApplicationInput(value.SoftwareApplication)
		if err != nil {
			return nil, err
		}
		created, err := s.applications.CreateDraft(ctx, actor, groupID, input)
		if err != nil {
			return nil, resourceError(err)
		}
		return createResponse(created.SoftwareApplicationID, created.VersionID, created.Revision, created.Status), nil
	case *metadatalifecyclev1.CreateResourceRequest_SoftwareSourceCode:
		if value.SoftwareSourceCode == nil {
			return nil, status.Error(codes.InvalidArgument, "software source code metadata is required")
		}
		input, err := softwareSourceCodeInput(value.SoftwareSourceCode)
		if err != nil {
			return nil, err
		}
		created, err := s.sourceCodes.CreateDraft(ctx, actor, groupID, input)
		if err != nil {
			return nil, resourceError(err)
		}
		return createResponse(created.SoftwareSourceCodeID, created.VersionID, created.Revision, created.Status), nil
	default:
		return nil, status.Error(codes.InvalidArgument, "resource metadata is required")
	}
}

func (s *ResourceServer) GetResourceVersion(ctx context.Context, request *metadatalifecyclev1.GetResourceVersionRequest) (*metadatalifecyclev1.GetResourceVersionResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "resource version request is required")
	}
	resourceID, err := parseUUID(request.GetResourceId(), "resource id")
	if err != nil {
		return nil, err
	}
	versionID, err := parseUUID(request.GetVersionId(), "version id")
	if err != nil {
		return nil, err
	}
	actor, err := optionalActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var value any
	var includeContacts bool
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		value, includeContacts, err = s.datasets.Read(ctx, actor, resourceID, versionID)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		value, includeContacts, err = s.applications.Read(ctx, actor, resourceID, versionID)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		value, includeContacts, err = s.sourceCodes.Read(ctx, actor, resourceID, versionID)
	default:
		return nil, status.Error(codes.InvalidArgument, "resource type is required") //nolint:wrapcheck
	}
	if err != nil {
		return nil, resourceReadError(err)
	}

	response := &metadatalifecyclev1.ResourceVersion{}
	switch item := value.(type) {
	case domain.DatasetVersion:
		response.ResourceId, response.VersionId = item.DatasetID.String(), item.VersionID.String()
		response.Revision, response.Status = item.Revision, string(item.Status)
		response.CreatedBy, response.GroupId, response.GroupName = string(item.CreatedBy), item.Group.ID.String(), item.Group.Name
		response.CreatedAt, response.UpdatedAt = timestamppb.New(item.CreatedAt), timestamppb.New(item.UpdatedAt)
		response.Metadata = &metadatalifecyclev1.ResourceVersion_Dataset{Dataset: datasetResourceMetadata(item.Metadata, includeContacts)}
	case domain.SoftwareApplicationVersion:
		response.ResourceId, response.VersionId = item.SoftwareApplicationID.String(), item.VersionID.String()
		response.Revision, response.Status = item.Revision, string(item.Status)
		response.CreatedBy, response.GroupId, response.GroupName = string(item.CreatedBy), item.Group.ID.String(), item.Group.Name
		response.CreatedAt, response.UpdatedAt = timestamppb.New(item.CreatedAt), timestamppb.New(item.UpdatedAt)
		response.Metadata = &metadatalifecyclev1.ResourceVersion_SoftwareApplication{SoftwareApplication: applicationResourceMetadata(item.Metadata, includeContacts)}
	case domain.SoftwareSourceCodeVersion:
		response.ResourceId, response.VersionId = item.SoftwareSourceCodeID.String(), item.VersionID.String()
		response.Revision, response.Status = item.Revision, string(item.Status)
		response.CreatedBy, response.GroupId, response.GroupName = string(item.CreatedBy), item.Group.ID.String(), item.Group.Name
		response.CreatedAt, response.UpdatedAt = timestamppb.New(item.CreatedAt), timestamppb.New(item.UpdatedAt)
		response.Metadata = &metadatalifecyclev1.ResourceVersion_SoftwareSourceCode{SoftwareSourceCode: sourceCodeResourceMetadata(item.Metadata, includeContacts)}
	default:
		return nil, status.Error(codes.Internal, "unsupported resource type") //nolint:wrapcheck
	}
	return &metadatalifecyclev1.GetResourceVersionResponse{Resource: response}, nil
}

func (s *ResourceServer) ReplaceResourceDraft(ctx context.Context, request *metadatalifecyclev1.ReplaceResourceDraftRequest) (*metadatalifecyclev1.ReplaceResourceDraftResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "resource draft request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := parseUUID(request.GetResourceId(), "resource id")
	if err != nil {
		return nil, err
	}
	versionID, err := parseUUID(request.GetVersionId(), "version id")
	if err != nil {
		return nil, err
	}

	switch value := request.GetMetadata().(type) {
	case *metadatalifecyclev1.ReplaceResourceDraftRequest_Dataset:
		if value.Dataset == nil {
			return nil, status.Error(codes.InvalidArgument, "dataset metadata is required")
		}
		metadata, err := datasetDraftInput(value.Dataset)
		if err != nil {
			return nil, err
		}
		input, err := domain.NewDatasetDraftInput(domain.DatasetDraftInput{Revision: request.GetRevision(), Metadata: metadata})
		if err != nil {
			return nil, resourceError(err)
		}
		result, err := s.datasets.ReplaceDraft(ctx, actor, resourceID, versionID, input)
		if err != nil {
			return nil, resourceError(err)
		}
		return &metadatalifecyclev1.ReplaceResourceDraftResponse{Resource: &metadatalifecyclev1.ResourceRecord{ResourceId: result.DatasetID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: string(result.Status)}}, nil
	case *metadatalifecyclev1.ReplaceResourceDraftRequest_SoftwareApplication:
		if value.SoftwareApplication == nil {
			return nil, status.Error(codes.InvalidArgument, "software application metadata is required")
		}
		metadata, err := softwareApplicationDraftInput(value.SoftwareApplication)
		if err != nil {
			return nil, err
		}
		input, err := domain.NewSoftwareApplicationDraftInput(domain.SoftwareApplicationDraftInput{Revision: request.GetRevision(), Metadata: metadata})
		if err != nil {
			return nil, resourceError(err)
		}
		result, err := s.applications.ReplaceDraft(ctx, actor, resourceID, versionID, input)
		if err != nil {
			return nil, resourceError(err)
		}
		return &metadatalifecyclev1.ReplaceResourceDraftResponse{Resource: &metadatalifecyclev1.ResourceRecord{ResourceId: result.SoftwareApplicationID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: string(result.Status)}}, nil
	case *metadatalifecyclev1.ReplaceResourceDraftRequest_SoftwareSourceCode:
		if value.SoftwareSourceCode == nil {
			return nil, status.Error(codes.InvalidArgument, "software source code metadata is required")
		}
		metadata, err := softwareSourceCodeDraftInput(value.SoftwareSourceCode)
		if err != nil {
			return nil, err
		}
		input, err := domain.NewSoftwareSourceCodeDraftInput(domain.SoftwareSourceCodeDraftInput{Revision: request.GetRevision(), Metadata: metadata})
		if err != nil {
			return nil, resourceError(err)
		}
		result, err := s.sourceCodes.ReplaceDraft(ctx, actor, resourceID, versionID, input)
		if err != nil {
			return nil, resourceError(err)
		}
		return &metadatalifecyclev1.ReplaceResourceDraftResponse{Resource: &metadatalifecyclev1.ResourceRecord{ResourceId: result.SoftwareSourceCodeID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: string(result.Status)}}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "resource metadata is required")
	}
}

func (s *ResourceServer) DiscardResourceDraft(ctx context.Context, request *metadatalifecyclev1.DiscardResourceDraftRequest) (*metadatalifecyclev1.DiscardResourceDraftResponse, error) {
	actor, resourceID, versionID, _, err := s.lifecycleRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		err = s.datasets.Discard(ctx, actor, resourceID, versionID)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		err = s.applications.Discard(ctx, actor, resourceID, versionID)
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		err = s.sourceCodes.Discard(ctx, actor, resourceID, versionID)
	default:
		return nil, status.Error(codes.InvalidArgument, "resource type is required") //nolint:wrapcheck
	}
	if err != nil {
		return nil, resourceError(err)
	}
	return &metadatalifecyclev1.DiscardResourceDraftResponse{}, nil
}

func (s *ResourceServer) SubmitResource(ctx context.Context, request *metadatalifecyclev1.SubmitResourceRequest) (*metadatalifecyclev1.SubmitResourceResponse, error) {
	record, err := s.lifecycleTransition(ctx, request, func(actor domain.Actor, id, version uuid.UUID) (any, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Submit(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Submit(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Submit(ctx, actor, id, version)
		default:
			return nil, status.Error(codes.InvalidArgument, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.SubmitResourceResponse{Resource: record}, nil
}

func (s *ResourceServer) WithdrawResource(ctx context.Context, request *metadatalifecyclev1.WithdrawResourceRequest) (*metadatalifecyclev1.WithdrawResourceResponse, error) {
	record, err := s.lifecycleTransition(ctx, request, func(actor domain.Actor, id, version uuid.UUID) (any, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Withdraw(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Withdraw(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Withdraw(ctx, actor, id, version)
		default:
			return nil, status.Error(codes.InvalidArgument, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.WithdrawResourceResponse{Resource: record}, nil
}

func (s *ResourceServer) RejectResource(ctx context.Context, request *metadatalifecyclev1.RejectResourceRequest) (*metadatalifecyclev1.RejectResourceResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "resource request is required") //nolint:wrapcheck
	}
	actor, resourceID, versionID, _, err := s.lifecycleRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		value, e := domain.NewDatasetRejectionInput(domain.DatasetRejectionInput{Note: request.GetNote()})
		if e != nil {
			return nil, resourceError(e)
		}
		result, e := s.datasets.Reject(ctx, actor, resourceID, versionID, value)
		if e != nil {
			return nil, resourceError(e)
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.RejectResourceResponse{Resource: record}, e
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		value, e := domain.NewSoftwareApplicationRejectionInput(domain.SoftwareApplicationRejectionInput{Note: request.GetNote()})
		if e != nil {
			return nil, resourceError(e)
		}
		result, e := s.applications.Reject(ctx, actor, resourceID, versionID, value)
		if e != nil {
			return nil, resourceError(e)
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.RejectResourceResponse{Resource: record}, e
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		value, e := domain.NewSoftwareSourceCodeRejectionInput(domain.SoftwareSourceCodeRejectionInput{Note: request.GetNote()})
		if e != nil {
			return nil, resourceError(e)
		}
		result, e := s.sourceCodes.Reject(ctx, actor, resourceID, versionID, value)
		if e != nil {
			return nil, resourceError(e)
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.RejectResourceResponse{Resource: record}, e
	default:
		return nil, status.Error(codes.InvalidArgument, "resource type is required") //nolint:wrapcheck
	}
}

func (s *ResourceServer) ReopenResource(ctx context.Context, request *metadatalifecyclev1.ReopenResourceRequest) (*metadatalifecyclev1.ReopenResourceResponse, error) {
	record, err := s.lifecycleTransition(ctx, request, func(actor domain.Actor, id, version uuid.UUID) (any, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Reopen(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Reopen(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Reopen(ctx, actor, id, version)
		default:
			return nil, status.Error(codes.InvalidArgument, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.ReopenResourceResponse{Resource: record}, nil
}

func (s *ResourceServer) PublishResource(ctx context.Context, request *metadatalifecyclev1.PublishResourceRequest) (*metadatalifecyclev1.PublishResourceResponse, error) {
	record, err := s.lifecycleTransition(ctx, request, func(actor domain.Actor, id, version uuid.UUID) (any, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Publish(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Publish(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Publish(ctx, actor, id, version)
		default:
			return nil, status.Error(codes.InvalidArgument, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.PublishResourceResponse{Resource: record}, nil
}

func (s *ResourceServer) UnpublishResource(ctx context.Context, request *metadatalifecyclev1.UnpublishResourceRequest) (*metadatalifecyclev1.UnpublishResourceResponse, error) {
	record, err := s.lifecycleTransition(ctx, request, func(actor domain.Actor, id, version uuid.UUID) (any, error) {
		switch request.GetResourceType() {
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
			return s.datasets.Unpublish(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
			return s.applications.Unpublish(ctx, actor, id, version)
		case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
			return s.sourceCodes.Unpublish(ctx, actor, id, version)
		default:
			return nil, status.Error(codes.InvalidArgument, "resource type is required")
		}
	})
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.UnpublishResourceResponse{Resource: record}, nil
}

func (s *ResourceServer) DraftResourceFromPublication(ctx context.Context, request *metadatalifecyclev1.DraftResourceFromPublicationRequest) (*metadatalifecyclev1.DraftResourceFromPublicationResponse, error) {
	record, err := s.draftResource(ctx, request, "publication")
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DraftResourceFromPublicationResponse{Resource: record}, nil
}

func (s *ResourceServer) DraftResourceFromLatestArchive(ctx context.Context, request *metadatalifecyclev1.DraftResourceFromLatestArchiveRequest) (*metadatalifecyclev1.DraftResourceFromLatestArchiveResponse, error) {
	record, err := s.draftResource(ctx, request, "archive")
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DraftResourceFromLatestArchiveResponse{Resource: record}, nil
}

func (s *ResourceServer) CopyPublishedResource(ctx context.Context, request *metadatalifecyclev1.CopyPublishedResourceRequest) (*metadatalifecyclev1.CopyPublishedResourceResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "resource request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	sourceID, err := parseUUID(request.GetResourceId(), "resource id")
	if err != nil {
		return nil, err
	}
	groupID, err := parseUUID(request.GetGroupId(), "group id")
	if err != nil {
		return nil, err
	}
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		result, e := s.datasets.CopyPublished(ctx, actor, sourceID, groupID)
		if e != nil {
			return nil, resourceError(e)
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.CopyPublishedResourceResponse{Resource: record}, e
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		result, e := s.applications.CopyPublished(ctx, actor, sourceID, groupID)
		if e != nil {
			return nil, resourceError(e)
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.CopyPublishedResourceResponse{Resource: record}, e
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		result, e := s.sourceCodes.CopyPublished(ctx, actor, sourceID, groupID)
		if e != nil {
			return nil, resourceError(e)
		}
		record, e := lifecycleResponse(result)
		return &metadatalifecyclev1.CopyPublishedResourceResponse{Resource: record}, e
	default:
		return nil, status.Error(codes.InvalidArgument, "resource type is required")
	}
}

type resourceVersionRequest interface {
	GetResourceType() metadatalifecyclev1.ResourceType
	GetResourceId() string
	GetVersionId() string
}

func (s *ResourceServer) lifecycleRequest(ctx context.Context, request resourceVersionRequest) (domain.Actor, uuid.UUID, uuid.UUID, any, error) {
	if request == nil {
		return domain.Actor(""), uuid.Nil, uuid.Nil, nil, status.Error(codes.InvalidArgument, "resource request is required")
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return domain.Actor(""), uuid.Nil, uuid.Nil, nil, err
	}
	id, err := parseUUID(request.GetResourceId(), "resource id")
	if err != nil {
		return domain.Actor(""), uuid.Nil, uuid.Nil, nil, err
	}
	version, err := parseUUID(request.GetVersionId(), "version id")
	if err != nil {
		return domain.Actor(""), uuid.Nil, uuid.Nil, nil, err
	}
	return actor, id, version, nil, nil
}

func (s *ResourceServer) lifecycleTransition(ctx context.Context, request resourceVersionRequest, call func(domain.Actor, uuid.UUID, uuid.UUID) (any, error)) (*metadatalifecyclev1.ResourceRecord, error) {
	actor, id, version, _, err := s.lifecycleRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	result, err := call(actor, id, version)
	if err != nil {
		return nil, resourceError(err)
	}
	return lifecycleResponse(result)
}

type draftResourceRequest interface {
	GetResourceType() metadatalifecyclev1.ResourceType
	GetResourceId() string
}

func (s *ResourceServer) draftResource(ctx context.Context, request draftResourceRequest, source string) (*metadatalifecyclev1.ResourceRecord, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "resource request is required") //nolint:wrapcheck
	}
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseUUID(request.GetResourceId(), "resource id")
	if err != nil {
		return nil, err
	}
	var result any
	switch request.GetResourceType() {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		if source == "publication" {
			result, err = s.datasets.DraftFromPublication(ctx, actor, id)
		} else {
			result, err = s.datasets.DraftFromLatestArchive(ctx, actor, id)
		}
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		if source == "publication" {
			result, err = s.applications.DraftFromPublication(ctx, actor, id)
		} else {
			result, err = s.applications.DraftFromLatestArchive(ctx, actor, id)
		}
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		if source == "publication" {
			result, err = s.sourceCodes.DraftFromPublication(ctx, actor, id)
		} else {
			result, err = s.sourceCodes.DraftFromLatestArchive(ctx, actor, id)
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "resource type is required") //nolint:wrapcheck
	}
	if err != nil {
		return nil, resourceError(err)
	}
	return lifecycleResponse(result)
}

func lifecycleResponse(value any) (*metadatalifecyclev1.ResourceRecord, error) {
	switch result := value.(type) {
	case domain.DatasetVersion:
		return &metadatalifecyclev1.ResourceRecord{ResourceId: result.DatasetID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: string(result.Status)}, nil
	case domain.SoftwareApplicationVersion:
		return &metadatalifecyclev1.ResourceRecord{ResourceId: result.SoftwareApplicationID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: string(result.Status)}, nil
	case domain.SoftwareSourceCodeVersion:
		return &metadatalifecyclev1.ResourceRecord{ResourceId: result.SoftwareSourceCodeID.String(), VersionId: result.VersionID.String(), Revision: result.Revision, Status: string(result.Status)}, nil
	default:
		return nil, status.Error(codes.Internal, "unsupported resource type") //nolint:wrapcheck
	}
}

func optionalActorFromContext(ctx context.Context) (*domain.Actor, error) {
	md, ok := grpcmetadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil
	}
	values := md.Get(actorMetadataKey)
	if len(values) == 0 {
		return nil, nil
	}
	if len(values) != 1 {
		return nil, status.Error(codes.InvalidArgument, "invalid actor") //nolint:wrapcheck
	}
	actor, err := domain.NewActor(values[0])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid actor") //nolint:wrapcheck
	}
	return &actor, nil
}

func datasetResourceMetadata(value domain.DatasetMetadata, includeContacts bool) *metadatalifecyclev1.DatasetResourceMetadata {
	result := &metadatalifecyclev1.DatasetResourceMetadata{Id: value.ID.String(), Title: value.Title, Description: value.Description, Keywords: value.Keywords, AccrualPeriodicity: value.AccrualPeriodicity, QualityAnnotation: value.QualityAnnotation, AccessRight: value.AccessRight, SpatialCoverage: geometryString(value.SpatialCoverage), TemporalCoverage: temporalOutput(value.TemporalCoverage), Created: timestampOutput(value.SourceCreated), Updated: timestampOutput(value.SourceUpdated), ContactsIncluded: includeContacts}
	for _, item := range value.PersistentIdentifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, identifierOutput(item))
	}
	for _, item := range value.Providers {
		result.Providers = append(result.Providers, referenceOutput(item.ID, item.Name))
	}
	for _, item := range value.Categories {
		result.Categories = append(result.Categories, referenceOutput(item.ID, item.Name))
	}
	if includeContacts {
		for _, item := range value.ContactPoints {
			result.ContactPoints = append(result.ContactPoints, item.ID.String())
		}
	}
	for _, item := range value.Distributions {
		result.Distributions = append(result.Distributions, distributionOutput(item, includeContacts))
	}
	return result
}

func applicationResourceMetadata(value domain.SoftwareApplicationMetadata, includeContacts bool) *metadatalifecyclev1.SoftwareApplicationResourceMetadata {
	result := &metadatalifecyclev1.SoftwareApplicationResourceMetadata{Id: value.ID.String(), Name: value.Name, Description: value.Description, MainEntityOfPage: value.MainEntityOfPage, License: value.License, Version: value.Version, Keywords: value.Keywords, DownloadUrl: value.DownloadURL, Requirements: value.Requirements, OperatingSystem: value.OperatingSystem, ContactsIncluded: includeContacts}
	for _, item := range value.Creators {
		result.Creators = append(result.Creators, referenceOutput(item.ID, item.Name))
	}
	for _, item := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierOutput(item))
	}
	for _, item := range value.Categories {
		result.Categories = append(result.Categories, referenceOutput(item.ID, item.Name))
	}
	if includeContacts {
		for _, item := range value.ContactPoints {
			result.ContactPoints = append(result.ContactPoints, item.ID.String())
		}
	}
	return result
}

func sourceCodeResourceMetadata(value domain.SoftwareSourceCodeMetadata, includeContacts bool) *metadatalifecyclev1.SoftwareSourceCodeResourceMetadata {
	result := &metadatalifecyclev1.SoftwareSourceCodeResourceMetadata{Id: value.ID.String(), Name: value.Name, Description: value.Description, ProgrammingLanguage: value.ProgrammingLanguage, Keywords: value.Keywords, Repository: value.Repository, License: value.License, SoftwareRequirements: value.SoftwareRequirements, RuntimePlatform: value.RuntimePlatform, MainEntityOfPage: value.MainEntityOfPage, Version: value.Version, ContactsIncluded: includeContacts}
	for _, item := range value.Creators {
		result.Creators = append(result.Creators, referenceOutput(item.ID, item.Name))
	}
	for _, item := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierOutput(item))
	}
	for _, item := range value.Categories {
		result.Categories = append(result.Categories, referenceOutput(item.ID, item.Name))
	}
	if includeContacts {
		for _, item := range value.ContactPoints {
			result.ContactPoints = append(result.ContactPoints, item.ID.String())
		}
	}
	return result
}

func referenceOutput(id uuid.UUID, name string) *metadatalifecyclev1.ResourceReference {
	return &metadatalifecyclev1.ResourceReference{Id: id.String(), Name: name}
}

func identifierOutput(value domain.Identifier) *metadatalifecyclev1.ResourceIdentifier {
	return &metadatalifecyclev1.ResourceIdentifier{Id: value.ID.String(), Type: string(value.Type), Value: value.Value}
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

func distributionOutput(value domain.Distribution, includeContacts bool) *metadatalifecyclev1.Distribution {
	result := &metadatalifecyclev1.Distribution{Id: value.ID.String(), Title: value.Title, Description: value.Description, License: value.License, Type: string(value.Type), Issued: timestampOutput(value.Issued), Modified: timestampOutput(value.Modified), ByteSize: value.ByteSize}
	if value.DownloadableFile != nil {
		result.Delivery = &metadatalifecyclev1.Distribution_DownloadableFile{DownloadableFile: &metadatalifecyclev1.DownloadableFile{Url: value.DownloadableFile.URL, Format: value.DownloadableFile.Format}}
	}
	if value.WebService != nil {
		result.Delivery = &metadatalifecyclev1.Distribution_WebService{WebService: webServiceOutput(*value.WebService, includeContacts)}
	}
	return result
}

func webServiceOutput(value domain.WebService, includeContacts bool) *metadatalifecyclev1.WebService {
	result := &metadatalifecyclev1.WebService{Id: value.ID.String(), Name: value.Name, Description: value.Description, TemplateUrl: value.TemplateURL, Format: value.Format, Documentation: documentationOutput(value.Documentation), SpatialCoverage: geometryString(value.SpatialCoverage), TemporalCoverage: temporalOutput(value.TemporalCoverage), ContactsIncluded: includeContacts}
	if value.Provider != nil {
		result.Provider = new(string)
		*result.Provider = value.Provider.ID.String()
		result.ProviderReference = referenceOutput(value.Provider.ID, value.Provider.Name)
	}
	if includeContacts {
		for _, item := range value.ContactPoints {
			result.ContactPoints = append(result.ContactPoints, item.ID.String())
		}
	}
	for _, item := range value.Parameters {
		result.Parameters = append(result.Parameters, parameterOutput(item))
	}
	return result
}

func documentationOutput(value *domain.Documentation) *metadatalifecyclev1.Documentation {
	if value == nil {
		return nil
	}
	return &metadatalifecyclev1.Documentation{Title: value.Title, Description: value.Description, Uri: value.URI}
}

func parameterOutput(value domain.Parameter) *metadatalifecyclev1.Parameter {
	metadata := value.GetMetadata()
	result := &metadatalifecyclev1.Parameter{Metadata: &metadatalifecyclev1.ParameterMetadata{Id: metadata.ID.String(), Kind: string(metadata.Kind), Variable: metadata.Variable, Property: metadata.Property, Label: metadata.Label, Required: metadata.Required, MultipleValues: metadata.MultipleValues, ReadOnly: metadata.ReadOnly}}
	switch item := value.(type) {
	case domain.StringParameter:
		result.Value = &metadatalifecyclev1.Parameter_String_{String_: &metadatalifecyclev1.StringParameter{Default: item.Default, AllowedValues: item.AllowedValues, Pattern: item.Pattern}}
	case domain.BooleanParameter:
		result.Value = &metadatalifecyclev1.Parameter_Boolean{Boolean: &metadatalifecyclev1.BooleanParameter{Default: item.Default}}
	case domain.IntegerParameter:
		result.Value = &metadatalifecyclev1.Parameter_Integer{Integer: &metadatalifecyclev1.IntegerParameter{Minimum: item.Minimum, Maximum: item.Maximum, Default: item.Default, AllowedValues: item.AllowedValues}}
	case domain.FloatParameter:
		result.Value = &metadatalifecyclev1.Parameter_Float{Float: &metadatalifecyclev1.FloatParameter{Minimum: item.Minimum, Maximum: item.Maximum, Default: item.Default, AllowedValues: item.AllowedValues}}
	case domain.DateTimeParameter:
		result.Value = &metadatalifecyclev1.Parameter_DateTime{DateTime: &metadatalifecyclev1.DateTimeParameter{Minimum: timestampOutput(item.Minimum), Maximum: timestampOutput(item.Maximum), Default: timestampOutput(item.Default), Pattern: item.Pattern}}
	}
	return result
}

func resourceReadError(err error) error {
	switch {
	case errors.Is(err, domain.ErrDatasetForbidden), errors.Is(err, domain.ErrSoftwareApplicationForbidden), errors.Is(err, domain.ErrSoftwareSourceCodeForbidden):
		return fmt.Errorf("read resource: %w", status.Error(codes.PermissionDenied, "resource operation is not permitted"))
	case errors.Is(err, domain.ErrDatasetNotFound), errors.Is(err, domain.ErrSoftwareApplicationNotFound), errors.Is(err, domain.ErrSoftwareSourceCodeNotFound):
		return fmt.Errorf("read resource: %w", status.Error(codes.NotFound, "resource or version not found"))
	default:
		return fmt.Errorf("read resource: %w", status.Error(codes.Internal, "read resource"))
	}
}

func datasetInput(value *metadatalifecyclev1.DatasetMetadata) (domain.DatasetInput, error) {
	contactPoints, err := resourceParseUUIDs(value.GetContactPoints(), "dataset contact point")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	providers, err := resourceParseUUIDs(value.GetProviders(), "dataset provider")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "dataset category")
	if err != nil {
		return domain.DatasetInput{}, err
	}
	temporal, err := temporalInput(value.GetTemporalCoverage())
	if err != nil {
		return domain.DatasetInput{}, err
	}
	spatial, err := spatialInput(value.SpatialCoverage)
	if err != nil {
		return domain.DatasetInput{}, err
	}
	distributions := make([]domain.DistributionInput, 0, len(value.GetDistributions()))
	for _, item := range value.GetDistributions() {
		mapped, err := distributionInput(item)
		if err != nil {
			return domain.DatasetInput{}, err
		}
		distributions = append(distributions, mapped)
	}
	identifiers, err := identifierInputs(value.GetPersistentIdentifiers())
	if err != nil {
		return domain.DatasetInput{}, err
	}
	created, err := timestampInput(value.GetCreated())
	if err != nil {
		return domain.DatasetInput{}, err
	}
	updated, err := timestampInput(value.GetUpdated())
	if err != nil {
		return domain.DatasetInput{}, err
	}
	return domain.DatasetInput{
		Title: value.GetTitle(), Description: value.GetDescription(), Keywords: value.GetKeywords(),
		AccrualPeriodicity: value.AccrualPeriodicity, QualityAnnotation: value.QualityAnnotation, AccessRight: value.AccessRight,
		Distributions: distributions, SpatialCoverage: spatial, TemporalCoverage: temporal,
		PersistentIdentifiers: identifiers, ContactPointIDs: contactPoints, ProviderIDs: providers, CategoryIDs: categories,
		SourceCreated: created, SourceUpdated: updated,
	}, nil
}

//nolint:dupl // Create and draft protobufs intentionally map identical domain fields.
func distributionInput(value *metadatalifecyclev1.Distribution) (domain.DistributionInput, error) {
	if value == nil {
		return domain.DistributionInput{}, status.Error(codes.InvalidArgument, "distribution is required") //nolint:wrapcheck
	}
	issued, err := timestampInput(value.GetIssued())
	if err != nil {
		return domain.DistributionInput{}, err
	}
	modified, err := timestampInput(value.GetModified())
	if err != nil {
		return domain.DistributionInput{}, err
	}
	result := domain.DistributionInput{Title: value.GetTitle(), Description: value.GetDescription(), License: value.License, Type: domain.DistributionType(value.GetType()), Issued: issued, Modified: modified, ByteSize: value.ByteSize}
	switch delivery := value.GetDelivery().(type) {
	case *metadatalifecyclev1.Distribution_WebService:
		mapped, err := webServiceInput(delivery.WebService)
		if err != nil {
			return domain.DistributionInput{}, err
		}
		result.WebService = &mapped
	case *metadatalifecyclev1.Distribution_DownloadableFile:
		if delivery.DownloadableFile == nil {
			return domain.DistributionInput{}, status.Error(codes.InvalidArgument, "downloadable file is required") //nolint:wrapcheck
		}
		result.DownloadableFile = &domain.DownloadableFile{URL: delivery.DownloadableFile.GetUrl(), Format: delivery.DownloadableFile.GetFormat()}
	default:
		return domain.DistributionInput{}, status.Error(codes.InvalidArgument, "distribution delivery method is required") //nolint:wrapcheck
	}
	return result, nil
}

func webServiceInput(value *metadatalifecyclev1.WebService) (domain.WebServiceInput, error) {
	if value == nil {
		return domain.WebServiceInput{}, status.Error(codes.InvalidArgument, "web service is required")
	}
	contactPoints, err := resourceParseUUIDs(value.GetContactPoints(), "web service contact point")
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	var provider *uuid.UUID
	if value.Provider != nil {
		id, err := parseUUID(value.GetProvider(), "web service provider")
		if err != nil {
			return domain.WebServiceInput{}, err
		}
		provider = &id
	}
	spatial, err := spatialInput(value.SpatialCoverage)
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	temporal, err := temporalInput(value.GetTemporalCoverage())
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	documentation, err := documentationInput(value.GetDocumentation())
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	parameters := make([]domain.Parameter, 0, len(value.GetParameters()))
	for _, item := range value.GetParameters() {
		mapped, err := parameterInput(item)
		if err != nil {
			return domain.WebServiceInput{}, err
		}
		parameters = append(parameters, mapped)
	}
	return domain.WebServiceInput{Name: value.GetName(), Description: value.GetDescription(), TemplateURL: value.GetTemplateUrl(), Parameters: parameters, Format: value.GetFormat(), ProviderID: provider, Documentation: documentation, SpatialCoverage: spatial, TemporalCoverage: temporal, ContactPointIDs: contactPoints}, nil
}

func parameterInput(value *metadatalifecyclev1.Parameter) (domain.Parameter, error) {
	if value == nil || value.GetMetadata() == nil {
		return nil, status.Error(codes.InvalidArgument, "parameter metadata is required")
	}
	m := value.GetMetadata()
	metadata := domain.ParameterMetadata{
		Variable:       m.GetVariable(),
		Property:       m.GetProperty(),
		Label:          m.GetLabel(),
		Required:       m.GetRequired(),
		MultipleValues: m.GetMultipleValues(),
		ReadOnly:       m.GetReadOnly(),
	}
	switch v := value.GetValue().(type) {
	case *metadatalifecyclev1.Parameter_String_:
		if v.String_ == nil {
			return nil, status.Error(codes.InvalidArgument, "string parameter is required")
		}
		return domain.StringParameter{
			Metadata:      metadata,
			Default:       v.String_.Default,
			AllowedValues: v.String_.GetAllowedValues(),
			Pattern:       v.String_.GetPattern(),
		}, nil
	case *metadatalifecyclev1.Parameter_Boolean:
		if v.Boolean == nil {
			return nil, status.Error(codes.InvalidArgument, "boolean parameter is required")
		}
		return domain.BooleanParameter{Metadata: metadata, Default: v.Boolean.Default}, nil
	case *metadatalifecyclev1.Parameter_Integer:
		if v.Integer == nil {
			return nil, status.Error(codes.InvalidArgument, "integer parameter is required")
		}
		return domain.IntegerParameter{Metadata: metadata, Minimum: v.Integer.Minimum, Maximum: v.Integer.Maximum, Default: v.Integer.Default, AllowedValues: v.Integer.GetAllowedValues()}, nil
	case *metadatalifecyclev1.Parameter_Float:
		if v.Float == nil {
			return nil, status.Error(codes.InvalidArgument, "float parameter is required")
		}
		return domain.FloatParameter{Metadata: metadata, Minimum: v.Float.Minimum, Maximum: v.Float.Maximum, Default: v.Float.Default, AllowedValues: v.Float.GetAllowedValues()}, nil
	case *metadatalifecyclev1.Parameter_DateTime:
		if v.DateTime == nil {
			return nil, status.Error(codes.InvalidArgument, "date-time parameter is required")
		}
		minimum, err := timestampInput(v.DateTime.GetMinimum())
		if err != nil {
			return nil, err
		}
		maximum, err := timestampInput(v.DateTime.GetMaximum())
		if err != nil {
			return nil, err
		}
		def, err := timestampInput(v.DateTime.GetDefault())
		if err != nil {
			return nil, err
		}
		return domain.DateTimeParameter{Metadata: metadata, Minimum: minimum, Maximum: maximum, Default: def, Pattern: v.DateTime.GetPattern()}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "parameter value is required")
	}
}

func documentationInput(value *metadatalifecyclev1.Documentation) (*domain.Documentation, error) {
	if value == nil {
		return nil, nil
	}
	return &domain.Documentation{Title: value.Title, Description: value.Description, URI: value.GetUri()}, nil
}

//nolint:dupl // Create and draft protobufs intentionally map identical domain fields.
func softwareApplicationInput(value *metadatalifecyclev1.SoftwareApplicationMetadata) (domain.SoftwareApplicationInput, error) {
	creators, err := resourceParseUUIDs(value.GetCreators(), "software application creator")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "software application category")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	contacts, err := resourceParseUUIDs(value.GetContactPoints(), "software application contact point")
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	identifiers, err := identifierInputs(value.GetIdentifiers())
	if err != nil {
		return domain.SoftwareApplicationInput{}, err
	}
	return domain.SoftwareApplicationInput{Name: value.GetName(), Description: value.GetDescription(), MainEntityOfPage: value.MainEntityOfPage, License: value.License, Version: value.Version, Keywords: value.GetKeywords(), DownloadURL: value.DownloadUrl, Requirements: value.Requirements, OperatingSystem: value.OperatingSystem, CreatorIDs: creators, ContactPointIDs: contacts, CategoryIDs: categories, Identifiers: identifiers}, nil
}

//nolint:dupl // Create and draft protobufs intentionally map identical domain fields.
func softwareSourceCodeInput(value *metadatalifecyclev1.SoftwareSourceCodeMetadata) (domain.SoftwareSourceCodeInput, error) {
	creators, err := resourceParseUUIDs(value.GetCreators(), "software source code creator")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	categories, err := resourceParseUUIDs(value.GetCategories(), "software source code category")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	contacts, err := resourceParseUUIDs(value.GetContactPoints(), "software source code contact point")
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	identifiers, err := identifierInputs(value.GetIdentifiers())
	if err != nil {
		return domain.SoftwareSourceCodeInput{}, err
	}
	return domain.SoftwareSourceCodeInput{Name: value.GetName(), Description: value.GetDescription(), ProgrammingLanguage: value.ProgrammingLanguage, Keywords: value.GetKeywords(), Repository: value.Repository, License: value.License, SoftwareRequirements: value.SoftwareRequirements, RuntimePlatform: value.RuntimePlatform, MainEntityOfPage: value.MainEntityOfPage, Version: value.Version, CreatorIDs: creators, ContactPointIDs: contacts, CategoryIDs: categories, Identifiers: identifiers}, nil
}

func identifierInputs(values []*metadatalifecyclev1.Identifier) ([]domain.IdentifierInput, error) {
	result := make([]domain.IdentifierInput, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, status.Error(codes.InvalidArgument, "identifier is required")
		}
		result = append(result, domain.IdentifierInput{Type: domain.IdentifierType(value.GetType()), Value: value.GetValue()})
	}
	return result, nil
}

func parseUUID(value, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid %s", field)
	}
	return id, nil
}

func resourceParseUUIDs(values []string, field string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := parseUUID(value, field)
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
		return nil, status.Error(codes.InvalidArgument, "invalid timestamp")
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
		return nil, err
	}
	end, err := timestampInput(value.GetEnd())
	if err != nil {
		return nil, err
	}
	return &domain.Temporal{Start: start, End: end}, nil
}

func spatialInput(value *string) (geom.T, error) {
	if value == nil {
		return nil, nil
	}
	geometry, err := wkt.Unmarshal(strings.TrimSpace(*value))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid spatial coverage WKT")
	}
	geometry, err = geom.SetSRID(geometry, 4326)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid spatial coverage SRID")
	}
	return geometry, nil
}

func createResponse(resourceID, versionID uuid.UUID, revision int32, resourceStatus domain.Status) *metadatalifecyclev1.CreateResourceResponse {
	return &metadatalifecyclev1.CreateResourceResponse{Resource: &metadatalifecyclev1.ResourceRecord{ResourceId: resourceID.String(), VersionId: versionID.String(), Revision: revision, Status: string(resourceStatus)}}
}

func resourceError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidDataset), errors.Is(err, domain.ErrSoftwareApplicationInvalid), errors.Is(err, domain.ErrSoftwareSourceCodeInvalid):
		return fmt.Errorf("create resource: %w", status.Error(codes.InvalidArgument, "invalid resource draft"))
	case errors.Is(err, domain.ErrDatasetForbidden), errors.Is(err, domain.ErrSoftwareApplicationForbidden), errors.Is(err, domain.ErrSoftwareSourceCodeForbidden):
		return fmt.Errorf("create resource: %w", status.Error(codes.PermissionDenied, "resource operation is not permitted"))
	case errors.Is(err, domain.ErrDatasetConflict), errors.Is(err, domain.ErrSoftwareApplicationConflict), errors.Is(err, domain.ErrSoftwareSourceCodeConflict):
		return fmt.Errorf("create resource: %w", status.Error(codes.AlreadyExists, "resource operation conflicts with existing data"))
	case errors.Is(err, domain.ErrDatasetNotFound), errors.Is(err, domain.ErrSoftwareApplicationNotFound), errors.Is(err, domain.ErrSoftwareSourceCodeNotFound):
		return fmt.Errorf("create resource: %w", status.Error(codes.NotFound, "resource or version not found"))
	default:
		return fmt.Errorf("create resource: %w", status.Error(codes.Internal, "create resource"))
	}
}
