package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoDatasetType(value gatewayapi.OptNilDatasetType) (*metadatalifecyclev1.DatasetType, error) {
	if !value.Set || value.Null {
		return nil, nil
	}
	if err := value.Value.Validate(); err != nil {
		return nil, fmt.Errorf("unsupported dataset type %q", value.Value)
	}
	number, ok := metadatalifecyclev1.DatasetType_value["DATASET_TYPE_"+string(value.Value)]
	if !ok || number == 0 {
		return nil, fmt.Errorf("unsupported dataset type %q", value.Value)
	}
	result := metadatalifecyclev1.DatasetType(number)
	return &result, nil
}

func toAPIDatasetType(value *metadatalifecyclev1.DatasetType) (gatewayapi.OptNilDatasetType, error) {
	result := gatewayapi.OptNilDatasetType{}
	if value == nil {
		return result, nil
	}
	name, ok := metadatalifecyclev1.DatasetType_name[int32(*value)]
	mapped := gatewayapi.DatasetType(strings.TrimPrefix(name, "DATASET_TYPE_"))
	if !ok || mapped.Validate() != nil {
		return result, fmt.Errorf("unsupported upstream dataset type %d", *value)
	}
	result.SetTo(mapped)
	return result, nil
}

func toClientStringPtr(v gatewayapi.OptNilString) *string {
	if !v.Set || v.Null {
		return nil
	}
	return &v.Value
}

func toAPIRequiredNilString(value *string) gatewayapi.NilString {
	if value == nil {
		return gatewayapi.NilString{Null: true}
	}
	return gatewayapi.NewNilString(*value)
}

func toAPIOptURI(field string, value *string) (gatewayapi.OptURI, error) {
	result := gatewayapi.OptURI{}
	if value == nil {
		return result, nil
	}
	if *value == "" {
		return result, fmt.Errorf("decode upstream %s: URI must not be blank", field)
	}
	parsed, err := parseURI(*value)
	if err != nil {
		return result, fmt.Errorf("decode upstream %s: %w", field, err)
	}
	result.SetTo(parsed)
	return result, nil
}

func parseURI(value string) (url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return url.URL{}, fmt.Errorf("parse URI: %w", err)
	}
	return *parsed, nil
}

func toProtoDatasetAccessRight(value gatewayapi.OptNilDatasetAccessRight) (*metadatalifecyclev1.DatasetAccessRight, error) {
	if !value.Set || value.Null {
		return nil, nil
	}
	var result metadatalifecyclev1.DatasetAccessRight
	switch value.Value {
	case gatewayapi.DatasetAccessRightNONPUBLIC:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_NON_PUBLIC
	case gatewayapi.DatasetAccessRightNORMAL:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_NORMAL
	case gatewayapi.DatasetAccessRightOPDATPRO:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_OP_DATPRO
	case gatewayapi.DatasetAccessRightPUBLIC:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_PUBLIC
	case gatewayapi.DatasetAccessRightRESTRICTED:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_RESTRICTED
	case gatewayapi.DatasetAccessRightCONFIDENTIAL:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_CONFIDENTIAL
	case gatewayapi.DatasetAccessRightSENSITIVE:
		result = metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_SENSITIVE
	default:
		return nil, fmt.Errorf("unsupported dataset access right %q", value.Value)
	}
	return &result, nil
}

func toAPIDatasetAccessRight(value *metadatalifecyclev1.DatasetAccessRight) (gatewayapi.OptNilDatasetAccessRight, error) {
	result := gatewayapi.OptNilDatasetAccessRight{}
	if value == nil {
		return result, nil
	}
	switch *value {
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_NON_PUBLIC:
		result.SetTo(gatewayapi.DatasetAccessRightNONPUBLIC)
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_NORMAL:
		result.SetTo(gatewayapi.DatasetAccessRightNORMAL)
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_OP_DATPRO:
		result.SetTo(gatewayapi.DatasetAccessRightOPDATPRO)
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_PUBLIC:
		result.SetTo(gatewayapi.DatasetAccessRightPUBLIC)
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_RESTRICTED:
		result.SetTo(gatewayapi.DatasetAccessRightRESTRICTED)
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_CONFIDENTIAL:
		result.SetTo(gatewayapi.DatasetAccessRightCONFIDENTIAL)
	case metadatalifecyclev1.DatasetAccessRight_DATASET_ACCESS_RIGHT_SENSITIVE:
		result.SetTo(gatewayapi.DatasetAccessRightSENSITIVE)
	default:
		return gatewayapi.OptNilDatasetAccessRight{}, fmt.Errorf("unsupported upstream dataset access right %d", *value)
	}
	return result, nil
}

func toProtoDatasetAccrualPeriodicity(value gatewayapi.OptNilDatasetAccrualPeriodicity) (*metadatalifecyclev1.DatasetAccrualPeriodicity, error) {
	if !value.Set || value.Null {
		return nil, nil
	}
	var result metadatalifecyclev1.DatasetAccrualPeriodicity
	switch value.Value {
	case gatewayapi.DatasetAccrualPeriodicityTRIENNIAL:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_TRIENNIAL
	case gatewayapi.DatasetAccrualPeriodicityBIENNIAL:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIENNIAL
	case gatewayapi.DatasetAccrualPeriodicityANNUAL:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_ANNUAL
	case gatewayapi.DatasetAccrualPeriodicitySEMIANNUAL:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIANNUAL
	case gatewayapi.DatasetAccrualPeriodicityTHREETIMESAYEAR:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_YEAR
	case gatewayapi.DatasetAccrualPeriodicityQUARTERLY:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_QUARTERLY
	case gatewayapi.DatasetAccrualPeriodicityBIMONTHLY:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIMONTHLY
	case gatewayapi.DatasetAccrualPeriodicityMONTHLY:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_MONTHLY
	case gatewayapi.DatasetAccrualPeriodicitySEMIMONTHLY:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIMONTHLY
	case gatewayapi.DatasetAccrualPeriodicityBIWEEKLY:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIWEEKLY
	case gatewayapi.DatasetAccrualPeriodicityTHREETIMESAMONTH:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_MONTH
	case gatewayapi.DatasetAccrualPeriodicityWEEKLY:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_WEEKLY
	case gatewayapi.DatasetAccrualPeriodicitySEMIWEEKLY:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIWEEKLY
	case gatewayapi.DatasetAccrualPeriodicityTHREETIMESAWEEK:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_WEEK
	case gatewayapi.DatasetAccrualPeriodicityDAILY:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_DAILY
	case gatewayapi.DatasetAccrualPeriodicityCONTINUOUS:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_CONTINUOUS
	case gatewayapi.DatasetAccrualPeriodicityIRREGULAR:
		result = metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_IRREGULAR
	default:
		return nil, fmt.Errorf("unsupported dataset accrual periodicity %q", value.Value)
	}
	return &result, nil
}

func toAPIDatasetAccrualPeriodicity(value *metadatalifecyclev1.DatasetAccrualPeriodicity) (gatewayapi.OptNilDatasetAccrualPeriodicity, error) {
	result := gatewayapi.OptNilDatasetAccrualPeriodicity{}
	if value == nil {
		return result, nil
	}
	switch *value {
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_TRIENNIAL:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityTRIENNIAL)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIENNIAL:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityBIENNIAL)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_ANNUAL:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityANNUAL)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIANNUAL:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicitySEMIANNUAL)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_YEAR:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityTHREETIMESAYEAR)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_QUARTERLY:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityQUARTERLY)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIMONTHLY:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityBIMONTHLY)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_MONTHLY:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityMONTHLY)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIMONTHLY:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicitySEMIMONTHLY)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_BIWEEKLY:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityBIWEEKLY)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_MONTH:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityTHREETIMESAMONTH)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_WEEKLY:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityWEEKLY)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_SEMIWEEKLY:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicitySEMIWEEKLY)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_THREE_TIMES_A_WEEK:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityTHREETIMESAWEEK)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_DAILY:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityDAILY)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_CONTINUOUS:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityCONTINUOUS)
	case metadatalifecyclev1.DatasetAccrualPeriodicity_DATASET_ACCRUAL_PERIODICITY_IRREGULAR:
		result.SetTo(gatewayapi.DatasetAccrualPeriodicityIRREGULAR)
	default:
		return gatewayapi.OptNilDatasetAccrualPeriodicity{}, fmt.Errorf("unsupported upstream dataset accrual periodicity %d", *value)
	}
	return result, nil
}

func toProtoDistributionType(value gatewayapi.DistributionDraftType) (metadatalifecyclev1.DistributionType, error) {
	switch value {
	case gatewayapi.WebServiceDistributionDraftDistributionDraft:
		return metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_WEB_SERVICE, nil
	case gatewayapi.DownloadableFileDistributionDraftDistributionDraft:
		return metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_DOWNLOADABLE_FILE, nil
	default:
		return metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_UNSPECIFIED, fmt.Errorf("unsupported distribution type %q", value)
	}
}

func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid upstream %s: %w", field, err)
	}
	return id, nil
}

func fromProtoTimestamp(field string, value *timestamppb.Timestamp) (time.Time, error) {
	if value == nil {
		return time.Time{}, fmt.Errorf("decode upstream %s: timestamp is required", field)
	}
	if err := value.CheckValid(); err != nil {
		return time.Time{}, fmt.Errorf("decode upstream %s: %w", field, err)
	}
	return value.AsTime(), nil
}

func toProtoKeywords(values []gatewayapi.Keyword) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}
	return result
}

func toAPIKeywords(values []string) []gatewayapi.Keyword {
	result := make([]gatewayapi.Keyword, len(values))
	for i, value := range values {
		result[i] = gatewayapi.Keyword(value)
	}
	return result
}
