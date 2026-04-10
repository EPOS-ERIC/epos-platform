package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Port    string        `mapstructure:"port" yaml:"port"`
	Timeout time.Duration `mapstructure:"request_timeout" yaml:"request_timeout"`
}

// RunServer starts the HTTP server with the given configuration.
func RunServer(cfg Config) {
	defer func() {
		_ = logging.Close()
	}()

	log := logging.Get("server")
	log.Info("initializing application")

	log.Info("configuration loaded", "port", cfg.Port, "dbPath", cfg.DBPath, "dbName", cfg.DBName, "cacheTTL", cfg.CacheTTL)

	db, err := db.NewSqliteDB(cfg.DBPath, cfg.DBName)
	if err != nil {
		log.Error("failed to initialize database", "error", err)
		panic(err)
	}
	defer func() {
		_ = db.Close()
	}()

	log.Debug("initializing services")
	movieService := services.NewMovieService(db, tmdbClient, cfg.CacheTTL)
	listService := services.NewListService(db, movieService)
	watchedService := services.NewWatchedService(db, listService, movieService)
	authService := services.NewAuthService(db, listService, cfg.SessionExpiry, cfg.HTTPS, cfg.AdminDefaultPassword)

	router := routes.NewRouter(db, movieService, watchedService, listService, authService)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	}

	ticker := time.NewTicker(cfg.SessionExpiry)
	done := make(chan bool)

	// clean expired sessions
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				err := authService.CleanupExpiredSessions(context.Background())
				if err != nil {
					log.Error("failed to cleanup expired sessions", "error", err)
					// Continue to next tick instead of returning
				} else {
					log.Debug("successfully cleaned up expired sessions")
				}
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed to start", "error", err)
			panic(err)
		}
	}()

	sig := <-sigChan
	log.Info("received signal, shutting down gracefully", "signal", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("server shutdown complete")
}
