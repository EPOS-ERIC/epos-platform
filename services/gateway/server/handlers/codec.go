package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/google/uuid"
)

var log = logging.Get("handlers")

func WriteError(ctx context.Context, w http.ResponseWriter, status int, code, message string, internalErr error) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)

	problem := api.Problem{Title: code, Status: status, Detail: message, DebugDetail: ""}
	if internalErr != nil && log.Enabled(ctx, slog.LevelDebug) {
		problem.DebugDetail = internalErr.Error()
	}
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		log.ErrorContext(ctx, "failed to encode problem response", "status", status, "error", err)
	}
}

func decodeJSON(r *http.Request, w http.ResponseWriter, value any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		WriteError(r.Context(), w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", err)
		return false
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid JSON request body", err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "request body must contain one JSON value", err)
		return false
	}

	return true
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
