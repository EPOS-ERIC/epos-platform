package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/docs"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter() http.Handler {
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

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

	// discoveryHandler := handlers.NewDiscovery(discoveryService)
	// resourcesHandler := handlers.NewResources(resourcesService)

	// r.Route("/v1", func(r chi.Router) {
	// 	// discovery
	// 	r.Get("/search", discoveryHandler.SearchMetadata)
	// 	r.Get("/concept", discoveryHandler.GetConceptTree)
	//
	// 	// resources
	// 	r.Get("/dataset/{id}", resourcesHandler.GetDatasetByID)
	// 	r.Get("/distribution/{id}", resourcesHandler.GetDistributionByID)
	// 	r.Get("/softwareapplication/{id}", resourcesHandler.GetSoftwareApplicationByID)
	// 	r.Get("/softwaresourcecode/{id}", resourcesHandler.GetSoftwareSourceCodeByID)
	// 	r.Get("/webservice/{id}", resourcesHandler.GetWebserviceByID)
	// 	r.Get("/organization/{id}", resourcesHandler.GetOrganizationByID)
	// 	r.Get("/concept/{id}", resourcesHandler.GetConceptByID)
	// 	r.Get("/conceptscheme/{id}", resourcesHandler.GetConceptSchemeByID)
	// })

	return observability.NewHTTPHandler(r, "/")
}
