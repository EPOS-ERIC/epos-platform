package main

import (
	"net/http"
	"os"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/server"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/storage"
)

func main() {
	log := logging.Get("user-group")
	defer func() {
		if err := logging.Close(); err != nil {
			log.Error("failed to close logging", "error", err)
		}
	}()

	addr := os.Getenv("USER_GROUP_LISTEN_ADDR")
	if addr == "" {
		addr = ":8090"
	}

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

	migrationsDir := os.Getenv("USER_GROUP_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	if err := storage.ApplyMigrations(db, migrationsDir); err != nil {
		log.Error("failed to apply migrations", "migrations_dir", migrationsDir, "error", err)
		os.Exit(1)
	}

	handler := server.NewRouter(
		repository.NewUserPostgresRepository(db),
		repository.NewGroupPostgresRepository(db),
		repository.NewMembershipPostgresRepository(db),
		repository.NewFriendshipPostgresRepository(db),
		db,
	)
	log.Info("starting http server", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}
