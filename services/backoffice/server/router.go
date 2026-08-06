package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/docs"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/server/handlers"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/service"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter(database *db.PostgresDB) http.Handler {
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handlers.NewHealth(database).ServeHTTP)

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

	usersSvc := service.NewUsers(database)
	groupsSvc := service.NewGroups(database)
	membershipsSvc := service.NewMemberships(database)
	friendshipsSvc := service.NewFriendships(database)
	authorizerSvc := service.NewAuthorizer(database)

	usersHandler := handlers.NewUsers(usersSvc)
	groupsHandler := handlers.NewGroups(groupsSvc)
	membershipsHandler := handlers.NewMemberships(membershipsSvc)
	friendshipsHandler := handlers.NewFriendships(friendshipsSvc)
	authorizeHandler := handlers.NewAuthorize(authorizerSvc)

	r.Route("/api/v2", func(r chi.Router) {
		r.Post("/authorize", authorizeHandler.Authorize)

		r.Route("/groups", func(r chi.Router) {
			r.Get("/", groupsHandler.List)
			r.Post("/", groupsHandler.Create)
			r.Route("/{groupId}", func(r chi.Router) {
				r.Get("/", groupsHandler.GetByID)
				r.Put("/", groupsHandler.Replace)
				r.Patch("/", groupsHandler.Patch)
				r.Delete("/", groupsHandler.Delete)
				r.Get("/memberships", membershipsHandler.ListByGroup)
				r.Put("/memberships/{userId}", membershipsHandler.Upsert)
				r.Delete("/memberships/{userId}", membershipsHandler.Delete)
			})
		})

		r.Route("/users", func(r chi.Router) {
			r.Get("/", usersHandler.List)
			r.Post("/", usersHandler.Create)
			r.Get("/by-auth/{authIdentifier}", usersHandler.GetByAuthIdentifier)
			r.Route("/{userId}", func(r chi.Router) {
				r.Get("/", usersHandler.GetByID)
				r.Put("/", usersHandler.Replace)
				r.Patch("/", usersHandler.Patch)
				r.Delete("/", usersHandler.Delete)
				r.Post("/friends/{friendUserId}", friendshipsHandler.Create)
				r.Delete("/friends/{friendUserId}", friendshipsHandler.Delete)
			})
		})
	})

	return observability.NewHTTPHandler(r, "/")
}
