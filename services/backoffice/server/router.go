package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/docs"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/server/handlers"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter(database *db.PostgresDB) http.Handler {
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// TODO: maybe we can have this endpoint be outside of the logger middleware so that we can siplify the middleware to not manually exclude helath logs
	r.Get("/health", handlers.NewHealth(database).ServeHTTP)
	r.Get("/openapi.yaml", serveOpenAPISpec)
	r.Get("/scalar", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scalar/", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Handle("/scalar/*", http.StripPrefix("/scalar", http.FileServer(http.FS(docs.GetScalarUI()))))
	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(docs.GetUI()))))

	return observability.NewHTTPHandler(r, "/")
}

func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	if _, err := w.Write(docs.GetSpec()); err != nil {
		log.Error("failed to write openapi spec", "error", err)
	}
}
