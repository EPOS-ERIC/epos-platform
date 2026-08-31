package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/google/uuid"
)

func WriteError(log *slog.Logger, ctx context.Context, w http.ResponseWriter, status int, code, message string, internalErr error) {
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

func decodeJSON(log *slog.Logger, r *http.Request, w http.ResponseWriter, value any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		WriteError(log, r.Context(), w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", err)
		return false
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		if mbErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			WriteError(log, r.Context(), w, http.StatusRequestEntityTooLarge, "request_too_large",
				fmt.Sprintf("request body exceeds %d bytes", mbErr.Limit), err)
			return false
		}
		WriteError(log, r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid JSON request body", err)
		return false
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		WriteError(log, r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid JSON request body", err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		WriteError(log, r.Context(), w, http.StatusBadRequest, "invalid_request", "request body must contain one JSON value", err)
		return false
	}

	return true
}

func validUUID(value string) bool {
	// TODO: parse don't validate
	_, err := uuid.Parse(value)
	return err == nil
}
