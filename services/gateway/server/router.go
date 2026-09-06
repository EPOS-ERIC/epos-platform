package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/middleware"
	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/docs"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
	"github.com/go-chi/chi/v5"
)

func NewRouter(
	log *slog.Logger,
	identityService *identity.Service,
	metadataLifecycleClient *clients.MetadataLifecycle,
	authService *auth.Service,
	corsAllowOrigin string,
) http.Handler {
	log = log.With("component", "http")
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	if corsAllowOrigin != "" {
		r.Use(middleware.CORSAllowOrigin(corsAllowOrigin))
	}
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recoverer(log))

	// TODO: maybe we can have this endpoint be outside of the logger middleware so that we can siplify the middleware to not manually exclude health logs
	r.Get("/health", health)

	// TODO: modify the docs library to have this shared, like returning the routes directly
	r.Get("/openapi.yaml", serveOpenAPISpec(log))
	r.Get("/scalar", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scalar/", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Handle("/scalar/*", http.StripPrefix("/scalar", http.FileServer(http.FS(docs.GetScalarUI()))))
	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(docs.GetUI()))))

	generatedHandler := handlers.NewOgenHandler(
		identityService,
		metadataLifecycleClient,
	)
	generatedServer, err := gatewayapi.NewServer(
		generatedHandler,
		handlers.NewSecurityHandler(authService, identityService),
		gatewayapi.WithPathPrefix("/v1"),
		gatewayapi.WithErrorHandler(handlers.NewErrorHandler(log)),
	)
	if err != nil {
		panic(fmt.Sprintf("create generated gateway server: %v", err))
	}
	r.Mount("/v1", requestBodyLimit(generatedServer))

	return observability.NewHTTPHandler(r, "/")
}

func requestBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		next.ServeHTTP(w, r)
	})
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
