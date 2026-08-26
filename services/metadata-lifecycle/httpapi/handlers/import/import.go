// Package imports exposes metadata import HTTP handlers.
package imports

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadataimport"
)

type importService interface {
	Import(context.Context, string, io.Reader) (metadataimport.Result, error)
}

type handler struct {
	service importService
}

var log = logging.Get("import-handler")

// New creates a Turtle import handler.
func New(service importService) *handler {
	return &handler{service: service}
}

func (h *handler) Turtle(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/turtle" {
		http.Error(w, "Content-Type must be text/turtle", http.StatusUnsupportedMediaType)
		return
	}
	result, err := h.service.Import(r.Context(), actorID, http.MaxBytesReader(w, r.Body, 50<<20))
	if err != nil {
		writeImportError(r, w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, resultResponse{
		Groups:               result.Groups,
		Categories:           result.Categories,
		Organizations:        result.Organizations,
		ContactPoints:        result.ContactPoints,
		Datasets:             result.Datasets,
		SoftwareApplications: result.SoftwareApplications,
		SoftwareSourceCodes:  result.SoftwareSourceCodes,
		Warnings:             diagnostics(result.Warnings),
	})
}

type resultResponse struct {
	Groups               int          `json:"groups"`
	Categories           int          `json:"categories"`
	Organizations        int          `json:"organizations"`
	ContactPoints        int          `json:"contact_points"`
	Datasets             int          `json:"datasets"`
	SoftwareApplications int          `json:"software_applications"`
	SoftwareSourceCodes  int          `json:"software_source_codes"`
	Warnings             []diagnostic `json:"warnings"`
}

type diagnostic struct {
	Severity  metadataimport.Severity `json:"severity"`
	Subject   string                  `json:"subject"`
	Predicate string                  `json:"predicate"`
	Message   string                  `json:"message"`
}

func diagnostics(values []metadataimport.Diagnostic) []diagnostic {
	result := make([]diagnostic, 0, len(values))
	for _, value := range values {
		result = append(result, diagnostic{
			Severity:  value.Severity,
			Subject:   value.Subject,
			Predicate: value.Predicate,
			Message:   value.Message,
		})
	}
	return result
}

func writeImportError(r *http.Request, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, metadataimport.ErrDenied):
		transport.WriteProblem(w, http.StatusForbidden, "forbidden", "global admin required")
	case errors.Is(err, metadataimport.ErrInvalid):
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_import", err.Error())
	case errors.Is(err, metadataimport.ErrConflict):
		transport.WriteProblem(w, http.StatusConflict, "import_conflict", err.Error())
	default:
		log.ErrorContext(r.Context(), "metadata import failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "metadata import failed")
	}
}
