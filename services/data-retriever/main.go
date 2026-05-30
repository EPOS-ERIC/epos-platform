package main

import (
	"net/http"
	"os"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/server"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/storage"
)

func main() {
	log := logging.Get("data-retriever")
	defer func() {
		if err := logging.Close(); err != nil {
			log.Error("failed to close logging", "error", err)
		}
	}()

	addr := getenv("DATA_RETRIEVER_LISTEN_ADDR", ":8100")

	db, err := storage.OpenPostgres()
	if err != nil {
		log.Error("failed to open postgres", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close postgres", "error", err)
		}
	}()

	migrationsDir := getenv("DATA_RETRIEVER_MIGRATIONS_DIR", "migrations")
	if err := storage.ApplyMigrations(db, migrationsDir); err != nil {
		log.Error("failed to apply migrations", "migrations_dir", migrationsDir, "error", err)
		os.Exit(1)
	}

	timeout, err := time.ParseDuration(getenv("DATA_RETRIEVER_HTTP_TIMEOUT", "30s"))
	if err != nil {
		log.Error("failed to parse timeout", "error", err)
		os.Exit(1)
	}

	maxBodyBytes := int64(1 << 20)
	handler := server.NewRouter(
		repository.NewRetrievalPostgresRepository(db),
		retriever.NewClient(timeout, maxBodyBytes),
		db,
	)
	printBanner(addr)
	log.Info("starting http server", "addr", addr, "timeout", timeout.String())

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func printBanner(addr string) {
	logoHollow := `⣠⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣤⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⣤⣶⣶⣶⣶⣶⣤⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣠⣤⣤⣶⣶⣦⣤⣤⣀⡀⠀⠀⠀
⣿⣿⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⣿⣿⠛⠛⠛⠛⠛⠛⠛⠛⠿⠿⣿⣿⣷⣄⠀⠀⠀⣀⣴⣿⣿⡿⠿⠛⠛⠛⠛⠛⠻⢿⣿⣿⣦⣄⠀⠀⠀⣠⣾⣿⣿⠿⠟⠛⠛⠛⠿⠿⣿⣿⣷⣤⡀
⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⢻⣿⣷⣀⣾⣿⡿⠛⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠙⢿⣿⣷⣄⣾⣿⡟⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⣻⣿⣷
⣿⣿⠀⠀⠀⠀⢠⣤⣤⣤⣤⣤⣤⣤⣤⣿⣿⠀⠀⠀⠀⢀⣴⣶⣷⣶⣄⠀⠀⠀⢻⣿⣿⣿⠏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣿⣿⣿⡟⠀⠀⠀⠀⣠⣤⣤⣤⣄⡀⣠⣾⣿⡿⠁
⣿⣿⠀⠀⠀⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⠀⠀⠀⢼⣿⣟⢙⣿⣿⠀⠀⠀⠘⣿⣿⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⣿⣿⡇⠀⠀⠀⢸⣿⣿⣿⣿⣿⣿⣿⡿⠋⠀⠀
⣿⣿⠀⠀⠀⠀⠈⠉⠉⠉⠉⠉⠉⢹⣿⣿⣿⠀⠀⠀⠀⠘⢿⣿⣿⣿⠟⠀⠀⠀⢸⣿⡏⠀⠀⠀⠀⠀⠀⠀⣠⣶⣶⣶⣤⠀⠀⠀⠀⠀⠀⠀⢸⣿⣇⠀⠀⠀⠀⠈⠉⠙⠛⠛⠿⢿⣿⣷⣄⠀
⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣰⣿⣿⡇⠀⠀⠀⠀⠀⠀⠰⣿⣿⣋⣿⣿⡇⠀⠀⠀⠀⠀⠀⢸⣿⣿⣦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣿⣆
⣿⣿⠀⠀⠀⠀⢰⣶⣶⣶⣶⣶⣶⣾⣿⣿⣿⠀⠀⠀⠀⣀⣀⣀⣀⣠⣤⣴⣿⣿⣿⣿⣇⠀⠀⠀⠀⠀⠀⠀⠻⣿⣿⣿⠟⠀⠀⠀⠀⠀⠀⠀⢸⣿⣿⣿⣿⣿⣷⣶⣶⣦⣤⡀⠀⠀⠀⢸⣿⣿
⣿⣿⠀⠀⠀⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⠀⠀⠀⣿⣿⡿⠿⠿⠿⠟⠛⠁⠸⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⡀⠀⠀⠀⠀⠀⠀⢀⣿⣿⣿⣿⢿⣿⣿⣿⣿⣿⣿⡇⠀⠀⠀⢠⣿⣿
⣿⣿⠀⠀⠀⠀⠈⠉⠉⠉⠉⠉⠉⠉⠉⣿⣿⠀⠀⠀⠀⣿⣿⡇⠀⠀⠀⠀⠀⠀⠀⠹⣿⣿⣄⠀⠀⠀⠀⠀⢸⣿⣿⣿⣇⠀⠀⠀⠀⠀⢠⣾⣿⣿⠟⠁⠀⠀⠉⠉⠉⠉⠁⠀⠀⠀⠀⣼⣿⡟
⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⠀⠀⠀⠀⣿⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣿⣷⣄⠀⠀⢀⣿⣿⠟⣿⣿⡄⠀⠀⣠⣴⣿⣿⣿⣷⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣠⣾⣿⡟⠁
⣿⣿⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣿⣿⣶⣶⣶⣶⣿⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣿⣿⣶⣾⣿⡿⠀⢻⣿⣷⣶⣿⣿⡿⠋⠁⠉⠻⢿⣿⣿⣷⣶⣶⣶⣶⣾⣿⣿⣿⠟⠋⠀⠀
⠈⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠙⠛⠛⠋⠀⠀⠀⠙⠛⠛⠋⠁⠀⠀⠀⠀⠀⠀⠀⠉⠉⠛⠛⠛⠛⠋⠉⠁⠀⠀⠀⠀⠀`

	println()
	println(logoHollow)
	println()
	println("EPOS Data Retriever Service")
	println("PostgreSQL | HTTP API")
	println("Copyright (c) 2026 EPOS ERIC")
	println("https://www.epos-eu.org/")
	println("Server      http://localhost" + addr)
	println("------------------------------------------------------------")
}
