package domain

import (
	"io"

	"github.com/google/uuid"
)

// DownloadTarget is a metadata-authorized file request for Data Retriever.
type DownloadTarget struct {
	// TODO: maybe we sholud return the distribution id too?
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	URL        string
}

type DownloadResponse struct {
	StatusCode int
	Body       io.ReadCloser
}
