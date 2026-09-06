// Package httpapi provides the metadata lifecycle HTTP transport.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/docs"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/categories"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/health"
	imports "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/import"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/organizations"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/resource"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/handlers/users"
	"github.com/go-chi/chi/v5"
)

type Router struct {
	Imports       *imports.Handler
	Users         *users.Handler
	Categories    *categories.Handler
	ContactPoints *contactpoints.Handler
	Organizations *organizations.Handler
	Resources     *resource.Handler
	Health        *health.Handler

	Log *slog.Logger
}

// NewRouter builds the metadata lifecycle HTTP handler and routes.
// TODO: inject services inside of here, don't create them in the function
func NewRouter(router Router) http.Handler {
	log := router.Log.With("component", "routes")
	log.Info("creating HTTP handler")

	r := chi.NewRouter()
	r.Use(auditMiddleware(router.Log))
	r.Use(middleware.Logger(router.Log))
	r.Use(middleware.Recoverer(router.Log))
	r.Get("/health", router.Health.ServeHTTP)
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
		// TODO: group the endpoints better with r.Route

		r.Post("/imports/turtle", router.Imports.Turtle)

		r.Get("/users/{user_id}", router.Users.GetByID)
		r.Post("/internal/users", router.Users.Create)

		r.Post("/groups/{group_id}/categories", router.Categories.Create)
		r.Put("/groups/{group_id}/categories/{category_id}", router.Categories.Replace)
		r.Delete("/groups/{group_id}/categories/{category_id}", router.Categories.Delete)
	})

	return observability.NewHTTPHandler(r, "/")
}
