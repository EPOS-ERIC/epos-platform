package handlers

import (
	"fmt"
	"net/url"
	"time"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
