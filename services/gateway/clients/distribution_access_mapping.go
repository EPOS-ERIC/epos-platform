package clients

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

func fromProtoPreparedDistributionDownload(response *metadatalifecyclev1.PrepareDistributionDownloadResponse) (domain.DownloadTarget, error) {
	if response == nil {
		return domain.DownloadTarget{}, apperror.Default(apperror.CodeInternal)
	}

	resourceID, err := uuid.Parse(response.GetResourceId())
	if err != nil || resourceID == uuid.Nil {
		return domain.DownloadTarget{}, apperror.Default(apperror.CodeInternal)
	}

	versionID, err := uuid.Parse(response.GetVersionId())
	if err != nil || versionID == uuid.Nil {
		return domain.DownloadTarget{}, apperror.Default(apperror.CodeInternal)
	}

	if response.GetUrl() == "" {
		return domain.DownloadTarget{}, apperror.Default(apperror.CodeInternal)
	}

	return domain.DownloadTarget{
		ResourceID: resourceID,
		VersionID:  versionID,
		URL:        response.GetUrl(),
	}, nil
}

func fromProtoDownloadStreamHeaders(response *dataretrieverv1.RetrieveStreamResponse) (int, error) {
	if response == nil || response.GetHeaders() == nil {
		return 0, apperror.Default(apperror.CodeInternal)
	}

	status := int(response.GetHeaders().GetUpstreamStatus())
	if status < http.StatusOK || status > 599 {
		return 0, apperror.Default(apperror.CodeInternal)
	}

	return status, nil
}
