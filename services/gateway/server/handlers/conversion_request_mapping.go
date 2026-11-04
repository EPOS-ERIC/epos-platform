package handlers

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func toProtoConversionRequest(value *gatewayapi.RequestDistributionConversionReq, distributionID uuid.UUID) *metadatalifecyclev1.RequestDistributionConversionRequest {
	return &metadatalifecyclev1.RequestDistributionConversionRequest{
		DistributionId: distributionID.String(),
		InputFormat:    value.InputFormat,
		Visualizations: toProtoWebServiceVisibleOn(gatewayapi.NewOptNilWebServiceVisibleOnArray(value.Visualizations)),
		Message:        value.Message,
	}
}
