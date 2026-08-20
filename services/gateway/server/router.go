package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/docs"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/imports"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/lifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter(
	lifecycleService *lifecycle.Service,
	identityService *identity.Service,
	backofficeClient *clients.Backoffice,
	authService *auth.Service,
	corsAllowOrigin string,
) http.Handler {
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	if corsAllowOrigin != "" {
		r.Use(middleware.CORSAllowOrigin(corsAllowOrigin))
	}
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// TODO: maybe we can have this endpoint be outside of the logger middleware so that we can siplify the middleware to not manually exclude helath logs
	r.Get("/health", health)
	r.Get("/openapi.yaml", serveOpenAPISpec)
	r.Get("/scalar", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scalar/", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Handle("/scalar/*", http.StripPrefix("/scalar", http.FileServer(http.FS(docs.GetScalarUI()))))
	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(docs.GetUI()))))

	lifecycleHandler := handlers.NewLifecycle(lifecycleService)
	usersHandler := handlers.NewUsers(identityService)
	importsHandler := handlers.NewImports(imports.NewService(backofficeClient))

	r.Route("/v1", func(r chi.Router) {
		r.With(authMiddleware(authService, backofficeClient)).Post("/imports/turtle", importsHandler.Turtle)
		// discovery
		r.With(authMiddlewareOptional(authService, backofficeClient)).Get("/search", handlers.NotImplemented)
		r.Get("/concept", handlers.NotImplemented)

		r.Route("/users", func(r chi.Router) {
			r.Use(authMiddleware(authService, backofficeClient))
			r.Get("/", usersHandler.List)
			r.Get("/{id}", usersHandler.Get)
		})

		r.Route("/groups", func(r chi.Router) {
			r.Use(authMiddleware(authService, backofficeClient))
			r.Get("/", handlers.NotImplemented)
			r.Get("/{groupId}", handlers.NotImplemented)
			r.Get("/{groupId}/memberships", handlers.NotImplemented)
			r.Delete("/{groupId}", handlers.NotImplemented)
			r.Delete("/{groupId}/memberships/{userId}", handlers.NotImplemented)
			r.Post("/{groupId}/memberships", handlers.NotImplemented)
			r.Post("/", handlers.NotImplemented)
			r.Post("/{groupId}/resources/{resourceType}", lifecycleHandler.Create)
			r.Post("/{groupId}/resources/{resourceType}/{id}/copy", lifecycleHandler.Copy)
		})

		r.Route("/resources", func(r chi.Router) {
			r.With(authMiddlewareOptional(authService, backofficeClient)).Get("/{resourceType}/{id}/versions/{versionId}", lifecycleHandler.Get)
			r.With(authMiddleware(authService, backofficeClient)).Put("/{resourceType}/{id}/versions/{versionId}/draft", lifecycleHandler.ReplaceDraft)
			r.With(authMiddleware(authService, backofficeClient)).Delete("/{resourceType}/{id}/versions/{versionId}/draft", lifecycleHandler.Discard)
			r.With(authMiddleware(authService, backofficeClient)).Post("/{resourceType}/{id}/versions/{versionId}/{action}", lifecycleHandler.Action)
			r.With(authMiddleware(authService, backofficeClient)).Post("/{resourceType}/{id}/draft-from-publication", lifecycleHandler.DraftFromPublication)
			r.With(authMiddleware(authService, backofficeClient)).Post("/{resourceType}/{id}/draft-from-archive", lifecycleHandler.DraftFromArchive)
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

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
