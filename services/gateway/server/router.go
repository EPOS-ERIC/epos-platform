package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/observability"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/auth"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/contactrequest"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/converter"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/distributionaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/docs"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/membership"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resource"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
	sharingservice "github.com/EPOS-ERIC/epos-platform/services/gateway/sharing"
	"github.com/go-chi/chi/v5"
)

func NewRouter(
	log *slog.Logger,
	identityService *identity.Service,
	membershipService *membership.Service,
	metadataLifecycleClient *clients.MetadataLifecycle,
	contactRequestSender contactrequest.EmailSender,
	dataRetrieverClient *clients.DataRetriever,
	converterClient *clients.Converter,
	sharingService *sharingservice.Service,
	authService *auth.Service,
	corsAllowOrigin string,
	requestTimeout time.Duration,
) http.Handler {
	log = log.With("component", "http")
	log.Info("creating HTTP handler")

	r := chi.NewRouter()

	if corsAllowOrigin != "" {
		r.Use(corsMiddleware(corsAllowOrigin))
	}
	r.Use(requestLogger(log))
	r.Use(recoverer(log))

	// TODO: maybe we can have this endpoint be outside of the logger middleware so that we can siplify the middleware to not manually exclude health logs
	r.Get("/health", health)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scalar/", http.StatusFound)
	})
	r.Get("/openapi.yaml", serveOpenAPISpec(log))
	r.Get("/scalar", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/scalar/", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Handle("/scalar/*", http.StripPrefix("/scalar", http.FileServer(http.FS(docs.GetScalarUI()))))
	r.Handle("/swagger/*", http.StripPrefix("/swagger", http.FileServer(http.FS(docs.GetUI()))))

	securityHandler := handlers.NewSecurityHandler(authService, identityService)
	accessService := distributionaccess.NewService(metadataLifecycleClient, dataRetrieverClient, converterClient)
	contactRequestService := contactrequest.New(metadataLifecycleClient, contactRequestSender)
	converterService := converter.New(converterClient, metadataLifecycleClient)
	resourceService := resource.New(log, metadataLifecycleClient, converterService)
	generatedHandler := handlers.NewOgenHandler(
		identityService,
		membershipService,
		metadataLifecycleClient,
		sharingService,
		accessService,
		contactRequestService,
		resourceService,
		converterService,
		log,
	)
	generatedServer, err := gatewayapi.NewServer(
		generatedHandler,
		securityHandler,
		gatewayapi.WithPathPrefix("/v1"),
		gatewayapi.WithErrorHandler(handlers.NewErrorHandler(log)),
	)
	if err != nil {
		panic(fmt.Sprintf("create generated gateway server: %v", err))
	}

	r.Mount("/v1", handlers.RequestTimeout(generatedServer, requestTimeout))

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
