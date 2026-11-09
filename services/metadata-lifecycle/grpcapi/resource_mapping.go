package grpcapi

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toDomainResourceType(value metadatalifecyclev1.ResourceType) (domain.ResourceType, bool) {
	switch value {
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET:
		return domain.Dataset, true
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION:
		return domain.SoftwareApplication, true
	case metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE:
		return domain.SoftwareSourceCode, true
	default:
		return "", false
	}
}

func toProtoDistributionIDs(ids []uuid.UUID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, id.String())
	}
	return result
}

func toProtoDistributionIDPairs(pairs map[uuid.UUID]uuid.UUID) map[string]string {
	result := make(map[string]string, len(pairs))
	for sourceID, targetID := range pairs {
		result[sourceID.String()] = targetID.String()
	}
	return result
}

func toDomainDistributionAccessInput(value *metadatalifecyclev1.PrepareDistributionAccessRequest) (domain.DistributionAccessInput, error) {
	if value == nil {
		return domain.DistributionAccessInput{}, apperror.Field("type", "Request type is required.", nil)
	}
	var parameters []*metadatalifecyclev1.DistributionAccessParameterInput
	switch input := value.Input.(type) {
	case *metadatalifecyclev1.PrepareDistributionAccessRequest_Http:
		if input.Http == nil {
			return domain.DistributionAccessInput{}, apperror.Field("http", "HTTP input is required.", nil)
		}
		parameters = input.Http.Parameters
	case *metadatalifecyclev1.PrepareDistributionAccessRequest_Wfs:
		if input.Wfs == nil {
			return domain.DistributionAccessInput{}, apperror.Field("wfs", "WFS input is required.", nil)
		}
		parameters = input.Wfs.Parameters
	case *metadatalifecyclev1.PrepareDistributionAccessRequest_Dynamic:
		if input.Dynamic == nil {
			return domain.DistributionAccessInput{}, apperror.Field("dynamic", "DYNAMIC input is required.", nil)
		}
		parameters = input.Dynamic.Parameters
	default:
		return domain.DistributionAccessInput{}, apperror.Field("type", "Request type is required.", nil)
	}
	result := domain.DistributionAccessInput{
		OutputFormat: value.OutputFormat,
	}
	mappedParameters := make([]domain.DistributionAccessParameter, len(parameters))
	for i, parameter := range parameters {
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

		mappedParameters[i] = domain.DistributionAccessParameter{ID: id, Value: mapped}
	}
	switch input := value.Input.(type) {
	case *metadatalifecyclev1.PrepareDistributionAccessRequest_Http:
		result.Input = domain.HTTPAccessInput{Parameters: mappedParameters}
	case *metadatalifecyclev1.PrepareDistributionAccessRequest_Wfs:
		wfs := domain.WFSAccessInput{
			FeatureType: input.Wfs.FeatureType,
			ResultLimit: input.Wfs.ResultLimit,
			Parameters:  mappedParameters,
		}
		if bbox := input.Wfs.Bbox; bbox != nil {
			wfs.BBox = &domain.WFSBoundingBox{
				West: bbox.West, South: bbox.South, East: bbox.East, North: bbox.North,
			}
		}
		result.Input = wfs
	case *metadatalifecyclev1.PrepareDistributionAccessRequest_Dynamic:
		result.Input = domain.DynamicAccessInput{
			Longitude:  input.Dynamic.Longitude,
			Latitude:   input.Dynamic.Latitude,
			Parameters: mappedParameters,
		}
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
		Id:            value.VersionID.String(),
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

func toDomainAccrualPeriodicity(value *metadatalifecyclev1.DatasetAccrualPeriodicity) (*domain.AccrualPeriodicity, error) {
	if value == nil {
		return nil, nil
	}
	var result domain.AccrualPeriodicity
	switch *value {
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_TRIENNIAL:
		result = domain.AccrualPeriodicityTriennial
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIENNIAL:
		result = domain.AccrualPeriodicityBiennial
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_ANNUAL:
		result = domain.AccrualPeriodicityAnnual
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIANNUAL:
		result = domain.AccrualPeriodicitySemiannual
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_YEAR:
		result = domain.AccrualPeriodicityThreeTimesAYear
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_QUARTERLY:
		result = domain.AccrualPeriodicityQuarterly
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIMONTHLY:
		result = domain.AccrualPeriodicityBimonthly
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_MONTHLY:
		result = domain.AccrualPeriodicityMonthly
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIMONTHLY:
		result = domain.AccrualPeriodicitySemimonthly
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIWEEKLY:
		result = domain.AccrualPeriodicityBiweekly
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_MONTH:
		result = domain.AccrualPeriodicityThreeTimesAMonth
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_WEEKLY:
		result = domain.AccrualPeriodicityWeekly
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIWEEKLY:
		result = domain.AccrualPeriodicitySemiweekly
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_WEEK:
		result = domain.AccrualPeriodicityThreeTimesAWeek
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_DAILY:
		result = domain.AccrualPeriodicityDaily
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_CONTINUOUS:
		result = domain.AccrualPeriodicityContinuous
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_IRREGULAR:
		result = domain.AccrualPeriodicityIrregular
	default:
		return nil, apperror.Field("accrual_periodicity", "Accrual periodicity is not supported.", nil)
	}
	return &result, nil
}

func toProtoAccrualPeriodicity(value *domain.AccrualPeriodicity) *metadatalifecyclev1.DatasetAccrualPeriodicity {
	if value == nil {
		return nil
	}
	var result metadatalifecyclev1.DatasetAccrualPeriodicity
	switch *value {
	case domain.AccrualPeriodicityTriennial:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_TRIENNIAL
	case domain.AccrualPeriodicityBiennial:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIENNIAL
	case domain.AccrualPeriodicityAnnual:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_ANNUAL
	case domain.AccrualPeriodicitySemiannual:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIANNUAL
	case domain.AccrualPeriodicityThreeTimesAYear:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_YEAR
	case domain.AccrualPeriodicityQuarterly:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_QUARTERLY
	case domain.AccrualPeriodicityBimonthly:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIMONTHLY
	case domain.AccrualPeriodicityMonthly:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_MONTHLY
	case domain.AccrualPeriodicitySemimonthly:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIMONTHLY
	case domain.AccrualPeriodicityBiweekly:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIWEEKLY
	case domain.AccrualPeriodicityThreeTimesAMonth:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_MONTH
	case domain.AccrualPeriodicityWeekly:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_WEEKLY
	case domain.AccrualPeriodicitySemiweekly:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIWEEKLY
	case domain.AccrualPeriodicityThreeTimesAWeek:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_WEEK
	case domain.AccrualPeriodicityDaily:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_DAILY
	case domain.AccrualPeriodicityContinuous:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_CONTINUOUS
	case domain.AccrualPeriodicityIrregular:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_IRREGULAR
	default:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_UNSPECIFIED
	}
	return &result
}

func toDomainDistributionType(value metadatalifecyclev1.DistributionType) (domain.DistributionType, error) {
	switch value {
	case metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_UNSPECIFIED:
		return "", nil
	case metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_WEB_SERVICE:
		return domain.DistributionTypeWebService, nil
	case metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_DOWNLOADABLE_FILE:
		return domain.DistributionTypeDownloadableFile, nil
	default:
		return "", apperror.Field("type", "Distribution type is not supported.", nil)
	}
}

func toProtoDistributionType(value domain.DistributionType) metadatalifecyclev1.DistributionType {
	switch value {
	case domain.DistributionTypeWebService:
		return metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_WEB_SERVICE
	case domain.DistributionTypeDownloadableFile:
		return metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_DOWNLOADABLE_FILE
	default:
		return metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_UNSPECIFIED
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
		Id:            value.VersionID.String(),
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
		Id:            value.VersionID.String(),
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
