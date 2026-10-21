package grpcapi

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toDomainDistributionAccessInput(value *metadatalifecyclev1.PrepareDistributionAccessRequest) (domain.DistributionAccessInput, error) {
	result := domain.DistributionAccessInput{
		OutputFormat: value.OutputFormat,
		Parameters:   make([]domain.DistributionAccessParameter, len(value.Parameters)),
	}
	for i, parameter := range value.Parameters {
		target := fmt.Sprintf("parameters[%d]", i)
		if parameter == nil {
			return domain.DistributionAccessInput{}, apperror.Field(target, "Parameter is required.", nil)
		}

		id, err := parseUUIDAt(parameter.Id, "parameter id", target+".id")
		if err != nil {
			return domain.DistributionAccessInput{}, err
		}

		var mapped domain.AccessValue
		switch input := parameter.Value.(type) {
		case *metadatalifecyclev1.DistributionAccessParameterInput_Scalar:
			mapped, err = toDomainAccessScalar(input.Scalar)
		case *metadatalifecyclev1.DistributionAccessParameterInput_Array:
			if input.Array == nil {
				err = apperror.Field("value", "Array is required.", nil)
				break
			}

			items := make(domain.AccessArray, len(input.Array.Values))
			for j, item := range input.Array.Values {
				items[j], err = toDomainAccessScalar(item)
				if err != nil {
					err = apperror.PrefixTarget(err, fmt.Sprintf("array.values[%d]", j))
					break
				}
			}
			mapped = items
		default:
			err = apperror.Field("value", "Scalar or array value is required.", nil)
		}

		if err != nil {
			return domain.DistributionAccessInput{}, apperror.PrefixTarget(err, target)
		}

		result.Parameters[i] = domain.DistributionAccessParameter{ID: id, Value: mapped}
	}

	return result, nil
}

func toDomainAccessScalar(value *metadatalifecyclev1.DistributionAccessScalarValue) (domain.AccessScalar, error) {
	if value == nil {
		return nil, apperror.Field("value", "Scalar value is required.", nil)
	}

	switch value := value.Value.(type) {
	case *metadatalifecyclev1.DistributionAccessScalarValue_StringValue:
		return domain.AccessString(value.StringValue), nil
	case *metadatalifecyclev1.DistributionAccessScalarValue_BooleanValue:
		return domain.AccessBoolean(value.BooleanValue), nil
	case *metadatalifecyclev1.DistributionAccessScalarValue_IntegerValue:
		return domain.AccessInteger(value.IntegerValue), nil
	case *metadatalifecyclev1.DistributionAccessScalarValue_FloatValue:
		return domain.AccessFloat(value.FloatValue), nil
	default:
		return nil, apperror.Field("value", "Scalar value is required.", nil)
	}
}

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
