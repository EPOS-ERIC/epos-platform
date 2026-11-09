package grpcapi

import (
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toDomainConversionRequestInput(value *metadatalifecyclev1.RequestDistributionConversionRequest) (domain.ConversionRequestInput, error) {
	visualizations, err := webServiceVisibleOnInput(value.GetVisualizations())
	if err != nil {
		return domain.ConversionRequestInput{}, err
	}

	return domain.ConversionRequestInput{
		InputFormat:    value.GetInputFormat(),
		Visualizations: visualizations,
		Message:        value.GetMessage(),
	}, nil
}
