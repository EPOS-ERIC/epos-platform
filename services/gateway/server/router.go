package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/docs"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/lifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/usergroup"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter(
	discoveryService *discovery.Service,
	resourcesService *resources.Service,
	lifecycleService *lifecycle.Service,
	usergroupService *usergroup.Service,
	corsAllowOrigin string,
) http.Handler {
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	if corsAllowOrigin != "" {
		r.Use(middleware.CORSAllowOrigin(corsAllowOrigin))
	}
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/openapi.yaml", serveOpenAPISpec)
	r.Get("/scalar", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scalar/", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Handle("/scalar/*", http.StripPrefix("/scalar", http.FileServer(http.FS(docs.GetScalarUI()))))
	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(docs.GetUI()))))

	discoveryHandler := handlers.NewDiscovery(discoveryService)
	resourcesHandler := handlers.NewResources(resourcesService)
	lifecycleHandler := handlers.NewLifecycle(lifecycleService)
	usersHandler := handlers.NewUsers(usergroupService)

	r.Route("/v1", func(r chi.Router) {
		// discovery
		r.Get("/search", discoveryHandler.SearchMetadata)
		r.Get("/concept", discoveryHandler.GetConceptTree)

		r.Route("/users", func(r chi.Router) {
			r.Get("/{id}", usersHandler.GetUser)
		})

		r.Route("/resources", func(r chi.Router) {
			r.Get("/{resourceType}/{id}", resourcesHandler.GetResourceByID)
			// lifecycle TODO
			r.Post("/{resourceType}/{id}/draft", lifecycleHandler.DraftResource)
		})
	})

	return observability.NewHTTPHandler(r, "/")
}

func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	if _, err := w.Write(docs.GetSpec()); err != nil {
		log.Error("failed to write openapi spec", "error", err)
	}
}
