package main

import (
	"log"
	"net/http"
	"os"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/server"
)

func main() {
	addr := os.Getenv("USER_GROUP_LISTEN_ADDR")
	if addr == "" {
		addr = ":8090"
	}

	handler := server.NewRouter()
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
