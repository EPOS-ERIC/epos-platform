package server

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/server/handlers"
)

func NewRouter(openAPIPath string) http.Handler {
	mux := http.NewServeMux()
	groupsHandler := handlers.NewGroupsHandler(repository.NewGroupMemoryRepository())

	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("POST /api/v2/authorize", handlers.Authorize)
	mux.HandleFunc("GET /openapi.yaml", handlers.OpenAPISpec(openAPIPath))
	mux.HandleFunc("GET /swagger", handlers.SwaggerRedirect)
	mux.HandleFunc("GET /swagger/", handlers.SwaggerUI)
	mux.HandleFunc("GET /api/v2/groups", groupsHandler.List)
	mux.HandleFunc("POST /api/v2/groups", groupsHandler.Create)
	mux.HandleFunc("GET /api/v2/groups/{groupId}", groupsHandler.GetByID)
	mux.HandleFunc("PUT /api/v2/groups/{groupId}", groupsHandler.Replace)
	mux.HandleFunc("PATCH /api/v2/groups/{groupId}", groupsHandler.Patch)
	mux.HandleFunc("DELETE /api/v2/groups/{groupId}", groupsHandler.Delete)

	return mux
}
