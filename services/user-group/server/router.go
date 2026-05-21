package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/server/handlers"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/swagger"
)

func NewRouter(usersRepo repository.UserRepository, groupsRepo repository.GroupRepository) http.Handler {
	mux := http.NewServeMux()
	groupsHandler := handlers.NewGroupsHandler(groupsRepo)
	usersHandler := handlers.NewUsersHandler(usersRepo)

	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("POST /api/v2/authorize", handlers.Authorize)
	mux.HandleFunc("GET /openapi.yaml", handlers.OpenAPISpec)
	mux.HandleFunc("GET /swagger", handlers.SwaggerRedirect)
	mux.Handle("/swagger/", http.StripPrefix("/swagger", http.FileServer(http.FS(swagger.GetUI()))))
	mux.HandleFunc("GET /api/v2/groups", groupsHandler.List)
	mux.HandleFunc("POST /api/v2/groups", groupsHandler.Create)
	mux.HandleFunc("GET /api/v2/groups/{groupId}", groupsHandler.GetByID)
	mux.HandleFunc("PUT /api/v2/groups/{groupId}", groupsHandler.Replace)
	mux.HandleFunc("PATCH /api/v2/groups/{groupId}", groupsHandler.Patch)
	mux.HandleFunc("DELETE /api/v2/groups/{groupId}", groupsHandler.Delete)
	mux.HandleFunc("GET /api/v2/users", usersHandler.List)
	mux.HandleFunc("POST /api/v2/users", usersHandler.Create)
	mux.HandleFunc("GET /api/v2/users/by-auth/{authIdentifier}", usersHandler.GetByAuthIdentifier)
	mux.HandleFunc("GET /api/v2/users/{userId}", usersHandler.GetByID)
	mux.HandleFunc("PUT /api/v2/users/{userId}", usersHandler.Replace)
	mux.HandleFunc("PATCH /api/v2/users/{userId}", usersHandler.Patch)
	mux.HandleFunc("DELETE /api/v2/users/{userId}", usersHandler.Delete)

	return mux
}
