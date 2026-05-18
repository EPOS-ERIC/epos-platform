package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/middleware"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/swagger"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter(discoveryService *discovery.Service, resourcesService *resources.Service) http.Handler {
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/openapi.yaml", serveOpenAPISpec)
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(swagger.GetUI()))))

	discoveryHandler := handlers.NewDiscovery(discoveryService)
	resourcesHandler := handlers.NewResources(resourcesService)

	r.Route("/v1", func(r chi.Router) {
		// discovery
		r.Get("/search", discoveryHandler.SearchMetadata)
		r.Get("/concept", discoveryHandler.GetConceptTree)

		// resources
		r.Get("/dataset/{id}", resourcesHandler.GetDatasetByID)
		r.Get("/distribution/{id}", resourcesHandler.GetDistributionByID)
		r.Get("/software-application/{id}", resourcesHandler.GetSoftwareApplicationByID)
		r.Get("/software-source-code/{id}", resourcesHandler.GetSoftwareSourceCodeByID)
		r.Get("/concept/{id}", resourcesHandler.GetConceptByID)
	})

	return r
}

func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	if _, err := w.Write(swagger.GetSpec()); err != nil {
		log.Error("failed to write openapi spec", "error", err)
	}
}
