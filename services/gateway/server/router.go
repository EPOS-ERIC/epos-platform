package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/canonical"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/docs"
	groupservice "github.com/EPOS-ERIC/epos-platform/services/gateway/groups"
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
	groupsHandler := handlers.NewGroups(groupservice.NewService(backofficeClient))
	importsHandler := handlers.NewImports(imports.NewService(backofficeClient))
	canonicalHandler := handlers.NewCanonical(canonical.NewService(backofficeClient))
	discoveryHandler := handlers.NewDiscovery(discovery.NewService(backofficeClient))

	r.Route("/v1", func(r chi.Router) {
		r.With(authMiddleware(authService, backofficeClient)).Post("/imports/turtle", importsHandler.Turtle)

		r.Route("/contact-points", func(r chi.Router) {
			r.Use(authMiddleware(authService, backofficeClient))
			r.Get("/", canonicalHandler.ListContactPoints)
			r.Post("/", canonicalHandler.CreateContactPoint)
			r.Get("/{contactPointId}", canonicalHandler.GetContactPoint)
			r.Put("/{contactPointId}", canonicalHandler.ReplaceContactPoint)
			r.Delete("/{contactPointId}", canonicalHandler.DeleteContactPoint)
		})

		r.Route("/organizations", func(r chi.Router) {
			r.Get("/", canonicalHandler.ListOrganizations)
			r.With(authMiddleware(authService, backofficeClient)).Post("/", canonicalHandler.CreateOrganization)
			r.Get("/{organizationId}", canonicalHandler.GetOrganization)
			r.With(authMiddleware(authService, backofficeClient)).Put("/{organizationId}", canonicalHandler.ReplaceOrganization)
			r.With(authMiddleware(authService, backofficeClient)).Delete("/{organizationId}", canonicalHandler.DeleteOrganization)
		})

		// discovery
		r.With(authMiddlewareOptional(authService, backofficeClient)).Get("/search", discoveryHandler.Search)

		r.Route("/users", func(r chi.Router) {
			r.Use(authMiddleware(authService, backofficeClient))
			r.Get("/", usersHandler.List)
			r.Get("/{id}", usersHandler.Get)
			r.Put("/{id}/global-permissions/{permission}", usersHandler.GrantGlobalPermission)
			r.Delete("/{id}/global-permissions/{permission}", usersHandler.RevokeGlobalPermission)
			r.Put("/{id}/global-roles/{role}", usersHandler.AssignGlobalRole)
			r.Delete("/{id}/global-roles/{role}", usersHandler.RemoveGlobalRole)
			r.Post("/{id}/block", usersHandler.BlockUser)
			r.Post("/{id}/unblock", usersHandler.UnblockUser)
		})

		r.Route("/groups", func(r chi.Router) {
			r.Get("/", groupsHandler.List)
			r.With(authMiddleware(authService, backofficeClient)).Post("/", groupsHandler.Create)
			r.Get("/{groupId}", groupsHandler.Get)
			r.With(authMiddleware(authService, backofficeClient)).Put("/{groupId}", groupsHandler.Replace)
			r.With(authMiddleware(authService, backofficeClient)).Delete("/{groupId}", groupsHandler.Delete)
			r.With(authMiddleware(authService, backofficeClient)).Get("/{groupId}/memberships", groupsHandler.ListMemberships)

			r.With(authMiddleware(authService, backofficeClient)).Post("/{groupId}/membership-requests", groupsHandler.RequestMembership)
			r.With(authMiddleware(authService, backofficeClient)).Delete("/{groupId}/membership-requests", groupsHandler.WithdrawMembershipRequest)
			r.With(authMiddleware(authService, backofficeClient)).Patch("/{groupId}/memberships/{userId}", groupsHandler.DecideMembership)
			r.With(authMiddleware(authService, backofficeClient)).Delete("/{groupId}/memberships/{userId}", groupsHandler.DeleteMembership)
			r.With(authMiddleware(authService, backofficeClient)).Put("/{groupId}/memberships/{userId}/roles/{role}", groupsHandler.AssignGroupRole)
			r.With(authMiddleware(authService, backofficeClient)).Delete("/{groupId}/memberships/{userId}/roles/{role}", groupsHandler.RemoveGroupRole)
			r.With(authMiddleware(authService, backofficeClient)).Put("/{groupId}/memberships/{userId}/permissions/{permission}", groupsHandler.GrantGroupPermission)
			r.With(authMiddleware(authService, backofficeClient)).Delete("/{groupId}/memberships/{userId}/permissions/{permission}", groupsHandler.RevokeGroupPermission)
			r.With(authMiddleware(authService, backofficeClient)).Post("/{groupId}/role-requests", groupsHandler.RequestRole)
			r.With(authMiddleware(authService, backofficeClient)).Patch("/{groupId}/role-requests/{requestId}", groupsHandler.ApplyRoleRequestDecision)
			r.With(authMiddleware(authService, backofficeClient)).Delete("/{groupId}/role-requests/{requestId}", groupsHandler.WithdrawRoleRequest)
			r.With(authMiddleware(authService, backofficeClient)).Post("/{groupId}/resources", lifecycleHandler.Create)
			r.With(authMiddleware(authService, backofficeClient)).Post("/{groupId}/resources/{resourceType}/{id}/copy", lifecycleHandler.Copy)

			r.Route("/{groupId}/categories", func(r chi.Router) {
				r.Get("/", groupsHandler.ListCategories)
				r.With(authMiddleware(authService, backofficeClient)).Post("/", groupsHandler.CreateCategory)
				r.Get("/{categoryId}", groupsHandler.GetCategory)
				r.With(authMiddleware(authService, backofficeClient)).Put("/{categoryId}", groupsHandler.ReplaceCategory)
				r.With(authMiddleware(authService, backofficeClient)).Delete("/{categoryId}", groupsHandler.DeleteCategory)
			})
		})

		r.Route("/resources", func(r chi.Router) {
			r.With(authMiddlewareOptional(authService, backofficeClient)).Get("/{resourceType}/{id}/versions/{versionId}", lifecycleHandler.Get)
			r.With(authMiddleware(authService, backofficeClient)).Put("/{id}/versions/{versionId}/draft", lifecycleHandler.ReplaceDraft)
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
