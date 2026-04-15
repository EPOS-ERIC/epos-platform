package routes

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayserver"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/middleware"
	gatewayswagger "github.com/EPOS-ERIC/epos-platform/services/gateway/swagger"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter() chi.Router {
	log.Info("creating HTTP router")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/openapi.yaml", serveOpenAPISpec)
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(gatewayswagger.GetUI()))))

	service := newOpenAPIService()
	controllers := []gatewayserver.Router{
		gatewayserver.NewDiscoveryAPIController(service),
		gatewayserver.NewResourcesAPIController(service),
	}

	for _, controller := range controllers {
		for _, route := range controller.OrderedRoutes() {
			r.With(middleware.JSONMiddleware).Method(route.Method, route.Pattern, route.HandlerFunc)
		}
	}

	return r
}

func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	if _, err := w.Write(gatewayswagger.GetSpec()); err != nil {
		log.Error("failed to write openapi spec", "error", err)
	}
}
