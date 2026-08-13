// Package httpapi provides the backoffice HTTP transport.
package httpapi

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authorization"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/docs"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/dataset"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/health"
	imports "github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/import"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadataimport"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/turtle"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter(database *db.PostgresDB) http.Handler {
	log.Info("creating HTTP handler")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/health", health.New(database).ServeHTTP)
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		if _, err := w.Write(docs.GetSpec()); err != nil {
			log.Error("failed to write openapi spec", "error", err)
		}
	})
	r.Get("/scalar", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scalar/", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	r.Handle("/scalar/*", http.StripPrefix("/scalar", http.FileServer(http.FS(docs.GetScalarUI()))))
	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(docs.GetUI()))))
	r.Route("/api/v1", func(r chi.Router) {
		authorizer := authorization.New(database.Authorization)
		datasets := dataset.New(datasetlifecycle.New(authorizer, database.Datasets))
		importsHandler := imports.New(metadataimport.New(turtle.Parser{}, authorizer, database.Imports))

		r.Post("/imports/turtle", importsHandler.Turtle)
		r.Post("/groups/{group_id}/datasets", datasets.Create)
		r.Get("/datasets/{id}/{version_id}", datasets.Get)
		r.Put("/datasets/{id}/versions/{version_id}/draft", datasets.ReplaceDraft)
		r.Post("/datasets/{id}/versions/{version_id}/submit", datasets.Submit)
		r.Post("/datasets/{id}/versions/{version_id}/withdraw", datasets.Withdraw)
		r.Post("/datasets/{id}/versions/{version_id}/reject", datasets.Reject)
		r.Post("/datasets/{id}/versions/{version_id}/reopen", datasets.Reopen)
		r.Post("/datasets/{id}/versions/{version_id}/publish", datasets.Publish)
		r.Post("/datasets/{id}/versions/{version_id}/unpublish", datasets.Unpublish)
		r.Post("/datasets/{id}/draft-from-publication", datasets.DraftFromPublication)
		r.Post("/datasets/{id}/draft-from-archive", datasets.DraftFromArchive)
		r.Delete("/datasets/{id}/versions/{version_id}/draft", datasets.Discard)
		r.Post("/groups/{group_id}/datasets/{id}/copy", datasets.Copy)
	})

	return observability.NewHTTPHandler(r, "/")
}
