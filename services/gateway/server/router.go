package server

import (
	"log/slog"
	"net/http"

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

func NewRouter(
	log *slog.Logger,
	lifecycleService *lifecycle.Service,
	identityService *identity.Service,
	metadataLifecycleClient *clients.MetadataLifecycle,
	authService *auth.Service,
	corsAllowOrigin string,
) http.Handler {
	handlerLog := log
	log = log.With("component", "http")
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	if corsAllowOrigin != "" {
		r.Use(middleware.CORSAllowOrigin(corsAllowOrigin))
	}
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recoverer(log))

	// TODO: maybe we can have this endpoint be outside of the logger middleware so that we can siplify the middleware to not manually exclude helath logs
	r.Get("/health", health)
	r.Get("/openapi.yaml", serveOpenAPISpec(log))
	r.Get("/scalar", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scalar/", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Handle("/scalar/*", http.StripPrefix("/scalar", http.FileServer(http.FS(docs.GetScalarUI()))))
	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(docs.GetUI()))))

	lifecycleHandler := handlers.NewLifecycle(handlerLog, lifecycleService)
	usersHandler := handlers.NewUsers(handlerLog, identityService, metadataLifecycleClient.Proxy)
	groupsHandler := handlers.NewGroups(handlerLog, groupservice.NewService(metadataLifecycleClient))
	importsHandler := handlers.NewImports(handlerLog, imports.NewService(metadataLifecycleClient))
	canonicalHandler := handlers.NewCanonical(handlerLog, canonical.NewService(metadataLifecycleClient))
	discoveryHandler := handlers.NewDiscovery(handlerLog, discovery.NewService(metadataLifecycleClient))

	r.Route("/v1", func(r chi.Router) {
		r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/imports/turtle", importsHandler.Turtle)

		r.Route("/contact-points", func(r chi.Router) {
			r.Use(authMiddleware(log, authService, metadataLifecycleClient))
			r.Get("/", canonicalHandler.ListContactPoints)
			r.Post("/", canonicalHandler.CreateContactPoint)
			r.Get("/{contactPointId}", canonicalHandler.GetContactPoint)
			r.Put("/{contactPointId}", canonicalHandler.ReplaceContactPoint)
			r.Delete("/{contactPointId}", canonicalHandler.DeleteContactPoint)
		})

		r.Route("/organizations", func(r chi.Router) {
			r.Get("/", canonicalHandler.ListOrganizations)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/", canonicalHandler.CreateOrganization)
			r.Get("/{organizationId}", canonicalHandler.GetOrganization)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Put("/{organizationId}", canonicalHandler.ReplaceOrganization)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{organizationId}", canonicalHandler.DeleteOrganization)
		})

		// discovery
		r.With(authMiddlewareOptional(log, authService, metadataLifecycleClient)).Get("/search", discoveryHandler.Search)

		r.Route("/users", func(r chi.Router) {
			r.Use(authMiddleware(log, authService, metadataLifecycleClient))
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
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/", groupsHandler.Create)
			r.Get("/{groupId}", groupsHandler.Get)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Put("/{groupId}", groupsHandler.Replace)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{groupId}", groupsHandler.Delete)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Get("/{groupId}/memberships", groupsHandler.ListMemberships)

			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/{groupId}/membership-requests", groupsHandler.RequestMembership)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{groupId}/membership-requests", groupsHandler.WithdrawMembershipRequest)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Patch("/{groupId}/memberships/{userId}", groupsHandler.DecideMembership)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{groupId}/memberships/{userId}", groupsHandler.DeleteMembership)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Put("/{groupId}/memberships/{userId}/roles/{role}", groupsHandler.AssignGroupRole)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{groupId}/memberships/{userId}/roles/{role}", groupsHandler.RemoveGroupRole)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Put("/{groupId}/memberships/{userId}/permissions/{permission}", groupsHandler.GrantGroupPermission)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{groupId}/memberships/{userId}/permissions/{permission}", groupsHandler.RevokeGroupPermission)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/{groupId}/role-requests", groupsHandler.RequestRole)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Patch("/{groupId}/role-requests/{requestId}", groupsHandler.ApplyRoleRequestDecision)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{groupId}/role-requests/{requestId}", groupsHandler.WithdrawRoleRequest)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/{groupId}/resources", lifecycleHandler.Create)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/{groupId}/resources/{resourceType}/{id}/copy", lifecycleHandler.Copy)

			r.Route("/{groupId}/categories", func(r chi.Router) {
				r.Get("/", groupsHandler.ListCategories)
				r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/", groupsHandler.CreateCategory)
				r.Get("/{categoryId}", groupsHandler.GetCategory)
				r.With(authMiddleware(log, authService, metadataLifecycleClient)).Put("/{categoryId}", groupsHandler.ReplaceCategory)
				r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{categoryId}", groupsHandler.DeleteCategory)
			})
		})

		r.Route("/resources", func(r chi.Router) {
			r.With(authMiddlewareOptional(log, authService, metadataLifecycleClient)).Get("/{resourceType}/{id}/versions/{versionId}", lifecycleHandler.Get)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Put("/{id}/versions/{versionId}/draft", lifecycleHandler.ReplaceDraft)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Delete("/{resourceType}/{id}/versions/{versionId}/draft", lifecycleHandler.Discard)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/{resourceType}/{id}/versions/{versionId}/{action:submit|withdraw|reject|reopen|publish|unpublish}", lifecycleHandler.Action)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/{resourceType}/{id}/draft-from-publication", lifecycleHandler.DraftFromPublication)
			r.With(authMiddleware(log, authService, metadataLifecycleClient)).Post("/{resourceType}/{id}/draft-from-archive", lifecycleHandler.DraftFromArchive)
		})
	})

	return observability.NewHTTPHandler(r, "/")
}

func serveOpenAPISpec(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		if _, err := w.Write(docs.GetSpec()); err != nil {
			log.Error("failed to write openapi spec", "error", err)
		}
	}
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
