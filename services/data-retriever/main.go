package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/server"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/storage"
)

func main() {
	addr := getenv("DATA_RETRIEVER_LISTEN_ADDR", ":8100")

	db, err := storage.OpenPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()

	migrationsDir := getenv("DATA_RETRIEVER_MIGRATIONS_DIR", "migrations")
	if err := storage.ApplyMigrations(db, migrationsDir); err != nil {
		log.Fatal(err)
	}

	timeout, err := time.ParseDuration(getenv("DATA_RETRIEVER_HTTP_TIMEOUT", "30s"))
	if err != nil {
		log.Fatal(err)
	}

	maxBodyBytes := int64(1 << 20)
	handler := server.NewRouter(
		repository.NewRetrievalPostgresRepository(db),
		retriever.NewClient(timeout, maxBodyBytes),
	)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
