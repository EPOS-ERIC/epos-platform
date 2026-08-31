// Package imports exposes metadata import HTTP handlers.
package imports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
)

type service interface {
	Import(context.Context, domain.Actor, domain.ImportBatch) (domain.ImportResult, error)
}

type parser interface {
	Parse(io.Reader) (domain.ImportBatch, error)
}

type Handler struct {
	service service
	parser  parser
	log     *slog.Logger
}

// New creates a Turtle import handler.
func New(log *slog.Logger, service service, parser parser) *Handler {
	return &Handler{
		service: service,
		parser:  parser,
		log:     log.With("component", "import-handler"),
	}
}

func (h *Handler) Turtle(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/turtle" {
		http.Error(w, "Content-Type must be text/turtle", http.StatusUnsupportedMediaType)
		return
	}
	batch, err := h.parser.Parse(http.MaxBytesReader(w, r.Body, 50<<20))
	if err != nil {
		writeImportError(h.log, r, w, fmt.Errorf("parse Turtle: %w: %w", domain.ErrInvalidImport, err))
		return
	}

	result, err := h.service.Import(r.Context(), actor, batch)
	if err != nil {
		writeImportError(h.log, r, w, err)
		return
	}
	transport.WriteJSON(h.log, w, http.StatusOK, toResponse(result))
}

func writeImportError(log *slog.Logger, r *http.Request, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrImportForbidden):
		transport.WriteProblem(log, w, http.StatusForbidden, "forbidden", "global admin required")
	case errors.Is(err, domain.ErrInvalidImport):
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_import", err.Error())
	case errors.Is(err, domain.ErrImportConflict):
		transport.WriteProblem(log, w, http.StatusConflict, "import_conflict", err.Error())
	default:
		log.ErrorContext(r.Context(), "metadata import failed", "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", "metadata import failed")
	}
}
