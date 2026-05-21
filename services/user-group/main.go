package main

import (
	"log"
	"net/http"
	"os"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/server"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/storage"
)

func main() {
	addr := os.Getenv("USER_GROUP_LISTEN_ADDR")
	if addr == "" {
		addr = ":8090"
	}

	db, err := storage.OpenPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()

	handler := server.NewRouter(
		repository.NewUserPostgresRepository(db),
		repository.NewGroupPostgresRepository(db),
	)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
