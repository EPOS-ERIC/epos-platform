package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/swagger"
)

type Handler struct {
	repo   repository.RetrievalRepository
	client retrievalClient
}

type retrievalClient interface {
	Execute(rctx context.Context, req domain.ValidatedRequest) (retriever.Result, error)
}

func NewRouter(repo repository.RetrievalRepository, client retrievalClient) http.Handler {
	h := &Handler{repo: repo, client: client}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /openapi.yaml", openAPISpec)
	mux.HandleFunc("GET /swagger", swaggerRedirect)
	mux.Handle("/swagger/", http.StripPrefix("/swagger", http.FileServer(http.FS(swagger.GetUI()))))
	mux.HandleFunc("POST /api/v2/retrieve", h.retrieve)
	mux.HandleFunc("GET /api/v2/retrieve/{id}", h.getByID)
	mux.HandleFunc("GET /api/v2/search", h.search)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func swaggerRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
}

func openAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(swagger.GetSpec())
}

func (h *Handler) retrieve(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()

	var req domain.RetrieveRequest
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	validated, err := req.Validate()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.client.Execute(r.Context(), validated)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	record, err := h.repo.Create(r.Context(), validated, result)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	record, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "record not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "q is required")
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
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": results, "total": len(results)})
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

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}
