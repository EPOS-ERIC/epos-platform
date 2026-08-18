// Package httpapi provides the backoffice HTTP transport.
package httpapi

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/docs"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/dataset"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/health"
	imports "github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/import"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/resource"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/softwareapplication"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/softwaresourcecode"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadataimport"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/softwarelifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/turtle"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

// NewRouter builds the backoffice HTTP handler and routes.
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
		authorizer := authz.New(database.Authorization)
		datasets := dataset.New(datasetlifecycle.New(authorizer, database.Datasets))
		applications := softwareapplication.New(softwarelifecycle.NewApplicationService(authorizer, database.SoftwareApplications))
		sourceCodes := softwaresourcecode.New(softwarelifecycle.NewSourceCodeService(authorizer, database.SoftwareSourceCodes))
		resources := resource.New(datasets, applications, sourceCodes)
		importsHandler := imports.New(metadataimport.New(turtle.Parser{}, authorizer, database.Imports))

		r.Post("/imports/turtle", importsHandler.Turtle)
		r.Post("/groups/{group_id}/resources/{resource_type}", resources.Create)
		r.Get("/resources/{resource_type}/{id}/versions/{version_id}", resources.Get)
		r.Put("/resources/{resource_type}/{id}/versions/{version_id}/draft", resources.ReplaceDraft)
		r.Delete("/resources/{resource_type}/{id}/versions/{version_id}/draft", resources.Discard)
		r.Post("/resources/{resource_type}/{id}/versions/{version_id}/submit", resources.Submit)
		r.Post("/resources/{resource_type}/{id}/versions/{version_id}/withdraw", resources.Withdraw)
		r.Post("/resources/{resource_type}/{id}/versions/{version_id}/reject", resources.Reject)
		r.Post("/resources/{resource_type}/{id}/versions/{version_id}/reopen", resources.Reopen)
		r.Post("/resources/{resource_type}/{id}/versions/{version_id}/publish", resources.Publish)
		r.Post("/resources/{resource_type}/{id}/versions/{version_id}/unpublish", resources.Unpublish)
		r.Post("/resources/{resource_type}/{id}/draft-from-publication", resources.DraftFromPublication)
		r.Post("/resources/{resource_type}/{id}/draft-from-archive", resources.DraftFromArchive)
		r.Post("/groups/{group_id}/resources/{resource_type}/{id}/copy", resources.Copy)
	})

	return observability.NewHTTPHandler(r, "/")
}
