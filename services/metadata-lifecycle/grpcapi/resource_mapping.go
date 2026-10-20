package grpcapi

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoNativeFormats(values []domain.NativeFormat) []*metadatalifecyclev1.NativeFormat {
	result := make([]*metadatalifecyclev1.NativeFormat, 0, len(values))
	for _, value := range values {
		result = append(result, &metadatalifecyclev1.NativeFormat{
			OutputFormat: value.OutputFormat,
			Label:        value.Label,
		})
	}
	return result
}

func toProtoDatasetVersion(value domain.DatasetVersion) *metadatalifecyclev1.ResourceVersion {
	return &metadatalifecyclev1.ResourceVersion{
		ResourceId:    value.DatasetID.String(),
		VersionId:     value.VersionID.String(),
		Revision:      value.Revision,
		Status:        toGRPCVersioningStatus(value.Status),
		CreatedBy:     string(value.CreatedBy),
		GroupId:       value.Group.ID.String(),
		GroupName:     value.Group.Name,
		CreatedAt:     timestamppb.New(value.CreatedAt),
		UpdatedAt:     timestamppb.New(value.UpdatedAt),
		Metadata:      &metadatalifecyclev1.ResourceVersion_Dataset{Dataset: datasetResourceMetadata(value.Metadata)},
		RejectedBy:    value.RejectedBy,
		RejectedAt:    timestampOutput(value.RejectedAt),
		RejectionNote: value.RejectionNote,
	}
}

func toProtoDistributionDownload(value domain.DistributionDownload) *metadatalifecyclev1.PrepareDistributionDownloadResponse {
	return &metadatalifecyclev1.PrepareDistributionDownloadResponse{
		ResourceId: value.ResourceID.String(),
		VersionId:  value.VersionID.String(),
		Url:        value.URL,
	}
}

func toProtoSoftwareApplicationVersion(value domain.SoftwareApplicationVersion) *metadatalifecyclev1.ResourceVersion {
	return &metadatalifecyclev1.ResourceVersion{
		ResourceId:    value.SoftwareApplicationID.String(),
		VersionId:     value.VersionID.String(),
		Revision:      value.Revision,
		Status:        toGRPCVersioningStatus(value.Status),
		CreatedBy:     string(value.CreatedBy),
		GroupId:       value.Group.ID.String(),
		GroupName:     value.Group.Name,
		CreatedAt:     timestamppb.New(value.CreatedAt),
		UpdatedAt:     timestamppb.New(value.UpdatedAt),
		Metadata:      &metadatalifecyclev1.ResourceVersion_SoftwareApplication{SoftwareApplication: applicationResourceMetadata(value.Metadata)},
		RejectedBy:    value.RejectedBy,
		RejectedAt:    timestampOutput(value.RejectedAt),
		RejectionNote: value.RejectionNote,
	}
}

func toProtoSoftwareSourceCodeVersion(value domain.SoftwareSourceCodeVersion) *metadatalifecyclev1.ResourceVersion {
	return &metadatalifecyclev1.ResourceVersion{
		ResourceId:    value.SoftwareSourceCodeID.String(),
		VersionId:     value.VersionID.String(),
		Revision:      value.Revision,
		Status:        toGRPCVersioningStatus(value.Status),
		CreatedBy:     string(value.CreatedBy),
		GroupId:       value.Group.ID.String(),
		GroupName:     value.Group.Name,
		CreatedAt:     timestamppb.New(value.CreatedAt),
		UpdatedAt:     timestamppb.New(value.UpdatedAt),
		Metadata:      &metadatalifecyclev1.ResourceVersion_SoftwareSourceCode{SoftwareSourceCode: sourceCodeResourceMetadata(value.Metadata)},
		RejectedBy:    value.RejectedBy,
		RejectedAt:    timestampOutput(value.RejectedAt),
		RejectionNote: value.RejectionNote,
	}
}
