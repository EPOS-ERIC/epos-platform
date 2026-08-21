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
	groupservice "github.com/EPOS-ERIC/epos-platform/services/backoffice/groups"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/dataset"
	groupsapi "github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/groups"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/health"
	identityapi "github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/identity"
	imports "github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/import"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/resource"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/softwareapplication"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/handlers/softwaresourcecode"
	identityservice "github.com/EPOS-ERIC/epos-platform/services/backoffice/identity"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadataimport"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/softwarelifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/turtle"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

// NewRouter builds the backoffice HTTP handler and routes.
// TODO: inject services inside of here, don't create them in the function
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

		importsHandler := imports.New(metadataimport.New(turtle.Parser{}, authorizer, database.Imports))

		r.Post("/imports/turtle", importsHandler.Turtle)

		identity := identityapi.New(identityservice.New(database.Identity, authorizer))

		r.Get("/users/{user_id}", identity.GetByID)
		r.Get("/users", identity.List)
		r.Put("/users/{user_id}/global-permissions/{permission}", identity.GrantGlobalPermission)
		r.Delete("/users/{user_id}/global-permissions/{permission}", identity.RevokeGlobalPermission)
		r.Put("/users/{user_id}/global-roles/{role}", identity.AssignGlobalRole)
		r.Delete("/users/{user_id}/global-roles/{role}", identity.RemoveGlobalRole)
		r.Post("/users/{user_id}/block", identity.BlockUser)
		r.Post("/users/{user_id}/unblock", identity.UnblockUser)
		r.Post("/internal/users", identity.Create)

		groups := groupsapi.New(groupservice.New(database.Groups, authorizer))
		groupAccess := groupsapi.NewAccess(groupservice.NewAccess(database.Groups, authorizer))

		r.Get("/groups", groups.List)
		r.Post("/groups", groups.Create)
		r.Get("/groups/{group_id}", groups.Get)
		r.Get("/groups/{group_id}/memberships", groupAccess.ListMemberships)
		r.Post("/groups/{group_id}/membership-requests", groupAccess.RequestMembership)
		r.Delete("/groups/{group_id}/membership-requests", groupAccess.WithdrawMembershipRequest)
		r.Patch("/groups/{group_id}/memberships/{user_id}", groupAccess.DecideMembership)
		r.Delete("/groups/{group_id}/memberships/{user_id}", groupAccess.RevokeMembership)
		r.Post("/groups/{group_id}/role-requests", groupAccess.RequestRole)
		r.Patch("/groups/{group_id}/role-requests/{request_id}", groupAccess.ApplyRoleRequestDecision)
		r.Delete("/groups/{group_id}/role-requests/{request_id}", groupAccess.WithdrawRoleRequest)
		r.Put("/groups/{group_id}/memberships/{user_id}/roles/{role}", groupAccess.AssignRole)
		r.Delete("/groups/{group_id}/memberships/{user_id}/roles/{role}", groupAccess.RemoveRole)
		r.Put("/groups/{group_id}/memberships/{user_id}/permissions/{permission}", groupAccess.GrantPermission)
		r.Delete("/groups/{group_id}/memberships/{user_id}/permissions/{permission}", groupAccess.RevokePermission)

		datasets := dataset.New(datasetlifecycle.New(authorizer, database.Datasets))
		applications := softwareapplication.New(softwarelifecycle.NewApplicationService(authorizer, database.SoftwareApplications))
		sourceCodes := softwaresourcecode.New(softwarelifecycle.NewSourceCodeService(authorizer, database.SoftwareSourceCodes))
		resources := resource.New(datasets, applications, sourceCodes)

		r.Post("/resources/{resource_type}", resources.Create)
		r.Get("/resources/{resource_type}/{resource_id}/versions/{version_id}", resources.Get)
		r.Put("/resources/{resource_type}/{resource_id}/versions/{version_id}/draft", resources.ReplaceDraft)
		r.Delete("/resources/{resource_type}/{resource_id}/versions/{version_id}/draft", resources.Discard)
		r.Post("/resources/{resource_type}/{resource_id}/versions/{version_id}/submit", resources.Submit)
		r.Post("/resources/{resource_type}/{resource_id}/versions/{version_id}/withdraw", resources.Withdraw)
		r.Post("/resources/{resource_type}/{resource_id}/versions/{version_id}/reject", resources.Reject)
		r.Post("/resources/{resource_type}/{resource_id}/versions/{version_id}/reopen", resources.Reopen)
		r.Post("/resources/{resource_type}/{resource_id}/versions/{version_id}/publish", resources.Publish)
		r.Post("/resources/{resource_type}/{resource_id}/versions/{version_id}/unpublish", resources.Unpublish)
		r.Post("/resources/{resource_type}/{resource_id}/drafts/from-publication", resources.DraftFromPublication)
		r.Post("/resources/{resource_type}/{resource_id}/drafts/from-archive", resources.DraftFromArchive)
		r.Post("/resources/{resource_type}/{resource_id}/copies", resources.Copy)
	})

	return observability.NewHTTPHandler(r, "/")
}
