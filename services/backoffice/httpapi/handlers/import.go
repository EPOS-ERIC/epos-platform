package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/identity"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadataimport"
	"github.com/google/uuid"
)

type MetadataImport struct {
	importer *metadataimport.MetadataImporter
	log      *slog.Logger
}

func NewMetadataImport(importer *metadataimport.MetadataImporter) *MetadataImport {
	return &MetadataImport{importer: importer, log: logging.Get("metadata-import-handler")}
}

type metadataImportResponse struct {
	Datasets        int               `json:"datasets"`
	CategorySchemes int               `json:"categorySchemes"`
	Categories      int               `json:"categories"`
	Organizations   int               `json:"organizations"`
	ContactPoints   int               `json:"contactPoints"`
	Warnings        []importerWarning `json:"warnings,omitempty"`
}

type importerWarning struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (h *MetadataImport) Import(w http.ResponseWriter, r *http.Request) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		writeMessage(w, http.StatusBadRequest, "invalid content type")
		return
	}

	var input io.Reader
	var editorID, groupID uuid.UUID
	switch contentType {
	case "multipart/form-data":
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			writeMessage(w, http.StatusBadRequest, "request must be multipart form data")
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			writeMessage(w, http.StatusBadRequest, "file is required")
			return
		}
		defer func(ctx context.Context) {
			if closeErr := file.Close(); closeErr != nil {
				h.log.WarnContext(ctx, "close uploaded metadata file failed", "error", closeErr)
			}
		}(r.Context())
		input = file
		editorID, err = parseUUID(r.URL.Query().Get("editorId"), "editorId")
		if err != nil {
			writeMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		groupID, err = parseUUID(r.URL.Query().Get("groupId"), "groupId")
		if err != nil {
			writeMessage(w, http.StatusBadRequest, err.Error())
			return
		}
	case "text/turtle":
		input = http.MaxBytesReader(w, r.Body, 50<<20)
		editorID, err = parseUUID(r.URL.Query().Get("editorId"), "editorId")
		if err != nil {
			writeMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		groupID, err = parseUUID(r.URL.Query().Get("groupId"), "groupId")
		if err != nil {
			writeMessage(w, http.StatusBadRequest, err.Error())
			return
		}
	default:
		writeMessage(w, http.StatusBadRequest, "content type must be multipart/form-data or text/turtle")
		return
	}

	status, err := parseStatus(r.URL.Query().Get("status"))
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	batch, err := h.importer.Import(r.Context(), input, editorID, groupID, status)
	if err != nil {
		h.log.ErrorContext(r.Context(), "metadata import failed", "error", err)
		switch {
		case errors.Is(err, metadataimport.ErrMetadataImportPersist):
			writeMessage(w, http.StatusInternalServerError, "metadata import persistence failed")
		case errors.Is(err, identity.ErrNotFound):
			writeServiceError(w, err, "editor or group")
		case errors.Is(err, metadataimport.ErrMetadataImportStatus):
			writeMessage(w, http.StatusForbidden, err.Error())
		default:
			writeMessage(w, http.StatusUnprocessableEntity, err.Error())
		}
		return
	}

	warnings := make([]importerWarning, 0, len(batch.Warnings))
	for _, warning := range batch.Warnings {
		warnings = append(warnings, importerWarning{Subject: warning.Subject, Message: warning.Message})
	}
	writeJSON(w, http.StatusCreated, metadataImportResponse{
		Datasets: len(batch.Datasets), CategorySchemes: len(batch.CategorySchemes), Categories: len(batch.Categories),
		Organizations: len(batch.Organizations), ContactPoints: len(batch.ContactPoints), Warnings: warnings,
	})
}

func parseStatus(raw string) (metadata.DatasetStatus, error) {
	status := metadata.DatasetStatus(strings.ToUpper(strings.TrimSpace(raw)))
	if status == "" {
		return metadata.DatasetStatusDraft, nil
	}
	if !status.Valid() {
		return "", fmt.Errorf("invalid status %q", raw)
	}
	return status, nil
}

func parseUUID(raw, name string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, fmt.Errorf("%s is required", name)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", name)
	}
	return id, nil
}
