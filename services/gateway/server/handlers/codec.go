package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
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
