package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/swagger"
)

type Handler struct {
	repo       repository.RetrievalRepository
	client     retrievalClient
	healthPing healthPinger
	log        *slog.Logger
}

type healthPinger interface {
	PingContext(ctx context.Context) error
}

type retrievalClient interface {
	Execute(rctx context.Context, req domain.ValidatedRequest) (retriever.Result, error)
}

func NewRouter(log *slog.Logger, repo repository.RetrievalRepository, client retrievalClient, healthPing healthPinger) http.Handler {
	h := &Handler{
		repo:       repo,
		client:     client,
		healthPing: healthPing,
		log:        log.With("component", "data-retriever-handler"),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /openapi.yaml", h.openAPISpec)
	mux.HandleFunc("GET /swagger", swaggerRedirect)
	mux.Handle("/swagger/", http.StripPrefix("/swagger", http.FileServer(http.FS(swagger.GetUI()))))
	mux.HandleFunc("POST /api/v2/retrieve", h.retrieve)
	mux.HandleFunc("GET /api/v2/retrieve/{id}", h.getByID)
	mux.HandleFunc("GET /api/v2/search", h.search)
	return observability.NewHTTPHandler(mux, "/")
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if h.healthPing == nil {
		writeJSON(h.log, w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := h.healthPing.PingContext(ctx); err != nil {
		h.log.ErrorContext(r.Context(), "health check failed", "component", "database", "error", err)
		writeJSON(h.log, w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "reason": "database unreachable"})
		return
	}

	writeJSON(h.log, w, http.StatusOK, map[string]string{"status": "ok"})
}

func swaggerRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
}

func (h *Handler) openAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	if _, err := w.Write(swagger.GetSpec()); err != nil {
		h.log.ErrorContext(r.Context(), "failed to write openapi spec", "error", err)
	}
}

func (h *Handler) retrieve(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()

	var req domain.RetrieveRequest
	if err := decoder.Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid retrieve request body", "error", err)
		writeError(h.log, w, http.StatusBadRequest, err.Error())
		return
	}

	validated, err := req.Validate()
	if err != nil {
		h.log.WarnContext(r.Context(), "invalid retrieve request", "source_id", req.SourceID, "error", err)
		writeError(h.log, w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.client.Execute(r.Context(), validated)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed retrieval upstream call", "source_id", validated.SourceID, "method", validated.Method, "url", validated.GeneratedURL, "error", err)
		writeError(h.log, w, http.StatusBadGateway, err.Error())
		return
	}

	record, err := h.repo.Create(r.Context(), validated, result)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed storing retrieval record", "source_id", validated.SourceID, "method", validated.Method, "url", validated.GeneratedURL, "error", err)
		writeError(h.log, w, http.StatusInternalServerError, err.Error())
		return
	}

	if result.IsBinary {
		http.Redirect(w, r, validated.GeneratedURL, http.StatusSeeOther)
		return
	}

	if result.ContentType != "" {
		w.Header().Set("Content-Type", result.ContentType)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.Header().Set("X-Retrieval-Id", strconv.FormatInt(record.ID, 10))
	w.WriteHeader(result.StatusCode)
	_, _ = w.Write(result.Payload)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.log.WarnContext(r.Context(), "invalid retrieval id", "id", r.PathValue("id"), "error", err)
		writeError(h.log, w, http.StatusBadRequest, "invalid id")
		return
	}

	record, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.log.WarnContext(r.Context(), "retrieval record not found", "id", id)
			writeError(h.log, w, http.StatusNotFound, "record not found")
			return
		}
		h.log.ErrorContext(r.Context(), "failed reading retrieval by id", "id", id, "error", err)
		writeError(h.log, w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(h.log, w, http.StatusOK, record)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		h.log.WarnContext(r.Context(), "missing search query")
		writeError(h.log, w, http.StatusBadRequest, "q is required")
		return
	}

	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	results, err := h.repo.Search(
		r.Context(),
		q,
		strings.TrimSpace(r.URL.Query().Get("source_id")),
		strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("method"))),
		page,
		pageSize,
	)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed searching retrieval records", "query", q, "error", err)
		writeError(h.log, w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(h.log, w, http.StatusOK, map[string]any{"items": results, "total": len(results)})
}

func parsePositiveInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func writeJSON(log *slog.Logger, w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error("failed to encode JSON response", "status", code, "error", err)
	}
}

func writeError(log *slog.Logger, w http.ResponseWriter, code int, message string) {
	writeJSON(log, w, code, map[string]string{"error": message})
}
